// Package frontend records simulations into browser-friendly snapshots and
// serves the packing visualizer.
package frontend

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"

	"packing_simulator/backend"
	"packing_simulator/backend/evaluator"
	"packing_simulator/backend/policy"
)

const DefaultFrameDelayMS = 250

type SimulationSpec struct {
	ID                    string
	Workload              string
	Config                backend.SimulationConfig
	Iterations            int
	PolicyName            string
	ContainerSelectorName string
	ContainerSelectorK    int
	EvaluationTypes       []evaluator.EvaluationType
}

type SimulationRecording struct {
	ID                string               `json:"id"`
	Workload          string               `json:"workload"`
	Policy            string               `json:"policy"`
	ContainerSelector string               `json:"containerSelector"`
	Seed              int64                `json:"seed"`
	QueueLimit        int                  `json:"queueLimit"`
	FrameDelayMS      int                  `json:"frameDelayMs"`
	Containers        []ContainerRecording `json:"containers"`
	Frames            []SimulationFrame    `json:"frames"`
}

// ContainerRecording stores immutable container metadata and each box once.
// CreatedFrame and each box's FrameIndex are zero-based recording-frame indexes;
// frame zero is the initial state before any simulation timestamp is processed.
type ContainerRecording struct {
	ID           int           `json:"id"`
	Width        int           `json:"width"`
	Height       int           `json:"height"`
	CreatedFrame int           `json:"createdFrame"`
	Boxes        []RecordedBox `json:"boxes"`
}

type RecordedBox struct {
	FrameIndex int `json:"frameIndex"`
	ID         int `json:"id"`
	X          int `json:"x"`
	Y          int `json:"y"`
	Width      int `json:"width"`
	Height     int `json:"height"`
}

type SimulationFrame struct {
	Timestamp   *int              `json:"timestamp"`
	QueueCount  int               `json:"queueCount"`
	Stats       SimulationStats   `json:"stats"`
	Evaluations []EvaluationValue `json:"evaluations"`
}

type SimulationStats struct {
	Iterations   int  `json:"iterations"`
	Generated    int  `json:"generated"`
	Placed       int  `json:"placed"`
	Rotated      int  `json:"rotated"`
	Rejected     int  `json:"rejected"`
	Batches      int  `json:"batches"`
	StoppedEarly bool `json:"stoppedEarly"`
}

type placedBox struct {
	ID     int `json:"id"`
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type recordingBuilder struct {
	recording        *SimulationRecording
	evaluationTypes  []evaluator.EvaluationType
	containerIndexes map[int]int
	seenBoxes        map[[2]int]struct{}
}

type EvaluationValue struct {
	Name   string  `json:"name"`
	Value  float64 `json:"value"`
	Format string  `json:"format"`
}

// RecordSimulation runs one simulation and captures an initial state followed
// by one immutable frame for every processed timestamp.
func RecordSimulation(spec SimulationSpec) (SimulationRecording, error) {
	if spec.Iterations < 0 {
		return SimulationRecording{}, fmt.Errorf("iterations cannot be negative")
	}

	engine, err := backend.NewSimulationEngine(spec.Config)
	if err != nil {
		return SimulationRecording{}, fmt.Errorf("create simulation engine: %w", err)
	}

	placementPolicy, err := policy.NewPlacementPolicy(spec.PolicyName)
	if err != nil {
		return SimulationRecording{}, err
	}
	containerSelector, err := policy.NewContainerSelector(spec.ContainerSelectorName, spec.ContainerSelectorK)
	if err != nil {
		return SimulationRecording{}, fmt.Errorf("create container selector: %w", err)
	}
	id := spec.ID
	if id == "" {
		id = "simulation-1"
	}

	recording := SimulationRecording{
		ID:                id,
		Workload:          spec.Workload,
		Policy:            placementPolicy.Name(),
		ContainerSelector: containerSelector.Name(),
		Seed:              spec.Config.Seed,
		QueueLimit:        engine.World().Queue.Limit,
		FrameDelayMS:      DefaultFrameDelayMS,
		Containers:        make([]ContainerRecording, 0, 1),
		Frames:            make([]SimulationFrame, 0, spec.Iterations+1),
	}
	evaluationTypes := append([]evaluator.EvaluationType(nil), spec.EvaluationTypes...)
	if len(evaluationTypes) == 0 {
		evaluationTypes = evaluator.AllEvaluationTypes()
	}
	builder := recordingBuilder{
		recording:        &recording,
		evaluationTypes:  evaluationTypes,
		containerIndexes: make(map[int]int),
		seenBoxes:        make(map[[2]int]struct{}),
	}

	builder.captureFrame(engine, nil, backend.SimulationResult{})
	_, err = engine.RunWithProgressObserver(
		containerSelector,
		placementPolicy,
		spec.Iterations,
		func(progress backend.SimulationProgress, _ *backend.World) error {
			timestamp := progress.Timestamp
			builder.captureFrame(engine, &timestamp, progress.Result)
			return nil
		},
	)
	if err != nil {
		return SimulationRecording{}, fmt.Errorf("run simulation: %w", err)
	}

	return recording, nil
}

// RecordSimulations records independent simulations concurrently while
// returning results in spec order. A worker count of zero uses GOMAXPROCS.
func RecordSimulations(specs []SimulationSpec, workers int) ([]SimulationRecording, error) {
	return recordSimulations(specs, workers, RecordSimulation)
}

func recordSimulations(
	specs []SimulationSpec,
	workers int,
	record func(SimulationSpec) (SimulationRecording, error),
) ([]SimulationRecording, error) {
	if workers < 0 {
		return nil, errors.New("workers cannot be negative")
	}
	if len(specs) == 0 {
		return []SimulationRecording{}, nil
	}
	if workers == 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if workers > len(specs) {
		workers = len(specs)
	}

	type outcome struct {
		index     int
		recording SimulationRecording
		err       error
	}
	jobs := make(chan int)
	outcomes := make(chan outcome, len(specs))
	var workersDone sync.WaitGroup
	workersDone.Add(workers)
	for range workers {
		go func() {
			defer workersDone.Done()
			for index := range jobs {
				recording, err := record(specs[index])
				if err != nil {
					err = fmt.Errorf("record simulation %q: %w", specs[index].ID, err)
				}
				outcomes <- outcome{index: index, recording: recording, err: err}
			}
		}()
	}
	go func() {
		for index := range specs {
			jobs <- index
		}
		close(jobs)
		workersDone.Wait()
		close(outcomes)
	}()

	recordings := make([]SimulationRecording, len(specs))
	var recordingErrors []error
	for outcome := range outcomes {
		if outcome.err != nil {
			recordingErrors = append(recordingErrors, outcome.err)
			continue
		}
		recordings[outcome.index] = outcome.recording
	}
	if len(recordingErrors) > 0 {
		return nil, errors.Join(recordingErrors...)
	}
	return recordings, nil
}

func (builder *recordingBuilder) captureFrame(engine *backend.SimulationEngine, timestamp *int, result backend.SimulationResult) {
	world := engine.World()
	frameIndex := len(builder.recording.Frames)
	containers := append([]*backend.Container(nil), world.Containers...)
	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Id() < containers[j].Id()
	})
	for _, container := range containers {
		containerIndex, exists := builder.containerIndexes[container.Id()]
		if !exists {
			containerIndex = len(builder.recording.Containers)
			builder.containerIndexes[container.Id()] = containerIndex
			builder.recording.Containers = append(builder.recording.Containers, ContainerRecording{
				ID:           container.Id(),
				Width:        container.Width(),
				Height:       container.Height(),
				CreatedFrame: frameIndex,
				Boxes:        make([]RecordedBox, 0),
			})
		}

		for _, box := range placedBoxes(container) {
			key := [2]int{container.Id(), box.ID}
			if _, exists := builder.seenBoxes[key]; exists {
				continue
			}
			builder.seenBoxes[key] = struct{}{}
			recorded := RecordedBox{
				FrameIndex: frameIndex,
				ID:         box.ID,
				X:          box.X,
				Y:          box.Y,
				Width:      box.Width,
				Height:     box.Height,
			}
			builder.recording.Containers[containerIndex].Boxes = append(builder.recording.Containers[containerIndex].Boxes, recorded)
		}
	}

	builder.recording.Frames = append(builder.recording.Frames, SimulationFrame{
		Timestamp:   timestamp,
		QueueCount:  len(world.Queue.Items),
		Stats:       statsFromResult(result),
		Evaluations: evaluationValues(engine, builder.evaluationTypes),
	})
}

func statsFromResult(result backend.SimulationResult) SimulationStats {
	return SimulationStats{
		Iterations:   result.Iterations,
		Generated:    result.Generated,
		Placed:       result.Placed,
		Rotated:      result.Rotated,
		Rejected:     result.Rejected,
		Batches:      result.Batches,
		StoppedEarly: result.StoppedEarly,
	}
}

type boxBounds struct {
	minX int
	maxX int
	minY int
	maxY int
}

func placedBoxes(container *backend.Container) []placedBox {
	boundsByID := make(map[int]boxBounds)
	for y := 0; y < container.Height(); y++ {
		for x := 0; x < container.Width(); x++ {
			id, err := container.Cell(x, y)
			if err != nil || id == backend.EmptyCell {
				continue
			}

			bounds, exists := boundsByID[id]
			if !exists {
				boundsByID[id] = boxBounds{minX: x, maxX: x, minY: y, maxY: y}
				continue
			}
			if x < bounds.minX {
				bounds.minX = x
			}
			if x > bounds.maxX {
				bounds.maxX = x
			}
			if y < bounds.minY {
				bounds.minY = y
			}
			if y > bounds.maxY {
				bounds.maxY = y
			}
			boundsByID[id] = bounds
		}
	}

	ids := make([]int, 0, len(boundsByID))
	for id := range boundsByID {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	boxes := make([]placedBox, 0, len(ids))
	for _, id := range ids {
		bounds := boundsByID[id]
		boxes = append(boxes, placedBox{
			ID:     id,
			X:      bounds.minX,
			Y:      bounds.minY,
			Width:  bounds.maxX - bounds.minX + 1,
			Height: bounds.maxY - bounds.minY + 1,
		})
	}
	return boxes
}

func evaluationValues(engine *backend.SimulationEngine, evaluationTypes []evaluator.EvaluationType) []EvaluationValue {
	evaluations := make([]EvaluationValue, 0, len(evaluationTypes))
	for _, evaluationType := range evaluationTypes {
		format := "decimal"
		if evaluationType == evaluator.ContainerUtilization || evaluationType == evaluator.FutureFitProbabilityMetric {
			format = "percent"
		}
		evaluations = append(evaluations, EvaluationValue{
			Name:   evaluationType.String(),
			Value:  evaluator.EvaluateSimulation(engine, evaluationType),
			Format: format,
		})
	}
	return evaluations
}

// Package frontend records simulations into browser-friendly snapshots and
// serves the packing visualizer.
package frontend

import (
	"fmt"
	"sort"

	"packing_simulator/backend"
)

const DefaultFrameDelayMS = 250

type SimulationSpec struct {
	ID         string
	Config     backend.SimulationConfig
	Iterations int
	PolicyName string
}

type SimulationRecording struct {
	ID           string            `json:"id"`
	Policy       string            `json:"policy"`
	Seed         int64             `json:"seed"`
	Width        int               `json:"width"`
	Height       int               `json:"height"`
	QueueLimit   int               `json:"queueLimit"`
	FrameDelayMS int               `json:"frameDelayMs"`
	Frames       []SimulationFrame `json:"frames"`
}

type SimulationFrame struct {
	Timestamp   *int              `json:"timestamp"`
	QueueCount  int               `json:"queueCount"`
	Stats       SimulationStats   `json:"stats"`
	Boxes       []PlacedBox       `json:"boxes"`
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

type PlacedBox struct {
	ID     int `json:"id"`
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
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

	policy, err := backend.NewPolicy(spec.PolicyName)
	if err != nil {
		return SimulationRecording{}, err
	}

	id := spec.ID
	if id == "" {
		id = "simulation-1"
	}

	recording := SimulationRecording{
		ID:           id,
		Policy:       policy.Name(),
		Seed:         spec.Config.Seed,
		Width:        engine.World().Container.Width(),
		Height:       engine.World().Container.Height(),
		QueueLimit:   engine.World().Queue.Limit,
		FrameDelayMS: DefaultFrameDelayMS,
		Frames:       make([]SimulationFrame, 0, spec.Iterations+1),
	}

	recording.Frames = append(recording.Frames, captureFrame(engine, nil, backend.SimulationResult{}))
	_, err = engine.RunWithProgressObserver(
		policy,
		spec.Iterations,
		func(progress backend.SimulationProgress, _ *backend.World) error {
			timestamp := progress.Timestamp
			recording.Frames = append(recording.Frames, captureFrame(engine, &timestamp, progress.Result))
			return nil
		},
	)
	if err != nil {
		return SimulationRecording{}, fmt.Errorf("run simulation: %w", err)
	}

	return recording, nil
}

func captureFrame(engine *backend.SimulationEngine, timestamp *int, result backend.SimulationResult) SimulationFrame {
	world := engine.World()
	return SimulationFrame{
		Timestamp:   timestamp,
		QueueCount:  len(world.Queue.Items),
		Stats:       statsFromResult(result),
		Boxes:       placedBoxes(&world.Container),
		Evaluations: evaluationValues(engine),
	}
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

func placedBoxes(container *backend.Container) []PlacedBox {
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

	boxes := make([]PlacedBox, 0, len(ids))
	for _, id := range ids {
		bounds := boundsByID[id]
		boxes = append(boxes, PlacedBox{
			ID:     id,
			X:      bounds.minX,
			Y:      bounds.minY,
			Width:  bounds.maxX - bounds.minX + 1,
			Height: bounds.maxY - bounds.minY + 1,
		})
	}
	return boxes
}

func evaluationValues(engine *backend.SimulationEngine) []EvaluationValue {
	evaluations := make([]EvaluationValue, 0, len(backend.AllEvaluationTypes()))
	for _, evaluationType := range backend.AllEvaluationTypes() {
		format := "decimal"
		if evaluationType == backend.ContainerUtilization || evaluationType == backend.FutureFitProbabilityMetric {
			format = "percent"
		}
		evaluations = append(evaluations, EvaluationValue{
			Name:   evaluationType.String(),
			Value:  backend.EvaluateSimulation(engine, evaluationType),
			Format: format,
		})
	}
	return evaluations
}

package frontend

import (
	"math"
	"testing"
	"time"

	"packing_simulator/backend"
	"packing_simulator/backend/evaluator"
	"packing_simulator/backend/policy"
)

func TestRecordSimulationCapturesInitialAndTimestampFrames(t *testing.T) {
	recording, err := RecordSimulation(SimulationSpec{
		ID: "test-simulation",
		Config: backend.SimulationConfig{
			ContainerHeight: 2,
			ContainerWidth:  3,
			QueueSize:       1,
			MinBoxHeight:    1,
			MaxBoxHeight:    1,
			MinBoxWidth:     1,
			MaxBoxWidth:     1,
			Seed:            7,
		},
		Iterations: 2,
		PolicyName: policy.BottomLeftPolicyName,
	})
	if err != nil {
		t.Fatal(err)
	}

	if recording.ID != "test-simulation" || recording.QueueLimit != 1 {
		t.Errorf("recording metadata = %+v", recording)
	}
	if len(recording.Containers) != 1 || recording.Containers[0].Width != 3 || recording.Containers[0].Height != 2 || recording.Containers[0].CreatedFrame != 0 {
		t.Errorf("recorded containers = %+v", recording.Containers)
	}
	if recording.ContainerSelector != policy.ContainerSelectorFirstFitName {
		t.Errorf("recording container selector = %q; want %q", recording.ContainerSelector, policy.ContainerSelectorFirstFitName)
	}
	if len(recording.Frames) != 3 {
		t.Fatalf("frame count = %d; want initial plus 2 timestamps", len(recording.Frames))
	}
	initial := recording.Frames[0]
	if initial.Timestamp != nil || initial.Stats != (SimulationStats{}) {
		t.Errorf("initial frame = %+v", initial)
	}
	if len(initial.Evaluations) != len(evaluator.AllEvaluationTypes()) {
		t.Errorf("initial evaluations = %d; want %d", len(initial.Evaluations), len(evaluator.AllEvaluationTypes()))
	}

	final := recording.Frames[2]
	if final.Timestamp == nil || *final.Timestamp != 1 {
		t.Errorf("final timestamp = %v; want 1", final.Timestamp)
	}
	if final.Stats.Generated != 2 || final.Stats.Placed != 2 || final.Stats.Batches != 2 {
		t.Errorf("final stats = %+v", final.Stats)
	}
	boxes := recording.Containers[0].Boxes
	if len(boxes) != 2 || boxes[0].ID != 1 || boxes[0].FrameIndex != 1 || boxes[1].ID != 2 || boxes[1].FrameIndex != 2 {
		t.Errorf("recorded boxes = %+v", boxes)
	}
	if got := evaluationByName(t, final, evaluator.ContainerUtilization.String()); math.Abs(got-2.0/6.0) > 1e-9 {
		t.Errorf("utilization = %f; want %f", got, 2.0/6.0)
	}
}

func TestRecordSimulationCapturesMultipleContainersAsPlacementEvents(t *testing.T) {
	recording, err := RecordSimulation(SimulationSpec{
		ID:       "multi",
		Workload: "tiny-bins",
		Config: backend.SimulationConfig{
			ContainerHeight: 1,
			ContainerWidth:  1,
			MaxContainers:   -1,
			QueueSize:       1,
			MinBoxHeight:    1,
			MaxBoxHeight:    1,
			MinBoxWidth:     1,
			MaxBoxWidth:     1,
			Seed:            5,
		},
		Iterations:            3,
		PolicyName:            policy.BottomLeftPolicyName,
		ContainerSelectorName: policy.ContainerSelectorNextFitName,
		EvaluationTypes:       []evaluator.EvaluationType{evaluator.BinCount, evaluator.ContainerUtilization},
	})
	if err != nil {
		t.Fatal(err)
	}
	if recording.Workload != "tiny-bins" || len(recording.Containers) != 3 {
		t.Fatalf("recording = %+v; want three containers", recording)
	}
	for i, container := range recording.Containers {
		wantFrame := i + 1
		if i == 0 {
			wantFrame = 0
		}
		if container.ID != i+1 || container.CreatedFrame != wantFrame || len(container.Boxes) != 1 {
			t.Errorf("container %d = %+v", i, container)
			continue
		}
		if container.Boxes[0].ID != i+1 || container.Boxes[0].FrameIndex != i+1 {
			t.Errorf("container %d box = %+v", i, container.Boxes[0])
		}
	}
	for i, frame := range recording.Frames {
		if len(frame.Evaluations) != 2 || frame.Evaluations[0].Name != evaluator.BinCount.String() || frame.Evaluations[1].Name != evaluator.ContainerUtilization.String() {
			t.Errorf("frame %d evaluations = %+v", i, frame.Evaluations)
		}
	}
}

func TestRecordSimulationsPreservesSpecOrder(t *testing.T) {
	specs := []SimulationSpec{testRecordingSpec("first", 1), testRecordingSpec("second", 2)}
	recordings, err := RecordSimulations(specs, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(recordings) != 2 || recordings[0].ID != "first" || recordings[1].ID != "second" {
		t.Errorf("RecordSimulations() = %+v", recordings)
	}
	if _, err := RecordSimulations(specs, -1); err == nil {
		t.Fatal("RecordSimulations() accepted a negative worker count")
	}
	if recordings, err := RecordSimulations(specs, 0); err != nil || len(recordings) != len(specs) {
		t.Fatalf("RecordSimulations() with automatic workers = %d recordings, %v", len(recordings), err)
	}
}

func TestRecordSimulationsHonorsWorkerLimit(t *testing.T) {
	specs := []SimulationSpec{{ID: "one"}, {ID: "two"}, {ID: "three"}}
	started := make(chan string, len(specs))
	release := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		_, err := recordSimulations(specs, 2, func(spec SimulationSpec) (SimulationRecording, error) {
			started <- spec.ID
			<-release
			return SimulationRecording{ID: spec.ID}, nil
		})
		done <- err
	}()

	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("two workers did not start")
		}
	}
	select {
	case id := <-started:
		t.Fatalf("third recording %q started before a worker was released", id)
	default:
	}

	close(release)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("third recording did not start after a worker was released")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestRecordSimulationZeroIterations(t *testing.T) {
	recording, err := RecordSimulation(SimulationSpec{
		Config: backend.SimulationConfig{
			ContainerHeight: 1,
			ContainerWidth:  1,
			QueueSize:       1,
			MinBoxHeight:    1,
			MaxBoxHeight:    1,
			MinBoxWidth:     1,
			MaxBoxWidth:     1,
		},
		PolicyName: policy.BottomLeftPolicyName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if recording.ID != "simulation-1" || len(recording.Frames) != 1 {
		t.Errorf("recording = %+v; want default ID and one initial frame", recording)
	}
}

func TestRecordSimulationRejectsNegativeIterations(t *testing.T) {
	_, err := RecordSimulation(SimulationSpec{Iterations: -1})
	if err == nil {
		t.Fatal("RecordSimulation() unexpectedly accepted negative iterations")
	}
}

func TestPlacedBoxesReturnsRectanglesInIDOrder(t *testing.T) {
	container, err := backend.NewContainer(4, 5, 1) // id doesn't matter here
	if err != nil {
		t.Fatal(err)
	}
	if err := container.Place(backend.Box{ID: 9, Width: 2, Height: 2}, 3, 0, false); err != nil {
		t.Fatal(err)
	}
	if err := container.Place(backend.Box{ID: 2, Width: 3, Height: 1}, 0, 3, false); err != nil {
		t.Fatal(err)
	}

	got := placedBoxes(container)
	want := []placedBox{
		{ID: 2, X: 0, Y: 3, Width: 3, Height: 1},
		{ID: 9, X: 3, Y: 0, Width: 2, Height: 2},
	}
	if len(got) != len(want) {
		t.Fatalf("placedBoxes() = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("box %d = %+v; want %+v", i, got[i], want[i])
		}
	}
}

func testRecordingSpec(id string, seed int64) SimulationSpec {
	return SimulationSpec{
		ID: id,
		Config: backend.SimulationConfig{
			ContainerHeight: 1,
			ContainerWidth:  1,
			QueueSize:       1,
			MinBoxHeight:    1,
			MaxBoxHeight:    1,
			MinBoxWidth:     1,
			MaxBoxWidth:     1,
			Seed:            seed,
		},
		Iterations: 1,
		PolicyName: policy.BottomLeftPolicyName,
	}
}

func evaluationByName(t *testing.T, frame SimulationFrame, name string) float64 {
	t.Helper()
	for _, evaluation := range frame.Evaluations {
		if evaluation.Name == name {
			return evaluation.Value
		}
	}
	t.Fatalf("evaluation %q not found", name)
	return 0
}

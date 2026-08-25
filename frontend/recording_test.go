package frontend

import (
	"math"
	"testing"

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

	if recording.ID != "test-simulation" || recording.Width != 3 || recording.Height != 2 || recording.QueueLimit != 1 {
		t.Errorf("recording metadata = %+v", recording)
	}
	if len(recording.Frames) != 3 {
		t.Fatalf("frame count = %d; want initial plus 2 timestamps", len(recording.Frames))
	}
	initial := recording.Frames[0]
	if initial.Timestamp != nil || initial.Stats != (SimulationStats{}) || len(initial.Boxes) != 0 {
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
	if len(final.Boxes) != 2 || final.Boxes[0].ID != 1 || final.Boxes[1].ID != 2 {
		t.Errorf("final boxes = %+v", final.Boxes)
	}
	if got := evaluationByName(t, final, evaluator.ContainerUtilization.String()); math.Abs(got-2.0/6.0) > 1e-9 {
		t.Errorf("utilization = %f; want %f", got, 2.0/6.0)
	}
	if len(recording.Frames[1].Boxes) != 1 {
		t.Error("earlier frame changed after subsequent simulation steps")
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
	container, err := backend.NewContainer(4, 5)
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
	want := []PlacedBox{
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

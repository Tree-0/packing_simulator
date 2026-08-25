package backend_test

import (
	"errors"
	"testing"

	"packing_simulator/backend"
	"packing_simulator/backend/policy"
)

func TestRunWithProgressObserverReportsCumulativeResults(t *testing.T) {
	engine := newProgressTestEngine(t, 2, 2)
	var progress []backend.SimulationProgress

	result, err := engine.RunWithProgressObserver(
		newBottomLeftPolicy(t),
		2,
		func(step backend.SimulationProgress, _ *backend.World) error {
			progress = append(progress, step)
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(progress) != 2 {
		t.Fatalf("observer calls = %d; want 2", len(progress))
	}
	if progress[0].Timestamp != 0 || progress[0].Result.Generated != 1 || progress[0].Result.Placed != 1 || progress[0].Result.Batches != 1 {
		t.Errorf("first progress = %+v; want timestamp 0 with one generated and placed box", progress[0])
	}
	if progress[1].Timestamp != 1 || progress[1].Result != result {
		t.Errorf("final progress = %+v; want timestamp 1 and final result %+v", progress[1], result)
	}
}

func TestRunWithProgressObserverReportsEarlyStop(t *testing.T) {
	engine := newProgressTestEngine(t, 1, 1)
	var final backend.SimulationProgress

	result, err := engine.RunWithProgressObserver(newBottomLeftPolicy(t), 3, func(step backend.SimulationProgress, _ *backend.World) error {
		final = step
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if !result.StoppedEarly || !final.Result.StoppedEarly {
		t.Fatalf("stopped early = %v, observer = %v; want both true", result.StoppedEarly, final.Result.StoppedEarly)
	}
	if final.Timestamp != 1 || result.Generated != 2 || result.Placed != 1 || result.Rejected != 1 {
		t.Errorf("early-stop result = %+v at timestamp %d", result, final.Timestamp)
	}
}

func TestRunWithProgressObserverZeroIterations(t *testing.T) {
	engine := newProgressTestEngine(t, 2, 2)
	called := false
	result, err := engine.RunWithProgressObserver(newBottomLeftPolicy(t), 0, func(backend.SimulationProgress, *backend.World) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Error("observer called for a zero-iteration simulation")
	}
	if result != (backend.SimulationResult{}) {
		t.Errorf("result = %+v; want zero value", result)
	}
}

func TestRunWithProgressObserverWrapsErrors(t *testing.T) {
	engine := newProgressTestEngine(t, 2, 2)
	want := errors.New("stop recording")
	_, err := engine.RunWithProgressObserver(newBottomLeftPolicy(t), 1, func(backend.SimulationProgress, *backend.World) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v; want wrapped observer error", err)
	}
}

func newProgressTestEngine(t *testing.T, height, width int) *backend.SimulationEngine {
	t.Helper()
	engine, err := backend.NewSimulationEngine(backend.SimulationConfig{
		ContainerHeight: height,
		ContainerWidth:  width,
		QueueSize:       1,
		MinBoxHeight:    1,
		MaxBoxHeight:    1,
		MinBoxWidth:     1,
		MaxBoxWidth:     1,
		Seed:            1,
	})
	if err != nil {
		t.Fatal(err)
	}
	return engine
}

func newBottomLeftPolicy(t *testing.T) backend.Policy {
	t.Helper()
	p, err := policy.NewPolicy(policy.BottomLeftPolicyName)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

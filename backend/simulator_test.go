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
		policy.ContainerSelectorFirstFit{},
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

// Early stopping functionality was removed

/*
func TestRunWithProgressObserverReportsEarlyStop(t *testing.T) {
	engine := newProgressTestEngine(t, 1, 1)
	var final backend.SimulationProgress

	result, err := engine.RunWithProgressObserver(policy.ContainerSelectorFirstFit{}, newBottomLeftPolicy(t), 3, func(step backend.SimulationProgress, _ *backend.World) error {
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
*/

func TestRunWithProgressObserverZeroIterations(t *testing.T) {
	engine := newProgressTestEngine(t, 2, 2)
	called := false
	result, err := engine.RunWithProgressObserver(policy.ContainerSelectorFirstFit{}, newBottomLeftPolicy(t), 0, func(backend.SimulationProgress, *backend.World) error {
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
	_, err := engine.RunWithProgressObserver(policy.ContainerSelectorFirstFit{}, newBottomLeftPolicy(t), 1, func(backend.SimulationProgress, *backend.World) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v; want wrapped observer error", err)
	}
}

func TestRunAllocatesContainerWhenExistingContainersCannotFit(t *testing.T) {
	engine := newContainerLimitTestEngine(t, 2)

	result, err := engine.Run(policy.ContainerSelectorFirstFit{}, newBottomLeftPolicy(t), 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Placed != 2 || result.Rejected != 0 {
		t.Fatalf("result = %+v; want two placements and no rejections", result)
	}
	if got := len(engine.World().Containers); got != 2 {
		t.Fatalf("container count = %d; want 2", got)
	}
	for index, wantBoxID := range []int{1, 2} {
		container := engine.World().Containers[index]
		if container.Id() != index+1 {
			t.Errorf("container %d ID = %d; want %d", index, container.Id(), index+1)
		}
		cell, err := container.Cell(0, 0)
		if err != nil {
			t.Fatal(err)
		}
		if cell != wantBoxID {
			t.Errorf("container %d Cell(0, 0) = %d; want box %d", index+1, cell, wantBoxID)
		}
	}
}

func TestRunRejectsBoxWhenContainerLimitIsReached(t *testing.T) {
	engine := newContainerLimitTestEngine(t, 1)

	result, err := engine.Run(policy.ContainerSelectorFirstFit{}, newBottomLeftPolicy(t), 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Placed != 1 || result.Rejected != 1 {
		t.Fatalf("result = %+v; want one placement and one rejection", result)
	}
	if got := len(engine.World().Containers); got != 1 {
		t.Errorf("container count = %d; want 1", got)
	}
}

func TestNewSimulationEngineRejectsInvalidContainerLimit(t *testing.T) {
	_, err := backend.NewSimulationEngine(backend.SimulationConfig{
		ContainerHeight: 1,
		ContainerWidth:  1,
		MaxContainers:   -2,
		QueueSize:       1,
		MinBoxHeight:    1,
		MaxBoxHeight:    1,
		MinBoxWidth:     1,
		MaxBoxWidth:     1,
	})
	if err == nil {
		t.Fatal("NewSimulationEngine() accepted a max container value below -1")
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

func newContainerLimitTestEngine(t *testing.T, maxContainers int) *backend.SimulationEngine {
	t.Helper()
	engine, err := backend.NewSimulationEngine(backend.SimulationConfig{
		ContainerHeight: 1,
		ContainerWidth:  1,
		MaxContainers:   maxContainers,
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

func newBottomLeftPolicy(t *testing.T) backend.PlacementPolicy {
	t.Helper()
	p, err := policy.NewPlacementPolicy(policy.BottomLeftPolicyName)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

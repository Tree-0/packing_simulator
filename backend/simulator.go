/*


 */

package backend

import (
	"errors"
	"fmt"
)

type SimulationConfig struct {
	ContainerHeight int
	ContainerWidth  int
	// MaxContainers limits the total number of containers. -1 is unbounded;
	// zero is also treated as unbounded for backwards-compatible zero-value
	// configuration literals.
	MaxContainers    int
	QueueSize        int
	MinBoxHeight     int
	MaxBoxHeight     int
	MinBoxWidth      int
	MaxBoxWidth      int
	Seed             int64
	AllowBoxRotation bool
}

type SimulationResult struct {
	Iterations   int
	Generated    int
	Placed       int
	Rejected     int
	Batches      int
	Rotated      int
	StoppedEarly bool
}

// StepObserver is called after each timestamp has been processed.
type StepObserver func(timestamp int, world *World) error

// SimulationProgress describes the cumulative result after one timestamp.
type SimulationProgress struct {
	Timestamp int
	Result    SimulationResult
}

// ProgressObserver is called after each timestamp has been processed. The
// supplied World is the engine's live world; observers that retain state must
// copy the data they need before returning.
type ProgressObserver func(progress SimulationProgress, world *World) error

type SimulationEngine struct {
	world           *World
	generator       BoxGenerator
	distribution    UniformBoxDistribution
	containerHeight int
	containerWidth  int
	maxContainers   int
}

func NewSimulationEngine(config SimulationConfig) (*SimulationEngine, error) {
	if config.MaxBoxHeight > config.ContainerHeight || config.MaxBoxWidth > config.ContainerWidth {
		return nil, errors.New("maximum box dimensions cannot exceed container dimensions")
	}
	maxContainers, err := normalizedMaxContainers(config.MaxContainers)
	if err != nil {
		return nil, err
	}

	world, err := NewWorld(config.ContainerHeight, config.ContainerWidth, config.QueueSize)
	if err != nil {
		return nil, err
	}

	distribution := UniformBoxDistribution{
		MinWidth:  config.MinBoxWidth,
		MaxWidth:  config.MaxBoxWidth,
		MinHeight: config.MinBoxHeight,
		MaxHeight: config.MaxBoxHeight,
	}

	generator, err := NewRandomBoxGenerator(
		config.Seed,
		config.MinBoxWidth,
		config.MaxBoxWidth,
		config.MinBoxHeight,
		config.MaxBoxHeight,
		config.AllowBoxRotation,
	)
	if err != nil {
		return nil, err
	}

	return &SimulationEngine{
		world:           world,
		generator:       generator,
		distribution:    distribution,
		containerHeight: config.ContainerHeight,
		containerWidth:  config.ContainerWidth,
		maxContainers:   maxContainers,
	}, nil
}

func normalizedMaxContainers(maxContainers int) (int, error) {
	if maxContainers == 0 {
		return -1, nil
	}
	if maxContainers < -1 {
		return 0, errors.New("max containers must be -1 (unbounded) or positive")
	}
	return maxContainers, nil
}

func (eng *SimulationEngine) World() *World {
	return eng.world
}

func (eng *SimulationEngine) UniformBoxDistribution() UniformBoxDistribution {
	return eng.distribution
}

// Run generates one box per iteration and processes boxes whenever the queue
// reaches its configured limit. A final partial queue is processed as a batch.
func (eng *SimulationEngine) Run(cs ContainerSelector, p PlacementPolicy, iterations int) (SimulationResult, error) {
	return eng.run(cs, p, iterations, nil)
}

// RunWithObserver runs the simulation and calls observer after each timestamp.
// The observer may be nil when no per-step output is needed.
func (eng *SimulationEngine) RunWithObserver(
	cs ContainerSelector,
	p PlacementPolicy,
	iterations int,
	observer StepObserver,
) (SimulationResult, error) {
	if observer == nil {
		return eng.run(cs, p, iterations, nil)
	}

	return eng.run(cs, p, iterations, func(progress SimulationProgress, world *World) error {
		return observer(progress.Timestamp, world)
	})
}

// RunWithProgressObserver runs the simulation and reports the cumulative
// result and world state after every timestamp.
func (eng *SimulationEngine) RunWithProgressObserver(
	cs ContainerSelector,
	p PlacementPolicy,
	iterations int,
	observer ProgressObserver,
) (SimulationResult, error) {
	return eng.run(cs, p, iterations, observer)
}

func (eng *SimulationEngine) run(
	cs ContainerSelector,
	p PlacementPolicy,
	iterations int,
	observer ProgressObserver,
) (SimulationResult, error) {
	result := SimulationResult{}
	if eng == nil || eng.world == nil || eng.generator == nil {
		return result, errors.New("simulation engine is not initialized")
	}
	if cs == nil {
		return result, errors.New("container selector is required")
	}
	if p == nil {
		return result, errors.New("policy is required")
	}
	if iterations < 0 {
		return result, errors.New("iterations cannot be negative")
	}

	for t := 0; t < iterations; t++ {
		queue := &eng.world.Queue
		if !queue.Enqueue(eng.generator.Next(t)) {
			return result, errors.New("queue unexpectedly reached its limit")
		}
		result.Generated++
		result.Iterations = t + 1

		// stopped := false
		if queue.Full() || t == iterations-1 {
			batch := queue.Drain()
			placed, rotated, err := eng.processBatch(t, cs, p, batch)
			if err != nil {
				return result, err
			}
			result.Batches++
			result.Placed += placed
			result.Rotated += rotated
			result.Rejected += len(batch) - placed

			// early stopping if nothing can be placed (see below comment)
			// if placed == 0 {
			// 	result.StoppedEarly = true
			// 	stopped = true
			// }
		}

		if observer != nil {
			progress := SimulationProgress{Timestamp: t, Result: result}
			if err := observer(progress, eng.world); err != nil {
				return result, fmt.Errorf("observing timestamp %d: %w", t, err)
			}
		}

		// We used to stop if no boxes in current queue can be placed.
		// Now, for the sake of running experiments where every (simulation, policy)
		// pair receives the same full set of boxes, we mark them as rejected and
		// continue.

		// if stopped {
		// 	return result, nil
		// }
	}

	return result, nil
}

func (eng *SimulationEngine) processBatch(
	t int,
	cs ContainerSelector,
	p PlacementPolicy,
	batch []QueuedBox,
) (int, int, error) {
	placed := 0
	rotated := 0
	if cs == nil {
		return placed, rotated, errors.New("container selector is required")
	}

	batch = p.OrderBatch(batch)

	for i, queued := range batch {
		remaining := batch[i:]

		selectionContext := ContainerSelectionContext{
			Timestamp:  t,
			Containers: eng.World().ContainerSnapshots(),
			Batch:      append([]QueuedBox(nil), remaining...),
		}
		orderedContainerIDs := cs.Rank(selectionContext, queued.Box)

		boxPlaced := false
		for _, containerID := range orderedContainerIDs {
			if containerID < 1 {
				return placed, rotated, fmt.Errorf("container selector returned invalid container ID %d", containerID)
			}

			container, err := eng.World().ContainerById(containerID)
			if err != nil {
				return placed, rotated, fmt.Errorf("container selector returned unknown container ID %d: %w", containerID, err)
			}

			found, wasRotated, err := eng.placeInContainer(t, p, container, remaining, queued)
			if err != nil {
				return placed, rotated, err
			}
			if found {
				placed++
				if wasRotated {
					rotated++
				}
				boxPlaced = true
				break
			}
		}

		// If this box could not be placed in any existing container
		// AND we can't put it in a new one, skip to the next box in the batch
		if boxPlaced || !eng.canAllocateContainer() {
			continue
		}

		// must make a new container to place the box
		container, err := eng.World().NewContainer(eng.containerHeight, eng.containerWidth)
		if err != nil {
			return placed, rotated, fmt.Errorf("allocate a new container: %w", err)
		}

		found, wasRotated, err := eng.placeInContainer(t, p, container, remaining, queued)
		if err != nil {
			return placed, rotated, err
		}
		if !found {
			return placed, rotated, fmt.Errorf("policy %q found no placement for box %d in a new empty container", p.Name(), queued.Box.ID)
		}

		placed++
		if wasRotated {
			rotated++
		}
	}

	return placed, rotated, nil
}

func (eng *SimulationEngine) canAllocateContainer() bool {
	return eng.maxContainers < 0 || len(eng.world.Containers) < eng.maxContainers
}

// placeInContainer asks the placement policy for a position in container and,
// when found, places the box there. It returns whether a placement was found,
// whether that placement rotated the box, and any policy or placement error.
func (eng *SimulationEngine) placeInContainer(
	t int,
	p PlacementPolicy,
	container *Container,
	remaining []QueuedBox,
	queued QueuedBox,
) (bool, bool, error) {
	placementContext := PlacementContext{
		Timestamp: t,
		Container: container.ContainerSnapshot(),
		Batch:     append([]QueuedBox(nil), remaining...),
	}
	decision, found := p.FindPlacement(placementContext, queued.Box)
	if !found {
		return false, false, nil
	}

	box := queued.Box
	if decision.Rotated {
		var err error
		box, err = queued.Box.TryRotate()
		if err != nil {
			return false, false, fmt.Errorf("policy %q produced a rotated decision for an unrotatable box %d: %w", p.Name(), queued.Box.ID, err)
		}
	}

	if err := container.Place(box, decision.Point.X, decision.Point.Y, decision.Rotated); err != nil {
		return false, false, fmt.Errorf("policy %q produced an invalid placement for box %d: %w", p.Name(), queued.Box.ID, err)
	}

	return true, decision.Rotated, nil
}

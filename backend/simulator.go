/*


 */

package backend

import (
	"errors"
	"fmt"
)

type SimulationConfig struct {
	ContainerHeight  int
	ContainerWidth   int
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
	world        *World
	generator    BoxGenerator
	distribution UniformBoxDistribution
}

func NewSimulationEngine(config SimulationConfig) (*SimulationEngine, error) {
	if config.MaxBoxHeight > config.ContainerHeight || config.MaxBoxWidth > config.ContainerWidth {
		return nil, errors.New("maximum box dimensions cannot exceed container dimensions")
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
		world:        world,
		generator:    generator,
		distribution: distribution,
	}, nil
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

			placementContext := PlacementContext{
				Timestamp: t,
				Container: container.ContainerSnapshot(),
				Batch:     append([]QueuedBox(nil), remaining...),
			}
			decision, found := p.FindPlacement(placementContext, queued.Box)
			if !found {
				continue
			}

			box := queued.Box
			// Rotate the box if the placement demands
			if decision.Rotated {
				box, err = queued.Box.TryRotate()
				if err != nil {
					return placed, rotated, fmt.Errorf("policy %q produced a rotated decision for an unrotatable box %d: %w",
						p.Name(), queued.Box.ID, err)
				}
			}

			if err := container.Place(box, decision.Point.X, decision.Point.Y, decision.Rotated); err != nil {
				return placed, rotated, fmt.Errorf("policy %q produced an invalid placement for box %d: %w", p.Name(), queued.Box.ID, err)
			}

			placed++
			if decision.Rotated {
				rotated++
			}
			boxPlaced = true
			break
		}

		// We couldn't place box into an existing container, so allocate new
		// TODO: verify logic here
		// TODO: add a test to verify a new container gets added when a box cannot fit into any previous containers
		// TODO: abstract the placement context construction and actual placement into a helper?
		// I think there might be some redundancy with the logic in the container loop above, but with
		// error propagation and passing arguments down maybe it wouldn't be much cleaner...
		if !boxPlaced {
			container, err := eng.World().NewContainer(
				eng.World().Containers[0].Height(),
				eng.World().Containers[0].Width(),
			)
			if err != nil {
				return placed, rotated, fmt.Errorf(
					"Unable to allocate new container after being unable to place box into existing container",
				)
			}

			placementContext := PlacementContext{
				Timestamp: t,
				Container: container.ContainerSnapshot(),
				Batch: 	   append([]QueuedBox(nil), remaining...),
			}
			box := queued.Box
			decision, _ := p.FindPlacement(placementContext, box) // ignoring found bc container is newly allocated and empty
			if err := container.Place(box, decision.Point.X, decision.Point.Y, decision.Rotated); err != nil {
				return placed, rotated, fmt.Errorf("policy %q produced an invalid placement for box %d: %w", p.Name(), queued.Box.ID, err)
			}

			placed++
			if decision.Rotated {
				rotated++
			}
		}
	}

	return placed, rotated, nil
}

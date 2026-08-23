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
func (eng *SimulationEngine) Run(p Policy, iterations int) (SimulationResult, error) {
	return eng.run(p, iterations, nil)
}

// RunWithObserver runs the simulation and calls observer after each timestamp.
// The observer may be nil when no per-step output is needed.
func (eng *SimulationEngine) RunWithObserver(
	p Policy,
	iterations int,
	observer StepObserver,
) (SimulationResult, error) {
	if observer == nil {
		return eng.run(p, iterations, nil)
	}

	return eng.run(p, iterations, func(progress SimulationProgress, world *World) error {
		return observer(progress.Timestamp, world)
	})
}

// RunWithProgressObserver runs the simulation and reports the cumulative
// result and world state after every timestamp.
func (eng *SimulationEngine) RunWithProgressObserver(
	p Policy,
	iterations int,
	observer ProgressObserver,
) (SimulationResult, error) {
	return eng.run(p, iterations, observer)
}

func (eng *SimulationEngine) run(
	p Policy,
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

		stopped := false
		if queue.Full() || t == iterations-1 {
			batch := queue.Drain()
			placed, rotated, err := eng.processBatch(p, batch)
			if err != nil {
				return result, err
			}
			result.Batches++
			result.Placed += placed
			result.Rotated += rotated
			result.Rejected += len(batch) - placed
			if placed == 0 {
				result.StoppedEarly = true
				stopped = true
			}
		}

		if observer != nil {
			progress := SimulationProgress{Timestamp: t, Result: result}
			if err := observer(progress, eng.world); err != nil {
				return result, fmt.Errorf("observing timestamp %d: %w", t, err)
			}
		}

		if stopped {
			return result, nil
		}
	}

	return result, nil
}

func (eng *SimulationEngine) processBatch(p Policy, batch []QueuedBox) (int, int, error) {
	placed := 0
	rotated := 0

	p.OrderBatch(batch)

	for _, queued := range batch {
		decision, found := p.FindPlacement(&eng.world.Container, queued.Box)
		if !found {
			continue
		}

		box := queued.Box

		// place the rotated box instead, if that's what the policy decided
		if decision.Rotated {
			rotated += 1
			var err error
			box, err = queued.Box.TryRotate()
			if err != nil {
				return placed, rotated, fmt.Errorf("policy %q produced a rotated decision for an unrotatable box %d: %w",
					p.Name(), queued.Box.ID, err)
			}
		}

		if err := eng.world.Container.Place(
			box,
			decision.Point.X,
			decision.Point.Y,
			decision.Rotated,
		); err != nil {
			return placed, rotated, fmt.Errorf("policy %q produced an invalid placement for box %d: %w", p.Name(), queued.Box.ID, err)
		}
		placed++
	}

	return placed, rotated, nil
}

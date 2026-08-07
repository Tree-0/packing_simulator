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
	QueueSize       int
	MinBoxHeight    int
	MaxBoxHeight    int
	MinBoxWidth     int
	MaxBoxWidth     int
	Seed            int64
}

type SimulationResult struct {
	Iterations   int
	Generated    int
	Placed       int
	Rejected     int
	Batches      int
	StoppedEarly bool
}

// StepObserver is called after each timestamp has been processed.
type StepObserver func(timestamp int, world *World) error

type SimulationEngine struct {
	world     *World
	generator BoxGenerator
}

func NewSimulationEngine(config SimulationConfig) (*SimulationEngine, error) {
	if config.MaxBoxHeight > config.ContainerHeight || config.MaxBoxWidth > config.ContainerWidth {
		return nil, errors.New("maximum box dimensions cannot exceed container dimensions")
	}

	world, err := NewWorld(config.ContainerHeight, config.ContainerWidth, config.QueueSize)
	if err != nil {
		return nil, err
	}

	generator, err := NewRandomBoxGenerator(
		config.Seed,
		config.MinBoxWidth,
		config.MaxBoxWidth,
		config.MinBoxHeight,
		config.MaxBoxHeight,
	)
	if err != nil {
		return nil, err
	}

	return &SimulationEngine{
		world:     world,
		generator: generator,
	}, nil
}

func (eng *SimulationEngine) World() *World {
	return eng.world
}

// Run generates one box per iteration and processes boxes whenever the queue
// reaches its configured limit. A final partial queue is processed as a batch.
func (eng *SimulationEngine) Run(p Policy, iterations int) (SimulationResult, error) {
	return eng.RunWithObserver(p, iterations, nil)
}

// RunWithObserver runs the simulation and calls observer after each timestamp.
// The observer may be nil when no per-step output is needed.
func (eng *SimulationEngine) RunWithObserver(
	p Policy,
	iterations int,
	observer StepObserver,
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
			placed, err := eng.processBatch(p, batch)
			if err != nil {
				return result, err
			}
			result.Batches++
			result.Placed += placed
			result.Rejected += len(batch) - placed
			if placed == 0 {
				result.StoppedEarly = true
				stopped = true
			}
		}

		if observer != nil {
			if err := observer(t, eng.world); err != nil {
				return result, fmt.Errorf("observing timestamp %d: %w", t, err)
			}
		}

		if stopped {
			return result, nil
		}
	}

	return result, nil
}

func (eng *SimulationEngine) processBatch(p Policy, batch []QueuedBox) (int, error) {
	placed := 0
	for _, queued := range batch {
		decision, found := p.FindPlacement(&eng.world.Container, queued.Box)
		if !found {
			continue
		}

		if err := eng.world.Container.Place(
			queued.Box, 
			decision.Point.X, 
			decision.Point.Y,
		); err != nil {
			return placed, fmt.Errorf("policy %q produced an invalid placement for box %d: %w", p.Name(), queued.Box.ID, err)
		}
		placed++
	}

	return placed, nil
}

/*
Objective functions to evaluate the performance of a packing policy via various metrics.
*/

package evaluator

import (
	"fmt"
	"strings"

	"packing_simulator/backend"
)

type EvaluationType int

const (
	ContainerUtilization EvaluationType = iota
	ContainerFragmentation
	AreaWeightedContainerFragmentation // Lower is better
	ContainerCompactness               // higher is better
	FutureFitProbabilityMetric         // Requires a box-size distribution.
)

func AllEvaluationTypes() []EvaluationType {
	return []EvaluationType{
		ContainerUtilization,
		ContainerFragmentation,
		AreaWeightedContainerFragmentation,
		ContainerCompactness,
		FutureFitProbabilityMetric,
	}
}

func (evalType EvaluationType) String() string {
	switch evalType {
	case ContainerUtilization:
		return "Container utilization"
	case ContainerFragmentation:
		return "Container fragmentation"
	case AreaWeightedContainerFragmentation:
		return "Area-weighted fragmentation"
	case ContainerCompactness:
		return "Compactness"
	case FutureFitProbabilityMetric:
		return "Future fit probability"
	default:
		return "Unknown evaluation"
	}
}

// Get the evaluator name from config and return the type
func ParseEvaluation(name string) (EvaluationType, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "utilization":
		return ContainerUtilization, nil
	case "fragmentation":
		return ContainerFragmentation, nil
	case "area-weighted-fragmentation":
		return AreaWeightedContainerFragmentation, nil
	case "compactness":
		return ContainerCompactness, nil
	case "future-fit-probability":
		return FutureFitProbabilityMetric, nil
	default:
		return 0, fmt.Errorf(
			"unknown evaluator %q; choose one of: utilization, fragmentation, area-weighted-fragmentation, compactness, future-fit-probability",
			name,
		)
	}
}

// EvaluateSimulation evaluates the simulation's current world.
func EvaluateSimulation(sim *backend.SimulationEngine, evalType EvaluationType) float64 {
	if sim == nil {
		return 0
	}
	switch evalType {
	case FutureFitProbabilityMetric:
		return FutureFitProbability(
			sim.World(),
			sim.UniformBoxDistribution())
	}
	return EvaluateWorld(sim.World(), evalType)
}

func EvaluateWorld(world *backend.World, evalType EvaluationType) float64 {
	if world == nil {
		return 0
	}

	switch evalType {
	case ContainerUtilization:
		return Utilization(world)
	case ContainerFragmentation:
		// ignores the other metrics returned by Fragmentation score for now...
		return Fragmentation(world).FragmentationScore
	case AreaWeightedContainerFragmentation:
		return AreaWeightedFragmentation(world)
	case ContainerCompactness:
		return Compactness(world)
	default:
		return 0
	}

}

func InContainer(c *backend.Container, p backend.Point) bool {
	return 0 <= p.X && p.X < c.Width() && 0 <= p.Y && p.Y < c.Height()
}

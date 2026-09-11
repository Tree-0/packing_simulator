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
	BinCount EvaluationType = iota // Lower is better.
	ContainerUtilization
	ContainerFragmentation
	AreaWeightedContainerFragmentation // Lower is better.
	ContainerCompactness               // Higher is better.
	FutureFitProbabilityMetric         // Requires a box-size distribution.
)

func AllEvaluationTypes() []EvaluationType {
	return []EvaluationType{
		BinCount,
		ContainerUtilization,
		ContainerFragmentation,
		AreaWeightedContainerFragmentation,
		ContainerCompactness,
		FutureFitProbabilityMetric,
	}
}

func (evalType EvaluationType) String() string {
	switch evalType {
	case BinCount:
		return "Used bin count"
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

// ParseEvaluation gets an evaluator type from its config name.
func ParseEvaluation(name string) (EvaluationType, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "bin-count":
		return BinCount, nil
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
		evaluationTypes := AllEvaluationTypes()
		names := make([]string, len(evaluationTypes))
		for i, evaluationType := range evaluationTypes {
			names[i] = evaluationType.String()
		}
		return 0, fmt.Errorf("unknown evaluator %q; choose one of: %s", name, strings.Join(names, ", "))
	}
}

// EvaluateSimulation evaluates the simulation's current world.
func EvaluateSimulation(sim *backend.SimulationEngine, evalType EvaluationType) float64 {
	if sim == nil {
		return 0
	}
	if evalType == FutureFitProbabilityMetric {
		return WorldFutureFitProbability(sim.World(), sim.UniformBoxDistribution())
	}
	return EvaluateWorld(sim.World(), evalType)
}

// EvaluateWorld returns the requested scalar world-level metric.
func EvaluateWorld(world *backend.World, evalType EvaluationType) float64 {
	switch evalType {
	case BinCount:
		return float64(UsedContainerCount(world))
	case ContainerUtilization:
		return WorldUtilization(world)
	case ContainerFragmentation:
		return MeanFragmentation(world)
	case AreaWeightedContainerFragmentation:
		return WorldAreaWeightedFragmentation(world)
	case ContainerCompactness:
		return WorldCompactness(world)
	default:
		return 0
	}
}

// ContainerMetrics describes all geometric metrics for one used container.
type ContainerMetrics struct {
	ContainerID               int
	Utilization               float64
	Fragmentation             FragmentationMetrics
	AreaWeightedFragmentation float64
	Compactness               float64
	FutureFitProbability      float64
}

// EvaluateContainerMetrics returns one evaluation record for each used
// container, in the world's container order.
func EvaluateContainerMetrics(
	world *backend.World,
	distribution backend.UniformBoxDistribution,
) []ContainerMetrics {
	containers := usedContainers(world)
	metrics := make([]ContainerMetrics, 0, len(containers))
	for _, container := range containers {
		fragmentation := Fragmentation(container)
		areaWeightedFragmentation := areaWeightedFragmentation(fragmentation, containerArea(container))
		metrics = append(metrics, ContainerMetrics{
			ContainerID:               container.Id(),
			Utilization:               Utilization(container),
			Fragmentation:             fragmentation,
			AreaWeightedFragmentation: areaWeightedFragmentation,
			Compactness:               1 - areaWeightedFragmentation,
			FutureFitProbability:      FutureFitProbability(container, distribution),
		})
	}
	return metrics
}

func usedContainers(world *backend.World) []*backend.Container {
	if world == nil {
		return nil
	}

	containers := make([]*backend.Container, 0, len(world.Containers))
	for _, container := range world.Containers {
		if container != nil && container.OccupiedArea() > 0 {
			containers = append(containers, container)
		}
	}
	return containers
}

func containerArea(container *backend.Container) int {
	if container == nil {
		return 0
	}
	return container.Height() * container.Width()
}

func InContainer(container *backend.Container, point backend.Point) bool {
	return container != nil && 0 <= point.X && point.X < container.Width() && 0 <= point.Y && point.Y < container.Height()
}

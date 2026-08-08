/*
Objective functions to evaluate the performance of a packing policy via various metrics.
*/

package backend

type EvaluationType int

const (
	ContainerUtilization EvaluationType = iota
	ContainerFragmentation
	FutureFitProbabilityMetric // Requires a box-size distribution.
)

func AllEvaluationTypes() []EvaluationType {
	return []EvaluationType{
		ContainerUtilization,
		ContainerFragmentation,
		FutureFitProbabilityMetric,
	}
}

func (evalType EvaluationType) String() string {
	switch evalType {
	case ContainerUtilization:
		return "Container utilization"
	case ContainerFragmentation:
		return "Container fragmentation"
	case FutureFitProbabilityMetric:
		return "Future fit probability"
	default:
		return "Unknown evaluation"
	}
}

// EvaluateSimulation evaluates the simulation's current world.
func EvaluateSimulation(sim *SimulationEngine, evalType EvaluationType) float64 {
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

func EvaluateWorld(world *World, evalType EvaluationType) float64 {
	if world == nil {
		return 0
	}

	switch evalType {
	case ContainerUtilization:
		return Utilization(world)
	case ContainerFragmentation:
		// ignores the other metrics returned by Fragmentation score for now...
		return Fragmentation(world).FragmentationScore
	default:
		return 0
	}

}

func InContainer(c *Container, p Point) bool {
	return 0 <= p.X && p.X < c.Width() && 0 <= p.Y && p.Y < c.Height()
}

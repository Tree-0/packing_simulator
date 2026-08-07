/*
Objective functions to evaluate the performance of a packing policy via various metrics.
*/

package backend

type EvaluationType int

const (
	ContainerUtilization EvaluationType = iota
	ContainerFragmentation
	FutureFitProbability // the probability that a randomly generated future box fits somewhere
)

// EvaluateSimulation evaluates the simulation's current world.
func EvaluateSimulation(sim *SimulationEngine, evalType EvaluationType) float64 {
	if sim == nil {
		return 0
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

// ratio of used cells : total cells
func Utilization(world *World) float64 {
	container := &world.Container
	total := container.Height() * container.Width()
	if total == 0 {
		return 0
	}

	// sum used space
	occupied := 0
	for y := 0; y < container.Height(); y++ {
		for x := 0; x < container.Width(); x++ {
			cell, _ := container.Cell(x, y)
			if cell != EmptyCell {
				occupied++
			}
		}
	}

	return float64(occupied) / float64(total)
}

type FragmentationMetrics struct {
	RegionCount        int
	LargestRegionRatio float64
	FragmentationScore float64
}

// Fragmentation measures the distinct four-directionally connected regions of
// unused packing space.
func Fragmentation(world *World) FragmentationMetrics {
	if world == nil {
		return FragmentationMetrics{}
	}

	container := &world.Container
	total := container.Height() * container.Width()
	if total == 0 {
		return FragmentationMetrics{}
	}

	visitedCells := make(map[Point]struct{})

	fragments := 0
	emptyCells := 0
	fragmentSizes := make([]int, 0)
	directions := [...]Point{
		{X: 0, Y: 1},
		{X: 0, Y: -1},
		{X: 1, Y: 0},
		{X: -1, Y: 0},
	}

	// bfs from each tile to find distinct packing gaps
	for y := 0; y < container.Height(); y++ {
		for x := 0; x < container.Width(); x++ {
			start := Point{X: x, Y: y}
			if _, seen := visitedCells[start]; seen {
				continue
			}

			// don't explore occupied cells
			cell, err := container.Cell(x, y)
			if err != nil || cell != EmptyCell {
				continue
			}

			// found the start of a new empty fragment
			fragments++
			currFragmentSize := 0
			cellQueue := []Point{start}
			visitedCells[start] = struct{}{}

			for queueIndex := 0; queueIndex < len(cellQueue); queueIndex++ {
				cellPoint := cellQueue[queueIndex]
				// new contiguous empty cell is part of this fragment
				currFragmentSize++
				emptyCells++

				// explore in-range neighbors.
				for _, direction := range directions {
					next := Point{X: cellPoint.X + direction.X, Y: cellPoint.Y + direction.Y}
					if !InContainer(container, next) {
						continue
					}
					if _, seen := visitedCells[next]; seen {
						continue
					}

					nextCell, err := container.Cell(next.X, next.Y)
					if err != nil || nextCell != EmptyCell {
						continue
					}

					visitedCells[next] = struct{}{}
					cellQueue = append(cellQueue, next)
				}
			}

			fragmentSizes = append(fragmentSizes, currFragmentSize)
		}
	}

	squaredFragmentSum := float64(0)
	largestFragment := 0
	for _, fragmentSize := range fragmentSizes {
		squaredFragmentSum += float64(fragmentSize) * float64(fragmentSize)
		if fragmentSize > largestFragment {
			largestFragment = fragmentSize
		}
	}

	fragmentation := 0.0
	largestRegionRatio := 0.0
	if emptyCells > 0 {
		emptyCellCount := float64(emptyCells)
		fragmentation = 1 - squaredFragmentSum/(emptyCellCount*emptyCellCount)
		largestRegionRatio = float64(largestFragment) / emptyCellCount
	}

	return FragmentationMetrics{
		RegionCount:        fragments,
		LargestRegionRatio: largestRegionRatio,
		FragmentationScore: fragmentation,
	}
}

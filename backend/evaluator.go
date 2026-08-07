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
		return Fragmentation(world).FragmentationScore
	default:
		return 0
	}

}

func InContainer(c *Container, p Point) bool {
	return 0 <= p.X && p.X < c.Width() && 0 <= p.Y && p.Y < c.Height();
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
	RegionCount int
	LargestRegionRatio float64
	FragmentationScore float64
}

// the number of "islands" of unused packing space
func Fragmentation(world *World) FragmentationMetrics {
	container := &world.Container
	total := container.Height() * container.Width()
	if total == 0 {
		return FragmentationMetrics {
			RegionCount: 0,
			LargestRegionRatio: 0,
			FragmentationScore: 0,
		}
	}

	cellQueue := []Point{}
	visitedCells := make(map[Point]struct{})

	fragments := 0
	emptyCells := 0
	fragmentSizes := make([]int, 0)

	// bfs from each tile to find distinct packing gaps
	for y := 0; y < container.Height(); y++ {
		for x := 0; x < container.Width(); x++ {

			// don't explore occupied cells
			cell, err := container.Cell(x,y)
			if err != nil || cell != EmptyCell {
				continue
			}

			// found the start of a new empty fragment
			fragments += 1
			currFragmentSize := 0
			cellQueue = append(cellQueue, Point{x, y})

			for len(cellQueue) > 0 {
				cell_point := cellQueue[0]
				cell_val, err := container.Cell(cell_point.X, cell_point.Y)
				cellQueue = cellQueue[1:]

				if err != nil {
					// print something or return early? shouldn't be getting errors
					// and I want to propagate it
				}

				_, seen := visitedCells[cell_point]
				if seen || cell_val != EmptyCell {
					continue
				}

				visitedCells[cell_point] = struct{}{}
				
				// new contiguous empty cell is part of this fragment
				currFragmentSize += 1
				emptyCells += 1

				dirx := []int{0,0,1,-1}
				diry := []int{1,-1,0,0}
				
				// explore in-range neighbors.
				for i := 0; i < len(dirx); i++ {
					next := Point{cell_point.X + dirx[i], cell_point.Y + diry[i]}
					if InContainer(container, next){
						cellQueue = append(cellQueue, next)
					}
				}

			}

			fragmentSizes = append(fragmentSizes, currFragmentSize)
		}
	}

	squaredFragmentSum := 0
	for i := 0; i < len(fragmentSizes); i++ {
		squaredFragmentSum += fragmentSizes[i] * fragmentSizes[i]
	}
	
	fragmentation := float64(0)
	if emptyCells > 0 {
		fragmentation = 1 - (float64(squaredFragmentSum) / float64(emptyCells * emptyCells))
	}

	return FragmentationMetrics {
		RegionCount: fragments,
		LargestRegionRatio: -1,
		FragmentationScore: fragmentation,
	}

}

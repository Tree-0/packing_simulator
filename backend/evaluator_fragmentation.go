package backend

type FragmentationMetrics struct {
	RegionCount        int
	LargestRegionRatio float64
	FragmentationScore float64
	EmptyCells 		   int
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
		EmptyCells: 		emptyCells,
	}
}

func AreaWeightedFragmentation(world *World) float64 {

	fragmentation := Fragmentation(world)
	totalCells := world.Container.Height() * world.Container.Width()
	emptyFraction := float64(fragmentation.EmptyCells) / float64(totalCells)

	// a very small number of empty cells will reduce the impact
	// of those cells being highly fragmented
	return (emptyFraction * fragmentation.FragmentationScore)

}

// complement to AreaWeightedFragmentation.
// Larger value indicates better score.
func Compactness(world *World) float64 {
	return 1 - AreaWeightedFragmentation(world)
}
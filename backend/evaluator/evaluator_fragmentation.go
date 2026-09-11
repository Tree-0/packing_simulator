package evaluator

import "packing_simulator/backend"

type FragmentationMetrics struct {
	RegionCount        int
	LargestRegionRatio float64
	FragmentationScore float64
	EmptyCells         int
}

// Fragmentation measures the distinct four-directionally connected regions of
// unused packing space in one container.
func Fragmentation(container *backend.Container) FragmentationMetrics {
	if containerArea(container) == 0 {
		return FragmentationMetrics{}
	}

	visitedCells := make(map[backend.Point]struct{})
	fragmentSizes := make([]int, 0)
	directions := [...]backend.Point{
		{X: 0, Y: 1},
		{X: 0, Y: -1},
		{X: 1, Y: 0},
		{X: -1, Y: 0},
	}

	emptyCells := 0
	for y := 0; y < container.Height(); y++ {
		for x := 0; x < container.Width(); x++ {
			start := backend.Point{X: x, Y: y}
			if _, seen := visitedCells[start]; seen {
				continue
			}

			cell, err := container.Cell(x, y)
			if err != nil || cell != backend.EmptyCell {
				continue
			}

			fragmentSize := 0
			cellQueue := []backend.Point{start}
			visitedCells[start] = struct{}{}
			for queueIndex := 0; queueIndex < len(cellQueue); queueIndex++ {
				cellPoint := cellQueue[queueIndex]
				fragmentSize++
				emptyCells++

				for _, direction := range directions {
					next := backend.Point{X: cellPoint.X + direction.X, Y: cellPoint.Y + direction.Y}
					if !InContainer(container, next) {
						continue
					}
					if _, seen := visitedCells[next]; seen {
						continue
					}

					nextCell, err := container.Cell(next.X, next.Y)
					if err != nil || nextCell != backend.EmptyCell {
						continue
					}
					visitedCells[next] = struct{}{}
					cellQueue = append(cellQueue, next)
				}
			}
			fragmentSizes = append(fragmentSizes, fragmentSize)
		}
	}

	squaredFragmentSum := 0.0
	largestFragment := 0
	for _, fragmentSize := range fragmentSizes {
		squaredFragmentSum += float64(fragmentSize * fragmentSize)
		if fragmentSize > largestFragment {
			largestFragment = fragmentSize
		}
	}

	if emptyCells == 0 {
		return FragmentationMetrics{}
	}

	emptyCellCount := float64(emptyCells)
	return FragmentationMetrics{
		RegionCount:        len(fragmentSizes),
		LargestRegionRatio: float64(largestFragment) / emptyCellCount,
		FragmentationScore: 1 - squaredFragmentSum/(emptyCellCount*emptyCellCount),
		EmptyCells:         emptyCells,
	}
}

// AreaWeightedFragmentation discounts fragmentation when little empty space
// remains in one container.
func AreaWeightedFragmentation(container *backend.Container) float64 {
	return areaWeightedFragmentation(Fragmentation(container), containerArea(container))
}

func areaWeightedFragmentation(fragmentation FragmentationMetrics, totalCells int) float64 {
	if totalCells == 0 {
		return 0
	}
	return float64(fragmentation.EmptyCells) / float64(totalCells) * fragmentation.FragmentationScore
}

// Compactness is the complement of area-weighted fragmentation for one
// container. A nil or zero-sized container has no compactness score.
func Compactness(container *backend.Container) float64 {
	if containerArea(container) == 0 {
		return 0
	}
	return 1 - AreaWeightedFragmentation(container)
}

// MeanFragmentation is the unweighted mean fragmentation score across used
// containers.
func MeanFragmentation(world *backend.World) float64 {
	containers := usedContainers(world)
	if len(containers) == 0 {
		return 0
	}

	total := 0.0
	for _, container := range containers {
		total += Fragmentation(container).FragmentationScore
	}
	return total / float64(len(containers))
}

// WorldAreaWeightedFragmentation weights each used container's fragmentation
// by its empty cells relative to all used-container capacity.
func WorldAreaWeightedFragmentation(world *backend.World) float64 {
	weightedFragmentation := 0.0
	totalCapacity := 0
	for _, container := range usedContainers(world) {
		fragmentation := Fragmentation(container)
		weightedFragmentation += float64(fragmentation.EmptyCells) * fragmentation.FragmentationScore
		totalCapacity += containerArea(container)
	}
	if totalCapacity == 0 {
		return 0
	}
	return weightedFragmentation / float64(totalCapacity)
}

// WorldCompactness is the complement of world area-weighted fragmentation.
func WorldCompactness(world *backend.World) float64 {
	if UsedContainerCount(world) == 0 {
		return 0
	}
	return 1 - WorldAreaWeightedFragmentation(world)
}

package evaluator

import "packing_simulator/backend"

// FutureFitProbability is the probability that a box from the provided size
// distribution could fit in one container. Rotation is not considered.
func FutureFitProbability(container *backend.Container, distribution backend.UniformBoxDistribution) float64 {
	if container == nil || !validDistribution(distribution) {
		return 0
	}

	fitCount := 0
	totalPossibleSizes := distributionSizeCount(distribution)
	fitIndex := container.OccupancySnapshot()
	for height := distribution.MinHeight; height <= distribution.MaxHeight; height++ {
		for width := distribution.MinWidth; width <= distribution.MaxWidth; width++ {
			if fitIndex.CanFitDimensions(width, height) {
				fitCount++
			}
		}
	}

	return float64(fitCount) / float64(totalPossibleSizes)
}

// WorldFutureFitProbability is the probability that a generated box can fit
// in at least one used container. Rotation is not considered.
func WorldFutureFitProbability(world *backend.World, distribution backend.UniformBoxDistribution) float64 {
	containers := usedContainers(world)
	if len(containers) == 0 || !validDistribution(distribution) {
		return 0
	}

	fitIndices := make([]backend.OccupancySnapshot, len(containers))
	for i, container := range containers {
		fitIndices[i] = container.OccupancySnapshot()
	}

	fitCount := 0
	totalPossibleSizes := distributionSizeCount(distribution)
	for height := distribution.MinHeight; height <= distribution.MaxHeight; height++ {
		for width := distribution.MinWidth; width <= distribution.MaxWidth; width++ {
			for _, fitIndex := range fitIndices {
				if fitIndex.CanFitDimensions(width, height) {
					fitCount++
					break
				}
			}
		}
	}

	return float64(fitCount) / float64(totalPossibleSizes)
}

func validDistribution(distribution backend.UniformBoxDistribution) bool {
	return distribution.MinWidth > 0 && distribution.MinHeight > 0 &&
		distribution.MaxWidth >= distribution.MinWidth &&
		distribution.MaxHeight >= distribution.MinHeight
}

func distributionSizeCount(distribution backend.UniformBoxDistribution) int {
	return (distribution.MaxHeight - distribution.MinHeight + 1) *
		(distribution.MaxWidth - distribution.MinWidth + 1)
}

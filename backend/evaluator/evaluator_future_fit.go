package evaluator

import "packing_simulator/backend"

// The probability that a box from the provided size distribution could fit somewhere
// in the current world's container.
// TODO: account for rotation
func FutureFitProbability(world *backend.World, d backend.UniformBoxDistribution) float64 {
	if world == nil || d.MinWidth <= 0 || d.MinHeight <= 0 || d.MaxWidth < d.MinWidth || d.MaxHeight < d.MinHeight {
		return 0
	}

	fitCount := 0
	totalPossibleSizes := (d.MaxHeight - d.MinHeight + 1) * (d.MaxWidth - d.MinWidth + 1)

	// build a public wrapper of a reusable 2D prefix sum index for rectangular fit queries
	fitIndex := world.Container.OccupancySnapshot()

	for height := d.MinHeight; height <= d.MaxHeight; height++ {
		for width := d.MinWidth; width <= d.MaxWidth; width++ {
			if fitIndex.CanFitDimensions(width, height) {
				fitCount += 1
			}
		}
	}

	if totalPossibleSizes == 0 {
		return 0
	}

	return float64(fitCount) / float64(totalPossibleSizes)
}

// TODO: Current future fit only examines the probability of ANY future box fitting.
// It does not consider HOW MANY could fit, and currently does not account for rotation

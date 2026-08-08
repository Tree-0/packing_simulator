package backend

// The probability that a box from the provided size distribution could fit somewhere
// in the current world's container.
func FutureFitProbability(world *World, d UniformBoxDistribution) float64 {
	if world == nil || d.MinWidth <= 0 || d.MinHeight <= 0 || d.MaxWidth < d.MinWidth || d.MaxHeight < d.MinHeight {
		return 0
	}

	fitCount := 0
	totalPossibleSizes := (d.MaxHeight - d.MinHeight + 1) * (d.MaxWidth - d.MinWidth + 1)
	fitIndex := newOccupancyIndex(&world.Container)

	for height := d.MinHeight; height <= d.MaxHeight; height++ {
		for width := d.MinWidth; width <= d.MaxWidth; width++ {
			if fitIndex.canFitDimensions(width, height) {
				fitCount += 1
			}
		}
	}

	if totalPossibleSizes == 0 {
		return 0
	}

	return float64(fitCount) / float64(totalPossibleSizes)
}
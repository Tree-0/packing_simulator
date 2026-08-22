package backend

import "sort"

func (LargestAreaBottomLeftPolicy) OrderBatch(batch []QueuedBox) {
	sort.SliceStable(batch, func(i, j int) bool {
		areaI := batch[i].Box.Width * batch[i].Box.Height
		areaJ := batch[j].Box.Width * batch[j].Box.Height

		return areaI > areaJ
	})
}

// Same as BottomLeftPolicy, but the OrderBatch function will have been called by
// the simulation engine to change the order in which boxes are placed.
func (LargestAreaBottomLeftPolicy) FindPlacement(container *Container, box Box) (PlacementDecision, bool) {
	if container == nil {
		return PlacementDecision{}, false
	}

	decision, found := findPlacementHelper(container, box, false)

	// if no valid rotation, return our current placement (or none)
	if !box.CanRotate || box.Height == box.Width {
		return decision, found
	}

	rotatedBox := Box{
		ID:        box.ID,
		Height:    box.Width,
		Width:     box.Height,
		CanRotate: true,
	}

	rotatedDecision, rotatedFound := findPlacementHelper(container, rotatedBox, true)

	if !rotatedFound {
		return decision, found
	}

	// we placed the rotated version successfully,
	// but not the original
	if !found {
		return rotatedDecision, rotatedFound
	}

	// both valid, determine which is better.
	// if same placement, prefer original (unrotated)
	finalDecision := PlacementDecision{}

	// prioritize lowest, then leftmost
	switch {
	case decision.Point.Y > rotatedDecision.Point.Y:
		finalDecision = decision
	case decision.Point.Y < rotatedDecision.Point.Y:
		finalDecision = rotatedDecision
	default:
		switch {
		case rotatedDecision.Point.X < decision.Point.X:
			finalDecision = rotatedDecision
		default:
			finalDecision = decision
		}
	}

	return finalDecision, true
}
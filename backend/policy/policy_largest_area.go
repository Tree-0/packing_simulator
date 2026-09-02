package policy

import (
	"sort"

	"packing_simulator/backend"
)

// LargestAreaBottomLeftPolicy orders available boxes by width * height, placing them in that order.
type LargestAreaBottomLeftPolicy struct{}

func (LargestAreaBottomLeftPolicy) Name() string {
	return LargestAreaBottomLeftPolicyName
}

func (LargestAreaBottomLeftPolicy) OrderBatch(batch []backend.QueuedBox) []backend.QueuedBox {
	sorted := append([]backend.QueuedBox(nil), batch...)

	sort.SliceStable(sorted, func(i, j int) bool {
		areaI := sorted[i].Box.Width * sorted[i].Box.Height
		areaJ := sorted[j].Box.Width * sorted[j].Box.Height

		return areaI > areaJ
	})

	return sorted
}

// Same as BottomLeftPolicy, but the OrderBatch function will have been called by
// the simulation engine to change the order in which boxes are placed.
func (LargestAreaBottomLeftPolicy) FindPlacement(
	context backend.PolicyContext,
	box backend.Box,
) (backend.PlacementDecision, bool) {
	container := context.Container

	decision, found := findPlacementHelper(&container, box, false)

	// if no valid rotation, return our current placement (or none)
	if !box.CanRotate || box.Height == box.Width {
		return decision, found
	}

	rotatedBox := backend.Box{
		ID:        box.ID,
		Height:    box.Width,
		Width:     box.Height,
		CanRotate: true,
	}

	rotatedDecision, rotatedFound := findPlacementHelper(&container, rotatedBox, true)

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
	finalDecision := backend.PlacementDecision{}

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

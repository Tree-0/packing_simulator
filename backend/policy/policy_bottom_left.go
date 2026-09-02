//BottomLeftPolicy places boxes at the lowest, then leftmost, available position.

package policy

import "packing_simulator/backend"

type BottomLeftPolicy struct{}

func (BottomLeftPolicy) Name() string {
	return BottomLeftPolicyName
}

// no-op; policy does not sort batch
func (BottomLeftPolicy) OrderBatch(batch []backend.QueuedBox) []backend.QueuedBox {
	return append([]backend.QueuedBox(nil), batch...) // return a copy
}

// assumes the caller handles null containers
func findPlacementHelper(
	container *backend.ContainerSnapshot,
	box backend.Box,
	rotated bool,
) (backend.PlacementDecision, bool) {
	for y := container.Height() - box.Height; y >= 0; y-- {
		for x := 0; x <= container.Width()-box.Width; x++ {
			if container.CanPlace(box, x, y) {
				return backend.PlacementDecision{
						Point: backend.Point{X: x, Y: y}, Rotated: rotated,
					},
					true
			}
		}
	}

	return backend.PlacementDecision{}, false
}

func (BottomLeftPolicy) FindPlacement(
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

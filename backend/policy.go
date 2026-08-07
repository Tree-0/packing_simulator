/*
Policies to determine how objects are placed into the packing simulation
*/

package backend

type PlacementDecision struct {
	X int
	Y int
}

type Policy interface {
	FindPlacement(*Container, Box) (PlacementDecision, bool)
	Name() string
}

// BottomLeftPolicy places boxes at the lowest, then leftmost, available position.
type BottomLeftPolicy struct{}

func (BottomLeftPolicy) Name() string {
	return "bottom-left"
}

func (BottomLeftPolicy) FindPlacement(container *Container, box Box) (PlacementDecision, bool) {
	if container == nil {
		return PlacementDecision{}, false
	}

	for y := container.Height() - box.Height; y >= 0; y-- {
		for x := 0; x <= container.Width()-box.Width; x++ {
			if container.CanPlace(box, x, y) {
				return PlacementDecision{X: x, Y: y}, true
			}
		}
	}

	return PlacementDecision{}, false
}

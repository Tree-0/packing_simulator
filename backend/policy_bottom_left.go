package backend

// no-op; policy does not sort batch
func (BottomLeftPolicy) OrderBatch(batch []QueuedBox) {}

func (BottomLeftPolicy) FindPlacement(container *Container, box Box) (PlacementDecision, bool) {
	if container == nil {
		return PlacementDecision{}, false
	}

	for y := container.Height() - box.Height; y >= 0; y-- {
		for x := 0; x <= container.Width()-box.Width; x++ {
			if container.CanPlace(box, x, y) {
				return PlacementDecision{Point: Point{X: x, Y: y}}, true
			}
		}
	}

	return PlacementDecision{}, false
}
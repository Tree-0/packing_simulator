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

	for y := container.Height() - box.Height; y >= 0; y-- {
		for x := 0; x <= container.Width()-box.Width; x++ {
			if container.CanPlace(box, x, y) {
				return PlacementDecision{Point: Point{X: x, Y: y}}, true
			}
		}
	}

	return PlacementDecision{}, false
}
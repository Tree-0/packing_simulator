package evaluator

import "packing_simulator/backend"

// ratio of used cells : total cells
func Utilization(world *backend.World) float64 {
	container := &world.Container
	total := container.Height() * container.Width()
	if total == 0 {
		return 0
	}

	return float64(container.OccupiedArea()) / float64(total)
}

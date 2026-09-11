package evaluator

import "packing_simulator/backend"

// Utilization returns the ratio of occupied to total cells in one container.
func Utilization(container *backend.Container) float64 {
	total := containerArea(container)
	if total == 0 {
		return 0
	}
	return float64(container.OccupiedArea()) / float64(total)
}

// UsedContainerCount returns the number of containers with at least one
// occupied cell.
func UsedContainerCount(world *backend.World) int {
	return len(usedContainers(world))
}

// WorldUtilization is the capacity-weighted utilization across used
// containers.
func WorldUtilization(world *backend.World) float64 {
	occupiedArea := 0
	capacity := 0
	for _, container := range usedContainers(world) {
		occupiedArea += container.OccupiedArea()
		capacity += containerArea(container)
	}
	if capacity == 0 {
		return 0
	}
	return float64(occupiedArea) / float64(capacity)
}

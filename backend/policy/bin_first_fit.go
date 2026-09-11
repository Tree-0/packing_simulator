/*
Keeps all bins open.
*/
package policy

import (
	"packing_simulator/backend"
)

type ContainerSelectorFirstFit struct{}

// Since we want first available box, we don't do any ordering.
// returns the ids of all boxes in unchanged order.
func (cs ContainerSelectorFirstFit) Rank(
	context backend.ContainerSelectionContext,
	_ backend.Box,
) []int {
	results := make([]int, len(context.Containers))
	for i, container := range context.Containers {
		results[i] = container.Id()
	}
	return results
}

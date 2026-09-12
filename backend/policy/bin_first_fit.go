package policy

import (
	"packing_simulator/backend"
)

type ContainerSelectorFirstFit struct{}

func (ContainerSelectorFirstFit) Name() string {
	return ContainerSelectorFirstFitName
}

// Rank returns every existing container in creation order.
func (ContainerSelectorFirstFit) Rank(
	context backend.ContainerSelectionContext,
	_ backend.Box,
) []int {
	results := make([]int, len(context.Containers))
	for i, container := range context.Containers {
		results[i] = container.Id()
	}
	return results
}

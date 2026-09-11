/*
Keeps only the most recently opened bin(s) available.
*/
package policy

import (
	"packing_simulator/backend"
)

type ContainerSelectorNextFit struct{}

// Rank returns only the most recently opened container. If that container
// cannot accept the box, the simulation engine is responsible for opening a
// new one.
func (ContainerSelectorNextFit) Rank(
	context backend.ContainerSelectionContext,
	_ backend.Box,
) []int {
	if len(context.Containers) == 0 {
		return nil
	}

	return []int{context.Containers[len(context.Containers)-1].Id()}
}

// ContainerSelectorNextKFit keeps the K most recently opened containers
// available, trying the oldest of those containers first.
type ContainerSelectorNextKFit struct {
	K int
}

func (cs ContainerSelectorNextKFit) Rank(
	context backend.ContainerSelectionContext,
	_ backend.Box,
) []int {
	if cs.K <= 0 || len(context.Containers) == 0 {
		return nil
	}

	start := len(context.Containers) - cs.K
	if start < 0 {
		start = 0
	}

	results := make([]int, len(context.Containers)-start)
	for i, container := range context.Containers[start:] {
		results[i] = container.Id()
	}
	return results
}

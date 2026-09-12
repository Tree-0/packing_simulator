package policy

import (
	"fmt"
	"strings"

	"packing_simulator/backend"
)

const (
	BottomLeftPolicyName            = "bottom-left"
	LargestAreaBottomLeftPolicyName = "largest-area-bottom-left"

	ContainerSelectorFirstFitName = "first-fit"
	ContainerSelectorNextFitName  = "next-fit"
	ContainerSelectorNextKFitName = "next-k-fit"
)

func AvailablePolicyNames() []string {
	return []string{
		BottomLeftPolicyName,
		LargestAreaBottomLeftPolicyName,
	}
}

func NewPlacementPolicy(name string) (backend.PlacementPolicy, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case BottomLeftPolicyName:
		return BottomLeftPolicy{}, nil
	case LargestAreaBottomLeftPolicyName:
		return LargestAreaBottomLeftPolicy{}, nil
	default:
		return nil, fmt.Errorf(
			"unknown policy %q; choose one of: %s",
			name,
			strings.Join(AvailablePolicyNames(), ", "),
		)
	}
}

// AvailableContainerSelectorNames returns the accepted names for container
// selectors. Next K Fit additionally requires a positive K value.
func AvailableContainerSelectorNames() []string {
	return []string{
		ContainerSelectorFirstFitName,
		ContainerSelectorNextFitName,
		ContainerSelectorNextKFitName,
	}
}

// NewContainerSelector constructs a container selector from configuration.
// An omitted name preserves the historical First Fit default.
func NewContainerSelector(name string, k int) (backend.ContainerSelector, error) {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	if normalizedName == "" {
		normalizedName = ContainerSelectorFirstFitName
	}

	switch normalizedName {
	case ContainerSelectorFirstFitName:
		if k != 0 {
			return nil, fmt.Errorf("container selector %q does not use k", normalizedName)
		}
		return ContainerSelectorFirstFit{}, nil
	case ContainerSelectorNextFitName:
		if k != 0 {
			return nil, fmt.Errorf("container selector %q does not use k", normalizedName)
		}
		return ContainerSelectorNextFit{}, nil
	case ContainerSelectorNextKFitName:
		if k < 1 {
			return nil, fmt.Errorf("container selector %q requires k to be at least 1", normalizedName)
		}
		return ContainerSelectorNextKFit{K: k}, nil
	default:
		return nil, fmt.Errorf(
			"unknown container selector %q; choose one of: %s",
			name,
			strings.Join(AvailableContainerSelectorNames(), ", "),
		)
	}
}

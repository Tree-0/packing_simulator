package policy

import (
	"fmt"
	"strings"

	"packing_simulator/backend"
)

const (
	BottomLeftPolicyName            = "bottom-left"
	LargestAreaBottomLeftPolicyName = "largest-area-bottom-left"
)

func AvailablePolicyNames() []string {
	return []string{
		BottomLeftPolicyName,
		LargestAreaBottomLeftPolicyName,
	}
}

func NewPolicy(name string) (backend.Policy, error) {
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

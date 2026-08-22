/*
Policies to determine how objects are placed into the packing simulation
*/

package backend

import (
	"fmt"
	"strings"
)

type PlacementDecision struct {
	Point   Point
	Rotated bool
}

type Policy interface {
	OrderBatch([]QueuedBox)
	FindPlacement(*Container, Box) (PlacementDecision, bool)
	Name() string
}

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

func NewPolicy(name string) (Policy, error) {
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

// BottomLeftPolicy places boxes at the lowest, then leftmost, available position.
type BottomLeftPolicy struct{}

// LargestAreaBottomLeftPolicy orders available boxes by width * height, placing them in that order.
type LargestAreaBottomLeftPolicy struct{}

func (BottomLeftPolicy) Name() string {
	return BottomLeftPolicyName
}

func (LargestAreaBottomLeftPolicy) Name() string {
	return LargestAreaBottomLeftPolicyName
}

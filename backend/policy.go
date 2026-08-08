/*
Policies to determine how objects are placed into the packing simulation
*/

package backend

import (
	"fmt"
	"sort"
	"strings"
)

type PlacementDecision struct {
	Point Point
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

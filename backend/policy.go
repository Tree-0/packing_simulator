/*
Policies to determine how objects are placed into the packing simulation
*/

package backend

// Immutable snapshot provided to any Policy
type PolicyContext struct {
	Timestamp int
	Container ContainerSnapshot
	Batch     []QueuedBox // copied current batch
}

type PlacementDecision struct {
	Point   Point
	Rotated bool
}

type Policy interface {
	OrderBatch([]QueuedBox) ([]QueuedBox)
	FindPlacement(PolicyContext, Box) (PlacementDecision, bool)
	Name() string
}

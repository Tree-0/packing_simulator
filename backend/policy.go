/*
Policies to determine how objects are placed into the packing simulation
*/

package backend

type PlacementDecision struct {
	Point   Point
	Rotated bool
}

type Policy interface {
	OrderBatch([]QueuedBox)
	FindPlacement(*Container, Box) (PlacementDecision, bool)
	Name() string
}

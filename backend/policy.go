/*
Policies to determine how objects are placed into the packing simulation

PlacementPolicy orders boxes to iterate through
	-> ContainerSelector chooses container for that box, passes result back
		-> PlacementPolicy puts a box in that container.
*/

package backend

type ContainerSelectionContext struct {
	Timestamp  int
	Containers []ContainerSnapshot
	Batch      []QueuedBox // boxes from current batch not yet placed
}

// ContainerSelector ranks existing containers in the order the simulation
// should try them for a box.
type ContainerSelector interface {
	Rank(ContainerSelectionContext, Box) []int // container IDs, best first
	Name() string
}

// We feed the output of ContainerSelector.Rank() one-by-one into the PlacementPolicy
// until we successfully place.

// Context provided to a placement policy, for deciding where
// to place a box in the container previously selected by a container
// selection policy.
type PlacementContext struct {
	Timestamp int
	Container ContainerSnapshot
	Batch     []QueuedBox // boxes from current batch not yet placed
}

type PlacementPolicy interface {
	OrderBatch([]QueuedBox) []QueuedBox
	FindPlacement(PlacementContext, Box) (PlacementDecision, bool)
	Name() string
}

type PlacementDecision struct {
	// ContainerID is not included, the simulator loops over containers
	// when processing a batch. The placement policy never has knowledge
	// of the container ID it is exploring.
	Point   Point
	Rotated bool
}

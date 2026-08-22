package backend

import (
	"reflect"
	"testing"
)

func TestLargestAreaBottomLeftOrderBatch(t *testing.T) {
	batch := []QueuedBox{
		{Box: Box{ID: 1, Width: 1, Height: 1}}, // area 1
		{Box: Box{ID: 2, Width: 2, Height: 2}}, // area 4
		{Box: Box{ID: 3, Width: 1, Height: 3}}, // area 3
		{Box: Box{ID: 4, Width: 2, Height: 2}}, // area 4
	}

	LargestAreaBottomLeftPolicy{}.OrderBatch(batch)

	gotIDs := make([]int, len(batch))
	for i, queued := range batch {
		gotIDs[i] = queued.Box.ID
	}
	wantIDs := []int{2, 4, 3, 1}

	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Errorf("OrderBatch() IDs = %v; want %v", gotIDs, wantIDs)
	}
}

func TestNewPolicy(t *testing.T) {
	tests := []struct {
		name     string
		wantName string
		wantErr  bool
	}{
		{name: BottomLeftPolicyName, wantName: BottomLeftPolicyName},
		{name: " LARGEST-AREA-BOTTOM-LEFT ", wantName: LargestAreaBottomLeftPolicyName},
		{name: "not-a-policy", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, err := NewPolicy(tt.name)
			if tt.wantErr {
				if err == nil {
					t.Fatal("NewPolicy() returned nil error; want an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if policy.Name() != tt.wantName {
				t.Errorf("NewPolicy(%q).Name() = %q; want %q", tt.name, policy.Name(), tt.wantName)
			}
		})
	}
}

func TestBottomLeftPolicyOnlyRotatedOrientationFits(t *testing.T) {
	container, err := NewContainer(1, 2)
	if err != nil {
		t.Fatal(err)
	}

	// rotating is the only way to fit into the 1x2 container
	decision, found := BottomLeftPolicy{}.FindPlacement(container, Box{
		ID:        1,
		Width:     1,
		Height:    2,
		CanRotate: true,
	})

	if !found {
		t.Fatal("FindPlacement() found no placement; want the rotated orientation")
	}
	want := PlacementDecision{Point: Point{X: 0, Y: 0}, Rotated: true}
	if decision != want {
		t.Errorf("FindPlacement() = %+v; want %+v", decision, want)
	}
}

func TestProcessBatchRecordsRotatedPlacement(t *testing.T) {
	world, err := NewWorld(1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	engine := &SimulationEngine{world: world}

	placed, rotated, err := engine.processBatch(BottomLeftPolicy{}, []QueuedBox{{
		Box: Box{ID: 1, Width: 1, Height: 2, CanRotate: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if placed != 1 || rotated != 1 {
		t.Fatalf("processBatch() = (%d placed, %d rotated); want (1 placed, 1 rotated)", placed, rotated)
	}

	placement, found := world.Container.placements[1]
	if !found {
		t.Fatal("rotated box was not recorded in the container")
	}
	if !placement.Rotated {
		t.Error("rotated box placement was recorded with Rotated = false")
	}
}

func TestBottomLeftPolicyChoosesOriginalOrientationWhenItIsLower(t *testing.T) {
	container, err := NewContainer(3, 3)
	if err != nil {
		t.Fatal(err)
	}

	decision, found := BottomLeftPolicy{}.FindPlacement(container, Box{
		ID:        1,
		Width:     2,
		Height:    1,
		CanRotate: true,
	})

	if !found {
		t.Fatal("FindPlacement() found no placement; want the original orientation")
	}
	want := PlacementDecision{Point: Point{X: 0, Y: 2}, Rotated: false}
	if decision != want {
		t.Errorf("FindPlacement() = %+v; want %+v", decision, want)
	}
}

func TestBottomLeftPolicyChoosesRotatedOrientationWhenItIsLower(t *testing.T) {
	container, err := NewContainer(3, 3)
	if err != nil {
		t.Fatal(err)
	}

	decision, found := BottomLeftPolicy{}.FindPlacement(container, Box{
		ID:        1,
		Width:     1,
		Height:    2,
		CanRotate: true,
	})

	if !found {
		t.Fatal("FindPlacement() found no placement; want the rotated orientation")
	}
	want := PlacementDecision{Point: Point{X: 0, Y: 2}, Rotated: true}
	if decision != want {
		t.Errorf("FindPlacement() = %+v; want %+v", decision, want)
	}
}

func TestBottomLeftPolicyPrefersOriginalOrientationOnTie(t *testing.T) {
	container, err := NewContainer(3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := container.Place(Box{ID: 99, Width: 1, Height: 1}, 1, 2, false); err != nil {
		t.Fatal(err)
	}

	decision, found := BottomLeftPolicy{}.FindPlacement(container, Box{
		ID:        1,
		Width:     2,
		Height:    1,
		CanRotate: true,
	})

	if !found {
		t.Fatal("FindPlacement() found no placement; want an orientation tie")
	}
	want := PlacementDecision{Point: Point{X: 0, Y: 1}, Rotated: false}
	if decision != want {
		t.Errorf("FindPlacement() = %+v; want %+v", decision, want)
	}
}

func TestBottomLeftPolicyChoosesOriginalOrientationWhenItIsLeftmost(t *testing.T) {
	container, err := NewContainer(3, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := container.Place(Box{ID: 98, Width: 1, Height: 1}, 0, 2, false); err != nil {
		t.Fatal(err)
	}
	if err := container.Place(Box{ID: 99, Width: 1, Height: 1}, 2, 2, false); err != nil {
		t.Fatal(err)
	}

	// The original 2x1 footprint fits at (0, 1). Its rotated 1x2 footprint
	// fits on the same row, but only at (1, 1).
	decision, found := BottomLeftPolicy{}.FindPlacement(container, Box{
		ID:        1,
		Width:     2,
		Height:    1,
		CanRotate: true,
	})

	if !found {
		t.Fatal("FindPlacement() found no placement; want the leftmost original orientation")
	}
	want := PlacementDecision{Point: Point{X: 0, Y: 1}, Rotated: false}
	if decision != want {
		t.Errorf("FindPlacement() = %+v; want %+v", decision, want)
	}
}

func TestBottomLeftPolicyChoosesRotatedOrientationWhenItIsLeftmost(t *testing.T) {
	container, err := NewContainer(3, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := container.Place(Box{ID: 98, Width: 1, Height: 1}, 0, 2, false); err != nil {
		t.Fatal(err)
	}
	if err := container.Place(Box{ID: 99, Width: 1, Height: 1}, 2, 2, false); err != nil {
		t.Fatal(err)
	}

	// The original 1x2 footprint fits at (1, 1). Its rotated 2x1 footprint
	// fits on the same row at (0, 1), so bottom-left selects the rotation.
	decision, found := BottomLeftPolicy{}.FindPlacement(container, Box{
		ID:        1,
		Width:     1,
		Height:    2,
		CanRotate: true,
	})

	if !found {
		t.Fatal("FindPlacement() found no placement; want the leftmost rotated orientation")
	}
	want := PlacementDecision{Point: Point{X: 0, Y: 1}, Rotated: true}
	if decision != want {
		t.Errorf("FindPlacement() = %+v; want %+v", decision, want)
	}
}

func TestLargestAreaBottomLeftChangesPlacementOrder(t *testing.T) {
	// The small box fits first at (0, 1), forcing the large box to (1, 1).
	// When the large box is placed first, it gets (0, 1), and the small box
	// moves to (2, 1).
	batch := []QueuedBox{
		{Box: Box{ID: 1, Width: 1, Height: 2}}, // area 2
		{Box: Box{ID: 2, Width: 2, Height: 2}}, // area 4
	}

	fifoWorld := runPolicyBatch(t, BottomLeftPolicy{}, batch)
	largestFirstWorld := runPolicyBatch(t, LargestAreaBottomLeftPolicy{}, batch)

	assertBoxPlacement(t, fifoWorld, 1, Point{X: 0, Y: 1})
	assertBoxPlacement(t, fifoWorld, 2, Point{X: 1, Y: 1})
	assertBoxPlacement(t, largestFirstWorld, 2, Point{X: 0, Y: 1})
	assertBoxPlacement(t, largestFirstWorld, 1, Point{X: 2, Y: 1})
}

func TestLargestAreaBottomLeftMatchesFIFOWhenBatchIsAlreadySorted(t *testing.T) {
	batch := []QueuedBox{
		{Box: Box{ID: 2, Width: 2, Height: 2}}, // area 4
		{Box: Box{ID: 1, Width: 1, Height: 2}}, // area 2
	}

	fifoWorld := runPolicyBatch(t, BottomLeftPolicy{}, batch)
	largestFirstWorld := runPolicyBatch(t, LargestAreaBottomLeftPolicy{}, batch)

	if !reflect.DeepEqual(fifoWorld.Container.placements, largestFirstWorld.Container.placements) {
		t.Errorf(
			"already sorted batch produced different placements: FIFO = %v, largest-first = %v",
			fifoWorld.Container.placements,
			largestFirstWorld.Container.placements,
		)
	}
}

func runPolicyBatch(t *testing.T, policy Policy, batch []QueuedBox) *World {
	t.Helper()

	world, err := NewWorld(3, 4, len(batch))
	if err != nil {
		t.Fatal(err)
	}

	engine := &SimulationEngine{world: world}
	batchCopy := append([]QueuedBox(nil), batch...)
	// not testing for amount rotated herer
	placed, _, err := engine.processBatch(policy, batchCopy)
	if err != nil {
		t.Fatal(err)
	}
	if placed != len(batch) {
		t.Fatalf("processBatch() placed %d boxes; want %d", placed, len(batch))
	}

	return world
}

func assertBoxPlacement(t *testing.T, world *World, boxID int, want Point) {
	t.Helper()

	got, exists := world.Container.placements[boxID]
	if !exists {
		t.Fatalf("box %d was not placed", boxID)
	}
	if got.X != want.X || got.Y != want.Y {
		t.Errorf("box %d placed at (%d, %d); want (%d, %d)", boxID, got.X, got.Y, want.X, want.Y)
	}
}

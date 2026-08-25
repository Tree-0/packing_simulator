package policy

import (
	"reflect"
	"testing"

	"packing_simulator/backend"
)

func TestLargestAreaBottomLeftOrderBatch(t *testing.T) {
	batch := []backend.QueuedBox{
		{Box: backend.Box{ID: 1, Width: 1, Height: 1}}, // area 1
		{Box: backend.Box{ID: 2, Width: 2, Height: 2}}, // area 4
		{Box: backend.Box{ID: 3, Width: 1, Height: 3}}, // area 3
		{Box: backend.Box{ID: 4, Width: 2, Height: 2}}, // area 4
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
	container, err := backend.NewContainer(1, 2)
	if err != nil {
		t.Fatal(err)
	}

	// rotating is the only way to fit into the 1x2 container
	decision, found := BottomLeftPolicy{}.FindPlacement(container, backend.Box{
		ID:        1,
		Width:     1,
		Height:    2,
		CanRotate: true,
	})

	if !found {
		t.Fatal("FindPlacement() found no placement; want the rotated orientation")
	}
	want := backend.PlacementDecision{Point: backend.Point{X: 0, Y: 0}, Rotated: true}
	if decision != want {
		t.Errorf("FindPlacement() = %+v; want %+v", decision, want)
	}
}

func TestSimulationRecordsRotatedPlacement(t *testing.T) {
	engine, err := backend.NewSimulationEngine(backend.SimulationConfig{
		ContainerHeight:  2,
		ContainerWidth:   3,
		QueueSize:        1,
		MinBoxHeight:     2,
		MaxBoxHeight:     2,
		MinBoxWidth:      1,
		MaxBoxWidth:      1,
		Seed:             1,
		AllowBoxRotation: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := engine.Run(BottomLeftPolicy{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Placed != 1 || result.Rotated != 1 {
		t.Fatalf("Run() result = %+v; want one placed and one rotated box", result)
	}

	container := &engine.World().Container
	for x := 0; x < 2; x++ {
		cell, err := container.Cell(x, 1)
		if err != nil {
			t.Fatal(err)
		}
		if cell != 1 {
			t.Errorf("Cell(%d, 1) = %d; want rotated box ID 1", x, cell)
		}
	}
}

func TestBottomLeftPolicyChoosesOriginalOrientationWhenItIsLower(t *testing.T) {
	container, err := backend.NewContainer(3, 3)
	if err != nil {
		t.Fatal(err)
	}

	decision, found := BottomLeftPolicy{}.FindPlacement(container, backend.Box{
		ID:        1,
		Width:     2,
		Height:    1,
		CanRotate: true,
	})

	if !found {
		t.Fatal("FindPlacement() found no placement; want the original orientation")
	}
	want := backend.PlacementDecision{Point: backend.Point{X: 0, Y: 2}, Rotated: false}
	if decision != want {
		t.Errorf("FindPlacement() = %+v; want %+v", decision, want)
	}
}

func TestBottomLeftPolicyChoosesRotatedOrientationWhenItIsLower(t *testing.T) {
	container, err := backend.NewContainer(3, 3)
	if err != nil {
		t.Fatal(err)
	}

	decision, found := BottomLeftPolicy{}.FindPlacement(container, backend.Box{
		ID:        1,
		Width:     1,
		Height:    2,
		CanRotate: true,
	})

	if !found {
		t.Fatal("FindPlacement() found no placement; want the rotated orientation")
	}
	want := backend.PlacementDecision{Point: backend.Point{X: 0, Y: 2}, Rotated: true}
	if decision != want {
		t.Errorf("FindPlacement() = %+v; want %+v", decision, want)
	}
}

func TestBottomLeftPolicyPrefersOriginalOrientationOnTie(t *testing.T) {
	container, err := backend.NewContainer(3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := container.Place(backend.Box{ID: 99, Width: 1, Height: 1}, 1, 2, false); err != nil {
		t.Fatal(err)
	}

	decision, found := BottomLeftPolicy{}.FindPlacement(container, backend.Box{
		ID:        1,
		Width:     2,
		Height:    1,
		CanRotate: true,
	})

	if !found {
		t.Fatal("FindPlacement() found no placement; want an orientation tie")
	}
	want := backend.PlacementDecision{Point: backend.Point{X: 0, Y: 1}, Rotated: false}
	if decision != want {
		t.Errorf("FindPlacement() = %+v; want %+v", decision, want)
	}
}

func TestBottomLeftPolicyChoosesOriginalOrientationWhenItIsLeftmost(t *testing.T) {
	container, err := backend.NewContainer(3, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := container.Place(backend.Box{ID: 98, Width: 1, Height: 1}, 0, 2, false); err != nil {
		t.Fatal(err)
	}
	if err := container.Place(backend.Box{ID: 99, Width: 1, Height: 1}, 2, 2, false); err != nil {
		t.Fatal(err)
	}

	// The original 2x1 footprint fits at (0, 1). Its rotated 1x2 footprint
	// fits on the same row, but only at (1, 1).
	decision, found := BottomLeftPolicy{}.FindPlacement(container, backend.Box{
		ID:        1,
		Width:     2,
		Height:    1,
		CanRotate: true,
	})

	if !found {
		t.Fatal("FindPlacement() found no placement; want the leftmost original orientation")
	}
	want := backend.PlacementDecision{Point: backend.Point{X: 0, Y: 1}, Rotated: false}
	if decision != want {
		t.Errorf("FindPlacement() = %+v; want %+v", decision, want)
	}
}

func TestBottomLeftPolicyChoosesRotatedOrientationWhenItIsLeftmost(t *testing.T) {
	container, err := backend.NewContainer(3, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := container.Place(backend.Box{ID: 98, Width: 1, Height: 1}, 0, 2, false); err != nil {
		t.Fatal(err)
	}
	if err := container.Place(backend.Box{ID: 99, Width: 1, Height: 1}, 2, 2, false); err != nil {
		t.Fatal(err)
	}

	// The original 1x2 footprint fits at (1, 1). Its rotated 2x1 footprint
	// fits on the same row at (0, 1), so bottom-left selects the rotation.
	decision, found := BottomLeftPolicy{}.FindPlacement(container, backend.Box{
		ID:        1,
		Width:     1,
		Height:    2,
		CanRotate: true,
	})

	if !found {
		t.Fatal("FindPlacement() found no placement; want the leftmost rotated orientation")
	}
	want := backend.PlacementDecision{Point: backend.Point{X: 0, Y: 1}, Rotated: true}
	if decision != want {
		t.Errorf("FindPlacement() = %+v; want %+v", decision, want)
	}
}

func TestLargestAreaBottomLeftChangesPlacementOrder(t *testing.T) {
	// The small box fits first at (0, 1), forcing the large box to (1, 1).
	// When the large box is placed first, it gets (0, 1), and the small box
	// moves to (2, 1).
	batch := []backend.QueuedBox{
		{Box: backend.Box{ID: 1, Width: 1, Height: 2}}, // area 2
		{Box: backend.Box{ID: 2, Width: 2, Height: 2}}, // area 4
	}

	fifoContainer := runPolicyBatch(t, BottomLeftPolicy{}, batch)
	largestFirstContainer := runPolicyBatch(t, LargestAreaBottomLeftPolicy{}, batch)

	assertBoxPlacement(t, fifoContainer, 1, backend.Point{X: 0, Y: 1})
	assertBoxPlacement(t, fifoContainer, 2, backend.Point{X: 1, Y: 1})
	assertBoxPlacement(t, largestFirstContainer, 2, backend.Point{X: 0, Y: 1})
	assertBoxPlacement(t, largestFirstContainer, 1, backend.Point{X: 2, Y: 1})
}

func TestLargestAreaBottomLeftMatchesFIFOWhenBatchIsAlreadySorted(t *testing.T) {
	batch := []backend.QueuedBox{
		{Box: backend.Box{ID: 2, Width: 2, Height: 2}}, // area 4
		{Box: backend.Box{ID: 1, Width: 1, Height: 2}}, // area 2
	}

	fifoContainer := runPolicyBatch(t, BottomLeftPolicy{}, batch)
	largestFirstContainer := runPolicyBatch(t, LargestAreaBottomLeftPolicy{}, batch)

	fifoCells := containerCells(t, fifoContainer)
	largestFirstCells := containerCells(t, largestFirstContainer)
	if !reflect.DeepEqual(fifoCells, largestFirstCells) {
		t.Errorf(
			"already sorted batch produced different placements: FIFO = %v, largest-first = %v",
			fifoCells,
			largestFirstCells,
		)
	}
}

func runPolicyBatch(t *testing.T, p backend.Policy, batch []backend.QueuedBox) *backend.Container {
	t.Helper()

	container, err := backend.NewContainer(3, 4)
	if err != nil {
		t.Fatal(err)
	}

	batchCopy := append([]backend.QueuedBox(nil), batch...)
	p.OrderBatch(batchCopy)
	placed := 0
	for _, queued := range batchCopy {
		decision, found := p.FindPlacement(container, queued.Box)
		if !found {
			continue
		}

		box := queued.Box
		if decision.Rotated {
			box, err = box.TryRotate()
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := container.Place(box, decision.Point.X, decision.Point.Y, decision.Rotated); err != nil {
			t.Fatal(err)
		}
		placed++
	}
	if placed != len(batchCopy) {
		t.Fatalf("policy placed %d boxes; want %d", placed, len(batchCopy))
	}

	return container
}

func containerCells(t *testing.T, container *backend.Container) [][]int {
	t.Helper()

	cells := make([][]int, container.Height())
	for y := 0; y < container.Height(); y++ {
		cells[y] = make([]int, container.Width())
		for x := 0; x < container.Width(); x++ {
			cell, err := container.Cell(x, y)
			if err != nil {
				t.Fatal(err)
			}
			cells[y][x] = cell
		}
	}
	return cells
}

func assertBoxPlacement(t *testing.T, container *backend.Container, boxID int, want backend.Point) {
	t.Helper()

	for y := 0; y < container.Height(); y++ {
		for x := 0; x < container.Width(); x++ {
			cell, err := container.Cell(x, y)
			if err != nil {
				t.Fatal(err)
			}
			if cell != boxID {
				continue
			}

			got := backend.Point{X: x, Y: y}
			if got != want {
				t.Errorf("box %d placed at (%d, %d); want (%d, %d)", boxID, got.X, got.Y, want.X, want.Y)
			}
			return
		}
	}

	t.Fatalf("box %d was not placed", boxID)
}

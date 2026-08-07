package backend

import (
	"testing"
)

func TestCanPlace(t *testing.T) {
	container, err := NewContainer(4, 5)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		box  Box
		x    int
		y    int
		want bool
	}{
		{name: "fits", box: Box{ID: 1, Height: 2, Width: 3}, x: 2, y: 2, want: true},
		{name: "negative x", box: Box{ID: 1, Height: 1, Width: 1}, x: -1, y: 0, want: false},
		{name: "past right edge", box: Box{ID: 1, Height: 1, Width: 2}, x: 4, y: 0, want: false},
		{name: "past bottom edge", box: Box{ID: 1, Height: 2, Width: 1}, x: 0, y: 3, want: false},
		{name: "empty ID", box: Box{ID: EmptyCell, Height: 1, Width: 1}, x: 0, y: 0, want: false},
		{name: "zero width", box: Box{ID: 1, Height: 1, Width: 0}, x: 0, y: 0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := container.CanPlace(tt.box, tt.x, tt.y); got != tt.want {
				t.Errorf("CanPlace(%+v, %d, %d) = %v; want %v", tt.box, tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestPlaceFillsCellsAndPreventsOverlap(t *testing.T) {
	container, err := NewContainer(4, 5)
	if err != nil {
		t.Fatal(err)
	}

	box := Box{ID: 7, Height: 2, Width: 3}
	if err := container.Place(box, 1, 1); err != nil {
		t.Fatalf("Place() returned an unexpected error: %v", err)
	}

	for y := 1; y < 3; y++ {
		for x := 1; x < 4; x++ {
			if got := container.cells[y][x]; got != box.ID {
				t.Errorf("cells[%d][%d] = %d; want %d", y, x, got, box.ID)
			}
		}
	}

	if container.CanPlace(Box{ID: 8, Height: 1, Width: 1}, 2, 2) {
		t.Error("CanPlace() allowed a box to overlap occupied cells")
	}

	placement, exists := container.placements[box.ID]
	if !exists {
		t.Fatal("Place() did not record the placement")
	}
	if placement != (BoxPlacement{BoxID: box.ID, X: 1, Y: 1}) {
		t.Errorf("placement = %+v; want BoxID 7 at (1, 1)", placement)
	}
}

func TestCanFitDimensions(t *testing.T) {
	container, err := NewContainer(3, 3)
	if err != nil {
		t.Fatal(err)
	}

	if !container.CanFitDimensions(3, 3) {
		t.Error("an empty container should fit its full dimensions")
	}

	if err := container.Place(Box{ID: 1, Width: 1, Height: 1}, 1, 1); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		width, height int
		want          bool
	}{
		{name: "single cell", width: 1, height: 1, want: true},
		{name: "vertical gap", width: 1, height: 3, want: true},
		{name: "center prevents every two by two square", width: 2, height: 2, want: false},
		{name: "too wide", width: 4, height: 1, want: false},
		{name: "zero width", width: 0, height: 1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := container.CanFitDimensions(tt.width, tt.height); got != tt.want {
				t.Errorf("CanFitDimensions(%d, %d) = %v; want %v", tt.width, tt.height, got, tt.want)
			}
		})
	}
}

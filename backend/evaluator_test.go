package backend

import (
	"math"
	"testing"
)

func TestFragmentationEvaluator(t *testing.T) {
	world, _ := NewWorld(3, 3, 1)
	container := world.Container

	// create container and populate
	points := []Point{
		{X: 0, Y: 1},
		{X: 1, Y: 1},
		{X: 2, Y: 1},
		{X: 1, Y: 2},
	}
	for i := 1; i <= len(points); i++ {
		box := Box{ID: i, Height: 1, Width: 1}
		container.Place(box, points[i-1].X, points[i-1].Y)
	}

	fragmentation := Fragmentation(world)

	// 0 1 0
	// 0 1 1 
	// 0 1 0

	// 1 - (3^2 + 1^2 + 1^2) / (5 ^ 2)
	// 1 - (11 / 25)
	// 0.56

	expected := 0.56
	if math.Abs(fragmentation.FragmentationScore-expected) > 1e-9 {
		t.Fatalf("expected fragmentation score %.2f, got %.2f", expected, fragmentation.FragmentationScore)
	}
}

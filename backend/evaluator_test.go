package backend

import (
	"math"
	"testing"
)

func TestFragmentationEvaluator(t *testing.T) {
	world := newTestWorld(t, 3, 3)
	container := &world.Container

	// create container and populate
	points := []Point{
		{X: 0, Y: 1},
		{X: 1, Y: 1},
		{X: 2, Y: 1},
		{X: 1, Y: 2},
	}
	for i := 1; i <= len(points); i++ {
		box := Box{ID: i, Height: 1, Width: 1}
		if err := container.Place(box, points[i-1].X, points[i-1].Y, false); err != nil {
			t.Fatalf("Place() returned an unexpected error: %v", err)
		}
	}

	// 0 0 0
	// 1 1 1
	// 0 1 0

	// 1 - (3^2 + 1^2 + 1^2) / (5 ^ 2)
	// 1 - (11 / 25)
	// 0.56
	assertFragmentationMetrics(t, Fragmentation(world), FragmentationMetrics{
		RegionCount:        3,
		LargestRegionRatio: 3.0 / 5.0,
		FragmentationScore: 0.56,
	})
}

func TestFragmentationEvaluatorLarge(t *testing.T) {
	const (
		height = 100
		width  = 120
	)

	world := newTestWorld(t, height, width)
	container := &world.Container
	for i, x := range []int{40, 80} {
		wall := Box{ID: i + 1, Height: height, Width: 1}
		if err := container.Place(wall, x, 0, false); err != nil {
			t.Fatalf("Place() returned an unexpected error: %v", err)
		}
	}

	// The two occupied columns leave regions 4,000, 3,900, and 3,900
	// cells in size.
	const emptyCells = 11800.0
	assertFragmentationMetrics(t, Fragmentation(world), FragmentationMetrics{
		RegionCount:        3,
		LargestRegionRatio: 4000.0 / emptyCells,
		FragmentationScore: 1 - (4000.0*4000.0+3900.0*3900.0+3900.0*3900.0)/(emptyCells*emptyCells),
	})
}

func TestFragmentationEvaluatorEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		world func(t *testing.T) *World
		want  FragmentationMetrics
	}{
		{
			name: "empty container is one region",
			world: func(t *testing.T) *World {
				return newTestWorld(t, 4, 5)
			},
			want: FragmentationMetrics{
				RegionCount:        1,
				LargestRegionRatio: 1,
				FragmentationScore: 0,
			},
		},
		{
			name: "fully occupied container has no regions",
			world: func(t *testing.T) *World {
				world := newTestWorld(t, 4, 5)
				box := Box{ID: 1, Height: 4, Width: 5}
				if err := world.Container.Place(box, 0, 0, false); err != nil {
					t.Fatalf("Place() returned an unexpected error: %v", err)
				}
				return world
			},
			want: FragmentationMetrics{},
		},
		{
			name: "zero-sized container has no regions",
			world: func(t *testing.T) *World {
				return &World{}
			},
			want: FragmentationMetrics{},
		},
		{
			name: "diagonal empty cells are separate regions",
			world: func(t *testing.T) *World {
				world := newTestWorld(t, 3, 3)
				boxID := 1
				for y := 0; y < 3; y++ {
					for x := 0; x < 3; x++ {
						if x == y {
							continue
						}
						box := Box{ID: boxID, Height: 1, Width: 1}
						if err := world.Container.Place(box, x, y, false); err != nil {
							t.Fatalf("Place() returned an unexpected error: %v", err)
						}
						boxID++
					}
				}
				return world
			},
			want: FragmentationMetrics{
				RegionCount:        3,
				LargestRegionRatio: 1.0 / 3.0,
				FragmentationScore: 2.0 / 3.0,
			},
		},
		{
			name: "nil world has no regions",
			world: func(t *testing.T) *World {
				return nil
			},
			want: FragmentationMetrics{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertFragmentationMetrics(t, Fragmentation(tt.world(t)), tt.want)
		})
	}
}

func TestFutureFitProbability(t *testing.T) {
	world := newTestWorld(t, 3, 3)
	if err := world.Container.Place(Box{ID: 1, Width: 1, Height: 1}, 1, 1, false); err != nil {
		t.Fatal(err)
	}

	got := FutureFitProbability(world, UniformBoxDistribution{
		MinWidth:  1,
		MaxWidth:  2,
		MinHeight: 1,
		MaxHeight: 2,
	})

	// Sizes 1x1, 1x2, and 2x1 fit. Every 2x2 square includes the center cell.
	const want = 3.0 / 4.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("FutureFitProbability() = %.12f; want %.12f", got, want)
	}
}

func newTestWorld(t *testing.T, height, width int) *World {
	t.Helper()

	world, err := NewWorld(height, width, 1)
	if err != nil {
		t.Fatalf("NewWorld() returned an unexpected error: %v", err)
	}
	return world
}

func assertFragmentationMetrics(t *testing.T, got, want FragmentationMetrics) {
	t.Helper()

	if got.RegionCount != want.RegionCount {
		t.Errorf("RegionCount = %d; want %d", got.RegionCount, want.RegionCount)
	}
	if math.Abs(got.LargestRegionRatio-want.LargestRegionRatio) > 1e-9 {
		t.Errorf("LargestRegionRatio = %.12f; want %.12f", got.LargestRegionRatio, want.LargestRegionRatio)
	}
	if math.Abs(got.FragmentationScore-want.FragmentationScore) > 1e-9 {
		t.Errorf("FragmentationScore = %.12f; want %.12f", got.FragmentationScore, want.FragmentationScore)
	}
}

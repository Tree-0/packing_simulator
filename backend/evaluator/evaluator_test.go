package evaluator

import (
	"math"
	"testing"

	"packing_simulator/backend"
)

func TestFragmentation(t *testing.T) {
	container := newTestContainer(t, 3, 3, 1)
	points := []backend.Point{{X: 0, Y: 1}, {X: 1, Y: 1}, {X: 2, Y: 1}, {X: 1, Y: 2}}
	placePoints(t, container, points)

	assertFragmentationMetrics(t, Fragmentation(container), FragmentationMetrics{
		RegionCount:        3,
		LargestRegionRatio: 3.0 / 5.0,
		FragmentationScore: 0.56,
		EmptyCells:         5,
	})
}

func TestFragmentationEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		container func(t *testing.T) *backend.Container
		want      FragmentationMetrics
	}{
		{
			name: "empty container is one region",
			container: func(t *testing.T) *backend.Container {
				return newTestContainer(t, 4, 5, 1)
			},
			want: FragmentationMetrics{RegionCount: 1, LargestRegionRatio: 1, EmptyCells: 20},
		},
		{
			name: "fully occupied container has no regions",
			container: func(t *testing.T) *backend.Container {
				container := newTestContainer(t, 2, 3, 1)
				if err := container.Place(backend.Box{ID: 1, Height: 2, Width: 3}, 0, 0, false); err != nil {
					t.Fatal(err)
				}
				return container
			},
			want: FragmentationMetrics{},
		},
		{
			name: "nil container has no regions",
			container: func(t *testing.T) *backend.Container {
				return nil
			},
			want: FragmentationMetrics{},
		},
		{
			name: "diagonal empty cells are separate regions",
			container: func(t *testing.T) *backend.Container {
				container := newTestContainer(t, 3, 3, 1)
				boxID := 1
				for y := 0; y < 3; y++ {
					for x := 0; x < 3; x++ {
						if x == y {
							continue
						}
						if err := container.Place(backend.Box{ID: boxID, Height: 1, Width: 1}, x, y, false); err != nil {
							t.Fatal(err)
						}
						boxID++
					}
				}
				return container
			},
			want: FragmentationMetrics{RegionCount: 3, LargestRegionRatio: 1.0 / 3.0, FragmentationScore: 2.0 / 3.0, EmptyCells: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertFragmentationMetrics(t, Fragmentation(tt.container(t)), tt.want)
		})
	}
}

func TestFutureFitProbability(t *testing.T) {
	container := newTestContainer(t, 3, 3, 1)
	if err := container.Place(backend.Box{ID: 1, Width: 1, Height: 1}, 1, 1, false); err != nil {
		t.Fatal(err)
	}

	got := FutureFitProbability(container, backend.UniformBoxDistribution{
		MinWidth: 1, MaxWidth: 2, MinHeight: 1, MaxHeight: 2,
	})
	assertFloat(t, got, 3.0/4.0, "FutureFitProbability")
}

func TestWorldMetricsUseOnlyUsedContainers(t *testing.T) {
	world := newTestWorld(t, 3, 3)
	first := world.Containers[0]
	placePoints(t, first, []backend.Point{{X: 0, Y: 1}, {X: 1, Y: 1}, {X: 2, Y: 1}, {X: 1, Y: 2}})

	second, err := world.NewContainer(1, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Place(backend.Box{ID: 5, Width: 1, Height: 1}, 0, 0, false); err != nil {
		t.Fatal(err)
	}
	if _, err := world.NewContainer(10, 10); err != nil {
		t.Fatal(err)
	}

	if got := UsedContainerCount(world); got != 2 {
		t.Errorf("UsedContainerCount() = %d; want 2", got)
	}
	assertFloat(t, WorldUtilization(world), 5.0/13.0, "WorldUtilization")
	assertFloat(t, MeanFragmentation(world), 0.28, "MeanFragmentation")
	assertFloat(t, WorldAreaWeightedFragmentation(world), 2.8/13.0, "WorldAreaWeightedFragmentation")
	assertFloat(t, WorldCompactness(world), 1-2.8/13.0, "WorldCompactness")
	assertFloat(t, EvaluateWorld(world, BinCount), 2, "BinCount")
	assertFloat(t, EvaluateWorld(world, ContainerUtilization), 5.0/13.0, "EvaluateWorld utilization")
}

func TestWorldFutureFitProbabilityUsesAnyUsedContainer(t *testing.T) {
	world := newTestWorld(t, 2, 2)
	first := world.Containers[0]
	if err := first.Place(backend.Box{ID: 1, Width: 2, Height: 2}, 0, 0, false); err != nil {
		t.Fatal(err)
	}

	second, err := world.NewContainer(2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Place(backend.Box{ID: 2, Width: 1, Height: 1}, 0, 0, false); err != nil {
		t.Fatal(err)
	}

	distribution := backend.UniformBoxDistribution{MinWidth: 2, MaxWidth: 2, MinHeight: 1, MaxHeight: 1}
	assertFloat(t, FutureFitProbability(first, distribution), 0, "first container future fit")
	assertFloat(t, FutureFitProbability(second, distribution), 1, "second container future fit")
	assertFloat(t, WorldFutureFitProbability(world, distribution), 1, "world future fit")
}

func TestEvaluateContainerMetrics(t *testing.T) {
	world := newTestWorld(t, 2, 2)
	first := world.Containers[0]
	if err := first.Place(backend.Box{ID: 1, Width: 1, Height: 1}, 0, 0, false); err != nil {
		t.Fatal(err)
	}
	second, err := world.NewContainer(2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Place(backend.Box{ID: 2, Width: 1, Height: 1}, 1, 1, false); err != nil {
		t.Fatal(err)
	}
	if _, err := world.NewContainer(1, 1); err != nil {
		t.Fatal(err)
	}

	distribution := backend.UniformBoxDistribution{MinWidth: 1, MaxWidth: 2, MinHeight: 1, MaxHeight: 2}
	metrics := EvaluateContainerMetrics(world, distribution)
	if len(metrics) != 2 {
		t.Fatalf("EvaluateContainerMetrics() returned %d entries; want 2", len(metrics))
	}
	for i, container := range []*backend.Container{first, second} {
		got := metrics[i]
		if got.ContainerID != container.Id() {
			t.Errorf("metrics[%d].ContainerID = %d; want %d", i, got.ContainerID, container.Id())
		}
		assertFloat(t, got.Utilization, Utilization(container), "container utilization")
		assertFragmentationMetrics(t, got.Fragmentation, Fragmentation(container))
		assertFloat(t, got.AreaWeightedFragmentation, AreaWeightedFragmentation(container), "container area-weighted fragmentation")
		assertFloat(t, got.Compactness, Compactness(container), "container compactness")
		assertFloat(t, got.FutureFitProbability, FutureFitProbability(container, distribution), "container future fit")
	}
}

func TestWorldMetricsWithoutUsedContainersAreZero(t *testing.T) {
	world := newTestWorld(t, 2, 2)
	if got := EvaluateContainerMetrics(world, backend.UniformBoxDistribution{MinWidth: 1, MaxWidth: 1, MinHeight: 1, MaxHeight: 1}); len(got) != 0 {
		t.Errorf("EvaluateContainerMetrics() = %v; want no entries", got)
	}
	assertFloat(t, WorldUtilization(world), 0, "WorldUtilization")
	assertFloat(t, MeanFragmentation(world), 0, "MeanFragmentation")
	assertFloat(t, WorldAreaWeightedFragmentation(world), 0, "WorldAreaWeightedFragmentation")
	assertFloat(t, WorldCompactness(world), 0, "WorldCompactness")
	assertFloat(t, WorldFutureFitProbability(world, backend.UniformBoxDistribution{MinWidth: 1, MaxWidth: 1, MinHeight: 1, MaxHeight: 1}), 0, "WorldFutureFitProbability")
}

func TestBinCountNameAndParsing(t *testing.T) {
	if got := BinCount.String(); got != "Used bin count" {
		t.Errorf("BinCount.String() = %q; want %q", got, "Used bin count")
	}
	got, err := ParseEvaluation("bin-count")
	if err != nil {
		t.Fatal(err)
	}
	if got != BinCount {
		t.Errorf("ParseEvaluation(\"bin-count\") = %v; want %v", got, BinCount)
	}
}

func newTestContainer(t *testing.T, height, width, id int) *backend.Container {
	t.Helper()
	container, err := backend.NewContainer(height, width, id)
	if err != nil {
		t.Fatal(err)
	}
	return container
}

func newTestWorld(t *testing.T, height, width int) *backend.World {
	t.Helper()
	world, err := backend.NewWorld(height, width, 1)
	if err != nil {
		t.Fatal(err)
	}
	return world
}

func placePoints(t *testing.T, container *backend.Container, points []backend.Point) {
	t.Helper()
	for i, point := range points {
		if err := container.Place(backend.Box{ID: i + 1, Height: 1, Width: 1}, point.X, point.Y, false); err != nil {
			t.Fatal(err)
		}
	}
}

func assertFragmentationMetrics(t *testing.T, got, want FragmentationMetrics) {
	t.Helper()
	if got.RegionCount != want.RegionCount {
		t.Errorf("RegionCount = %d; want %d", got.RegionCount, want.RegionCount)
	}
	assertFloat(t, got.LargestRegionRatio, want.LargestRegionRatio, "LargestRegionRatio")
	assertFloat(t, got.FragmentationScore, want.FragmentationScore, "FragmentationScore")
	if got.EmptyCells != want.EmptyCells {
		t.Errorf("EmptyCells = %d; want %d", got.EmptyCells, want.EmptyCells)
	}
}

func assertFloat(t *testing.T, got, want float64, name string) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %.12f; want %.12f", name, got, want)
	}
}

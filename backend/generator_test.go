package backend

import (
	"reflect"
	"testing"
)

func TestRandomBoxGeneratorNextN(t *testing.T) {
	generator, err := NewRandomBoxGenerator(42, 2, 2, 3, 3, true)
	if err != nil {
		t.Fatal(err)
	}

	workload, err := generator.NextN(7, 3)
	if err != nil {
		t.Fatalf("NextN() returned an unexpected error: %v", err)
	}
	if workload == nil {
		t.Fatal("NextN() returned a nil workload")
	}
	if workload.ID != 1 {
		t.Errorf("workload ID = %d; want 1", workload.ID)
	}
	if len(workload.Arrivals) != 3 {
		t.Fatalf("arrival count = %d; want 3", len(workload.Arrivals))
	}

	for i, arrival := range workload.Arrivals {
		if arrival.Box.ID != i+1 {
			t.Errorf("arrival %d box ID = %d; want %d", i, arrival.Box.ID, i+1)
		}
		if arrival.ArrivedAt != 7 {
			t.Errorf("arrival %d timestamp = %d; want 7", i, arrival.ArrivedAt)
		}
		if arrival.Box.Width != 2 || arrival.Box.Height != 3 {
			t.Errorf("arrival %d dimensions = %dx%d; want 2x3", i, arrival.Box.Width, arrival.Box.Height)
		}
		if !arrival.Box.CanRotate {
			t.Errorf("arrival %d cannot rotate; want rotation enabled", i)
		}
	}
}

func TestRandomBoxGeneratorNextNAdvancesGeneratorState(t *testing.T) {
	generator, err := NewRandomBoxGenerator(42, 1, 1, 1, 1, false)
	if err != nil {
		t.Fatal(err)
	}

	first, err := generator.NextN(0, 2)
	if err != nil {
		t.Fatal(err)
	}
	second, err := generator.NextN(10, 1)
	if err != nil {
		t.Fatal(err)
	}

	if first.ID != 1 || second.ID != 2 {
		t.Errorf("workload IDs = (%d, %d); want (1, 2)", first.ID, second.ID)
	}
	if got := second.Arrivals[0].Box.ID; got != 3 {
		t.Errorf("second workload box ID = %d; want 3", got)
	}
}

func TestRandomBoxGeneratorNextNZeroBoxes(t *testing.T) {
	generator, err := NewRandomBoxGenerator(42, 1, 1, 1, 1, false)
	if err != nil {
		t.Fatal(err)
	}

	workload, err := generator.NextN(5, 0)
	if err != nil {
		t.Fatalf("NextN() returned an unexpected error: %v", err)
	}
	if workload == nil {
		t.Fatal("NextN() returned a nil workload")
	}
	if workload.ID != 1 {
		t.Errorf("workload ID = %d; want 1", workload.ID)
	}
	if len(workload.Arrivals) != 0 {
		t.Errorf("arrival count = %d; want 0", len(workload.Arrivals))
	}

	next := generator.Next(6)
	if next.Box.ID != 1 {
		t.Errorf("box ID after empty workload = %d; want 1", next.Box.ID)
	}
}

func TestRandomBoxGeneratorNextNRejectsNegativeCountWithoutAdvancingState(t *testing.T) {
	generator, err := NewRandomBoxGenerator(42, 1, 1, 1, 1, false)
	if err != nil {
		t.Fatal(err)
	}

	workload, err := generator.NextN(5, -1)
	if err == nil {
		t.Fatal("NextN() returned nil error for a negative box count")
	}
	if workload != nil {
		t.Errorf("NextN() workload = %+v; want nil", workload)
	}

	valid, err := generator.NextN(5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if valid.ID != 1 || valid.Arrivals[0].Box.ID != 1 {
		t.Errorf("state advanced after rejected call: workload = %+v", valid)
	}
}

func TestRandomBoxGeneratorNextNIsDeterministicForSeed(t *testing.T) {
	first, err := NewRandomBoxGenerator(99, 1, 5, 2, 6, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewRandomBoxGenerator(99, 1, 5, 2, 6, false)
	if err != nil {
		t.Fatal(err)
	}

	firstWorkload, err := first.NextN(4, 5)
	if err != nil {
		t.Fatal(err)
	}
	secondWorkload, err := second.NextN(4, 5)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(firstWorkload, secondWorkload) {
		t.Errorf("same seed produced different workloads:\nfirst:  %+v\nsecond: %+v", firstWorkload, secondWorkload)
	}
}

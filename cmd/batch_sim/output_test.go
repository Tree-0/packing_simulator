package main

import (
	"encoding/csv"
	"os"
	"reflect"
	"testing"

	"packing_simulator/backend"
	"packing_simulator/backend/evaluator"
)

func TestWriteExperimentResultsCSV(t *testing.T) {
	paths, err := writeExperimentResultsCSV(
		t.TempDir(),
		[]RunResult{{
			WorkloadName:          "small",
			PolicyName:            "bottom-left",
			ContainerSelectorName: "next-k-fit(k=3)",
			Seed:                  42,
			Simulation: backend.SimulationResult{
				Iterations: 4,
				Generated:  4,
				Placed:     3,
				Rejected:   1,
				Batches:    2,
			},
			Evaluations: []evaluationResult{{
				evaluation: evaluator.ContainerUtilization,
				value:      0.75,
			}},
		}},
		[]AggregateResult{{
			WorkloadName:          "small",
			PolicyName:            "bottom-left",
			ContainerSelectorName: "next-k-fit(k=3)",
			Evaluation: evaluationResult{
				evaluation: evaluator.ContainerUtilization,
				value:      0.75,
			},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("writeExperimentResultsCSV() returned %d paths; want 2", len(paths))
	}

	if got, want := readCSVRecords(t, paths[0]), [][]string{
		{"workload", "policy", "container_selector", "seed", "iterations", "generated", "placed", "rotated", "rejected", "batches", "stopped_early", "evaluation", "value"},
		{"small", "bottom-left", "next-k-fit(k=3)", "42", "4", "4", "3", "0", "1", "2", "false", "Container utilization", "0.75"},
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("run CSV records = %v; want %v", got, want)
	}
	if got, want := readCSVRecords(t, paths[1]), [][]string{
		{"workload", "policy", "container_selector", "evaluation", "mean"},
		{"small", "bottom-left", "next-k-fit(k=3)", "Container utilization", "0.75"},
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("aggregate CSV records = %v; want %v", got, want)
	}
}

func readCSVRecords(t *testing.T, path string) [][]string {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return records
}

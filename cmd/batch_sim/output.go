package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// writeExperimentResultsCSV writes the individual run results and their
// aggregate means to separate CSV files.
func writeExperimentResultsCSV(
	outputDir string,
	runResults []RunResult,
	aggregateResults []AggregateResult,
) ([]string, error) {
	runPath, err := csvOutputPath(outputDir, "experiment-runs")
	if err != nil {
		return nil, err
	}
	aggregatePath, err := csvOutputPath(outputDir, "experiment-aggregates")
	if err != nil {
		return nil, err
	}

	sortedRuns := append([]RunResult(nil), runResults...)
	sort.Slice(sortedRuns, func(i, j int) bool {
		left, right := sortedRuns[i], sortedRuns[j]
		if left.WorkloadName != right.WorkloadName {
			return left.WorkloadName < right.WorkloadName
		}
		if left.PolicyName != right.PolicyName {
			return left.PolicyName < right.PolicyName
		}
		return left.Seed < right.Seed
	})

	if err := writeCSV(runPath, func(writer *csv.Writer) error {
		if err := writer.Write([]string{
			"workload", "policy", "seed", "iterations", "generated", "placed", "rotated",
			"rejected", "batches", "stopped_early", "evaluation", "value",
		}); err != nil {
			return err
		}

		for _, result := range sortedRuns {
			for _, evaluation := range result.Evaluations {
				if err := writer.Write([]string{
					result.WorkloadName,
					result.PolicyName,
					strconv.FormatInt(result.Seed, 10),
					strconv.Itoa(result.Simulation.Iterations),
					strconv.Itoa(result.Simulation.Generated),
					strconv.Itoa(result.Simulation.Placed),
					strconv.Itoa(result.Simulation.Rotated),
					strconv.Itoa(result.Simulation.Rejected),
					strconv.Itoa(result.Simulation.Batches),
					strconv.FormatBool(result.Simulation.StoppedEarly),
					evaluation.evaluation.String(),
					strconv.FormatFloat(evaluation.value, 'g', -1, 64),
				}); err != nil {
					return err
				}
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("write experiment-run CSV: %w", err)
	}

	sortedAggregates := append([]AggregateResult(nil), aggregateResults...)
	sort.Slice(sortedAggregates, func(i, j int) bool {
		left, right := sortedAggregates[i], sortedAggregates[j]
		if left.WorkloadName != right.WorkloadName {
			return left.WorkloadName < right.WorkloadName
		}
		if left.PolicyName != right.PolicyName {
			return left.PolicyName < right.PolicyName
		}
		return left.Evaluation.evaluation < right.Evaluation.evaluation
	})

	if err := writeCSV(aggregatePath, func(writer *csv.Writer) error {
		if err := writer.Write([]string{"workload", "policy", "evaluation", "mean"}); err != nil {
			return err
		}

		for _, result := range sortedAggregates {
			if err := writer.Write([]string{
				result.WorkloadName,
				result.PolicyName,
				result.Evaluation.evaluation.String(),
				strconv.FormatFloat(result.Evaluation.value, 'g', -1, 64),
			}); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("write experiment-aggregate CSV: %w", err)
	}

	return []string{runPath, aggregatePath}, nil
}

func csvOutputPath(outputDir, prefix string) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output directory %q: %w", outputDir, err)
	}

	filename := fmt.Sprintf(
		"%s-%s.csv",
		prefix,
		time.Now().UTC().Format("20060102T150405.000000000Z"),
	)
	return filepath.Join(outputDir, filename), nil
}

func writeCSV(path string, writeRows func(*csv.Writer) error) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %q: %w", path, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writeRows(writer); err != nil {
		return err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}

	return nil
}

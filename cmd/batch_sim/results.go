/*
Runs experiments across multiple workloads and aggregates the results.
Compares policy performance across packing scenarios, evaluation metrics, and
seeds.
*/

package main

import (
	"fmt"
	"sort"

	"packing_simulator/backend"
	"packing_simulator/backend/evaluator"
	"packing_simulator/internal/simconfig"
)

type experimentConfig = simconfig.ExperimentFile
type workloadConfig = simconfig.ExperimentWorkload
type simulationConfig = simconfig.WorkloadSimulation
type containerSelectorConfig = simconfig.ContainerSelectorConfig

type RunResult struct {
	WorkloadName          string
	PolicyName            string
	ContainerSelectorName string
	Seed                  int64
	Simulation            backend.SimulationResult
	Evaluations           []evaluationResult
}

// the unique key used to aggregate RunResults
// into AggregateResults
type AggregateKey struct {
	WorkloadName          string
	PolicyName            string
	ContainerSelectorName string
	EvaluationType        evaluator.EvaluationType
}

// AggregateResult is the mean of RunResults across seeds for one workload,
// placement-policy, container-selector, and evaluation group.
type AggregateResult struct {
	WorkloadName          string
	PolicyName            string
	ContainerSelectorName string
	Evaluation            evaluationResult
}

func loadExperimentConfig(path string) (experimentConfig, error) {
	return simconfig.LoadExperiment(path)
}

// Run multiple defined workloads as part of one experiment, and return the results
// from each individual run. There are (Workloads * Placement Policies *
// Container Selectors * Seeds) distinct runs, all of which receive every
// evaluation type defined in the config.
func runExperiment(config experimentConfig) ([]RunResult, error) {
	specs, err := config.RunSpecs()
	if err != nil {
		return nil, err
	}
	return runSpecs(specs, config.Workers)
}

// Take in all RunResults and produce the aggregated results by (workload,
// placement policy, container selector, evaluation).
func AggregateResults(results []RunResult) ([]AggregateResult, error) {
	if len(results) == 0 {
		return make([]AggregateResult, 0), nil
	}

	// Assign RunResults to buckets we'll use to average results
	aggregateBuckets := make(map[AggregateKey][]RunResult)
	for _, result := range results {
		for _, eval := range result.Evaluations {
			key := AggregateKey{
				WorkloadName:          result.WorkloadName,
				PolicyName:            result.PolicyName,
				ContainerSelectorName: result.ContainerSelectorName,
				EvaluationType:        eval.evaluation,
			}

			aggregateBuckets[key] = append(
				aggregateBuckets[key],
				// append a copy of the RunResult with only the single
				// eval type we care about for this aggregate
				RunResult{
					WorkloadName:          result.WorkloadName,
					PolicyName:            result.PolicyName,
					ContainerSelectorName: result.ContainerSelectorName,
					Seed:                  result.Seed,
					Evaluations:           []evaluationResult{eval},
				},
			)
		}
	}

	// Perform the aggregation
	aggregateResults := make([]AggregateResult, 0, len(aggregateBuckets))
	for key, runResults := range aggregateBuckets {
		if len(runResults) == 0 {
			continue
		}

		var total float64
		total = 0
		for _, runResult := range runResults {
			// We only added one element to these aggregate run-results
			total += runResult.Evaluations[0].value
		}
		average := total / float64(len(runResults))
		aggregateResults = append(
			aggregateResults,
			AggregateResult{
				WorkloadName:          key.WorkloadName,
				PolicyName:            key.PolicyName,
				ContainerSelectorName: key.ContainerSelectorName,
				Evaluation: evaluationResult{
					evaluation: key.EvaluationType,
					value:      average,
				},
			},
		)
	}

	return aggregateResults, nil
}

func PrintRunResults(config experimentConfig, results []RunResult) {
	if len(results) == 0 {
		fmt.Println("No experiment runs were produced.")
		return
	}

	workloadsByName := make(map[string]workloadConfig, len(config.Workloads))
	for _, workload := range config.Workloads {
		workloadsByName[workload.Name] = workload
	}

	sortedResults := append([]RunResult(nil), results...)
	sort.Slice(sortedResults, func(i, j int) bool {
		left, right := sortedResults[i], sortedResults[j]
		if left.WorkloadName != right.WorkloadName {
			return left.WorkloadName < right.WorkloadName
		}
		if left.PolicyName != right.PolicyName {
			return left.PolicyName < right.PolicyName
		}
		if left.ContainerSelectorName != right.ContainerSelectorName {
			return left.ContainerSelectorName < right.ContainerSelectorName
		}
		return left.Seed < right.Seed
	})

	currentWorkload := ""
	for _, result := range sortedResults {
		if result.WorkloadName != currentWorkload {
			if currentWorkload != "" {
				fmt.Println()
			}

			workload, found := workloadsByName[result.WorkloadName]
			if found {
				printWorkload(workload)
			} else {
				fmt.Printf("Workload: %s\n", result.WorkloadName)
			}
			fmt.Println()
			currentWorkload = result.WorkloadName
		}

		fmt.Printf(
			"Placement policy: %s  Container selector: %s  Seed: %d\n",
			result.PolicyName,
			result.ContainerSelectorName,
			result.Seed,
		)
		fmt.Printf(
			"  Iterations: %d, generated: %d, placed: %d, rotated: %d, rejected: %d, batches: %d\n",
			result.Simulation.Iterations,
			result.Simulation.Generated,
			result.Simulation.Placed,
			result.Simulation.Rotated,
			result.Simulation.Rejected,
			result.Simulation.Batches,
		)

		for _, evaluation := range result.Evaluations {
			switch evaluation.evaluation {
			case evaluator.ContainerUtilization, evaluator.FutureFitProbabilityMetric:
				fmt.Printf("  %-24s %.1f%%\n", evaluation.evaluation, 100*evaluation.value)
			default:
				fmt.Printf("  %-24s %.4f\n", evaluation.evaluation, evaluation.value)
			}
		}
		fmt.Println()
	}
}

func printWorkload(workload workloadConfig) {
	fmt.Printf("Workload: %s\n", workload.Name)
	printSimulationConfig(workload.Simulation)
}

func PrintAggregateResults(config experimentConfig, results []AggregateResult) {
	if len(results) == 0 {
		fmt.Println("No aggregate results were produced.")
		return
	}

	workloadsByName := make(map[string]workloadConfig, len(config.Workloads))
	for _, workload := range config.Workloads {
		workloadsByName[workload.Name] = workload
	}

	sortedResults := append([]AggregateResult(nil), results...)
	sort.Slice(sortedResults, func(i, j int) bool {
		left, right := sortedResults[i], sortedResults[j]
		if left.WorkloadName != right.WorkloadName {
			return left.WorkloadName < right.WorkloadName
		}
		if left.PolicyName != right.PolicyName {
			return left.PolicyName < right.PolicyName
		}
		if left.ContainerSelectorName != right.ContainerSelectorName {
			return left.ContainerSelectorName < right.ContainerSelectorName
		}
		return left.Evaluation.evaluation < right.Evaluation.evaluation
	})

	fmt.Println("Aggregate results (mean across seeds):")
	currentWorkload := ""
	currentPolicy := ""
	currentContainerSelector := ""
	for _, result := range sortedResults {
		if result.WorkloadName != currentWorkload {
			if currentWorkload != "" {
				fmt.Println()
			}

			workload, found := workloadsByName[result.WorkloadName]
			if found {
				printWorkload(workload)
			} else {
				fmt.Printf("Workload: %s\n", result.WorkloadName)
			}
			fmt.Println()
			currentWorkload = result.WorkloadName
			currentPolicy = ""
			currentContainerSelector = ""
		}

		if result.PolicyName != currentPolicy {
			fmt.Printf("Placement policy: %s\n", result.PolicyName)
			currentPolicy = result.PolicyName
			currentContainerSelector = ""
		}

		if result.ContainerSelectorName != currentContainerSelector {
			fmt.Printf("Container selector: %s\n", result.ContainerSelectorName)
			currentContainerSelector = result.ContainerSelectorName
		}

		switch result.Evaluation.evaluation {
		case evaluator.ContainerUtilization, evaluator.FutureFitProbabilityMetric:
			fmt.Printf("  %-24s %.1f%%\n", result.Evaluation.evaluation, 100*result.Evaluation.value)
		default:
			fmt.Printf("  %-24s %.4f\n", result.Evaluation.evaluation, result.Evaluation.value)
		}
	}
}

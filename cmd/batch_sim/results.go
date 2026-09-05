/*
Runs batches for multiple different configs and aggregates the results.
Compares how different policies perform across different packing scenarios
and evaluation metrics. Includes averages across all seeds for a batch, as
well as individual-seed comparisons.
*/

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"packing_simulator/backend"
	"packing_simulator/backend/evaluator"
	"packing_simulator/backend/policy"

	"gopkg.in/yaml.v3"
)

// configs to determine how the experiment will be run
type experimentConfig struct {
	Workloads  []workloadConfig `yaml:"workloads"`
	Seeds      []int64          `yaml:"seeds"`
	Policies   []string         `yaml:"policies"`
	Evaluators []string         `yaml:"evaluators"`
	Workers    int              `yaml:"workers"`
}

type workloadConfig struct {
	Name       string                `yaml:"name"`
	Simulation batchSimulationConfig `yaml:"simulation"`
}

type RunResult struct {
	WorkloadName string
	PolicyName   string
	Seed         int64
	Evaluations  []evaluationResult
}

// the unique key used to aggregate RunResults
// into AggregateResults
type AggregateKey struct {
	WorkloadName   string
	PolicyName     string
	EvaluationType evaluator.EvaluationType
}

// An aggregation of RunResult objects across all seeds
// for that (Workload, Policy, Evaluation) group
type AggregateResult struct {
	WorkloadName string
	PolicyName   string
	Evaluation   evaluationResult
}

// loadExperimentConfig reads and validates one experiment configuration file.
// A relative path is interpreted relative to the process's current directory.
func loadExperimentConfig(path string) (experimentConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return experimentConfig{}, fmt.Errorf("open experiment config %q: %w", path, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	var config experimentConfig
	if err := decoder.Decode(&config); err != nil {
		return experimentConfig{}, fmt.Errorf("decode experiment config %q: %w", path, err)
	}

	var extraDocument any
	if err := decoder.Decode(&extraDocument); err != io.EOF {
		if err == nil {
			return experimentConfig{}, fmt.Errorf("decode experiment config %q: multiple YAML documents are not supported", path)
		}
		return experimentConfig{}, fmt.Errorf("decode experiment config %q: %w", path, err)
	}

	if err := config.validate(); err != nil {
		return experimentConfig{}, fmt.Errorf("invalid experiment config %q: %w", path, err)
	}

	return config, nil
}

func (config experimentConfig) validate() error {
	if len(config.Workloads) == 0 {
		return errors.New("at least one workload is required")
	}
	if len(config.Seeds) == 0 {
		return errors.New("at least one seed is required")
	}
	if len(config.Policies) == 0 {
		return errors.New("at least one policy is required")
	}
	if len(config.Evaluators) == 0 {
		return errors.New("at least one evaluator is required")
	}
	if config.Workers < 0 {
		return errors.New("workers cannot be negative")
	}

	workloadNames := make(map[string]struct{}, len(config.Workloads))
	for _, workload := range config.Workloads {
		if workload.Name == "" {
			return errors.New("workload name is required")
		}
		if _, exists := workloadNames[workload.Name]; exists {
			return fmt.Errorf("duplicate workload name %q", workload.Name)
		}
		workloadNames[workload.Name] = struct{}{}

		if workload.Simulation.Iterations < 0 {
			return fmt.Errorf("workload %q: simulation.iterations cannot be negative", workload.Name)
		}
		if _, err := backend.NewSimulationEngine(workload.Simulation.toBackendConfig(config.Seeds[0])); err != nil {
			return fmt.Errorf("workload %q: simulation: %w", workload.Name, err)
		}
	}

	for _, name := range config.Policies {
		if _, err := policy.NewPolicy(name); err != nil {
			return err
		}
	}
	for _, name := range config.Evaluators {
		if _, err := evaluator.ParseEvaluation(name); err != nil {
			return err
		}
	}

	return nil
}

// Run multiple defined batch workloads as part of one experiment, and return the results
// from each individual run. There are (Workloads * Policies * Seeds) distinct runs,
// all of which receive every evaluation type defined in the config.
func runExperiment(config experimentConfig) ([]RunResult, error) {

	// setup the evaluators
	evaluations := make([]evaluator.EvaluationType, len(config.Evaluators))
	for i, name := range config.Evaluators {
		evaluation, err := evaluator.ParseEvaluation(name)
		if err != nil {
			return nil, err
		}
		evaluations[i] = evaluation
	}

	totalSimulations := len(config.Workloads) * len(config.Policies) * len(config.Seeds)
	runResults := make([]RunResult, 0, totalSimulations)

	for _, workload := range config.Workloads {
		batchConfig := batchConfig{
			workload.Simulation,
			config.Seeds,
			config.Policies,
			config.Evaluators,
			config.Workers,
		}

		batchResults, err := runBatch(batchConfig)
		if err != nil {
			return nil, fmt.Errorf("run batch for workload %q failed: %w", workload.Name, err)
		}

		for _, batchResult := range batchResults {
			runResults = append(runResults, RunResult{
				workload.Name,
				batchResult.policy,
				batchResult.seed,
				batchResult.evaluations,
			})
		}
	}

	return runResults, nil
}

// Take in all RunResults and produce the aggregated results
// by (workload, policy, evaluation)
func AggregateResults(results []RunResult) ([]AggregateResult, error) {
	if len(results) == 0 {
		return make([]AggregateResult, 0), nil
	}

	// Assign RunResults to buckets we'll use to average results
	aggregateBuckets := make(map[AggregateKey][]RunResult)
	for _, result := range results {
		for _, eval := range result.Evaluations {
			key := AggregateKey{
				result.WorkloadName,
				result.PolicyName,
				eval.evaluation,
			}

			aggregateBuckets[key] = append(
				aggregateBuckets[key],
				// append a copy of the RunResult with only the single
				// eval type we care about for this aggregate
				RunResult{
					result.WorkloadName,
					result.PolicyName,
					result.Seed,
					[]evaluationResult{eval},
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
				key.WorkloadName,
				key.PolicyName,
				evaluationResult{
					key.EvaluationType,
					average,
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
			"Policy: %s  Seed: %d\n",
			result.PolicyName,
			result.Seed,
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
	printBatchSimulationConfig(workload.Simulation)
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
		return left.Evaluation.evaluation < right.Evaluation.evaluation
	})

	fmt.Println("Aggregate results (mean across seeds):")
	currentWorkload := ""
	currentPolicy := ""
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
		}

		if result.PolicyName != currentPolicy {
			fmt.Printf("Policy: %s\n", result.PolicyName)
			currentPolicy = result.PolicyName
		}

		switch result.Evaluation.evaluation {
		case evaluator.ContainerUtilization, evaluator.FutureFitProbabilityMetric:
			fmt.Printf("  %-24s %.1f%%\n", result.Evaluation.evaluation, 100*result.Evaluation.value)
		default:
			fmt.Printf("  %-24s %.4f\n", result.Evaluation.evaluation, result.Evaluation.value)
		}
	}
}

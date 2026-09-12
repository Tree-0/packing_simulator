/*
Runs experiments across multiple workloads and aggregates the results.
Compares policy performance across packing scenarios, evaluation metrics, and
seeds.
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
	Workloads          []workloadConfig          `yaml:"workloads"`
	MaxContainers      int                       `yaml:"max_containers"`
	Seeds              []int64                   `yaml:"seeds"`
	Policies           []string                  `yaml:"policies"`
	ContainerSelectors []containerSelectorConfig `yaml:"container_selectors"`
	Evaluators         []string                  `yaml:"evaluators"`
	Workers            int                       `yaml:"workers"`
}

// containerSelectorConfig is the experiment equivalent of the shared
// single-run container_selector YAML object. K is used only by Next K Fit.
type containerSelectorConfig struct {
	Name string `yaml:"name"`
	K    int    `yaml:"k"`
}

// effectiveContainerSelectors returns the configured selectors, or the
// historical First Fit default when older experiment configuration omits them.
func (config experimentConfig) effectiveContainerSelectors() []containerSelectorConfig {
	if len(config.ContainerSelectors) == 0 {
		return []containerSelectorConfig{{Name: policy.ContainerSelectorFirstFitName}}
	}
	return config.ContainerSelectors
}

type workloadConfig struct {
	Name       string           `yaml:"name"`
	Simulation simulationConfig `yaml:"simulation"`
}

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
		if _, err := backend.NewSimulationEngine(workload.Simulation.toBackendConfig(config.Seeds[0], config.MaxContainers)); err != nil {
			return fmt.Errorf("workload %q: simulation: %w", workload.Name, err)
		}
	}

	for _, name := range config.Policies {
		if _, err := policy.NewPlacementPolicy(name); err != nil {
			return err
		}
	}
	for _, selector := range config.effectiveContainerSelectors() {
		if _, err := policy.NewContainerSelector(selector.Name, selector.K); err != nil {
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

// Run multiple defined workloads as part of one experiment, and return the results
// from each individual run. There are (Workloads * Placement Policies *
// Container Selectors * Seeds) distinct runs, all of which receive every
// evaluation type defined in the config.
func runExperiment(config experimentConfig) ([]RunResult, error) {

	totalSimulations := len(config.Workloads) * len(config.Policies) * len(config.effectiveContainerSelectors()) * len(config.Seeds)
	runResults := make([]RunResult, 0, totalSimulations)

	for _, workload := range config.Workloads {
		workloadResults, err := runWorkload(workload, config)
		if err != nil {
			return nil, fmt.Errorf("run workload %q: %w", workload.Name, err)
		}
		runResults = append(runResults, workloadResults...)
	}

	return runResults, nil
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

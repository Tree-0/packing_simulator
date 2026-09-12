/*
Run a group of simulations over the configured workload, seed, placement-policy,
and container-selector combinations.
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"

	"packing_simulator/backend"
	"packing_simulator/backend/evaluator"
	"packing_simulator/backend/policy"
	"packing_simulator/internal/simconfig"
)

type evaluationResult struct {
	evaluation evaluator.EvaluationType
	value      float64
}

type jobOutcome struct {
	index  int
	result RunResult
	err    error
}

func main() {
	configPath := flag.String("config", simconfig.DefaultExperimentPath, "path to the experiment YAML config")
	outputDir := flag.String("output-dir", "", "directory in which to write CSV results; omit to disable CSV output")
	flag.Parse()
	if flag.NArg() != 0 {
		log.Fatalf("unexpected positional arguments: %s", strings.Join(flag.Args(), " "))
	}

	config, err := loadExperimentConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	results, err := runExperiment(config)
	if err != nil {
		log.Fatal(err)
	}
	PrintRunResults(config, results)

	aggregates, err := AggregateResults(results)
	if err != nil {
		log.Fatal(err)
	}
	PrintAggregateResults(config, aggregates)

	if *outputDir != "" {
		paths, err := writeExperimentResultsCSV(*outputDir, results, aggregates)
		if err != nil {
			log.Fatal(err)
		}
		for _, path := range paths {
			fmt.Printf("Wrote CSV results to %s\n", path)
		}
	}
}

func runSpecs(specs []simconfig.ExperimentRunSpec, workers int) ([]RunResult, error) {
	if len(specs) == 0 {
		return []RunResult{}, nil
	}
	if workers == 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if workers > len(specs) {
		workers = len(specs)
	}

	// Unbuffered: the producer waits until a worker is ready for another spec.
	jobQueue := make(chan simconfig.ExperimentRunSpec)
	// buffered; can receive outcomes without waiting for a collector to consume the results
	outcomes := make(chan jobOutcome, len(specs))

	// 1. consumers
	// only run `worker` parallel jobs at a time
	var workersDone sync.WaitGroup
	workersDone.Add(workers)
	for range workers {
		// each worker repeatedly receives jobs
		go func() {
			defer workersDone.Done()
			for spec := range jobQueue { // waits for work
				result, err := runJob(spec)
				outcomes <- jobOutcome{index: spec.Index, result: result, err: err}
			}
		}()
	}

	// 2. producer
	// insert all the jobs into the queue--this is what is feeding work
	// into the previous goroutine.
	go func() {
		for _, spec := range specs {
			jobQueue <- spec
		}
		close(jobQueue)
		workersDone.Wait()
		close(outcomes)
	}()

	// 3. collect results from the outcome channel's buffer
	results := make([]RunResult, len(specs))
	var jobErrors []error
	for outcome := range outcomes {
		if outcome.err != nil {
			jobErrors = append(jobErrors, outcome.err)
			continue
		}
		results[outcome.index] = outcome.result
	}
	if len(jobErrors) > 0 {
		return nil, errors.Join(jobErrors...)
	}

	return results, nil
}

func runJob(spec simconfig.ExperimentRunSpec) (RunResult, error) {
	selectorName := spec.ContainerSelector.Name
	engine, err := backend.NewSimulationEngine(spec.Config)
	if err != nil {
		return RunResult{}, fmt.Errorf("workload %q, placement policy %q, container selector %q, seed %d: create engine: %w", spec.WorkloadName, spec.PolicyName, selectorName, spec.Config.Seed, err)
	}

	placementPolicy, err := policy.NewPlacementPolicy(spec.PolicyName)
	if err != nil {
		return RunResult{}, fmt.Errorf("workload %q, placement policy %q, container selector %q, seed %d: %w", spec.WorkloadName, spec.PolicyName, selectorName, spec.Config.Seed, err)
	}
	containerSelector, err := policy.NewContainerSelector(selectorName, spec.ContainerSelector.K)
	if err != nil {
		return RunResult{}, fmt.Errorf("workload %q, placement policy %q, container selector %q, seed %d: %w", spec.WorkloadName, spec.PolicyName, selectorName, spec.Config.Seed, err)
	}

	simulation, err := engine.Run(containerSelector, placementPolicy, spec.Iterations)
	if err != nil {
		return RunResult{}, fmt.Errorf("workload %q, placement policy %q, container selector %q, seed %d: run simulation: %w", spec.WorkloadName, spec.PolicyName, containerSelector.Name(), spec.Config.Seed, err)
	}

	result := RunResult{
		WorkloadName:          spec.WorkloadName,
		PolicyName:            placementPolicy.Name(),
		ContainerSelectorName: containerSelector.Name(),
		Seed:                  spec.Config.Seed,
		Simulation:            simulation,
		Evaluations:           make([]evaluationResult, len(spec.Evaluations)),
	}
	for i, evaluation := range spec.Evaluations {
		result.Evaluations[i] = evaluationResult{
			evaluation: evaluation,
			value:      evaluator.EvaluateSimulation(engine, evaluation),
		}
	}

	return result, nil
}
func printSimulationConfig(config simulationConfig) {
	fmt.Println("Simulation configuration:")
	fmt.Printf("  Container: %d wide x %d high\n", config.ContainerWidth, config.ContainerHeight)
	fmt.Printf("  Queue size: %d\n", config.QueueSize)
	fmt.Printf("  Box width: %d-%d\n", config.MinBoxWidth, config.MaxBoxWidth)
	fmt.Printf("  Box height: %d-%d\n", config.MinBoxHeight, config.MaxBoxHeight)
	fmt.Printf("  Iterations: %d\n", config.Iterations)
	fmt.Printf("  Rotation allowed: %t\n", config.AllowBoxRotation)
}

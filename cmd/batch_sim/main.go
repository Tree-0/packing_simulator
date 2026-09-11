/*
run a group of simulations off of the same seed.
runs all combinations of the provided evaluators and packing policies.
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
)

type simulationConfig struct {
	ContainerHeight  int  `yaml:"container_height"`
	ContainerWidth   int  `yaml:"container_width"`
	QueueSize        int  `yaml:"queue_size"`
	MinBoxHeight     int  `yaml:"min_box_height"`
	MaxBoxHeight     int  `yaml:"max_box_height"`
	MinBoxWidth      int  `yaml:"min_box_width"`
	MaxBoxWidth      int  `yaml:"max_box_width"`
	Iterations       int  `yaml:"iterations"`
	AllowBoxRotation bool `yaml:"allow_box_rotation"`
}

type workloadJob struct {
	index      int
	policyName string
	seed       int64
}

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
	configPath := flag.String("config", "config/experiment/config.yml", "path to the experiment YAML config")
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

func (config simulationConfig) toBackendConfig(seed int64) backend.SimulationConfig {
	return backend.SimulationConfig{
		ContainerHeight:  config.ContainerHeight,
		ContainerWidth:   config.ContainerWidth,
		QueueSize:        config.QueueSize,
		MinBoxHeight:     config.MinBoxHeight,
		MaxBoxHeight:     config.MaxBoxHeight,
		MinBoxWidth:      config.MinBoxWidth,
		MaxBoxWidth:      config.MaxBoxWidth,
		Seed:             seed,
		AllowBoxRotation: config.AllowBoxRotation,
	}
}

func runWorkload(workload workloadConfig, config experimentConfig) ([]RunResult, error) {
	evaluations := make([]evaluator.EvaluationType, len(config.Evaluators))
	for i, name := range config.Evaluators {
		evaluation, err := evaluator.ParseEvaluation(name)
		if err != nil {
			return nil, err
		}
		evaluations[i] = evaluation
	}

	// Create jobs for all combinations of policy and seed for this workload.
	jobs := make([]workloadJob, 0, len(config.Policies)*len(config.Seeds))
	for _, policyName := range config.Policies {
		for _, seed := range config.Seeds {
			jobs = append(jobs, workloadJob{
				index:      len(jobs),
				policyName: policyName,
				seed:       seed,
			})
		}
	}

	workers := config.Workers
	if workers == 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if workers > len(jobs) {
		workers = len(jobs)
	}

	// unbuffered; producer waits until worker is available to send a job
	jobQueue := make(chan workloadJob)
	// buffered; can receive outcomes without waiting for a collector to consume the results
	outcomes := make(chan jobOutcome, len(jobs))

	// 1. consumers
	// only run `worker` parallel jobs at a time
	var workersDone sync.WaitGroup
	workersDone.Add(workers)
	for range workers {
		// each worker repeatedly receives jobs
		go func() {
			defer workersDone.Done()
			for job := range jobQueue { // waits for work
				result, err := runJob(workload.Name, workload.Simulation, evaluations, job)
				outcomes <- jobOutcome{index: job.index, result: result, err: err}
			}
		}()
	}

	// 2. producer
	// insert all the jobs into the queue--this is what is feeding work
	// into the previous goroutine.
	go func() {
		for _, job := range jobs {
			jobQueue <- job
		}
		close(jobQueue)
		workersDone.Wait()
		close(outcomes)
	}()

	// 3. collect results from the outcome channel's buffer
	results := make([]RunResult, len(jobs))
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

func runJob(
	workloadName string,
	simulationConfig simulationConfig,
	evaluations []evaluator.EvaluationType,
	job workloadJob,
) (RunResult, error) {
	engine, err := backend.NewSimulationEngine(simulationConfig.toBackendConfig(job.seed))
	if err != nil {
		return RunResult{}, fmt.Errorf("policy %q, seed %d: create engine: %w", job.policyName, job.seed, err)
	}

	containerSelector := policy.ContainerSelectorFirstFit{}
	policy, err := policy.NewPlacementPolicy(job.policyName)
	if err != nil {
		return RunResult{}, fmt.Errorf("policy %q, seed %d: %w", job.policyName, job.seed, err)
	}

	simulation, err := engine.Run(containerSelector, policy, simulationConfig.Iterations)
	if err != nil {
		return RunResult{}, fmt.Errorf("policy %q, seed %d: run simulation: %w", job.policyName, job.seed, err)
	}

	result := RunResult{
		WorkloadName: workloadName,
		PolicyName:   policy.Name(),
		Seed:         job.seed,
		Simulation:   simulation,
		Evaluations:  make([]evaluationResult, len(evaluations)),
	}
	for i, evaluation := range evaluations {
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

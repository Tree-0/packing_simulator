/*
run a group of simulations off of the same seed.
runs all combinations of the provided evaluators and packing policies.
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"

	"packing_simulator/backend"
	"packing_simulator/backend/evaluator"
	"packing_simulator/backend/policy"

	"gopkg.in/yaml.v3"
)

// batchConfig is the YAML representation of a set of comparable simulations.
// Each policy is run once per seed; all selected evaluators are then applied to
// that completed simulation.
type batchConfig struct {
	Simulation batchSimulationConfig `yaml:"simulation"`
	Seeds      []int64               `yaml:"seeds"`
	Policies   []string              `yaml:"policies"`
	Evaluators []string              `yaml:"evaluators"`
	Workers    int                   `yaml:"workers"`
}

type batchSimulationConfig struct {
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

type batchJob struct {
	index      int
	policyName string
	seed       int64
}

type evaluationResult struct {
	evaluation evaluator.EvaluationType
	value      float64
}

type batchResult struct {
	policy      string
	seed        int64
	simulation  backend.SimulationResult
	evaluations []evaluationResult
}

type jobOutcome struct {
	index  int
	result batchResult
	err    error
}

func main() {
	mode := flag.String("mode", "batch", "the simulation mode we are running: 'batch' or 'experiment'")
	configPath := flag.String("config", "config/batch_sim/config.yml", "path to the batch simulation YAML config")
	flag.Parse()
	if flag.NArg() != 0 {
		log.Fatalf("unexpected positional arguments: %s", strings.Join(flag.Args(), " "))
	}

	// Run an experiment on a set of simulation scenarios and aggregate results
	if *mode == "experiment" {
		experimentConfig, err := loadExperimentConfig(*configPath)
		if err != nil {
			log.Fatal(err)
		}

		// run multiple different batches (different simulation parameters)
		// on the same set of policies, seeds, and evaluators
		results, err := runExperiment(experimentConfig)
		if err != nil {
			log.Fatal(err)
		}

		// display experiment results
		PrintRunResults(experimentConfig, results)

		aggregates, err := AggregateResults(results)
		if err != nil {
			log.Fatal(err)
		}

		// display aggregate results
		PrintAggregateResults(experimentConfig, aggregates)

		// Run a batch of simulations (one workload, multiple seeds and policies)
	} else if *mode == "batch" {
		batchConfig, err := loadConfig(*configPath)
		if err != nil {
			log.Fatal(err)
		}

		results, err := runBatch(batchConfig)
		if err != nil {
			log.Fatal(err)
		}

		printResults(results)

		// unrecognized simulation mode type
	} else {
		log.Fatal(
			fmt.Errorf("Unrecognized simulation mode: got %q, expected 'batch' or 'experiment'", *mode),
		)
	}
}

// loadConfig reads and validates one YAML configuration file. A relative path
// is intentionally interpreted relative to the process's current directory.
func loadConfig(path string) (batchConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return batchConfig{}, fmt.Errorf("open config %q: %w", path, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	var config batchConfig
	if err := decoder.Decode(&config); err != nil {
		return batchConfig{}, fmt.Errorf("decode config %q: %w", path, err)
	}

	var extraDocument any
	if err := decoder.Decode(&extraDocument); err != io.EOF {
		if err == nil {
			return batchConfig{}, fmt.Errorf("decode config %q: multiple YAML documents are not supported", path)
		}
		return batchConfig{}, fmt.Errorf("decode config %q: %w", path, err)
	}

	if err := config.validate(); err != nil {
		return batchConfig{}, fmt.Errorf("invalid config %q: %w", path, err)
	}

	return config, nil
}

func (config batchConfig) validate() error {
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
	if config.Simulation.Iterations < 0 {
		return errors.New("simulation.iterations cannot be negative")
	}

	if _, err := backend.NewSimulationEngine(config.Simulation.toBackendConfig(config.Seeds[0])); err != nil {
		return fmt.Errorf("simulation: %w", err)
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

func (config batchSimulationConfig) toBackendConfig(seed int64) backend.SimulationConfig {
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

func runBatch(config batchConfig) ([]batchResult, error) {
	evaluations := make([]evaluator.EvaluationType, len(config.Evaluators))
	for i, name := range config.Evaluators {
		evaluation, err := evaluator.ParseEvaluation(name)
		if err != nil {
			return nil, err
		}
		evaluations[i] = evaluation
	}

	// create jobs for all combinations of (policy, seed)
	jobs := make([]batchJob, 0, len(config.Policies)*len(config.Seeds))
	for _, policyName := range config.Policies {
		for _, seed := range config.Seeds {
			jobs = append(jobs, batchJob{
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
	jobQueue := make(chan batchJob)
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
				result, err := runJob(config.Simulation, evaluations, job)
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
	results := make([]batchResult, len(jobs))
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
	simulationConfig batchSimulationConfig,
	evaluations []evaluator.EvaluationType,
	job batchJob,
) (batchResult, error) {
	engine, err := backend.NewSimulationEngine(simulationConfig.toBackendConfig(job.seed))
	if err != nil {
		return batchResult{}, fmt.Errorf("policy %q, seed %d: create engine: %w", job.policyName, job.seed, err)
	}

	policy, err := policy.NewPolicy(job.policyName)
	if err != nil {
		return batchResult{}, fmt.Errorf("policy %q, seed %d: %w", job.policyName, job.seed, err)
	}

	simulation, err := engine.Run(policy, simulationConfig.Iterations)
	if err != nil {
		return batchResult{}, fmt.Errorf("policy %q, seed %d: run simulation: %w", job.policyName, job.seed, err)
	}

	result := batchResult{
		policy:      policy.Name(),
		seed:        job.seed,
		simulation:  simulation,
		evaluations: make([]evaluationResult, len(evaluations)),
	}
	for i, evaluation := range evaluations {
		result.evaluations[i] = evaluationResult{
			evaluation: evaluation,
			value:      evaluator.EvaluateSimulation(engine, evaluation),
		}
	}

	return result, nil
}

func printResults(results []batchResult) {
	for _, result := range results {
		fmt.Printf("Policy: %s  Seed: %d\n", result.policy, result.seed)
		fmt.Printf(
			"  Iterations: %d, generated: %d, placed: %d, rotated: %d, rejected: %d, batches: %d\n",
			result.simulation.Iterations,
			result.simulation.Generated,
			result.simulation.Placed,
			result.simulation.Rotated,
			result.simulation.Rejected,
			result.simulation.Batches,
		)
		if result.simulation.StoppedEarly {
			fmt.Println("  Stopped early: no box in a batch could be placed.")
		}

		for _, evaluation := range result.evaluations {
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

func printBatchSimulationConfig(config batchSimulationConfig) {
	fmt.Println("Simulation configuration:")
	fmt.Printf("  Container: %d wide x %d high\n", config.ContainerWidth, config.ContainerHeight)
	fmt.Printf("  Queue size: %d\n", config.QueueSize)
	fmt.Printf("  Box width: %d-%d\n", config.MinBoxWidth, config.MaxBoxWidth)
	fmt.Printf("  Box height: %d-%d\n", config.MinBoxHeight, config.MaxBoxHeight)
	fmt.Printf("  Iterations: %d\n", config.Iterations)
	fmt.Printf("  Rotation allowed: %t\n", config.AllowBoxRotation)
}

package simconfig

import (
	"errors"
	"fmt"
	"io"
	"os"

	"packing_simulator/backend"
	"packing_simulator/backend/evaluator"
	"packing_simulator/backend/policy"

	"gopkg.in/yaml.v3"
)

const DefaultExperimentPath = "config/experiment/config.yml"

// ExperimentFile describes the shared inputs for a batch of simulation runs.
type ExperimentFile struct {
	Workloads          []ExperimentWorkload      `yaml:"workloads"`
	MaxContainers      int                       `yaml:"max_containers"`
	Seeds              []int64                   `yaml:"seeds"`
	Policies           []string                  `yaml:"policies"`
	ContainerSelectors []ContainerSelectorConfig `yaml:"container_selectors"`
	Evaluators         []string                  `yaml:"evaluators"`
	Workers            int                       `yaml:"workers"`
}

type ExperimentWorkload struct {
	Name       string             `yaml:"name"`
	Simulation WorkloadSimulation `yaml:"simulation"`
}

// WorkloadSimulation contains the simulation values that vary by workload.
// Seeds and the container limit are supplied by the enclosing experiment.
type WorkloadSimulation struct {
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

func (simulation WorkloadSimulation) BackendConfig(seed int64, maxContainers int) backend.SimulationConfig {
	return backend.SimulationConfig{
		ContainerHeight:  simulation.ContainerHeight,
		ContainerWidth:   simulation.ContainerWidth,
		MaxContainers:    maxContainers,
		QueueSize:        simulation.QueueSize,
		MinBoxHeight:     simulation.MinBoxHeight,
		MaxBoxHeight:     simulation.MaxBoxHeight,
		MinBoxWidth:      simulation.MinBoxWidth,
		MaxBoxWidth:      simulation.MaxBoxWidth,
		Seed:             seed,
		AllowBoxRotation: simulation.AllowBoxRotation,
	}
}

// ExperimentRunSpec is one deterministic workload × policy × selector × seed
// permutation. Evaluations preserve their order in the experiment file.
type ExperimentRunSpec struct {
	Index             int
	WorkloadName      string
	Config            backend.SimulationConfig
	Iterations        int
	PolicyName        string
	ContainerSelector ContainerSelectorConfig
	Evaluations       []evaluator.EvaluationType
}

// EffectiveContainerSelectors preserves First Fit as the default for older
// experiment files that do not declare container_selectors.
func (config ExperimentFile) EffectiveContainerSelectors() []ContainerSelectorConfig {
	if len(config.ContainerSelectors) == 0 {
		return []ContainerSelectorConfig{{Name: policy.ContainerSelectorFirstFitName}}
	}
	return append([]ContainerSelectorConfig(nil), config.ContainerSelectors...)
}

// RunSpecs validates and expands an experiment in workload, placement-policy,
// container-selector, then seed order.
func (config ExperimentFile) RunSpecs() ([]ExperimentRunSpec, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	evaluations := make([]evaluator.EvaluationType, len(config.Evaluators))
	for i, name := range config.Evaluators {
		evaluation, err := evaluator.ParseEvaluation(name)
		if err != nil {
			return nil, err
		}
		evaluations[i] = evaluation
	}

	selectors := config.EffectiveContainerSelectors()
	total := len(config.Workloads) * len(config.Policies) * len(selectors) * len(config.Seeds)
	specs := make([]ExperimentRunSpec, 0, total)
	for _, workload := range config.Workloads {
		for _, policyName := range config.Policies {
			for _, selector := range selectors {
				for _, seed := range config.Seeds {
					specs = append(specs, ExperimentRunSpec{
						Index:             len(specs),
						WorkloadName:      workload.Name,
						Config:            workload.Simulation.BackendConfig(seed, config.MaxContainers),
						Iterations:        workload.Simulation.Iterations,
						PolicyName:        policyName,
						ContainerSelector: selector,
						Evaluations:       append([]evaluator.EvaluationType(nil), evaluations...),
					})
				}
			}
		}
	}
	return specs, nil
}

func (config ExperimentFile) Validate() error {
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
		if _, err := backend.NewSimulationEngine(workload.Simulation.BackendConfig(config.Seeds[0], config.MaxContainers)); err != nil {
			return fmt.Errorf("workload %q: simulation: %w", workload.Name, err)
		}
	}

	for _, name := range config.Policies {
		if _, err := policy.NewPlacementPolicy(name); err != nil {
			return err
		}
	}
	for _, selector := range config.EffectiveContainerSelectors() {
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

// LoadExperiment reads and validates one experiment YAML file.
func LoadExperiment(path string) (ExperimentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return ExperimentFile{}, fmt.Errorf("open experiment config %q: %w", path, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	var config ExperimentFile
	if err := decoder.Decode(&config); err != nil {
		return ExperimentFile{}, fmt.Errorf("decode experiment config %q: %w", path, err)
	}

	var extraDocument any
	if err := decoder.Decode(&extraDocument); err != io.EOF {
		if err == nil {
			return ExperimentFile{}, fmt.Errorf("decode experiment config %q: multiple YAML documents are not supported", path)
		}
		return ExperimentFile{}, fmt.Errorf("decode experiment config %q: %w", path, err)
	}

	if err := config.Validate(); err != nil {
		return ExperimentFile{}, fmt.Errorf("invalid experiment config %q: %w", path, err)
	}
	return config, nil
}

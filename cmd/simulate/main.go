package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"packing_simulator/backend"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "config/simulate/config.yml"

type simulateFileConfig struct {
	Simulation simulateConfig `yaml:"simulation"`
	Policy     string         `yaml:"policy"`
	Animate    int            `yaml:"animate"`
}

type simulateConfig struct {
	ContainerHeight  int   `yaml:"container_height"`
	ContainerWidth   int   `yaml:"container_width"`
	QueueSize        int   `yaml:"queue_size"`
	MinBoxHeight     int   `yaml:"min_box_height"`
	MaxBoxHeight     int   `yaml:"max_box_height"`
	MinBoxWidth      int   `yaml:"min_box_width"`
	MaxBoxWidth      int   `yaml:"max_box_width"`
	Iterations       int   `yaml:"iterations"`
	Seed             int64 `yaml:"seed"`
	AllowBoxRotation bool  `yaml:"allow_box_rotation"`
}

func main() {
	configPath, err := configPathFromArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	// Load the YAML values before declaring flags: a supplied flag then replaces
	// just that one YAML default.
	flag.String("config", configPath, "path to the single-simulation YAML config")
	height := flag.Int("height", config.Simulation.ContainerHeight, "container height")
	width := flag.Int("width", config.Simulation.ContainerWidth, "container width")
	queueSize := flag.Int("queue-size", config.Simulation.QueueSize, "number of boxes processed per batch")
	minBoxHeight := flag.Int("min-box-height", config.Simulation.MinBoxHeight, "minimum random box height")
	maxBoxHeight := flag.Int("max-box-height", config.Simulation.MaxBoxHeight, "maximum random box height")
	minBoxWidth := flag.Int("min-box-width", config.Simulation.MinBoxWidth, "minimum random box width")
	maxBoxWidth := flag.Int("max-box-width", config.Simulation.MaxBoxWidth, "maximum random box width")
	iterations := flag.Int("iterations", config.Simulation.Iterations, "maximum number of boxes to generate")
	seed := flag.Int64("seed", config.Simulation.Seed, "random seed")
	animate := flag.Int("animate", config.Animate, "animate each timestamp with this delay in milliseconds; use -1 to disable")
	policyName := flag.String("policy", config.Policy, "packing policy: bottom-left or largest-area-bottom-left")
	allowBoxRotation := flag.Bool("allow-box-rotation", config.Simulation.AllowBoxRotation, "whether the simulation can rotate boxes when attempting to place them")

	flag.Parse()

	if *animate < -1 {
		log.Fatal("animate delay cannot be negative")
	}

	engine, err := backend.NewSimulationEngine(backend.SimulationConfig{
		ContainerHeight:  *height,
		ContainerWidth:   *width,
		QueueSize:        *queueSize,
		MinBoxHeight:     *minBoxHeight,
		MaxBoxHeight:     *maxBoxHeight,
		MinBoxWidth:      *minBoxWidth,
		MaxBoxWidth:      *maxBoxWidth,
		Seed:             *seed,
		AllowBoxRotation: *allowBoxRotation,
	})
	if err != nil {
		log.Fatal(err)
	}

	policy, err := backend.NewPolicy(*policyName)
	if err != nil {
		log.Fatal(err)
	}
	var result backend.SimulationResult
	if *animate >= 0 {
		firstFrame := true
		fmt.Print("\033[?25l") // Hide the cursor while animating.
		result, err = engine.RunWithObserver(
			policy,
			*iterations,
			func(timestamp int, world *backend.World) error {
				if !firstFrame {
					time.Sleep(time.Duration(*animate) * time.Millisecond)
				}
				firstFrame = false

				fmt.Print("\033[2J\033[H")
				fmt.Printf(
					"Timestamp: %d  Queue: %d/%d\n\n",
					timestamp,
					len(world.Queue.Items),
					world.Queue.Limit,
				)
				return printContainer(&world.Container)
			},
		)
		fmt.Print("\033[?25h") // Restore the cursor before reporting errors.
		if firstFrame && err == nil {
			fmt.Print("\033[2J\033[H")
			err = printContainer(&engine.World().Container)
		}
		fmt.Println()
	} else {
		result, err = engine.Run(policy, *iterations)
	}
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Policy: %s\n", policy.Name())
	fmt.Printf("Seed: %d\n", *seed)
	fmt.Printf(
		"Iterations: %d, generated: %d, placed: %d, rotated: %d, rejected: %d, batches: %d\n",
		result.Iterations,
		result.Generated,
		result.Placed,
		result.Rotated,
		result.Rejected,
		result.Batches,
	)
	if result.StoppedEarly {
		fmt.Println("Simulation stopped because no box in a batch could be placed.")
	}

	fmt.Println("Evaluations:")
	for _, evalType := range backend.AllEvaluationTypes() {
		value := backend.EvaluateSimulation(engine, evalType)
		switch evalType {
		case backend.ContainerUtilization, backend.FutureFitProbabilityMetric:
			fmt.Printf("  %-24s %.1f%%\n", evalType, 100*value)
		default:
			fmt.Printf("  %-24s %.4f\n", evalType, value)
		}
	}
	fmt.Println()

	if *animate < 0 {
		if err := printContainer(&engine.World().Container); err != nil {
			log.Fatal(err)
		}
	}
}

// configPathFromArgs finds -config before the standard flag parser runs so the
// selected file can provide defaults for every other flag.
func configPathFromArgs(args []string) (string, error) {
	path := defaultConfigPath
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-config" || arg == "--config":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return "", fmt.Errorf("%s requires a config file path", arg)
			}
			path = args[i+1]
			i++
		case strings.HasPrefix(arg, "-config="):
			path = strings.TrimPrefix(arg, "-config=")
		case strings.HasPrefix(arg, "--config="):
			path = strings.TrimPrefix(arg, "--config=")
		}
	}
	if path == "" {
		return "", fmt.Errorf("config file path cannot be empty")
	}
	return path, nil
}

func loadConfig(path string) (simulateFileConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return simulateFileConfig{}, fmt.Errorf("open config %q: %w", path, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	var config simulateFileConfig
	if err := decoder.Decode(&config); err != nil {
		return simulateFileConfig{}, fmt.Errorf("decode config %q: %w", path, err)
	}

	var extraDocument any
	if err := decoder.Decode(&extraDocument); err != io.EOF {
		if err == nil {
			return simulateFileConfig{}, fmt.Errorf("decode config %q: multiple YAML documents are not supported", path)
		}
		return simulateFileConfig{}, fmt.Errorf("decode config %q: %w", path, err)
	}

	return config, nil
}

func printContainer(container *backend.Container) error {
	for y := 0; y < container.Height(); y++ {
		for x := 0; x < container.Width(); x++ {
			cell, err := container.Cell(x, y)
			if err != nil {
				return err
			}

			if cell == backend.EmptyCell {
				fmt.Print("  .")
			} else {
				fmt.Printf("%3d", cell)
			}
		}
		fmt.Println()
	}

	return nil
}

// Package simconfig loads and binds configuration shared by the single-run
// console and browser commands.
package simconfig

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"packing_simulator/backend"

	"gopkg.in/yaml.v3"
)

const DefaultPath = "config/simulate/config.yml"

type File struct {
	Simulation Simulation `yaml:"simulation"`
	Policy     string     `yaml:"policy"`
	Animate    int        `yaml:"animate"`
}

type Simulation struct {
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

// Flags contains shared single-simulation command-line values.
type Flags struct {
	ConfigPath       *string
	Height           *int
	Width            *int
	QueueSize        *int
	MinBoxHeight     *int
	MaxBoxHeight     *int
	MinBoxWidth      *int
	MaxBoxWidth      *int
	Iterations       *int
	Seed             *int64
	PolicyName       *string
	AllowBoxRotation *bool
}

// BindFlags declares the flags shared by the console and browser commands.
// Call PathFromArgs and Load first so the chosen YAML file supplies defaults.
func BindFlags(fs *flag.FlagSet, configPath string, config File) Flags {
	return Flags{
		ConfigPath:       fs.String("config", configPath, "path to the single-simulation YAML config"),
		Height:           fs.Int("height", config.Simulation.ContainerHeight, "container height"),
		Width:            fs.Int("width", config.Simulation.ContainerWidth, "container width"),
		QueueSize:        fs.Int("queue-size", config.Simulation.QueueSize, "number of boxes processed per batch"),
		MinBoxHeight:     fs.Int("min-box-height", config.Simulation.MinBoxHeight, "minimum random box height"),
		MaxBoxHeight:     fs.Int("max-box-height", config.Simulation.MaxBoxHeight, "maximum random box height"),
		MinBoxWidth:      fs.Int("min-box-width", config.Simulation.MinBoxWidth, "minimum random box width"),
		MaxBoxWidth:      fs.Int("max-box-width", config.Simulation.MaxBoxWidth, "maximum random box width"),
		Iterations:       fs.Int("iterations", config.Simulation.Iterations, "maximum number of boxes to generate"),
		Seed:             fs.Int64("seed", config.Simulation.Seed, "random seed"),
		PolicyName:       fs.String("policy", config.Policy, "packing policy: bottom-left or largest-area-bottom-left"),
		AllowBoxRotation: fs.Bool("allow-box-rotation", config.Simulation.AllowBoxRotation, "whether the simulation can rotate boxes when attempting to place them"),
	}
}

func (values Flags) BackendConfig() backend.SimulationConfig {
	return backend.SimulationConfig{
		ContainerHeight:  *values.Height,
		ContainerWidth:   *values.Width,
		QueueSize:        *values.QueueSize,
		MinBoxHeight:     *values.MinBoxHeight,
		MaxBoxHeight:     *values.MaxBoxHeight,
		MinBoxWidth:      *values.MinBoxWidth,
		MaxBoxWidth:      *values.MaxBoxWidth,
		Seed:             *values.Seed,
		AllowBoxRotation: *values.AllowBoxRotation,
	}
}

// PathFromArgs finds -config before flag parsing so the selected file can
// provide defaults for all other flags.
func PathFromArgs(args []string, defaultPath string) (string, error) {
	path := defaultPath
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

func Load(path string) (File, error) {
	file, err := os.Open(path)
	if err != nil {
		return File{}, fmt.Errorf("open config %q: %w", path, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	var config File
	if err := decoder.Decode(&config); err != nil {
		return File{}, fmt.Errorf("decode config %q: %w", path, err)
	}

	var extraDocument any
	if err := decoder.Decode(&extraDocument); err != io.EOF {
		if err == nil {
			return File{}, fmt.Errorf("decode config %q: multiple YAML documents are not supported", path)
		}
		return File{}, fmt.Errorf("decode config %q: %w", path, err)
	}

	return config, nil
}

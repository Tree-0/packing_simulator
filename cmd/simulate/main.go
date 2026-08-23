package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"packing_simulator/backend"
	"packing_simulator/internal/simconfig"
)

func main() {
	configPath, err := simconfig.PathFromArgs(os.Args[1:], simconfig.DefaultPath)
	if err != nil {
		log.Fatal(err)
	}
	config, err := simconfig.Load(configPath)
	if err != nil {
		log.Fatal(err)
	}

	// Load the YAML values before declaring flags: a supplied flag then replaces
	// just that one YAML default.
	values := simconfig.BindFlags(flag.CommandLine, configPath, config)
	animate := flag.Int("animate", config.Animate, "animate each timestamp with this delay in milliseconds; use -1 to disable")

	flag.Parse()

	if *animate < -1 {
		log.Fatal("animate delay cannot be negative")
	}

	engine, err := backend.NewSimulationEngine(values.BackendConfig())
	if err != nil {
		log.Fatal(err)
	}

	policy, err := backend.NewPolicy(*values.PolicyName)
	if err != nil {
		log.Fatal(err)
	}
	var result backend.SimulationResult
	if *animate >= 0 {
		firstFrame := true
		fmt.Print("\033[?25l") // Hide the cursor while animating.
		result, err = engine.RunWithObserver(
			policy,
			*values.Iterations,
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
		result, err = engine.Run(policy, *values.Iterations)
	}
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Policy: %s\n", policy.Name())
	fmt.Printf("Seed: %d\n", *values.Seed)
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

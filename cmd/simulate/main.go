package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"packing_simulator/backend"
)

func main() {
	height := flag.Int("height", 10, "container height")
	width := flag.Int("width", 20, "container width")
	queueSize := flag.Int("queue-size", 5, "number of boxes processed per batch")
	minBoxHeight := flag.Int("min-box-height", 1, "minimum random box height")
	maxBoxHeight := flag.Int("max-box-height", 4, "maximum random box height")
	minBoxWidth := flag.Int("min-box-width", 1, "minimum random box width")
	maxBoxWidth := flag.Int("max-box-width", 5, "maximum random box width")
	iterations := flag.Int("iterations", 50, "maximum number of boxes to generate")
	seed := flag.Int64("seed", time.Now().UnixNano(), "random seed; defaults to a new seed each run")
	animate := flag.Int("animate", -1, "animate each timestamp with this delay in milliseconds; omit to disable")
	policyName := flag.String("policy", backend.BottomLeftPolicyName, "packing policy: bottom-left or largest-area-bottom-left")

	flag.Parse()

	if *animate < -1 {
		log.Fatal("animate delay cannot be negative")
	}

	engine, err := backend.NewSimulationEngine(backend.SimulationConfig{
		ContainerHeight: *height,
		ContainerWidth:  *width,
		QueueSize:       *queueSize,
		MinBoxHeight:    *minBoxHeight,
		MaxBoxHeight:    *maxBoxHeight,
		MinBoxWidth:     *minBoxWidth,
		MaxBoxWidth:     *maxBoxWidth,
		Seed:            *seed,
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
		"Iterations: %d, generated: %d, placed: %d, rejected: %d, batches: %d\n",
		result.Iterations,
		result.Generated,
		result.Placed,
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

package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"packing_simulator/backend/evaluator"
	"packing_simulator/frontend"
	"packing_simulator/internal/simconfig"
)

func main() {
	recordings, address, err := prepareRecordings(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("listen on %s: %v", address, err)
	}

	server := &http.Server{
		Handler:           frontend.NewHandler(recordings),
		ReadHeaderTimeout: 5 * time.Second,
	}

	stopRequests := make(chan string, 2)
	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignals)
	go func() {
		received := <-shutdownSignals
		// Restore the default behavior so a second interrupt can force an exit
		// if graceful shutdown ever stalls.
		signal.Stop(shutdownSignals)
		stopRequests <- fmt.Sprintf("received %s", received)
	}()

	terminalInput := false
	if info, err := os.Stdin.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
		terminalInput = true
		go func() {
			if _, err := bufio.NewReader(os.Stdin).ReadString('\n'); err == nil {
				stopRequests <- "Enter pressed"
			}
		}()
	}

	if err := serveUntilStopped(server, listener, stopRequests, os.Stdout, terminalInput); err != nil {
		log.Fatal(err)
	}
}

func prepareRecordings(args []string) ([]frontend.SimulationRecording, string, error) {
	experimentPath, hasExperiment, err := explicitPathFlag(args, "experiment-config")
	if err != nil {
		return nil, "", err
	}
	_, hasSingleConfig, err := explicitPathFlag(args, "config")
	if err != nil {
		return nil, "", err
	}
	if hasExperiment && hasSingleConfig {
		return nil, "", errors.New("-config and -experiment-config are mutually exclusive")
	}

	if hasExperiment {
		return prepareExperimentRecordings(args, experimentPath)
	}
	return prepareSingleRecording(args)
}

func prepareSingleRecording(args []string) ([]frontend.SimulationRecording, string, error) {
	configPath, err := simconfig.PathFromArgs(args, simconfig.DefaultPath)
	if err != nil {
		return nil, "", err
	}
	config, err := simconfig.Load(configPath)
	if err != nil {
		return nil, "", err
	}

	flags := flag.NewFlagSet("visualize", flag.ContinueOnError)
	values := simconfig.BindFlags(flags, configPath, config)
	address := flags.String("addr", "127.0.0.1:8080", "local address for the visualizer server")
	if err := flags.Parse(args); err != nil {
		return nil, "", err
	}
	if flags.NArg() != 0 {
		return nil, "", fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}

	recording, err := frontend.RecordSimulation(frontend.SimulationSpec{
		ID:                    "simulation-1",
		Config:                values.BackendConfig(),
		Iterations:            *values.Iterations,
		PolicyName:            *values.PolicyName,
		ContainerSelectorName: *values.ContainerSelectorName,
		ContainerSelectorK:    *values.ContainerSelectorK,
	})
	if err != nil {
		return nil, "", err
	}
	return []frontend.SimulationRecording{recording}, *address, nil
}

func prepareExperimentRecordings(args []string, experimentPath string) ([]frontend.SimulationRecording, string, error) {
	flags := flag.NewFlagSet("visualize", flag.ContinueOnError)
	path := flags.String("experiment-config", experimentPath, "path to the experiment YAML config")
	address := flags.String("addr", "127.0.0.1:8080", "local address for the visualizer server")
	if err := flags.Parse(args); err != nil {
		return nil, "", err
	}
	if flags.NArg() != 0 {
		return nil, "", fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}

	config, err := simconfig.LoadExperiment(*path)
	if err != nil {
		return nil, "", err
	}
	runSpecs, err := config.RunSpecs()
	if err != nil {
		return nil, "", err
	}
	specs := make([]frontend.SimulationSpec, len(runSpecs))
	for i, run := range runSpecs {
		specs[i] = frontend.SimulationSpec{
			ID:                    fmt.Sprintf("simulation-%03d", i+1),
			Workload:              run.WorkloadName,
			Config:                run.Config,
			Iterations:            run.Iterations,
			PolicyName:            run.PolicyName,
			ContainerSelectorName: run.ContainerSelector.Name,
			ContainerSelectorK:    run.ContainerSelector.K,
			EvaluationTypes:       append([]evaluator.EvaluationType(nil), run.Evaluations...),
		}
	}
	recordings, err := frontend.RecordSimulations(specs, config.Workers)
	if err != nil {
		return nil, "", err
	}
	return recordings, *address, nil
}

func explicitPathFlag(args []string, name string) (string, bool, error) {
	short := "-" + name
	long := "--" + name
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == short || arg == long:
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return "", true, fmt.Errorf("%s requires a config file path", arg)
			}
			return args[i+1], true, nil
		case strings.HasPrefix(arg, short+"="):
			path := strings.TrimPrefix(arg, short+"=")
			if path == "" {
				return "", true, fmt.Errorf("%s cannot be empty", short)
			}
			return path, true, nil
		case strings.HasPrefix(arg, long+"="):
			path := strings.TrimPrefix(arg, long+"=")
			if path == "" {
				return "", true, fmt.Errorf("%s cannot be empty", long)
			}
			return path, true, nil
		}
	}
	return "", false, nil
}

func serveUntilStopped(
	server *http.Server,
	listener net.Listener,
	stopRequests <-chan string,
	output io.Writer,
	terminalInput bool,
) error {
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(listener)
	}()

	fmt.Fprintf(output, "Packing visualizer: http://%s\n", listener.Addr())
	if terminalInput {
		fmt.Fprintln(output, "Press Ctrl+C or Enter to stop.")
	}

	select {
	case reason := <-stopRequests:
		fmt.Fprintf(output, "Stopping visualizer (%s)…\n", reason)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shut down visualizer: %w", err)
		}
		serveErr := <-serverErrors
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return serveErr
		}
		return nil
	case serveErr := <-serverErrors:
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return serveErr
		}
		return nil
	}
}

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

	"packing_simulator/frontend"
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

	values := simconfig.BindFlags(flag.CommandLine, configPath, config)
	address := flag.String("addr", "127.0.0.1:8080", "local address for the visualizer server")
	flag.Parse()
	if flag.NArg() != 0 {
		log.Fatalf("unexpected positional arguments: %s", strings.Join(flag.Args(), " "))
	}

	recording, err := frontend.RecordSimulation(frontend.SimulationSpec{
		ID:         "simulation-1",
		Config:     values.BackendConfig(),
		Iterations: *values.Iterations,
		PolicyName: *values.PolicyName,
	})
	if err != nil {
		log.Fatal(err)
	}

	listener, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatalf("listen on %s: %v", *address, err)
	}

	server := &http.Server{
		Handler:           frontend.NewHandler([]frontend.SimulationRecording{recording}),
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

package main

import (
	"bytes"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"packing_simulator/backend/evaluator"
	"packing_simulator/backend/policy"
)

func TestPrepareRecordingsRejectsSingleAndExperimentConfigsTogether(t *testing.T) {
	_, _, err := prepareRecordings([]string{"-config", "single.yml", "-experiment-config", "experiment.yml"})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("prepareRecordings() error = %v; want mutually exclusive config error", err)
	}
}

func TestPrepareExperimentRecordingsExpandsRunsInOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "experiment.yml")
	contents := `workloads:
  - name: tiny
    simulation:
      container_height: 2
      container_width: 2
      queue_size: 1
      min_box_height: 1
      max_box_height: 1
      min_box_width: 1
      max_box_width: 1
      iterations: 1
      allow_box_rotation: false
max_containers: 1
seeds: [7, 8]
policies: [bottom-left]
container_selectors:
  - name: first-fit
  - name: next-fit
evaluators: [bin-count]
workers: 2
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	recordings, address, err := prepareRecordings([]string{"-experiment-config", path, "-addr", "127.0.0.1:9090"})
	if err != nil {
		t.Fatal(err)
	}
	if address != "127.0.0.1:9090" || len(recordings) != 4 {
		t.Fatalf("address = %q, recordings = %d; want address override and four runs", address, len(recordings))
	}
	wantSelectors := []string{
		policy.ContainerSelectorFirstFitName,
		policy.ContainerSelectorFirstFitName,
		policy.ContainerSelectorNextFitName,
		policy.ContainerSelectorNextFitName,
	}
	wantIDs := []string{"simulation-001", "simulation-002", "simulation-003", "simulation-004"}
	for i, recording := range recordings {
		if recording.ID != wantIDs[i] || recording.Workload != "tiny" || recording.ContainerSelector != wantSelectors[i] {
			t.Errorf("recording %d = %+v", i, recording)
		}
		if len(recording.Frames) == 0 || len(recording.Frames[0].Evaluations) != 1 || recording.Frames[0].Evaluations[0].Name != evaluator.BinCount.String() {
			t.Errorf("recording %d evaluations = %+v", i, recording.Frames)
		}
	}
}

func TestServeUntilStoppedClosesListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})}
	stopRequests := make(chan string, 1)
	var output bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- serveUntilStopped(server, listener, stopRequests, &output, true)
	}()

	response, err := http.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	stopRequests <- "test request"
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	got := output.String()
	if !strings.Contains(got, "Press Ctrl+C or Enter to stop.") || !strings.Contains(got, "Stopping visualizer (test request)") {
		t.Errorf("output = %q", got)
	}
	if _, err := net.Dial("tcp", listener.Addr().String()); err == nil {
		t.Fatal("listener still accepts connections after shutdown")
	}
}

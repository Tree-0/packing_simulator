package main

import (
	"bytes"
	"net"
	"net/http"
	"strings"
	"testing"
)

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

package simconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathFromArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{name: "default", want: "default.yml"},
		{name: "separate short flag", args: []string{"-config", "custom.yml"}, want: "custom.yml"},
		{name: "long equals flag", args: []string{"--config=custom.yml"}, want: "custom.yml"},
		{name: "missing value", args: []string{"-config"}, wantErr: true},
		{name: "empty value", args: []string{"-config="}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PathFromArgs(tt.args, "default.yml")
			if (err != nil) != tt.wantErr {
				t.Fatalf("PathFromArgs() error = %v; wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("PathFromArgs() = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	path := writeConfig(t, `simulation:
  container_height: 4
  container_width: 7
  queue_size: 2
  min_box_height: 1
  max_box_height: 2
  min_box_width: 1
  max_box_width: 3
  iterations: 9
  seed: 42
  allow_box_rotation: true
policy: bottom-left
animate: -1
`)

	config, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.Simulation.ContainerHeight != 4 || config.Simulation.ContainerWidth != 7 || config.Simulation.Iterations != 9 {
		t.Errorf("simulation config = %+v", config.Simulation)
	}
	if config.Policy != "bottom-left" || config.Animate != -1 {
		t.Errorf("file config = %+v", config)
	}
}

func TestLoadRejectsUnknownFieldsAndMultipleDocuments(t *testing.T) {
	for _, contents := range []string{
		"unknown: true\n",
		"policy: bottom-left\n---\npolicy: bottom-left\n",
	} {
		path := writeConfig(t, contents)
		if _, err := Load(path); err == nil {
			t.Fatalf("Load(%q) unexpectedly succeeded", strings.TrimSpace(contents))
		}
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

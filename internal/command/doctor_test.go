package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/railzwaylabs/macro/internal/project"
)

type fakeEnvironment struct {
	available map[string]bool
	version   string
}

func (environment fakeEnvironment) LookPath(name string) (string, error) {
	if environment.available[name] {
		return "/test/bin/" + name, nil
	}
	return "", os.ErrNotExist
}

func (environment fakeEnvironment) Output(context.Context, string, ...string) (string, error) {
	if environment.version == "" {
		return "", errors.New("no version")
	}
	return environment.version, nil
}

func TestDoctorReportsGoAndTreatsOptionalToolsAsWarnings(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)
	before, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = runDoctor(context.Background(), &output, fakeEnvironment{
		available: map[string]bool{"go": true, "git": true},
		version:   "go version go1.25.7 test/amd64",
	})
	if err != nil {
		t.Fatalf("optional tools made doctor fatal: %v", err)
	}
	for _, expected := range []string{"go1.25.7", "✓ Git", "! Buf not found", "optional tool warning"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("doctor output missing %q: %s", expected, output.String())
		}
	}
	after, err := os.ReadDir(directory)
	if err != nil || len(before) != len(after) {
		t.Fatal("doctor mutated filesystem")
	}
}

func TestDoctorReportsInvalidProjectManifest(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(directory+"/macro.yaml", []byte("name: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(directory)
	var output bytes.Buffer
	err := runDoctor(context.Background(), &output, fakeEnvironment{
		available: map[string]bool{"go": true}, version: "go version go1.25.7 test/amd64",
	})
	if err == nil || !strings.Contains(output.String(), "macro.yaml invalid") {
		t.Fatalf("error = %v, output = %s", err, output.String())
	}
}

func TestDoctorValidatesProjectAndWorkspaceManifests(t *testing.T) {
	parent := t.TempDir()
	if err := os.WriteFile(parent+"/macro.workspace.yaml", []byte("name: test\nprojects: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	generated, err := project.Generate(project.GenerateOptions{Parent: parent, Name: "billing", Type: project.TypeService})
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(generated.Directory)
	var output bytes.Buffer
	err = runDoctor(context.Background(), &output, fakeEnvironment{
		available: map[string]bool{"go": true}, version: "go version go1.25.7 test/amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"macro.yaml valid", "workspace consistent"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("doctor output missing %q: %s", expected, output.String())
		}
	}
}

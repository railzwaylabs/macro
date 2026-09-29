package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/railzwaylabs/macro/internal/project"
	"github.com/railzwaylabs/macro/internal/workspace"
)

func TestInitDryRunDoesNotMutateOrRunCommands(t *testing.T) {
	parent := t.TempDir()
	root, err := workspace.Init(parent, "commerce")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	runner := &fakeRunner{bufAvailable: true}
	var output bytes.Buffer
	command := newRootCommand(&output, &bytes.Buffer{}, BuildInfo{}, runner)
	command.SetArgs([]string{"init", "billing", "--type", "service", "--dry-run"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "billing")); !os.IsNotExist(err) {
		t.Fatal("dry-run created project files")
	}
	if len(runner.commands) != 0 {
		t.Fatalf("dry-run commands = %#v", runner.commands)
	}
	manifest, err := workspace.Read(root)
	if err != nil || len(manifest.Projects) != 0 {
		t.Fatalf("workspace mutated: %#v, %v", manifest, err)
	}
	for _, expected := range []string{"api/proto/billing/v1/billing.proto", "buf generate", "go mod tidy", "commerce", "No changes were written."} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("plan missing %q: %s", expected, output.String())
		}
	}
}

func TestAddModuleDryRunDoesNotMutate(t *testing.T) {
	generated, err := project.Generate(project.GenerateOptions{Parent: t.TempDir(), Name: "billing", Type: project.TypeService})
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(generated.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(generated.Directory)
	var output bytes.Buffer
	command := newRootCommand(&output, &bytes.Buffer{}, BuildInfo{}, &fakeRunner{})
	command.SetArgs([]string{"add", "module", "invoice", "--dry-run"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(generated.ManifestPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("dry-run changed macro.yaml")
	}
	if _, err := os.Stat(filepath.Join(generated.Directory, "internal", "invoice")); !os.IsNotExist(err) {
		t.Fatal("dry-run created module")
	}
	for _, expected := range []string{"internal/invoice/application/service.go", "_invoice.up.sql", "macro.yaml", "No changes were written."} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("plan missing %q: %s", expected, output.String())
		}
	}
}

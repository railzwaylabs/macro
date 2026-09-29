package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/railzwaylabs/macro/internal/project"
)

func TestWorkspaceCommands(t *testing.T) {
	parent := t.TempDir()
	t.Chdir(parent)

	var stdout bytes.Buffer
	cmd := New(&stdout, &bytes.Buffer{}, BuildInfo{})
	cmd.SetArgs([]string{"workspace", "init", "commerce"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("workspace init: %v", err)
	}
	if !strings.Contains(stdout.String(), "✓ Created workspace commerce") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	commerce := filepath.Join(parent, "commerce")
	external, err := project.Generate(project.GenerateOptions{Parent: parent, Name: "catalog", Type: project.TypeService})
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(commerce)
	stdout.Reset()
	cmd = New(&stdout, &bytes.Buffer{}, BuildInfo{})
	cmd.SetArgs([]string{"workspace", "add", external.Directory})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("workspace add: %v", err)
	}

	stdout.Reset()
	cmd = New(&stdout, &bytes.Buffer{}, BuildInfo{})
	cmd.SetArgs([]string{"workspace", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("workspace list: %v", err)
	}
	output := stdout.String()
	for _, value := range []string{"NAME", "TYPE", "PATH", "catalog", "service", "1 projects"} {
		if !strings.Contains(output, value) {
			t.Errorf("workspace list missing %q: %q", value, output)
		}
	}

	if _, err := os.Stat(filepath.Join(commerce, "macro.workspace.yaml")); err != nil {
		t.Fatalf("workspace manifest: %v", err)
	}
}

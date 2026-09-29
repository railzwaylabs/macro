package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/railzwaylabs/macro/internal/project"
)

func TestAddModuleCommand(t *testing.T) {
	generated, err := project.Generate(project.GenerateOptions{
		Parent: t.TempDir(),
		Name:   "billing",
		Type:   project.TypeService,
	})
	if err != nil {
		t.Fatal(err)
	}

	nestedDirectory := filepath.Join(generated.Directory, "cmd", "service")
	t.Chdir(nestedDirectory)

	var output bytes.Buffer
	command := New(&output, &bytes.Buffer{}, BuildInfo{})
	command.SetArgs([]string{"add", "module", "invoice-processing"})
	if err := command.Execute(); err != nil {
		t.Fatalf("execute add module: %v", err)
	}

	handlerPath := filepath.Join(
		generated.Directory,
		"internal",
		"invoice-processing",
		"transport",
		"grpc",
		"handler.go",
	)
	handler, err := os.ReadFile(handlerPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(handler), "type Handler struct{}") || !strings.Contains(string(handler), "func NewHandler() *Handler") {
		t.Fatalf("handler.go = %q", handler)
	}
	migrations, err := filepath.Glob(filepath.Join(generated.Directory, "migrations", "*_invoice_processing.*.sql"))
	if err != nil || len(migrations) != 2 {
		t.Fatalf("migrations = %#v, error = %v", migrations, err)
	}
	if !strings.Contains(output.String(), "✓ Created module invoice-processing") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestAddModuleCommandRequiresProject(t *testing.T) {
	t.Chdir(t.TempDir())

	command := New(&bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	command.SetArgs([]string{"add", "module", "invoice"})
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "macro project not found") {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestQuietModePreservesActionableDuplicateError(t *testing.T) {
	generated, err := project.Generate(project.GenerateOptions{Parent: t.TempDir(), Name: "billing", Type: project.TypeService})
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(generated.Directory)
	command := New(&bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	command.SetArgs([]string{"add", "module", "invoice"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	command = New(&bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	command.SetArgs([]string{"add", "module", "invoice", "--quiet"})
	err = command.Execute()
	if err == nil || !strings.Contains(err.Error(), "choose another module name") {
		t.Fatalf("duplicate error = %v", err)
	}
}

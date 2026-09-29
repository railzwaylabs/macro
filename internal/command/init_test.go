package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/railzwaylabs/macro/internal/project"
	macroworkspace "github.com/railzwaylabs/macro/internal/workspace"
)

func TestInitCommandCreatesServiceProject(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)

	var stdout bytes.Buffer
	cmd := newRootCommand(&stdout, &bytes.Buffer{}, BuildInfo{}, &fakeRunner{bufAvailable: false})
	cmd.SetArgs([]string{"init", "billing"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute init: %v", err)
	}

	manifest, err := project.Read(filepath.Join(directory, "billing"))
	if err != nil {
		t.Fatalf("read generated manifest: %v", err)
	}
	if manifest.Name != "billing" || manifest.Type != project.TypeService {
		t.Fatalf("manifest = %#v", manifest)
	}
	for _, path := range []string{
		filepath.Join("cmd", "service", "main.go"),
		filepath.Join("api", "proto", "billing", "v1", "billing.proto"),
		"migrations",
		"internal",
		"tests",
		"Dockerfile",
		"buf.yaml",
		"buf.gen.yaml",
		"Makefile",
		"README.md",
	} {
		if _, err := os.Stat(filepath.Join(directory, "billing", path)); err != nil {
			t.Errorf("generated path %s: %v", path, err)
		}
	}
	if !strings.Contains(stdout.String(), "✓ Generated macro.yaml") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Buf is required") || !strings.Contains(stdout.String(), "✓ Resolved Go modules") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.HasPrefix(stdout.String(), "Creating service \"billing\"\n\n") {
		t.Fatalf("stdout heading = %q", stdout.String())
	}
	if _, err := macroworkspace.Find(directory); !errors.Is(err, macroworkspace.ErrNotFound) {
		t.Fatalf("standalone project unexpectedly created a workspace: %v", err)
	}
}

func TestInitCommandCreatesWorkerAndJob(t *testing.T) {
	for _, kind := range []string{"worker", "job"} {
		t.Run(kind, func(t *testing.T) {
			directory := t.TempDir()
			t.Chdir(directory)
			cmd := newRootCommand(&bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{}, &fakeRunner{})
			cmd.SetArgs([]string{"init", "billing_" + kind, "--type", kind})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("execute init: %v", err)
			}
			root := filepath.Join(directory, "billing_"+kind)
			if _, err := os.Stat(filepath.Join(root, "cmd", kind, "main.go")); err != nil {
				t.Fatalf("generated main: %v", err)
			}
			if _, err := os.Stat(filepath.Join(root, "migrations")); !os.IsNotExist(err) {
				t.Fatalf("migrations exists for %s", kind)
			}
		})
	}
}

func TestInitCommandRejectsExistingDestination(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)
	if err := os.Mkdir("billing", 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand(&bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{}, &fakeRunner{})
	cmd.SetArgs([]string{"init", "billing"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestInitCommandRejectsInvalidType(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)
	cmd := newRootCommand(&bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{}, &fakeRunner{})
	cmd.SetArgs([]string{"init", "billing", "--type", "api"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "invalid project type") {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestInitCommandRegistersProjectInWorkspace(t *testing.T) {
	parent := t.TempDir()
	root, err := macroworkspace.Init(parent, "commerce")
	if err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "teams", "payments")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)

	var stdout bytes.Buffer
	cmd := newRootCommand(&stdout, &bytes.Buffer{}, BuildInfo{}, &fakeRunner{bufAvailable: false})
	cmd.SetArgs([]string{"init", "billing"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute init: %v", err)
	}

	manifest, err := macroworkspace.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Projects) != 1 || manifest.Projects[0].Path != "./teams/payments/billing" {
		t.Fatalf("projects = %#v", manifest.Projects)
	}
	if !strings.Contains(stdout.String(), "✓ Added billing to workspace commerce") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestInitRunsBufBeforeGoModTidyForService(t *testing.T) {
	runner := &fakeRunner{bufAvailable: true}
	if err := initializeProject(context.Background(), &bytes.Buffer{}, t.TempDir(), "billing", project.TypeService, "", runner, false); err != nil {
		t.Fatal(err)
	}
	want := []string{"buf generate", "go mod tidy"}
	if strings.Join(runner.commands, "|") != strings.Join(want, "|") {
		t.Fatalf("commands = %#v, want %#v", runner.commands, want)
	}
}

func TestInitDoesNotRunBufForWorkerOrJob(t *testing.T) {
	for _, kind := range []project.Type{project.TypeWorker, project.TypeJob} {
		runner := &fakeRunner{bufAvailable: true}
		if err := initializeProject(context.Background(), &bytes.Buffer{}, t.TempDir(), "task", kind, "", runner, false); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(runner.commands, "|"); got != "go mod tidy" {
			t.Fatalf("%s commands = %q", kind, got)
		}
	}
}

func TestInitTidyFailureDoesNotRegisterWorkspace(t *testing.T) {
	root, err := macroworkspace.Init(t.TempDir(), "commerce")
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{err: errors.New("tidy failed")}
	if err := initializeProject(context.Background(), &bytes.Buffer{}, root, "billing", project.TypeService, "", runner, false); err == nil {
		t.Fatal("initializeProject() error = nil")
	}
	manifest, err := macroworkspace.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Projects) != 0 {
		t.Fatalf("workspace projects = %#v", manifest.Projects)
	}
	if _, err := os.Stat(filepath.Join(root, "billing")); err != nil {
		t.Fatalf("generated project should remain available: %v", err)
	}
}

type fakeRunner struct {
	bufAvailable bool
	commands     []string
	err          error
}

func (runner *fakeRunner) LookPath(name string) (string, error) {
	if name == "buf" && runner.bufAvailable {
		return "/test/bin/buf", nil
	}
	return "", os.ErrNotExist
}

func (runner *fakeRunner) Run(_ context.Context, _ string, name string, arguments ...string) error {
	runner.commands = append(runner.commands, strings.Join(append([]string{name}, arguments...), " "))
	return runner.err
}

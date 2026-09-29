package command

import (
	"bytes"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := New(&stdout, &bytes.Buffer{}, BuildInfo{
		Version: "v0.1.0",
		Commit:  "abc123",
		Date:    "2026-09-28",
	})
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute version: %v", err)
	}

	const want = "macro v0.1.0 (commit: abc123, built: 2026-09-28)\n"
	if got := stdout.String(); got != want {
		t.Fatalf("unexpected output\nwant: %q\n got: %q", want, got)
	}
}

func TestVersionFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := New(&stdout, &bytes.Buffer{}, BuildInfo{
		Version: "v0.1.1",
		Commit:  "def456",
		Date:    "2026-09-29",
	})
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute --version: %v", err)
	}

	const want = "macro v0.1.1 (commit: def456, built: 2026-09-29)\n"
	if got := stdout.String(); got != want {
		t.Fatalf("unexpected output\nwant: %q\n got: %q", want, got)
	}
}

func TestRootCommandShowsHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := New(&stdout, &bytes.Buffer{}, BuildInfo{})
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute root command: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"A toolkit for scaffolding consistent Go services, workers, and jobs",
		"macro init billing",
		"macro workspace init commerce",
		"Usage:",
		"Common Commands:",
		"Management Commands:",
		"Commands:",
	} {
		if !bytes.Contains([]byte(got), []byte(want)) {
			t.Errorf("help output does not contain %q: %q", want, got)
		}
	}
}

func TestInitHelpExplainsWorkspaceRegistration(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := New(&stdout, &bytes.Buffer{}, BuildInfo{})
	cmd.SetArgs([]string{"init", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute init --help: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"service, worker, or job", "automatically registered", "--module"} {
		if !bytes.Contains([]byte(got), []byte(want)) {
			t.Errorf("init help does not contain %q: %q", want, got)
		}
	}
}

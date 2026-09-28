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

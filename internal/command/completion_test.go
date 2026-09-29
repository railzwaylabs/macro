package command

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletionSupportedShells(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			var output bytes.Buffer
			command := New(&output, &bytes.Buffer{}, BuildInfo{})
			command.SetArgs([]string{"completion", shell})
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			if output.Len() == 0 {
				t.Fatal("completion output is empty")
			}
		})
	}
}

func TestCompletionRejectsUnsupportedShell(t *testing.T) {
	command := New(&bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	command.SetArgs([]string{"completion", "nushell"})
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "use bash, zsh, fish, or powershell") {
		t.Fatalf("error = %v", err)
	}
}

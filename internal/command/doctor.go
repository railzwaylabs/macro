package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/railzwaylabs/macro/internal/project"
	"github.com/railzwaylabs/macro/internal/workspace"
)

type environmentChecker interface {
	LookPath(string) (string, error)
	Output(context.Context, string, ...string) (string, error)
}

type osEnvironmentChecker struct{}

func (osEnvironmentChecker) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (osEnvironmentChecker) Output(ctx context.Context, name string, arguments ...string) (string, error) {
	output, err := exec.CommandContext(ctx, name, arguments...).CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func newDoctorCommand(checker environmentChecker) *cobra.Command {
	return &cobra.Command{
		Use:     "doctor",
		Short:   "Check local development requirements",
		Long:    "Check local tools and validate the current Macro project and workspace context.",
		Example: "  macro doctor",
		Args:    cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return runDoctor(command.Context(), command.OutOrStdout(), checker)
		},
	}
}

func runDoctor(ctx context.Context, output io.Writer, checker environmentChecker) error {
	if _, err := fmt.Fprint(output, "Macro Doctor\n\n"); err != nil {
		return err
	}
	failures := 0
	warnings := 0

	if _, err := checker.LookPath("go"); err != nil {
		failures++
		_, _ = fmt.Fprintln(output, "✗ Go not found")
	} else if version, err := checker.Output(ctx, "go", "version"); err != nil {
		failures++
		_, _ = fmt.Fprintln(output, "✗ Go version could not be read")
	} else {
		_, _ = fmt.Fprintf(output, "✓ %s\n", strings.TrimPrefix(version, "go version "))
	}

	for _, tool := range []struct{ binary, label, guidance string }{
		{"git", "Git", "Required for source control."},
		{"buf", "Buf", "Required for service protobuf generation."},
		{"docker", "Docker", "Required for `make docker`."},
		{"golangci-lint", "golangci-lint", "Required for `make lint`."},
	} {
		if _, err := checker.LookPath(tool.binary); err != nil {
			warnings++
			_, _ = fmt.Fprintf(output, "! %s not found — %s\n", tool.label, tool.guidance)
		} else {
			_, _ = fmt.Fprintf(output, "✓ %s\n", tool.label)
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}
	if directory, findErr := project.Find(cwd); findErr == nil {
		if _, readErr := project.Read(directory); readErr != nil {
			failures++
			_, _ = fmt.Fprintf(output, "✗ macro.yaml invalid — %v\n", readErr)
		} else {
			_, _ = fmt.Fprintln(output, "✓ macro.yaml valid")
		}
	} else if !errors.Is(findErr, project.ErrNotFound) {
		failures++
		_, _ = fmt.Fprintf(output, "✗ project context invalid — %v\n", findErr)
	}

	if directory, findErr := workspace.Find(cwd); findErr == nil {
		if _, readErr := workspace.Read(directory); readErr != nil {
			failures++
			_, _ = fmt.Fprintf(output, "✗ macro.workspace.yaml invalid — %v\n", readErr)
		} else if _, listErr := workspace.List(directory); listErr != nil {
			failures++
			_, _ = fmt.Fprintf(output, "✗ workspace inconsistent — %v\n", listErr)
		} else {
			_, _ = fmt.Fprintln(output, "✓ workspace consistent")
		}
	} else if !errors.Is(findErr, workspace.ErrNotFound) {
		failures++
		_, _ = fmt.Fprintf(output, "✗ workspace context invalid — %v\n", findErr)
	}

	if failures > 0 {
		_, _ = fmt.Fprintf(output, "\n%d issues found.\n", failures)
		return fmt.Errorf("doctor found %d required issue(s)", failures)
	}
	if warnings > 0 {
		_, _ = fmt.Fprintf(output, "\nEnvironment ready with %d optional tool warning(s).\n", warnings)
		return nil
	}
	_, err = fmt.Fprintln(output, "\nAll checks passed.")
	return err
}

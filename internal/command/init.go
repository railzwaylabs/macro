package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/railzwaylabs/macro/internal/project"
	"github.com/railzwaylabs/macro/internal/ui"
	"github.com/railzwaylabs/macro/internal/workspace"
)

func newInitCommand(runner project.CommandRunner, cli *cliOptions) *cobra.Command {
	var (
		typeValue  string
		modulePath string
	)

	cmd := &cobra.Command{
		Use:   "init <name>",
		Short: "Create a new Macro project",
		Long: `Create a service, worker, or job in a new directory.

The project is automatically registered when the command is run inside a
Macro workspace. A service is created when --type is omitted.`,
		Example: `  macro init billing
  macro init billing --module github.com/example/billing
  macro init rating-worker --type worker
  macro init daily-report --type job`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, arguments []string) error {
			kind, err := project.ParseType(strings.ToLower(strings.TrimSpace(typeValue)))
			if err != nil {
				return err
			}
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get current directory: %w", err)
			}

			if cli.dryRun {
				return renderInitPlan(cmd.OutOrStdout(), cwd, arguments[0], kind, modulePath)
			}
			return initializeProject(cmd.Context(), cmd.OutOrStdout(), cwd, arguments[0], kind, modulePath, runner, cli.quiet)
		},
	}

	cmd.Flags().StringVarP(&typeValue, "type", "t", string(project.TypeService), "project type (service, worker, or job)")
	cmd.Flags().StringVar(&modulePath, "module", "", "Go module path (default: project name)")
	cmd.Flags().BoolVar(&cli.dryRun, "dry-run", false, "show planned changes without applying them")
	return cmd
}

func initializeProject(ctx context.Context, output io.Writer, parent, name string, kind project.Type, modulePath string, runner project.CommandRunner, quiet bool) error {
	statusOutput := output
	if quiet {
		statusOutput = io.Discard
	}
	status := ui.NewStatus(statusOutput)
	if err := status.Heading(fmt.Sprintf("Creating %s %q", kind, name)); err != nil {
		return fmt.Errorf("render project heading: %w", err)
	}

	generateOptions := project.GenerateOptions{
		Parent:     parent,
		Name:       name,
		Type:       kind,
		ModulePath: modulePath,
	}

	var generatedProject project.Result
	generateStep := ui.Step{
		Start:   "Generating project files",
		Success: fmt.Sprintf("Created %s", name),
		Failure: fmt.Sprintf("Failed to create %s", name),
		Run: func() error {
			var err error
			generatedProject, err = project.Generate(generateOptions)
			return err
		},
	}
	if err := status.Run(generateStep); err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	if err := status.Success("Generated " + project.ManifestName); err != nil {
		return err
	}

	if kind == project.TypeService {
		protoStep := ui.Step{
			Start:   "Generating protobuf code",
			Success: "Generated protobuf code",
			Failure: "Failed to generate protobuf code",
			Run:     func() error { return project.GenerateProto(ctx, generatedProject.Directory, runner) },
		}
		if err := status.Run(protoStep); err != nil {
			if !errors.Is(err, project.ErrBufNotFound) {
				return fmt.Errorf("generate protobuf code: %w", err)
			}
			if warningErr := status.Warning("Buf is required to generate protobuf Go code. Install Buf, then run `buf generate` and `go mod tidy`."); warningErr != nil {
				return errors.Join(err, warningErr)
			}
		}
	}

	tidyStep := ui.Step{
		Start:   "Resolving Go modules",
		Success: "Resolved Go modules",
		Failure: "Failed to resolve Go modules",
		Run:     func() error { return project.Tidy(ctx, generatedProject.Directory, runner) },
	}
	if err := status.Run(tidyStep); err != nil {
		return fmt.Errorf("resolve Go modules: %w", err)
	}

	workspaceDirectory, err := workspace.Find(parent)
	if errors.Is(err, workspace.ErrNotFound) {
		return status.Success("Project ready: ./" + name)
	}
	if err != nil {
		return fmt.Errorf("find workspace after creating project: %w", err)
	}

	registration, err := workspace.Add(workspaceDirectory, generatedProject.Directory)
	if err != nil {
		operationErr := fmt.Errorf("register project in workspace: %w", err)
		renderErr := status.Failure(fmt.Sprintf("Failed to add %s to workspace", name))
		return errors.Join(operationErr, renderErr)
	}

	message := fmt.Sprintf("Added %s to workspace %s", registration.ProjectName, registration.WorkspaceName)
	if err := status.Success(message); err != nil {
		return err
	}
	return status.Success("Project ready: ./" + name)
}

func renderInitPlan(output io.Writer, parent, name string, kind project.Type, modulePath string) error {
	plan, err := project.BuildPlan(project.GenerateOptions{Parent: parent, Name: name, Type: kind, ModulePath: modulePath})
	if err != nil {
		return fmt.Errorf("plan project: %w", err)
	}
	if _, err := fmt.Fprintln(output, "Would create:"); err != nil {
		return err
	}
	for _, path := range plan.CreateFiles {
		if _, err := fmt.Fprintf(output, "  %s\n", path); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(output, "\nWould run:"); err != nil {
		return err
	}
	for _, command := range plan.Commands {
		if _, err := fmt.Fprintf(output, "  %s\n", command); err != nil {
			return err
		}
	}
	workspaceDirectory, findErr := workspace.Find(parent)
	if findErr == nil {
		manifest, readErr := workspace.Read(workspaceDirectory)
		if readErr != nil {
			return fmt.Errorf("read workspace for dry run: %w", readErr)
		}
		if _, err := fmt.Fprintf(output, "\nWould register in workspace:\n  %s\n", manifest.Name); err != nil {
			return err
		}
	} else if !errors.Is(findErr, workspace.ErrNotFound) {
		return fmt.Errorf("find workspace for dry run: %w", findErr)
	}
	_, err = fmt.Fprintln(output, "\nNo changes were written.")
	return err
}

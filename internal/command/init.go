package command

import (
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

func newInitCommand() *cobra.Command {
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
  macro init close-cycle --type job`,
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

			return initializeProject(cmd.OutOrStdout(), cwd, arguments[0], kind, modulePath)
		},
	}

	cmd.Flags().StringVarP(&typeValue, "type", "t", string(project.TypeService), "project type (service, worker, or job)")
	cmd.Flags().StringVar(&modulePath, "module", "", "Go module path (default: project name)")
	return cmd
}

func initializeProject(output io.Writer, parent, name string, kind project.Type, modulePath string) error {
	status := ui.NewStatus(output)
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

	workspaceDirectory, err := workspace.Find(parent)
	if errors.Is(err, workspace.ErrNotFound) {
		return nil
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
	return status.Success(message)
}

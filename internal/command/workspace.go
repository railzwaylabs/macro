package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/railzwaylabs/macro/internal/ui"
	"github.com/railzwaylabs/macro/internal/workspace"
)

func newWorkspaceCommand(cli *cliOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage a collection of Macro projects",
		Long: `Manage projects using the nearest macro.workspace.yaml file.

Macro searches the current directory and its parents for the workspace.`,
		Example: `  macro workspace init commerce
  macro workspace add ../catalog
  macro workspace list`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddGroup(&cobra.Group{ID: "commands", Title: "Commands:"})
	initCommand := newWorkspaceInitCommand(cli)
	initCommand.GroupID = "commands"
	addCommand := newWorkspaceAddCommand(cli)
	addCommand.GroupID = "commands"
	listCommand := newWorkspaceListCommand()
	listCommand.GroupID = "commands"
	cmd.AddCommand(initCommand, addCommand, listCommand)
	return cmd
}

func newWorkspaceInitCommand(cli *cliOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "init <name>",
		Short: "Create a new Macro workspace",
		Long: `Create a directory containing an empty macro.workspace.yaml manifest.

Projects created beneath this directory are registered automatically.`,
		Example: `  macro workspace init commerce
  cd commerce
  macro init billing`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get current directory: %w", err)
			}

			return initializeWorkspace(cmd.OutOrStdout(), cwd, args[0], cli.quiet)
		},
	}
}

func initializeWorkspace(output io.Writer, parent, name string, quiet bool) error {
	if quiet {
		output = io.Discard
	}
	status := ui.NewStatus(output)
	createStep := ui.Step{
		Start:   fmt.Sprintf("Creating workspace %q", name),
		Success: fmt.Sprintf("Created workspace %s", name),
		Failure: fmt.Sprintf("Failed to create workspace %s", name),
		Run: func() error {
			_, err := workspace.Init(parent, name)
			return err
		},
	}

	if err := status.Run(createStep); err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	return status.Success("Created " + workspace.ManifestName)
}

func newWorkspaceAddCommand(cli *cliOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "Add an existing project to the nearest workspace",
		Long: `Validate and register an existing Macro project in the nearest workspace.

The path is stored relative to the workspace directory.`,
		Example: `  macro workspace add ./billing
  macro workspace add ../catalog`,
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		RunE: func(cmd *cobra.Command, arguments []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get current directory: %w", err)
			}

			workspaceDirectory, err := workspace.Find(cwd)
			if err != nil {
				return fmt.Errorf("find workspace: %w", err)
			}

			registration, err := workspace.Add(workspaceDirectory, arguments[0])
			if err != nil {
				statusOutput := cmd.OutOrStdout()
				if cli.quiet {
					statusOutput = io.Discard
				}
				status := ui.NewStatus(statusOutput)
				operationErr := fmt.Errorf("add project: %w", err)
				renderErr := status.Failure("Failed to add project to workspace")
				return errors.Join(operationErr, renderErr)
			}

			statusOutput := cmd.OutOrStdout()
			if cli.quiet {
				statusOutput = io.Discard
			}
			status := ui.NewStatus(statusOutput)
			message := fmt.Sprintf("Added %s to workspace %s", registration.ProjectName, registration.WorkspaceName)
			return status.Success(message)
		},
	}
}

func newWorkspaceListCommand() *cobra.Command {
	var pathsOnly bool
	var format string
	command := &cobra.Command{
		Use:   "list",
		Short: "List projects in the nearest workspace",
		Long: `List registered projects using metadata from each project's macro.yaml.

Macro reports missing or malformed project manifests as workspace errors.`,
		Example: `  macro workspace list`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get current directory: %w", err)
			}
			workspaceDirectory, err := workspace.Find(cwd)
			if err != nil {
				return fmt.Errorf("find workspace: %w", err)
			}

			entries, listErr := workspace.List(workspaceDirectory)
			if pathsOnly {
				if err := renderWorkspacePaths(cmd.OutOrStdout(), entries); err != nil {
					return err
				}
			} else {
				switch format {
				case "table":
					if err := renderWorkspaceProjects(cmd.OutOrStdout(), entries); err != nil {
						return err
					}
				case "json":
					if err := renderWorkspaceJSON(cmd.OutOrStdout(), entries); err != nil {
						return err
					}
				default:
					return fmt.Errorf("unsupported format %q: use table or json", format)
				}
			}
			if listErr != nil {
				return fmt.Errorf("workspace is inconsistent: %w", listErr)
			}
			return nil
		},
	}
	command.Flags().BoolVar(&pathsOnly, "paths", false, "print project paths only")
	command.Flags().StringVar(&format, "format", "table", "output format (table or json)")
	return command
}

func renderWorkspaceJSON(output io.Writer, entries []workspace.Entry) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entries)
}

func renderWorkspacePaths(output io.Writer, entries []workspace.Entry) error {
	for _, entry := range entries {
		if _, err := fmt.Fprintln(output, entry.Path); err != nil {
			return err
		}
	}
	return nil
}

func renderWorkspaceProjects(output io.Writer, entries []workspace.Entry) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "NAME\tTYPE\tPATH"); err != nil {
		return err
	}

	for _, entry := range entries {
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\n", entry.Name, entry.Type, entry.Path); err != nil {
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	_, err := fmt.Fprintf(output, "\n%d projects\n", len(entries))
	return err
}

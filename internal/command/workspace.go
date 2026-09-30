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
  macro workspace init commerce --profile traefik-nomad
  macro workspace infra add postgres
  macro workspace infra generate
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
	infraCommand := newWorkspaceInfrastructureCommand(cli)
	infraCommand.GroupID = "commands"
	cmd.AddCommand(initCommand, addCommand, listCommand, infraCommand)
	return cmd
}

func newWorkspaceInitCommand(cli *cliOptions) *cobra.Command {
	var profile string
	cmd := &cobra.Command{
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

			parsed := workspace.Profile(profile)
			if err := workspace.ValidateProfile(parsed); err != nil {
				return err
			}
			if cli.dryRun {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "Would create workspace %s with infrastructure profile %s.\nNo changes were written.\n", args[0], parsed)
				return err
			}
			return initializeWorkspace(cmd.OutOrStdout(), cwd, args[0], parsed, cli.quiet)
		},
	}
	cmd.Flags().StringVar(&profile, "profile", string(workspace.ProfileNginxCompose), "infrastructure profile (nginx-compose or traefik-nomad)")
	cmd.Flags().BoolVar(&cli.dryRun, "dry-run", false, "show planned changes without applying them")
	return cmd
}

func initializeWorkspace(output io.Writer, parent, name string, profile workspace.Profile, quiet bool) error {
	if quiet {
		output = io.Discard
	}
	status := ui.NewStatus(output)
	createStep := ui.Step{
		Start:   fmt.Sprintf("Creating workspace %q", name),
		Success: fmt.Sprintf("Created workspace %s", name),
		Failure: fmt.Sprintf("Failed to create workspace %s", name),
		Run: func() error {
			_, err := workspace.InitWithProfile(parent, name, profile)
			return err
		},
	}

	if err := status.Run(createStep); err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	return status.Success("Created " + workspace.ManifestName)
}

func newWorkspaceInfrastructureCommand(cli *cliOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "infra", Short: "Manage shared infrastructure configuration", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() }}
	cmd.AddCommand(newWorkspaceInfrastructureAddCommand(cli), newWorkspaceInfrastructureGenerateCommand(cli))
	return cmd
}

func currentWorkspace() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current directory: %w", err)
	}
	root, err := workspace.Find(cwd)
	if err != nil {
		return "", fmt.Errorf("find workspace: %w", err)
	}
	return root, nil
}

func newWorkspaceInfrastructureAddCommand(cli *cliOptions) *cobra.Command {
	var profile string
	cmd := &cobra.Command{Use: "add <postgres|redis|observability>", Short: "Enable an optional infrastructure component", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		root, err := currentWorkspace()
		if err != nil {
			return err
		}
		manifest, err := workspace.Read(root)
		if err != nil {
			return err
		}
		selected := workspace.Profile(profile)
		if manifest.Infrastructure != nil {
			selected = manifest.Infrastructure.Profile
		}
		if err := workspace.ValidateProfile(selected); err != nil {
			return err
		}
		if cli.dryRun {
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Would enable %s using profile %s.\nNo changes were written.\n", args[0], selected)
			return err
		}
		if err := workspace.AddInfrastructure(root, args[0], selected); err != nil {
			return fmt.Errorf("add infrastructure: %w", err)
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Enabled infrastructure component %s.\n", args[0])
		return err
	}}
	cmd.Flags().StringVar(&profile, "profile", string(workspace.ProfileNginxCompose), "profile used when initializing infrastructure in a legacy workspace")
	cmd.Flags().BoolVar(&cli.dryRun, "dry-run", false, "show planned changes without applying them")
	return cmd
}

func newWorkspaceInfrastructureGenerateCommand(cli *cliOptions) *cobra.Command {
	var force bool
	cmd := &cobra.Command{Use: "generate", Short: "Generate shared infrastructure configuration", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		root, err := currentWorkspace()
		if err != nil {
			return err
		}
		files, err := workspace.PlanInfrastructure(root)
		if err != nil {
			return fmt.Errorf("plan infrastructure: %w", err)
		}
		if cli.dryRun {
			fmt.Fprintln(cmd.OutOrStdout(), "Would generate:")
			for _, file := range files {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s/%s\n", workspace.InfrastructureDirectory, file.Path)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "\nNo changes were written.")
			return nil
		}
		if _, err = workspace.GenerateInfrastructure(root, force); err != nil {
			return fmt.Errorf("generate infrastructure: %w", err)
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Generated %d files in %s.\n", len(files), workspace.InfrastructureDirectory)
		return err
	}}
	cmd.Flags().BoolVar(&force, "force", false, "replace the generator-managed infrastructure directory")
	cmd.Flags().BoolVar(&cli.dryRun, "dry-run", false, "show planned changes without applying them")
	return cmd
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

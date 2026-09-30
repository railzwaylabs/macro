package command

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/railzwaylabs/macro/internal/grpcscaffold"
	modulegenerator "github.com/railzwaylabs/macro/internal/module"
	"github.com/railzwaylabs/macro/internal/project"
	"github.com/railzwaylabs/macro/internal/ui"
)

func newAddCommand(runner project.CommandRunner, cli *cliOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "add",
		Short: "Add a component to a Macro project",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}
	command.AddGroup(&cobra.Group{ID: "components", Title: "Components:"})

	moduleCommand := newAddModuleCommand(cli)
	moduleCommand.GroupID = "components"
	command.AddCommand(moduleCommand)
	grpcCommand := newAddGRPCCommand(runner, cli)
	grpcCommand.GroupID = "components"
	command.AddCommand(grpcCommand)

	return command
}

func newAddGRPCCommand(runner project.CommandRunner, cli *cliOptions) *cobra.Command {
	var gateway bool
	command := &cobra.Command{
		Use: "grpc <module>", Short: "Add a protobuf gRPC transport to a module",
		Example: "  macro add grpc product\n  macro add grpc product --gateway",
		Args:    cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			workingDirectory, err := os.Getwd()
			if err != nil {
				return err
			}
			projectDirectory, err := project.Find(workingDirectory)
			if err != nil {
				return fmt.Errorf("find Macro project: %w", err)
			}
			name := arguments[0]
			if cli.dryRun {
				_, err := fmt.Fprintf(command.OutOrStdout(), "Would generate gRPC transport for %s (gateway: %t).\nNo changes were written.\n", name, gateway)
				return err
			}
			statusOutput := command.OutOrStdout()
			if cli.quiet {
				statusOutput = io.Discard
			}
			status := ui.NewStatus(statusOutput)
			if err := status.Run(ui.Step{Start: fmt.Sprintf("Generating gRPC transport for %q", name), Success: fmt.Sprintf("Generated gRPC transport for %s", name), Failure: fmt.Sprintf("Failed to generate gRPC transport for %s", name), Run: func() error {
				_, err := grpcscaffold.Add(grpcscaffold.Options{ProjectDirectory: projectDirectory, Name: name, Gateway: gateway})
				return err
			}}); err != nil {
				return fmt.Errorf("add gRPC transport: %w", err)
			}
			if gateway {
				if err := runner.Run(command.Context(), projectDirectory, "buf", "dep", "update"); err != nil {
					return fmt.Errorf("resolve protobuf dependencies: %w", err)
				}
			}
			if err := project.GenerateProto(command.Context(), projectDirectory, runner); err != nil {
				return fmt.Errorf("generate protobuf code: %w", err)
			}
			if err := project.Tidy(command.Context(), projectDirectory, runner); err != nil {
				return fmt.Errorf("tidy generated project: %w", err)
			}
			return nil
		},
	}
	command.Flags().BoolVar(&gateway, "gateway", false, "also expose annotated HTTP/JSON routes")
	command.Flags().BoolVar(&cli.dryRun, "dry-run", false, "show planned changes without applying them")
	return command
}

func newAddModuleCommand(cli *cliOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "module <name>",
		Short: "Add minimal application package boundaries",
		Long: `Create minimal application and gRPC transport boundaries, a migration
pair, and register the module without inventing business behavior.`,
		Example: `  macro add module invoice
  macro add module invoice-processing`,
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(command *cobra.Command, arguments []string) error {
			workingDirectory, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get current directory: %w", err)
			}

			projectDirectory, err := project.Find(workingDirectory)
			if err != nil {
				return fmt.Errorf("find Macro project: %w", err)
			}

			name := arguments[0]
			if cli.dryRun {
				return renderModulePlan(command.OutOrStdout(), projectDirectory, name, time.Now().UTC())
			}
			statusOutput := command.OutOrStdout()
			if cli.quiet {
				statusOutput = io.Discard
			}
			status := ui.NewStatus(statusOutput)
			step := ui.Step{
				Start:   fmt.Sprintf("Creating module %q", name),
				Success: fmt.Sprintf("Created module %s", name),
				Failure: fmt.Sprintf("Failed to create module %s", name),
				Run: func() error {
					_, err := modulegenerator.Add(projectDirectory, name)
					return err
				},
			}

			if err := status.Run(step); err != nil {
				return fmt.Errorf("add module: %w", err)
			}

			return nil
		},
	}
	command.Flags().BoolVar(&cli.dryRun, "dry-run", false, "show planned changes without applying them")
	return command
}

func renderModulePlan(output io.Writer, projectDirectory, name string, timestamp time.Time) error {
	plan, err := modulegenerator.BuildPlan(projectDirectory, name, timestamp)
	if err != nil {
		return fmt.Errorf("plan module: %w", err)
	}
	if _, err := fmt.Fprintln(output, "Would create:"); err != nil {
		return err
	}
	for _, path := range plan.CreateFiles {
		relative, relErr := filepath.Rel(projectDirectory, path)
		if relErr != nil {
			return relErr
		}
		if _, err := fmt.Fprintf(output, "  %s\n", filepath.ToSlash(relative)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(output, "\nWould update:"); err != nil {
		return err
	}
	for _, path := range plan.UpdateFiles {
		if _, err := fmt.Fprintf(output, "  %s\n", filepath.Base(path)); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(output, "\nNo changes were written.")
	return err
}

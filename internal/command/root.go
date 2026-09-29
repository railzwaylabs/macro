package command

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/railzwaylabs/macro/internal/project"
)

const usageTemplate = `Usage:{{if eq .CommandPath "macro"}}
  macro <command> [flags]{{else}}{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{if eq .CommandPath "macro"}}Global Options:{{else}}Flags:{{end}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Run '{{.CommandPath}} COMMAND --help' for more information on a command.{{end}}
`

// BuildInfo contains version metadata injected into the CLI at build time.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

type cliOptions struct {
	dryRun bool
	quiet  bool
}

// String returns the human-readable CLI build information.
func (i BuildInfo) String() string {
	return fmt.Sprintf(
		"macro %s (commit: %s, built: %s)",
		valueOrDefault(i.Version, "dev"),
		valueOrDefault(i.Commit, "unknown"),
		valueOrDefault(i.Date, "unknown"),
	)
}

// New creates the root macro command.
func New(out, errOut io.Writer, info BuildInfo) *cobra.Command {
	return newRootCommand(out, errOut, info, project.ExecRunner{})
}

func newRootCommand(out, errOut io.Writer, info BuildInfo, runner project.CommandRunner) *cobra.Command {
	options := &cliOptions{}
	cmd := &cobra.Command{
		Use:   "macro <command> [flags]",
		Short: "Build consistent Go services, workers, and jobs",
		Long:  "Macro is a Go workload toolkit and CLI.",
		Example: `  macro init billing
  macro add module invoice
  macro init notifications --type worker
  macro workspace init commerce
  macro workspace list`,
		Version:       info.String(),
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetUsageTemplate(usageTemplate)
	cmd.SetVersionTemplate("{{.Version}}\n")
	cmd.SetHelpCommandGroupID("additional")
	cmd.SetCompletionCommandGroupID("additional")
	cmd.PersistentFlags().BoolVar(&options.quiet, "quiet", false, "suppress progress output")
	cmd.AddGroup(
		&cobra.Group{ID: "core", Title: "Core Commands:"},
		&cobra.Group{ID: "utility", Title: "Utility Commands:"},
		&cobra.Group{ID: "additional", Title: "Additional Commands:"},
	)

	initCommand := newInitCommand(runner, options)
	initCommand.GroupID = "core"
	addCommand := newAddCommand(options)
	addCommand.GroupID = "core"
	versionCommand := newVersionCommand(info)
	versionCommand.GroupID = "utility"
	workspaceCommand := newWorkspaceCommand(options)
	workspaceCommand.GroupID = "core"
	completionCommand := newCompletionCommand()
	completionCommand.GroupID = "utility"
	doctorCommand := newDoctorCommand(osEnvironmentChecker{})
	doctorCommand.GroupID = "utility"

	cmd.AddCommand(initCommand, addCommand, workspaceCommand, doctorCommand, completionCommand, versionCommand)

	return cmd
}

func newVersionCommand(info BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Macro CLI version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), info.String())
			return err
		},
	}
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

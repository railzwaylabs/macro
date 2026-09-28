package command

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// BuildInfo contains version metadata injected into the CLI at build time.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// New creates the root macro command.
func New(out, errOut io.Writer, info BuildInfo) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "macro",
		Short:         "Build and run services with the Macro framework",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.AddCommand(newVersionCommand(info))

	return cmd
}

func newVersionCommand(info BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Macro CLI version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"macro %s (commit: %s, built: %s)\n",
				valueOrDefault(info.Version, "dev"),
				valueOrDefault(info.Commit, "unknown"),
				valueOrDefault(info.Date, "unknown"),
			)
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

package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "completion <bash|zsh|fish|powershell>",
		Short:     "Generate shell completion scripts",
		Long:      "Generate a completion script on stdout. Macro does not modify shell profile files.",
		Example:   "  macro completion zsh > _macro",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(command *cobra.Command, arguments []string) error {
			switch arguments[0] {
			case "bash":
				return command.Root().GenBashCompletion(command.OutOrStdout())
			case "zsh":
				return command.Root().GenZshCompletion(command.OutOrStdout())
			case "fish":
				return command.Root().GenFishCompletion(command.OutOrStdout(), true)
			case "powershell":
				return command.Root().GenPowerShellCompletion(command.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell %q: use bash, zsh, fish, or powershell", arguments[0])
			}
		},
	}
}

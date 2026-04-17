package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:       "completion {bash|zsh|fish|powershell}",
		Short:     "Generate shell completion script",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(out, true)
			case "zsh":
				return cmd.Root().GenZshCompletion(out)
			case "fish":
				return cmd.Root().GenFishCompletion(out, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(out)
			}
			return fmt.Errorf("unsupported shell %q", args[0])
		},
	}
	c.Long = `Generate shell completion script. Source the output into your shell:

    # bash (one-shot)
    source <(triage completion bash)

    # bash (persistent)
    triage completion bash > /etc/bash_completion.d/triage

    # zsh (one-shot)
    source <(triage completion zsh)

    # zsh (persistent, oh-my-zsh)
    triage completion zsh > "${fpath[1]}/_triage"

    # fish
    triage completion fish | source
`
	return c
}

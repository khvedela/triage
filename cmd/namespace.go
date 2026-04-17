package cmd

import (
	"github.com/spf13/cobra"

	"github.com/OWNER/triage/internal/cli"
	"github.com/OWNER/triage/internal/findings"
)

func newNamespaceCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "namespace <name>",
		Aliases: []string{"ns"},
		Short:   "Diagnose every workload in a namespace",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := findings.Target{
				Kind: findings.TargetKindNamespace,
				Name: args[0],
			}
			code := cli.RunDiagnosis(cmd, target, cmd.OutOrStdout())
			setExitCode(cmd, code)
			return nil
		},
	}
	return c
}

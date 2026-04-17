package cmd

import (
	"github.com/spf13/cobra"

	"github.com/khvedela/triage/internal/cli"
	"github.com/khvedela/triage/internal/findings"
)

func newDeploymentCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "deployment <name>",
		Aliases: []string{"deploy"},
		Short:   "Diagnose a deployment (rolls in owned pod findings)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := cli.Get(cmd)
			target := findings.Target{
				Kind:      findings.TargetKindDeployment,
				Namespace: opts.Namespace,
				Name:      args[0],
			}
			code := cli.RunDiagnosis(cmd, target, cmd.OutOrStdout())
			setExitCode(cmd, code)
			return nil
		},
	}
	return c
}

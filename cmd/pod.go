package cmd

import (
	"github.com/spf13/cobra"

	"github.com/khvedela/kubediag/internal/cli"
	"github.com/khvedela/kubediag/internal/findings"
)

func newPodCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pod <name>",
		Short: "Diagnose a single pod",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := cli.Get(cmd)
			target := findings.Target{
				Kind:      findings.TargetKindPod,
				Namespace: opts.Namespace,
				Name:      args[0],
			}
			code := cli.RunDiagnosis(cmd, target, cmd.OutOrStdout())
			setExitCode(cmd, code)
			return nil
		},
	}
}

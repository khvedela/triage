package cmd

import (
	"github.com/spf13/cobra"

	"github.com/khvedela/triage/internal/cli"
	"github.com/khvedela/triage/internal/findings"
)

func newClusterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cluster",
		Short: "Diagnose cluster-wide conditions (node health, recent warnings)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			target := findings.Target{Kind: findings.TargetKindCluster}
			code := cli.RunDiagnosis(cmd, target, cmd.OutOrStdout())
			setExitCode(cmd, code)
			return nil
		},
	}
}

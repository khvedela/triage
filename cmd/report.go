package cmd

import (
	"github.com/spf13/cobra"

	"github.com/khvedela/kubediag/internal/cli"
	"github.com/khvedela/kubediag/internal/findings"
)

// newReportCmd implements `kubediag report {namespace,cluster}` — full markdown
// diagnostic reports with table of contents, tuned defaults, and generous
// max-findings caps.
func newReportCmd() *cobra.Command {
	r := &cobra.Command{
		Use:   "report",
		Short: "Generate a full markdown diagnostic report",
	}
	r.AddCommand(&cobra.Command{
		Use:   "namespace <name>",
		Short: "Write a full markdown report for a namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := cli.Get(cmd)
			opts.Output = "markdown"
			if opts.MaxFindings < 100 {
				opts.MaxFindings = 100
			}
			target := findings.Target{Kind: findings.TargetKindNamespace, Name: args[0]}
			code := cli.RunDiagnosis(cmd, target, cmd.OutOrStdout())
			setExitCode(cmd, code)
			return nil
		},
	})
	r.AddCommand(&cobra.Command{
		Use:   "cluster",
		Short: "Write a full markdown report for the entire cluster",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts := cli.Get(cmd)
			opts.Output = "markdown"
			if opts.MaxFindings < 100 {
				opts.MaxFindings = 100
			}
			target := findings.Target{Kind: findings.TargetKindCluster}
			code := cli.RunDiagnosis(cmd, target, cmd.OutOrStdout())
			setExitCode(cmd, code)
			return nil
		},
	})
	return r
}

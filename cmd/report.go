package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/OWNER/triage/internal/cli"
	"github.com/OWNER/triage/internal/findings"
)

// newReportCmd implements `triage report namespace <ns>` — same as
// `triage namespace <ns> -o markdown` with a few defaults tuned for report
// writing (include events/related on; higher max-findings cap).
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
			// Force markdown output and generous defaults for reports.
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
	// Stub for future: `triage report cluster`
	r.AddCommand(&cobra.Command{
		Use:   "cluster",
		Short: "Write a full markdown report for the cluster",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("report cluster: not yet implemented (tracking: v0.2)")
		},
	})
	return r
}

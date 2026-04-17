package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/OWNER/triage/internal/engine"
	"github.com/OWNER/triage/internal/findings"
	"github.com/OWNER/triage/internal/kube"
	"github.com/OWNER/triage/internal/output"
)

// RunDiagnosis is the shared orchestration used by every diagnosis command
// (pod, deployment, namespace, cluster). It builds the kube client, calls
// the engine, renders the report, and returns the appropriate exit code.
func RunDiagnosis(cmd *cobra.Command, target findings.Target, w io.Writer) int {
	opts := Get(cmd)
	ctx, cancel := context.WithTimeout(cmd.Context(), opts.Timeout)
	defer cancel()

	client, err := kube.NewClient(opts.KubeFlags)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "error: could not build Kubernetes client: %v\n", err)
		return ExitClusterError
	}

	if target.Namespace == "" && needsNamespace(target.Kind) {
		target.Namespace = client.CurrentNamespace()
	}

	sevMin, _ := findings.ParseSeverity(opts.SeverityMin)
	confMin, _ := findings.ParseConfidence(opts.ConfMin)

	report, err := engine.Run(ctx, client, target, engine.Options{
		MaxFindings:    opts.MaxFindings,
		SeverityMin:    sevMin,
		ConfidenceMin:  confMin,
		IncludeEvents:  opts.IncludeEvents,
		IncludeRelated: opts.IncludeRelated,
		DisabledRules:  opts.Config.Rules.Disabled,
		EnabledRules:   opts.Config.Rules.Enabled,
		Logger:         opts.Logger,
	})
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
		return ExitClusterError
	}

	format, err := output.ParseFormat(opts.Output)
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		return ExitUsage
	}

	if err := output.Render(w, report, format, output.RenderOptions{
		Color:       opts.Color,
		Verbose:     opts.Verbose,
		MaxFindings: opts.MaxFindings,
	}); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "error: rendering output: %v\n", err)
		return ExitInternal
	}

	if len(report.Findings) > 0 {
		return ExitFindings
	}
	return ExitOK
}

func needsNamespace(k findings.TargetKind) bool {
	switch k {
	case findings.TargetKindPod, findings.TargetKindDeployment:
		return true
	}
	return false
}

// Ctx returns ctx from cmd or the background context.
func Ctx(cmd *cobra.Command) context.Context {
	if c := cmd.Context(); c != nil {
		return c
	}
	return context.Background()
}

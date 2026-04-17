package rules

import (
	"context"
	"fmt"

	"github.com/OWNER/triage/internal/findings"
)

func init() { Register(&podInitFailed{}) }

type podInitFailed struct{}

func (r *podInitFailed) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-INIT-FAILED",
		Title:    "Pod init container failed and is not in CrashLoopBackOff",
		Category: findings.CategoryRuntime,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `An init container exited with a non-zero exit code, preventing the main
containers from starting. Unlike CrashLoopBackOff (repeated restarts),
this catches the case where the init container has failed but has not
been retried yet (or is blocked for other reasons).

Common causes:
- Init container command references a script/binary that doesn't exist in the image.
- Init container depends on a service (database, API) that is not yet ready.
- Network policy blocks the init container from reaching an external service.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/workloads/pods/init-containers/",
		},
		Priority: 84,
	}
}

func (r *podInitFailed) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil || !found {
		return nil, err
	}

	var out []findings.Finding
	ns, name := rc.Target.Namespace, rc.Target.Name

	for _, cs := range pod.Status.InitContainerStatuses {
		// Skip if it is in CrashLoopBackOff — that rule handles it.
		if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
			continue
		}
		// Detect terminated with non-zero exit.
		if cs.State.Terminated == nil {
			continue
		}
		t := cs.State.Terminated
		if t.ExitCode == 0 {
			continue
		}

		out = append(out, findings.Finding{
			ID:         "TRG-POD-INIT-FAILED",
			RuleID:     "TRG-POD-INIT-FAILED",
			Title:      fmt.Sprintf("Init container %q failed with exit code %d", cs.Name, t.ExitCode),
			Summary:    fmt.Sprintf("Init container %q exited with code %d (reason: %s). The main container(s) have not started.", cs.Name, t.ExitCode, t.Reason),
			Category:   findings.CategoryRuntime,
			Severity:   findings.SeverityHigh,
			Confidence: findings.ConfidenceHigh,
			Target:     rc.Target,
			Evidence: []findings.Evidence{
				{Kind: findings.EvidenceKindField, Source: fmt.Sprintf("pod.status.initContainerStatuses[%s].state.terminated.exitCode", cs.Name), Value: fmt.Sprintf("%d", t.ExitCode)},
				{Kind: findings.EvidenceKindField, Source: fmt.Sprintf("pod.status.initContainerStatuses[%s].state.terminated.reason", cs.Name), Value: t.Reason},
			},
			Remediation: findings.Remediation{
				Explanation: "Check the init container logs to find the cause of the non-zero exit.",
				NextCommands: []string{
					fmt.Sprintf("kubectl logs -n %s %s -c %s", ns, name, cs.Name),
					fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
				},
				SuggestedFix: "Fix the command, environment, or service dependency that causes the init container to exit non-zero.",
				DocsLinks:    []string{"https://kubernetes.io/docs/concepts/workloads/pods/init-containers/"},
			},
		})
	}
	return out, nil
}

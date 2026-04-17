package rules

import (
	"context"
	"fmt"

	"github.com/khvedela/triage/internal/findings"
)

func init() { Register(&podOOMKilled{}) }

type podOOMKilled struct{}

func (r *podOOMKilled) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-OOMKILLED",
		Title:    "Container was OOMKilled",
		Category: findings.CategoryResourcePressure,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `The container's process was killed by the kernel's out-of-memory (OOM) killer
because the container exceeded its memory limit. This shows up in
lastState.terminated.reason = "OOMKilled".

Common causes:
- The memory limit is set too low for the application's actual working set.
- A memory leak in the application.
- A burst-workload that requires more memory than the limit.

Remediation involves either increasing the memory limit or reducing the application's
memory footprint.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",
		},
		Priority: 90,
	}
}

func (r *podOOMKilled) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	var out []findings.Finding
	allStatuses := append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...)
	for _, cs := range allStatuses {
		t := cs.LastTerminationState.Terminated
		if t == nil || t.Reason != "OOMKilled" {
			continue
		}

		var limit string
		for _, c := range pod.Spec.Containers {
			if c.Name == cs.Name {
				if mem := c.Resources.Limits.Memory(); mem != nil {
					limit = mem.String()
				}
				break
			}
		}
		for _, c := range pod.Spec.InitContainers {
			if c.Name == cs.Name {
				if mem := c.Resources.Limits.Memory(); mem != nil {
					limit = mem.String()
				}
				break
			}
		}

		ev := []findings.Evidence{
			{
				Kind:   findings.EvidenceKindField,
				Source: fmt.Sprintf("pod.status.containerStatuses[%s].lastState.terminated.reason", cs.Name),
				Value:  "OOMKilled",
			},
			{
				Kind:   findings.EvidenceKindField,
				Source: fmt.Sprintf("pod.status.containerStatuses[%s].restartCount", cs.Name),
				Value:  fmt.Sprintf("%d", cs.RestartCount),
			},
		}
		if limit != "" {
			ev = append(ev, findings.Evidence{
				Kind:   findings.EvidenceKindField,
				Source: fmt.Sprintf("pod.spec.containers[%s].resources.limits.memory", cs.Name),
				Value:  limit,
			})
		}

		ns, name := rc.Target.Namespace, rc.Target.Name
		summary := fmt.Sprintf("Container %q was killed because it exceeded its memory limit", cs.Name)
		if limit != "" {
			summary += fmt.Sprintf(" (%s)", limit)
		}
		summary += ". The kernel OOM killer sent SIGKILL."

		out = append(out, findings.Finding{
			ID:         "TRG-POD-OOMKILLED",
			RuleID:     "TRG-POD-OOMKILLED",
			Title:      fmt.Sprintf("Container %q was OOMKilled", cs.Name),
			Summary:    summary,
			Category:   findings.CategoryResourcePressure,
			Severity:   findings.SeverityHigh,
			Confidence: findings.ConfidenceHigh,
			Target:     rc.Target,
			Evidence:   ev,
			Remediation: findings.Remediation{
				Explanation: "The container exceeded its memory limit. Either increase the limit or fix the memory leak.",
				NextCommands: []string{
					fmt.Sprintf("kubectl top pod -n %s %s --containers", ns, name),
					fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
				},
				SuggestedFix: "Increase resources.limits.memory in the container spec. " +
					"If the application has a memory leak, profile it with your language's memory profiler.",
				DocsLinks: []string{
					"https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",
				},
			},
		})
	}
	return out, nil
}

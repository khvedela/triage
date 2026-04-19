package rules

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/khvedela/kubediag/internal/findings"
)

func init() { Register(&podCrashLoopBackOff{}) }

type podCrashLoopBackOff struct{}

func (r *podCrashLoopBackOff) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-CRASHLOOPBACKOFF",
		Title:    "Container is in CrashLoopBackOff",
		Category: findings.CategoryRuntime,
		Severity: findings.SeverityCritical,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `The container has crashed repeatedly and Kubernetes is backing off before
restarting it again. The root cause is almost always in the container process itself:
the command exits non-zero on every start.

Common causes:
- The application panics or exits immediately due to a missing configuration file or env var.
- The entrypoint binary does not exist in the image.
- An OOMKilled container gets re-labelled as CrashLoopBackOff after several OOM events.
- A health check fails so fast that the container never stabilises.

Check the *previous* container logs for the actual exit reason.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-states",
		},
		Priority: 100,
	}
}

func (r *podCrashLoopBackOff) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	var out []findings.Finding
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting == nil || cs.State.Waiting.Reason != "CrashLoopBackOff" {
			continue
		}
		ev := crashEvidence(pod, cs)
		ns, name := rc.Target.Namespace, rc.Target.Name
		out = append(out, findings.Finding{
			ID:         "TRG-POD-CRASHLOOPBACKOFF",
			RuleID:     "TRG-POD-CRASHLOOPBACKOFF",
			Title:      fmt.Sprintf("Container %q is in CrashLoopBackOff (%d restarts)", cs.Name, cs.RestartCount),
			Summary:    crashSummary(cs),
			Category:   findings.CategoryRuntime,
			Severity:   findings.SeverityCritical,
			Confidence: findings.ConfidenceHigh,
			Target:     rc.Target,
			Evidence:   ev,
			Remediation: findings.Remediation{
				Explanation: "Inspect the previous container logs for the actual exit cause, then fix the underlying application or configuration issue.",
				NextCommands: []string{
					fmt.Sprintf("kubectl logs -n %s %s -c %s --previous", ns, name, cs.Name),
					fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
				},
				SuggestedFix: "Check the logs above for panic messages, missing files, or bad config. " +
					"Fix the application exit path or the referenced ConfigMap/Secret.",
				DocsLinks: []string{
					"https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-states",
				},
			},
		})
	}

	// Init containers
	for _, cs := range pod.Status.InitContainerStatuses {
		if cs.State.Waiting == nil || cs.State.Waiting.Reason != "CrashLoopBackOff" {
			continue
		}
		ns, name := rc.Target.Namespace, rc.Target.Name
		out = append(out, findings.Finding{
			ID:         "TRG-POD-INIT-CRASHLOOP",
			RuleID:     "TRG-POD-CRASHLOOPBACKOFF",
			Title:      fmt.Sprintf("Init container %q is in CrashLoopBackOff", cs.Name),
			Summary:    fmt.Sprintf("Init container %q keeps crashing. Pod will not start until all init containers succeed.", cs.Name),
			Category:   findings.CategoryRuntime,
			Severity:   findings.SeverityCritical,
			Confidence: findings.ConfidenceHigh,
			Target:     rc.Target,
			Evidence:   crashEvidence(pod, cs),
			Remediation: findings.Remediation{
				Explanation: "Init containers must exit 0. If this one keeps failing the pod never reaches Running.",
				NextCommands: []string{
					fmt.Sprintf("kubectl logs -n %s %s -c %s --previous", ns, name, cs.Name),
					fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
				},
			},
		})
	}
	return out, nil
}

func crashEvidence(pod *corev1.Pod, cs corev1.ContainerStatus) []findings.Evidence {
	ev := []findings.Evidence{
		{
			Kind:   findings.EvidenceKindField,
			Source: fmt.Sprintf("pod.status.containerStatuses[%s].state.waiting.reason", cs.Name),
			Value:  "CrashLoopBackOff",
		},
		{
			Kind:   findings.EvidenceKindComputed,
			Source: "restartCount",
			Value:  fmt.Sprintf("%d restarts", cs.RestartCount),
		},
	}
	if cs.LastTerminationState.Terminated != nil {
		t := cs.LastTerminationState.Terminated
		ev = append(ev, findings.Evidence{
			Kind:   findings.EvidenceKindField,
			Source: fmt.Sprintf("pod.status.containerStatuses[%s].lastState.terminated", cs.Name),
			Value:  fmt.Sprintf("reason=%s exitCode=%d", t.Reason, t.ExitCode),
		})
	}
	// Include relevant Warning events from the pod.
	for _, e := range pod.Status.Conditions {
		_ = e // conditions are informational; events come via cache in full engine runs
	}
	return ev
}

func crashSummary(cs corev1.ContainerStatus) string {
	if cs.LastTerminationState.Terminated != nil {
		t := cs.LastTerminationState.Terminated
		return fmt.Sprintf("Container %q last exited with code %d (reason: %s). "+
			"Kubernetes is backing off restarts — check previous logs for the root cause.",
			cs.Name, t.ExitCode, t.Reason)
	}
	return fmt.Sprintf("Container %q has restarted %d times and is now in CrashLoopBackOff. "+
		"Inspect previous logs with kubectl logs --previous.",
		cs.Name, cs.RestartCount)
}

// containerStatus helper used by other rules.
func podWaitingReason(pod *corev1.Pod, containerName, reason string) bool {
	for _, cs := range pod.Status.ContainerStatuses {
		if containerName != "" && cs.Name != containerName {
			continue
		}
		if cs.State.Waiting != nil && strings.HasPrefix(cs.State.Waiting.Reason, reason) {
			return true
		}
	}
	return false
}

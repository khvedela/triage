package rules

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"

	"github.com/khvedela/triage/internal/findings"
)

func init() {
	Register(&podReadinessProbe{})
	Register(&podLivenessProbe{})
	Register(&podStartupProbe{})
}

// ----- TRG-POD-READINESS-FAILING -----------------------------------------

type podReadinessProbe struct{}

func (r *podReadinessProbe) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-READINESS-FAILING",
		Title:    "Container readiness probe is failing",
		Category: findings.CategoryProbes,
		Severity: findings.SeverityMedium,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `The container's readiness probe is not returning success. Kubernetes removes
the pod from Service Endpoints while readiness fails, so traffic is not routed
to it. The pod stays Running but receives no traffic.

This is distinct from a liveness failure: failing readiness does not restart the
container; it only removes it from load balancing.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/",
		},
		Priority: 70,
	}
}

func (r *podReadinessProbe) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil || !found {
		return nil, err
	}

	events, _ := rc.Cache.ListEventsFor(ctx, "Pod", rc.Target.Namespace, rc.Target.Name)
	var out []findings.Finding

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			continue
		}
		// Only flag if the container is running (not crashing) — CrashLoop rule handles the latter.
		if cs.State.Running == nil {
			continue
		}
		// Check if the container has a readiness probe configured.
		probe := containerReadinessProbe(pod, cs.Name)
		if probe == nil {
			continue
		}
		// Check events for Unhealthy/readiness messages.
		probeMsg := probeEventMsg(events, cs.Name, "Readiness")

		ns, name := rc.Target.Namespace, rc.Target.Name
		ev := []findings.Evidence{
			{Kind: findings.EvidenceKindField, Source: fmt.Sprintf("pod.status.containerStatuses[%s].ready", cs.Name), Value: "false"},
			{Kind: findings.EvidenceKindField, Source: fmt.Sprintf("pod.status.containerStatuses[%s].state.running", cs.Name), Value: "true"},
			{Kind: findings.EvidenceKindComputed, Value: "Readiness probe: " + probeDesc(probe)},
		}
		if probeMsg != "" {
			ev = append(ev, findings.Evidence{Kind: findings.EvidenceKindEvent, Value: probeMsg})
		}

		out = append(out, findings.Finding{
			ID:         "TRG-POD-READINESS-FAILING",
			RuleID:     "TRG-POD-READINESS-FAILING",
			Title:      fmt.Sprintf("Container %q readiness probe failing — pod excluded from Service endpoints", cs.Name),
			Summary:    fmt.Sprintf("Container %q is Running but not Ready. Its readiness probe (%s) is returning failure. The pod is excluded from all Service Endpoints.", cs.Name, probeDesc(probe)),
			Category:   findings.CategoryProbes,
			Severity:   findings.SeverityMedium,
			Confidence: findings.ConfidenceMedium,
			Target:     rc.Target,
			Evidence:   ev,
			Remediation: findings.Remediation{
				Explanation: "The readiness probe is rejecting the container. The application may still be starting, or there may be an application-level error.",
				NextCommands: []string{
					fmt.Sprintf("kubectl logs -n %s %s -c %s", ns, name, cs.Name),
					fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
					fmt.Sprintf("kubectl get endpoints -n %s", ns),
				},
				SuggestedFix: "Check the application logs. If the app needs more startup time, " +
					"increase initialDelaySeconds or use a startupProbe. " +
					"Ensure the probe path/port is correct and the application responds there.",
				DocsLinks: []string{
					"https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/",
				},
			},
		})
	}
	return out, nil
}

// ----- TRG-POD-LIVENESS-FAILING ------------------------------------------

type podLivenessProbe struct{}

func (r *podLivenessProbe) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-LIVENESS-FAILING",
		Title:    "Container liveness probe is failing (causing container restarts)",
		Category: findings.CategoryProbes,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `The liveness probe is returning failure. When the probe fails enough times
(failureThreshold), Kubernetes kills and restarts the container. Repeated
liveness failures appear as increasing restart counts, sometimes leading to
CrashLoopBackOff.

A misconfigured liveness probe (wrong path, too-short timeout) is a very
common cause of unexpected pod restarts.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/",
		},
		Priority: 74,
	}
}

func (r *podLivenessProbe) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil || !found {
		return nil, err
	}

	events, _ := rc.Cache.ListEventsFor(ctx, "Pod", rc.Target.Namespace, rc.Target.Name)
	var out []findings.Finding

	for _, cs := range pod.Status.ContainerStatuses {
		// Liveness failures show up as high restart counts or in Unhealthy events.
		if cs.RestartCount == 0 {
			continue
		}
		probe := containerLivenessProbe(pod, cs.Name)
		if probe == nil {
			continue
		}
		probeMsg := probeEventMsg(events, cs.Name, "Liveness")
		if probeMsg == "" && cs.RestartCount < 2 {
			continue // Not enough signal to confidently blame liveness
		}

		ns, name := rc.Target.Namespace, rc.Target.Name
		ev := []findings.Evidence{
			{Kind: findings.EvidenceKindField, Source: fmt.Sprintf("pod.status.containerStatuses[%s].restartCount", cs.Name), Value: fmt.Sprintf("%d", cs.RestartCount)},
			{Kind: findings.EvidenceKindComputed, Value: "Liveness probe: " + probeDesc(probe)},
		}
		if probeMsg != "" {
			ev = append(ev, findings.Evidence{Kind: findings.EvidenceKindEvent, Value: probeMsg})
		}

		out = append(out, findings.Finding{
			ID:         "TRG-POD-LIVENESS-FAILING",
			RuleID:     "TRG-POD-LIVENESS-FAILING",
			Title:      fmt.Sprintf("Container %q liveness probe failing (%d restarts)", cs.Name, cs.RestartCount),
			Summary:    fmt.Sprintf("Container %q has restarted %d times. A failing liveness probe (%s) may be causing Kubernetes to kill it.", cs.Name, cs.RestartCount, probeDesc(probe)),
			Category:   findings.CategoryProbes,
			Severity:   findings.SeverityHigh,
			Confidence: findings.ConfidenceMedium,
			Target:     rc.Target,
			Evidence:   ev,
			Remediation: findings.Remediation{
				Explanation: "Liveness probe failures cause the kubelet to restart the container. Check whether the probe path/port/command is correct and the app responds within the timeout.",
				NextCommands: []string{
					fmt.Sprintf("kubectl logs -n %s %s -c %s --previous", ns, name, cs.Name),
					fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
				},
				SuggestedFix: "Verify the liveness probe is targeting the right path and port. " +
					"Increase timeoutSeconds or failureThreshold if the app is slow under load.",
			},
		})
	}
	return out, nil
}

// ----- TRG-POD-STARTUP-FAILING -------------------------------------------

type podStartupProbe struct{}

func (r *podStartupProbe) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-STARTUP-FAILING",
		Title:    "Container startup probe is failing",
		Category: findings.CategoryProbes,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `The startup probe is not passing within its timeout window. Until the startup
probe succeeds, both readiness and liveness probes are disabled. If the probe
never passes (failureThreshold × periodSeconds elapsed), the container is killed.

This is most commonly misconfigured when the application has variable cold-start
time that exceeds failureThreshold × periodSeconds.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-startup-probes",
		},
		Priority: 76,
	}
}

func (r *podStartupProbe) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil || !found {
		return nil, err
	}

	events, _ := rc.Cache.ListEventsFor(ctx, "Pod", rc.Target.Namespace, rc.Target.Name)
	var out []findings.Finding

	for _, cs := range pod.Status.ContainerStatuses {
		probe := containerStartupProbe(pod, cs.Name)
		if probe == nil {
			continue
		}
		probeMsg := probeEventMsg(events, cs.Name, "Startup")
		if probeMsg == "" {
			continue
		}

		ns, name := rc.Target.Namespace, rc.Target.Name
		out = append(out, findings.Finding{
			ID:         "TRG-POD-STARTUP-FAILING",
			RuleID:     "TRG-POD-STARTUP-FAILING",
			Title:      fmt.Sprintf("Container %q startup probe failing", cs.Name),
			Summary:    fmt.Sprintf("Container %q has a startup probe (%s) that is not passing.", cs.Name, probeDesc(probe)),
			Category:   findings.CategoryProbes,
			Severity:   findings.SeverityHigh,
			Confidence: findings.ConfidenceMedium,
			Target:     rc.Target,
			Evidence: []findings.Evidence{
				{Kind: findings.EvidenceKindEvent, Value: probeMsg},
				{Kind: findings.EvidenceKindComputed, Value: "Startup probe: " + probeDesc(probe)},
			},
			Remediation: findings.Remediation{
				Explanation: "The startup probe has not passed within its deadline. Check application startup time and probe configuration.",
				NextCommands: []string{
					fmt.Sprintf("kubectl logs -n %s %s -c %s", ns, name, cs.Name),
					fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
				},
				SuggestedFix: "Increase startupProbe.failureThreshold to give the app more time to start. " +
					"A common formula: maxStartupSeconds = failureThreshold × periodSeconds.",
			},
		})
	}
	return out, nil
}

// ----- probe helpers -------------------------------------------------------

func containerReadinessProbe(pod *corev1.Pod, name string) *corev1.Probe {
	for _, c := range pod.Spec.Containers {
		if c.Name == name {
			return c.ReadinessProbe
		}
	}
	return nil
}

func containerLivenessProbe(pod *corev1.Pod, name string) *corev1.Probe {
	for _, c := range pod.Spec.Containers {
		if c.Name == name {
			return c.LivenessProbe
		}
	}
	return nil
}

func containerStartupProbe(pod *corev1.Pod, name string) *corev1.Probe {
	for _, c := range pod.Spec.Containers {
		if c.Name == name {
			return c.StartupProbe
		}
	}
	return nil
}

func probeDesc(p *corev1.Probe) string {
	if p == nil {
		return "none"
	}
	switch {
	case p.HTTPGet != nil:
		return fmt.Sprintf("HTTP GET :%d%s", p.HTTPGet.Port.IntValue(), p.HTTPGet.Path)
	case p.TCPSocket != nil:
		return fmt.Sprintf("TCP :%d", p.TCPSocket.Port.IntValue())
	case p.Exec != nil:
		return fmt.Sprintf("exec [%s]", strings.Join(p.Exec.Command, " "))
	case p.GRPC != nil:
		return fmt.Sprintf("gRPC :%d", p.GRPC.Port)
	}
	return "unknown"
}

func probeEventMsg(events []eventsv1.Event, containerName, probeType string) string {
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		if e.Reason != "Unhealthy" {
			continue
		}
		note := strings.ToLower(e.Note)
		if strings.Contains(note, strings.ToLower(probeType)+" probe") {
			return e.Note
		}
	}
	return ""
}

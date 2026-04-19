package rules

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/khvedela/triage/internal/findings"
)

func init() { Register(&svcPortMismatch{}) }

// TRG-SVC-PORT-MISMATCH fires when a Service's targetPort does not match any
// containerPort declared by the pods selected by that Service.
type svcPortMismatch struct{}

func (r *svcPortMismatch) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-SVC-PORT-MISMATCH",
		Title:    "Service targetPort is not exposed by any selected pod",
		Category: findings.CategoryNetworking,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod, findings.TargetKindNamespace},
		Description: `A Service's targetPort (the port traffic is forwarded to inside the pod) does
not match any containerPort declared by the pods the Service selects.

Note: Kubernetes does not require containerPorts to be declared for traffic to
flow — kube-proxy uses iptables/ipvs rules regardless. However, when
containerPorts ARE declared and none match the targetPort, this almost always
indicates a misconfiguration: a typo, a renamed port, or a changed application
port that was not reflected in the Service spec.

This rule only fires when:
1. The Service has a non-empty selector (not a headless/external service).
2. At least one pod matches the selector.
3. Those pods declare containerPorts.
4. None of the declared containerPorts match the Service's targetPort.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service",
		},
		Priority: 84,
	}
}

func (r *svcPortMismatch) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	ns := rc.Target.Namespace
	if ns == "" {
		return nil, nil
	}

	svcs, err := rc.Cache.ListServices(ctx, ns)
	if err != nil {
		return nil, nil
	}
	pods, err := rc.Cache.ListPods(ctx, ns)
	if err != nil {
		return nil, nil
	}

	var out []findings.Finding
	for _, svc := range svcs {
		if svc.Spec.Type == corev1.ServiceTypeExternalName {
			continue
		}
		if len(svc.Spec.Selector) == 0 {
			continue
		}

		selected := podsMatchingSelector(svc.Spec.Selector, pods)
		if len(selected) == 0 {
			continue // TRG-SVC-SELECTOR-MISMATCH covers this
		}

		// Collect all containerPort numbers/names from selected pods.
		exposedPorts := collectExposedPorts(selected)
		if len(exposedPorts) == 0 {
			continue // pods don't declare ports — we can't make a judgement
		}

		for _, sp := range svc.Spec.Ports {
			tp := sp.TargetPort
			// Default: if TargetPort is zero-value, it equals the service Port.
			if tp.Type == intstr.Int && tp.IntVal == 0 {
				tp = intstr.FromInt(int(sp.Port))
			}

			if targetPortExposed(tp, exposedPorts) {
				continue
			}

			targetPortStr := targetPortString(tp)
			out = append(out, findings.Finding{
				ID:         "TRG-SVC-PORT-MISMATCH",
				RuleID:     "TRG-SVC-PORT-MISMATCH",
				Title:      fmt.Sprintf("Service %q targetPort %s not exposed by selected pods", svc.Name, targetPortStr),
				Summary: fmt.Sprintf(
					"Service %q/%q routes to targetPort %s, but none of the %d selected pod(s) declare a containerPort matching it. "+
						"Traffic will be forwarded to the port regardless, but this mismatch strongly suggests a misconfiguration.",
					ns, svc.Name, targetPortStr, len(selected),
				),
				Category:   findings.CategoryNetworking,
				Severity:   findings.SeverityHigh,
				Confidence: findings.ConfidenceMedium,
				Target:     rc.Target,
				Related: []findings.ResourceRef{
					{APIVersion: "v1", Kind: "Service", Namespace: ns, Name: svc.Name},
				},
				Evidence: []findings.Evidence{
					{
						Kind:   findings.EvidenceKindField,
						Source: fmt.Sprintf("service/%s.spec.ports[%s].targetPort", svc.Name, sp.Name),
						Value:  targetPortStr,
					},
					{
						Kind:  findings.EvidenceKindComputed,
						Value: fmt.Sprintf("Selected pods expose containerPorts: %s", joinPortSet(exposedPorts)),
					},
				},
				Remediation: findings.Remediation{
					Explanation: fmt.Sprintf(
						"The Service targetPort %s does not appear in the containerPorts of any selected pod. "+
							"Either update the Service targetPort to match the actual container port, or add/fix the containerPort in the pod spec.",
						targetPortStr,
					),
					NextCommands: []string{
						fmt.Sprintf("kubectl describe service -n %s %s", ns, svc.Name),
						fmt.Sprintf("kubectl get pods -n %s -l %s -o jsonpath='{.items[*].spec.containers[*].ports}'", ns, formatSelector(svc.Spec.Selector)),
					},
					SuggestedFix: fmt.Sprintf(
						"Check which port the application actually listens on, then update the Service:\n"+
							"  spec.ports[].targetPort: <actual-container-port>\n"+
							"Current targetPort: %s. Pods expose: %s.",
						targetPortStr, joinPortSet(exposedPorts),
					),
					DocsLinks: []string{
						"https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service",
					},
				},
			})
		}
	}
	return out, nil
}

// exposedPort represents a containerPort declared in a pod spec.
type exposedPort struct {
	number int32
	name   string
}

func collectExposedPorts(pods []corev1.Pod) []exposedPort {
	seen := map[string]bool{}
	var out []exposedPort
	for _, pod := range pods {
		for _, c := range pod.Spec.Containers {
			for _, p := range c.Ports {
				key := fmt.Sprintf("%d/%s", p.ContainerPort, p.Name)
				if !seen[key] {
					seen[key] = true
					out = append(out, exposedPort{number: p.ContainerPort, name: p.Name})
				}
			}
		}
	}
	return out
}

func targetPortExposed(tp intstr.IntOrString, exposed []exposedPort) bool {
	for _, ep := range exposed {
		switch tp.Type {
		case intstr.Int:
			if ep.number == tp.IntVal {
				return true
			}
		case intstr.String:
			if ep.name == tp.StrVal {
				return true
			}
			// Also allow numeric string match (e.g. targetPort: "8080")
			if n, err := strconv.Atoi(tp.StrVal); err == nil && ep.number == int32(n) {
				return true
			}
		}
	}
	return false
}

func targetPortString(tp intstr.IntOrString) string {
	if tp.Type == intstr.Int {
		return fmt.Sprintf("%d", tp.IntVal)
	}
	return tp.StrVal
}

func joinPortSet(ports []exposedPort) string {
	seen := map[string]bool{}
	var parts []string
	for _, p := range ports {
		s := fmt.Sprintf("%d", p.number)
		if p.name != "" {
			s = fmt.Sprintf("%d(%s)", p.number, p.name)
		}
		if !seen[s] {
			seen[s] = true
			parts = append(parts, s)
		}
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}

func podsMatchingSelector(selector map[string]string, pods []corev1.Pod) []corev1.Pod {
	var out []corev1.Pod
	for _, pod := range pods {
		if labelsMatch(selector, pod.Labels) {
			out = append(out, pod)
		}
	}
	return out
}

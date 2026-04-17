package rules

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/OWNER/triage/internal/findings"
)

func init() {
	Register(&svcNoEndpoints{})
	Register(&svcSelectorMismatch{})
}

// ----- TRG-SVC-NO-ENDPOINTS -----------------------------------------------

type svcNoEndpoints struct{}

func (r *svcNoEndpoints) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-SVC-NO-ENDPOINTS",
		Title:    "Service has no endpoints (no pods are selected)",
		Category: findings.CategoryNetworking,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod, findings.TargetKindNamespace},
		Description: `The Service's selector does not match any Running+Ready pod, so its Endpoints
object is empty. Traffic to this Service will be dropped or return connection
refused.

Common causes:
- Label mismatch between the Service selector and the pod labels.
- All pods are failing their readiness probes.
- No pods exist with the selected labels.
- The pods are in a different namespace.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service",
		},
		Priority: 82,
	}
}

func (r *svcNoEndpoints) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	ns := rc.Target.Namespace
	if ns == "" {
		return nil, nil // cluster-scope: skip
	}

	svcs, err := rc.Cache.ListServices(ctx, ns)
	if err != nil {
		return nil, nil
	}

	var out []findings.Finding
	for _, svc := range svcs {
		if svc.Spec.Type == corev1.ServiceTypeExternalName {
			continue
		}
		if len(svc.Spec.Selector) == 0 {
			continue // headless / external — intentional
		}

		ep, epFound, epErr := rc.Cache.GetEndpoints(ctx, ns, svc.Name)
		if epErr != nil || !epFound {
			continue
		}
		if endpointsHaveAddresses(ep) {
			continue
		}

		selectorStr := formatSelector(svc.Spec.Selector)
		out = append(out, findings.Finding{
			ID:         "TRG-SVC-NO-ENDPOINTS",
			RuleID:     "TRG-SVC-NO-ENDPOINTS",
			Title:      fmt.Sprintf("Service %q/%q has no endpoints", ns, svc.Name),
			Summary:    fmt.Sprintf("Service %q has selector %s but no Ready pods match it. All traffic to this Service is dropped.", svc.Name, selectorStr),
			Category:   findings.CategoryNetworking,
			Severity:   findings.SeverityHigh,
			Confidence: findings.ConfidenceHigh,
			Target:     rc.Target,
			Related: []findings.ResourceRef{
				{APIVersion: "v1", Kind: "Service", Namespace: ns, Name: svc.Name},
				{APIVersion: "v1", Kind: "Endpoints", Namespace: ns, Name: svc.Name},
			},
			Evidence: []findings.Evidence{
				{Kind: findings.EvidenceKindField, Source: fmt.Sprintf("service/%s.spec.selector", svc.Name), Value: selectorStr},
				{Kind: findings.EvidenceKindComputed, Value: fmt.Sprintf("kubectl get endpoints -n %s %s → no addresses", ns, svc.Name)},
			},
			Remediation: findings.Remediation{
				Explanation: "The Service selector does not match any Ready pod. Fix the labels on the pods or the selector on the Service.",
				NextCommands: []string{
					fmt.Sprintf("kubectl get endpoints -n %s %s", ns, svc.Name),
					fmt.Sprintf("kubectl get pods -n %s -l %s", ns, selectorStr),
					fmt.Sprintf("kubectl describe service -n %s %s", ns, svc.Name),
				},
				SuggestedFix: "Compare Service.spec.selector with pod.metadata.labels. " +
					"Fix the mismatch. If pods exist but are not Ready, resolve their readiness probe failures first.",
				DocsLinks: []string{
					"https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service",
				},
			},
		})
	}
	return out, nil
}

// ----- TRG-SVC-SELECTOR-MISMATCH -----------------------------------------

type svcSelectorMismatch struct{}

func (r *svcSelectorMismatch) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-SVC-SELECTOR-MISMATCH",
		Title:    "Service selector does not match any pod labels in the namespace",
		Category: findings.CategoryNetworking,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod, findings.TargetKindNamespace},
		Description: `The Service's selector labels do not appear on any pod in the namespace,
regardless of pod health or readiness. This is often a misconfiguration —
either the pod labels were changed or the Service selector was mistyped.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/services-networking/service/",
		},
		Priority: 83,
	}
}

func (r *svcSelectorMismatch) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
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
		if len(svc.Spec.Selector) == 0 {
			continue
		}
		if matchesAnyPod(svc.Spec.Selector, pods) {
			continue
		}
		// Check endpoints: if they exist and are non-empty, skip (some other mechanism is working)
		ep, epFound, _ := rc.Cache.GetEndpoints(ctx, ns, svc.Name)
		if epFound && endpointsHaveAddresses(ep) {
			continue
		}

		selectorStr := formatSelector(svc.Spec.Selector)
		out = append(out, findings.Finding{
			ID:         "TRG-SVC-SELECTOR-MISMATCH",
			RuleID:     "TRG-SVC-SELECTOR-MISMATCH",
			Title:      fmt.Sprintf("Service %q selector %s matches zero pods in namespace %q", svc.Name, selectorStr, ns),
			Summary:    fmt.Sprintf("Service %q has selector %s but no pod in namespace %q carries all those labels.", svc.Name, selectorStr, ns),
			Category:   findings.CategoryNetworking,
			Severity:   findings.SeverityHigh,
			Confidence: findings.ConfidenceMedium,
			Target:     rc.Target,
			Related: []findings.ResourceRef{
				{APIVersion: "v1", Kind: "Service", Namespace: ns, Name: svc.Name},
			},
			Evidence: []findings.Evidence{
				{Kind: findings.EvidenceKindField, Source: fmt.Sprintf("service/%s.spec.selector", svc.Name), Value: selectorStr},
				{Kind: findings.EvidenceKindComputed, Value: fmt.Sprintf("kubectl get pods -n %s -l %s → 0 results", ns, selectorStr)},
			},
			Remediation: findings.Remediation{
				Explanation: "The Service's selector labels don't match any existing pod, not even unhealthy ones.",
				NextCommands: []string{
					fmt.Sprintf("kubectl get pods -n %s --show-labels", ns),
					fmt.Sprintf("kubectl describe service -n %s %s", ns, svc.Name),
				},
				SuggestedFix: "Fix the typo in either the Service selector or the pod labels. " +
					"Run `kubectl get pods --show-labels` to see what labels are actually present.",
			},
		})
	}
	return out, nil
}

// ----- helpers ------------------------------------------------------------

func endpointsHaveAddresses(ep *corev1.Endpoints) bool {
	if ep == nil {
		return false
	}
	for _, s := range ep.Subsets {
		if len(s.Addresses) > 0 {
			return true
		}
	}
	return false
}

func matchesAnyPod(selector map[string]string, pods []corev1.Pod) bool {
	for _, pod := range pods {
		if labelsMatch(selector, pod.Labels) {
			return true
		}
	}
	return false
}

func labelsMatch(selector, podLabels map[string]string) bool {
	for k, v := range selector {
		if podLabels[k] != v {
			return false
		}
	}
	return true
}

func formatSelector(sel map[string]string) string {
	parts := make([]string, 0, len(sel))
	for k, v := range sel {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ",")
}

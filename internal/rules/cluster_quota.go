package rules

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/khvedela/triage/internal/findings"
)

func init() { Register(&clusterQuotaExhausted{}) }

// TRG-CLUSTER-QUOTA-EXHAUSTED fires when a namespace ResourceQuota has one or
// more resources at >=95% utilisation (hard limit nearly reached) or at 100%
// (hard limit hit — new pods/services will be rejected).
type clusterQuotaExhausted struct{}

func (r *clusterQuotaExhausted) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-CLUSTER-QUOTA-EXHAUSTED",
		Title:    "Namespace ResourceQuota is exhausted or nearly exhausted",
		Category: findings.CategoryResourcePressure,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindNamespace, findings.TargetKindCluster},
		Description: `A ResourceQuota in the namespace has one or more tracked resources at or near
its hard limit. When a quota is fully exhausted, the API server will reject new
object creation (pods, services, PVCs, etc.) with a "exceeded quota" error.

Thresholds used:
- ≥100% used → Critical: quota fully consumed, new workloads will be rejected.
- ≥95% used  → High: quota nearly consumed, plan to increase or clean up.

Common causes:
- Too many replicas scaled up without increasing quota.
- Stale completed/failed pods not cleaned up (they still count toward pod quota).
- Namespace was under-quota'd at creation and the workload has grown.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/policy/resource-quotas/",
		},
		Priority: 92,
	}
}

func (r *clusterQuotaExhausted) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	ns := rc.Target.Namespace
	if ns == "" {
		// Cluster-scoped: check all namespaces we can see.
		return r.evaluateCluster(ctx, rc)
	}
	return r.evaluateNamespace(ctx, rc, ns)
}

func (r *clusterQuotaExhausted) evaluateNamespace(ctx context.Context, rc *Context, ns string) ([]findings.Finding, error) {
	quotas, err := rc.Cache.ListResourceQuotas(ctx, ns)
	if err != nil {
		return nil, nil
	}
	return quotaFindings(rc, ns, quotas), nil
}

func (r *clusterQuotaExhausted) evaluateCluster(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	// For cluster-scoped target, list quotas across "" (all namespaces).
	quotas, err := rc.Cache.ListResourceQuotas(ctx, "")
	if err != nil {
		return nil, nil
	}
	// Group by namespace and produce findings.
	byNS := map[string][]corev1.ResourceQuota{}
	for _, q := range quotas {
		byNS[q.Namespace] = append(byNS[q.Namespace], q)
	}
	var out []findings.Finding
	for ns, qs := range byNS {
		out = append(out, quotaFindings(rc, ns, qs)...)
	}
	return out, nil
}

type quotaHit struct {
	quotaName string
	resource  string
	used      resource.Quantity
	hard      resource.Quantity
	pct       int // 0-100+
}

func quotaFindings(rc *Context, ns string, quotas []corev1.ResourceQuota) []findings.Finding {
	var out []findings.Finding

	for _, q := range quotas {
		var exhausted []quotaHit
		var nearlyExhausted []quotaHit

		for resourceName, hardQty := range q.Status.Hard {
			usedQty, ok := q.Status.Used[resourceName]
			if !ok {
				continue
			}
			if hardQty.IsZero() {
				continue
			}

			pct := usedPercent(usedQty, hardQty)
			hit := quotaHit{
				quotaName: q.Name,
				resource:  string(resourceName),
				used:      usedQty,
				hard:      hardQty,
				pct:       pct,
			}
			switch {
			case pct >= 100:
				exhausted = append(exhausted, hit)
			case pct >= 95:
				nearlyExhausted = append(nearlyExhausted, hit)
			}
		}

		if len(exhausted) == 0 && len(nearlyExhausted) == 0 {
			continue
		}

		severity := findings.SeverityHigh
		if len(exhausted) > 0 {
			severity = findings.SeverityCritical
		}

		all := append(exhausted, nearlyExhausted...)
		ev := make([]findings.Evidence, 0, len(all))
		for _, h := range all {
			status := fmt.Sprintf("used=%s hard=%s (%d%%)", h.used.String(), h.hard.String(), h.pct)
			if h.pct >= 100 {
				status += " — EXHAUSTED"
			}
			ev = append(ev, findings.Evidence{
				Kind:   findings.EvidenceKindField,
				Source: fmt.Sprintf("resourcequota/%s.status[%s]", h.quotaName, h.resource),
				Value:  status,
			})
		}

		resourceSummary := quotaSummaryLine(exhausted, nearlyExhausted)
		title := fmt.Sprintf("ResourceQuota %q in namespace %q: %s", q.Name, ns, resourceSummary)

		var summaryParts []string
		if len(exhausted) > 0 {
			names := resourceNames(exhausted)
			summaryParts = append(summaryParts, fmt.Sprintf("%d resource(s) fully exhausted (%s) — new workloads will be rejected", len(exhausted), strings.Join(names, ", ")))
		}
		if len(nearlyExhausted) > 0 {
			names := resourceNames(nearlyExhausted)
			summaryParts = append(summaryParts, fmt.Sprintf("%d resource(s) ≥95%% used (%s)", len(nearlyExhausted), strings.Join(names, ", ")))
		}

		out = append(out, findings.Finding{
			ID:         "TRG-CLUSTER-QUOTA-EXHAUSTED",
			RuleID:     "TRG-CLUSTER-QUOTA-EXHAUSTED",
			Title:      title,
			Summary:    fmt.Sprintf("Namespace %q ResourceQuota %q: %s.", ns, q.Name, strings.Join(summaryParts, "; ")),
			Category:   findings.CategoryResourcePressure,
			Severity:   severity,
			Confidence: findings.ConfidenceHigh,
			Target:     rc.Target,
			Related: []findings.ResourceRef{
				{APIVersion: "v1", Kind: "ResourceQuota", Namespace: ns, Name: q.Name},
			},
			Evidence: ev,
			Remediation: findings.Remediation{
				Explanation: "The ResourceQuota hard limit has been reached. The API server will reject new object creation until usage drops below the limit.",
				NextCommands: []string{
					fmt.Sprintf("kubectl describe resourcequota -n %s %s", ns, q.Name),
					fmt.Sprintf("kubectl get pods -n %s --field-selector=status.phase=Succeeded", ns),
					fmt.Sprintf("kubectl get pods -n %s --field-selector=status.phase=Failed", ns),
				},
				SuggestedFix: "Either increase the ResourceQuota hard limits, or reduce usage:\n" +
					"- Delete completed/failed pods (they still count toward pod quota).\n" +
					"- Scale down unused Deployments.\n" +
					"- Request a quota increase from the cluster admin.",
				DocsLinks: []string{
					"https://kubernetes.io/docs/concepts/policy/resource-quotas/",
				},
			},
		})
	}
	return out
}

// usedPercent returns used/hard as an integer percentage (can exceed 100).
func usedPercent(used, hard resource.Quantity) int {
	usedVal := used.MilliValue()
	hardVal := hard.MilliValue()
	if hardVal == 0 {
		return 0
	}
	return int(usedVal * 100 / hardVal)
}

func resourceNames(hits []quotaHit) []string {
	names := make([]string, len(hits))
	for i, h := range hits {
		names[i] = h.resource
	}
	return names
}

func quotaSummaryLine(exhausted, nearly []quotaHit) string {
	if len(exhausted) > 0 {
		return fmt.Sprintf("%d resource(s) exhausted", len(exhausted))
	}
	return fmt.Sprintf("%d resource(s) ≥95%% used", len(nearly))
}

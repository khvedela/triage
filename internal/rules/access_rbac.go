package rules

import (
	"context"
	"fmt"

	"github.com/OWNER/triage/internal/findings"
	"github.com/OWNER/triage/internal/kube"
)

func init() { Register(&accessRBAC{}) }

type accessRBAC struct{}

func (r *accessRBAC) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-ACCESS-INSUFFICIENT-READ",
		Title:    "triage has insufficient RBAC permissions; diagnosis is incomplete",
		Category: findings.CategoryAccess,
		Severity: findings.SeverityInfo,
		Scopes: []findings.TargetKind{
			findings.TargetKindPod,
			findings.TargetKindDeployment,
			findings.TargetKindNamespace,
			findings.TargetKindCluster,
		},
		Description: `triage was denied read access to one or more Kubernetes resources needed for
a complete diagnosis. Results may be incomplete.

This is not necessarily a problem with your workloads — it is a signal that
the user or service account running triage does not have the required RBAC
permissions to perform a full inspection.

To diagnose with full fidelity, grant read (get/list) on: pods, events,
deployments, replicasets, services, endpoints, configmaps, secrets,
persistentvolumeclaims, nodes.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/reference/access-authn-authz/rbac/",
		},
		Priority: 1, // lowest — this is a meta finding
	}
}

// resourceCheck describes one RBAC probe.
type resourceCheck struct {
	verb      string
	group     string
	resource  string
	namespace string
}

func (r *accessRBAC) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	ns := rc.Target.Namespace

	checks := []resourceCheck{
		{"get", "", "pods", ns},
		{"list", "", "events", ns},
		{"get", "apps", "deployments", ns},
		{"list", "apps", "replicasets", ns},
		{"list", "", "services", ns},
		{"list", "", "endpoints", ns},
	}
	if rc.Target.Kind == findings.TargetKindCluster {
		checks = append(checks, resourceCheck{"list", "", "nodes", ""})
	}

	var denied []string
	for _, check := range checks {
		allowed, err := rc.Cache.CanI(ctx, check.verb, check.group, check.resource, check.namespace)
		if err != nil {
			if kube.IsForbidden(err) {
				denied = append(denied, fmt.Sprintf("%s %s/%s", check.verb, check.group, check.resource))
			}
			continue
		}
		if !allowed {
			denied = append(denied, fmt.Sprintf("%s %s/%s", check.verb, check.group, check.resource))
		}
	}

	if len(denied) == 0 {
		return nil, nil
	}

	return []findings.Finding{{
		ID:         "TRG-ACCESS-INSUFFICIENT-READ",
		RuleID:     "TRG-ACCESS-INSUFFICIENT-READ",
		Title:      fmt.Sprintf("triage is missing RBAC access to %d resource type(s)", len(denied)),
		Summary:    fmt.Sprintf("Diagnosis is incomplete because triage cannot read: %v. Other rule findings may have lower confidence.", denied),
		Category:   findings.CategoryAccess,
		Severity:   findings.SeverityInfo,
		Confidence: findings.ConfidenceHigh,
		Target:     rc.Target,
		Evidence: []findings.Evidence{
			{Kind: findings.EvidenceKindComputed, Value: fmt.Sprintf("denied operations: %v", denied)},
		},
		Remediation: findings.Remediation{
			Explanation: "Grant get/list access to these resources for the account running triage.",
			NextCommands: []string{
				"kubectl auth can-i --list",
				fmt.Sprintf("kubectl auth can-i list pods -n %s", ns),
			},
			SuggestedFix: "Create a ClusterRole with read access to the required resources, " +
				"bind it to the account running triage, and re-run.",
			DocsLinks: []string{"https://kubernetes.io/docs/reference/access-authn-authz/rbac/"},
		},
	}}, nil
}

package rules

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/khvedela/triage/internal/findings"
)

func init() {
	Register(&deployRolloutStuck{})
	Register(&deployUnavailable{})
}

// ----- TRG-DEPLOY-ROLLOUT-STUCK ------------------------------------------

type deployRolloutStuck struct{}

func (r *deployRolloutStuck) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-DEPLOY-ROLLOUT-STUCK",
		Title:    "Deployment rollout has exceeded its progress deadline",
		Category: findings.CategoryRollout,
		Severity: findings.SeverityCritical,
		Scopes:   []findings.TargetKind{findings.TargetKindDeployment},
		Description: `The Deployment's progressDeadlineSeconds was exceeded. Kubernetes marks the
Deployment with condition Progressing=False / ProgressDeadlineExceeded.
The rollout is stuck — new pods are not becoming ready within the deadline.

Common causes:
- New pods crash (CrashLoopBackOff) and never become ready.
- New pods are unschedulable (insufficient resources, taint mismatch).
- The readiness probe on the new pods is never passing.
- A PVC or Secret is missing that the new pods need.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#failed-deployment",
		},
		Priority: 95,
	}
}

func (r *deployRolloutStuck) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	deploy, found, err := rc.Cache.GetDeployment(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil || !found {
		return nil, err
	}

	if !hasDeployCondition(deploy, "Progressing", "False", "ProgressDeadlineExceeded") {
		return nil, nil
	}

	ns, name := rc.Target.Namespace, rc.Target.Name
	desired := int32(0)
	if deploy.Spec.Replicas != nil {
		desired = *deploy.Spec.Replicas
	}
	ready := deploy.Status.ReadyReplicas

	return []findings.Finding{{
		ID:         "TRG-DEPLOY-ROLLOUT-STUCK",
		RuleID:     "TRG-DEPLOY-ROLLOUT-STUCK",
		Title:      fmt.Sprintf("Deployment %q rollout stuck: ProgressDeadlineExceeded", name),
		Summary:    fmt.Sprintf("Deployment %q in namespace %q has not progressed within its deadline. %d/%d replicas ready.", name, ns, ready, desired),
		Category:   findings.CategoryRollout,
		Severity:   findings.SeverityCritical,
		Confidence: findings.ConfidenceHigh,
		Target:     rc.Target,
		Evidence: []findings.Evidence{
			{Kind: findings.EvidenceKindField, Source: "deployment.status.conditions[Progressing].reason", Value: "ProgressDeadlineExceeded"},
			{Kind: findings.EvidenceKindComputed, Value: fmt.Sprintf("readyReplicas=%d desiredReplicas=%d", ready, desired)},
		},
		Remediation: findings.Remediation{
			Explanation: "The rollout has timed out. Check the pods in the new ReplicaSet for crash or scheduling failures.",
			NextCommands: []string{
				fmt.Sprintf("kubectl rollout status deployment/%s -n %s", name, ns),
				fmt.Sprintf("kubectl get pods -n %s -l app=%s", ns, name),
				fmt.Sprintf("kubectl describe deployment -n %s %s", ns, name),
				fmt.Sprintf("kubectl rollout history deployment/%s -n %s", name, ns),
			},
			SuggestedFix: "Inspect the failing pods in the new ReplicaSet. Common next steps:\n" +
				"1. `kubectl rollout undo deployment/" + name + " -n " + ns + "` to roll back.\n" +
				"2. Fix the underlying issue (crash, readiness, scheduling).\n" +
				"3. Re-deploy.",
			DocsLinks: []string{
				"https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#failed-deployment",
			},
		},
	}}, nil
}

// ----- TRG-DEPLOY-UNAVAILABLE-REPLICAS -----------------------------------

type deployUnavailable struct{}

func (r *deployUnavailable) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-DEPLOY-UNAVAILABLE-REPLICAS",
		Title:    "Deployment has unavailable replicas",
		Category: findings.CategoryRollout,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindDeployment},
		Description: `The Deployment has fewer ready replicas than desired. Some pods are either
not running, crashing, or failing their readiness probe. This reduces the
available capacity and may impact traffic.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/workloads/controllers/deployment/",
		},
		Priority: 85,
	}
}

func (r *deployUnavailable) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	deploy, found, err := rc.Cache.GetDeployment(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil || !found {
		return nil, err
	}

	desired := int32(0)
	if deploy.Spec.Replicas != nil {
		desired = *deploy.Spec.Replicas
	}
	ready := deploy.Status.ReadyReplicas
	unavail := deploy.Status.UnavailableReplicas

	if unavail == 0 || ready >= desired {
		return nil, nil
	}
	// Don't double-report if rollout-stuck already fired.
	if hasDeployCondition(deploy, "Progressing", "False", "ProgressDeadlineExceeded") {
		return nil, nil
	}

	ns, name := rc.Target.Namespace, rc.Target.Name
	return []findings.Finding{{
		ID:         "TRG-DEPLOY-UNAVAILABLE-REPLICAS",
		RuleID:     "TRG-DEPLOY-UNAVAILABLE-REPLICAS",
		Title:      fmt.Sprintf("Deployment %q: %d/%d replicas unavailable", name, unavail, desired),
		Summary:    fmt.Sprintf("%d of %d desired replicas are unavailable in Deployment %q.", unavail, desired, name),
		Category:   findings.CategoryRollout,
		Severity:   findings.SeverityHigh,
		Confidence: findings.ConfidenceHigh,
		Target:     rc.Target,
		Evidence: []findings.Evidence{
			{Kind: findings.EvidenceKindField, Source: "deployment.status.readyReplicas", Value: fmt.Sprintf("%d", ready)},
			{Kind: findings.EvidenceKindField, Source: "deployment.status.unavailableReplicas", Value: fmt.Sprintf("%d", unavail)},
			{Kind: findings.EvidenceKindField, Source: "deployment.spec.replicas", Value: fmt.Sprintf("%d", desired)},
		},
		Remediation: findings.Remediation{
			Explanation: "Some pods are not reaching Ready state. Diagnose individual pods to find the root cause.",
			NextCommands: []string{
				fmt.Sprintf("kubectl get pods -n %s -l app=%s", ns, name),
				fmt.Sprintf("kubectl describe deployment -n %s %s", ns, name),
			},
			SuggestedFix: "Run `triage pod <pod-name> -n " + ns + "` against one of the failing pods for a detailed diagnosis.",
		},
	}}, nil
}

// ----- deployment helpers -------------------------------------------------

func hasDeployCondition(d *appsv1.Deployment, condType, status, reason string) bool {
	for _, c := range d.Status.Conditions {
		if string(c.Type) == condType && string(c.Status) == status {
			if reason == "" || c.Reason == reason {
				return true
			}
		}
	}
	return false
}

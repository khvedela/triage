package rules

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"

	"github.com/khvedela/triage/internal/findings"
	"github.com/khvedela/triage/internal/kube"
)

func init() {
	Register(&podPendingResources{})
	Register(&podPendingTaint{})
	Register(&podPendingSelector{})
	Register(&podPendingPVC{})
}

// ----- TRG-POD-PENDING-INSUFFICIENT-RESOURCES ----------------------------

type podPendingResources struct{}

func (r *podPendingResources) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-PENDING-INSUFFICIENT-RESOURCES",
		Title:    "Pod is Pending due to insufficient CPU or memory on all nodes",
		Category: findings.CategoryScheduling,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `No node has enough allocatable CPU or memory to satisfy the pod's resource
requests. The pod will remain Pending until a node with sufficient capacity
becomes available.

Common causes:
- Resource requests are set too high relative to cluster capacity.
- All nodes are at capacity; scale up the cluster or reduce requests.
- Requests have been set in millicores/Mi that are unexpectedly large.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",
		},
		Priority: 75,
	}
}

func (r *podPendingResources) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil || !found {
		return nil, err
	}
	if pod.Status.Phase != corev1.PodPending {
		return nil, nil
	}

	events, _ := rc.Cache.ListEventsFor(ctx, "Pod", rc.Target.Namespace, rc.Target.Name)
	msg, matched := schedulingEventMsg(events, func(m string) bool {
		m = strings.ToLower(m)
		return strings.Contains(m, "insufficient cpu") ||
			strings.Contains(m, "insufficient memory") ||
			strings.Contains(m, "insufficient hugepages") ||
			strings.Contains(m, "insufficient ephemeral-storage")
	})
	if !matched {
		return nil, nil
	}

	requests := podResourceSummary(pod)
	ns, name := rc.Target.Namespace, rc.Target.Name

	ev := []findings.Evidence{
		{Kind: findings.EvidenceKindField, Source: "pod.status.phase", Value: "Pending"},
		{Kind: findings.EvidenceKindEvent, Source: "FailedScheduling", Value: truncate(msg, 200)},
	}
	if requests != "" {
		ev = append(ev, findings.Evidence{Kind: findings.EvidenceKindComputed, Value: "Pod resource requests: " + requests})
	}

	return []findings.Finding{{
		ID:         "TRG-POD-PENDING-INSUFFICIENT-RESOURCES",
		RuleID:     "TRG-POD-PENDING-INSUFFICIENT-RESOURCES",
		Title:      "Pod is unschedulable: insufficient CPU/memory across all nodes",
		Summary:    "No node has enough allocatable capacity to place this pod. Either reduce resource requests or add node capacity.",
		Category:   findings.CategoryScheduling,
		Severity:   findings.SeverityHigh,
		Confidence: findings.ConfidenceHigh,
		Target:     rc.Target,
		Evidence:   ev,
		Remediation: findings.Remediation{
			Explanation: "The scheduler could not find a node that satisfies the pod's resource requests.",
			NextCommands: []string{
				fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
				"kubectl describe nodes | grep -A5 'Allocated resources'",
				"kubectl top nodes",
			},
			SuggestedFix: "Lower resources.requests in the container spec, or add nodes to the cluster. " +
				"If using a Cluster Autoscaler, check why it has not already scaled up.",
		},
	}}, nil
}

// ----- TRG-POD-PENDING-TAINT-MISMATCH ------------------------------------

type podPendingTaint struct{}

func (r *podPendingTaint) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-PENDING-TAINT-MISMATCH",
		Title:    "Pod is Pending due to untolerated node taints",
		Category: findings.CategoryScheduling,
		Severity: findings.SeverityMedium,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `All nodes carry taints that this pod does not tolerate. The pod must declare
tolerations for all required taints to be scheduled.

Common in: dedicated node groups, spot instances, GPU nodes, or nodes marked
with NoSchedule for maintenance.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/",
		},
		Priority: 73,
	}
}

func (r *podPendingTaint) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil || !found {
		return nil, err
	}
	if pod.Status.Phase != corev1.PodPending {
		return nil, nil
	}

	events, _ := rc.Cache.ListEventsFor(ctx, "Pod", rc.Target.Namespace, rc.Target.Name)
	msg, matched := schedulingEventMsg(events, func(m string) bool {
		m = strings.ToLower(m)
		return strings.Contains(m, "untolerated taint") ||
			strings.Contains(m, "had taint") ||
			strings.Contains(m, "toleration")
	})
	if !matched {
		return nil, nil
	}

	ns, name := rc.Target.Namespace, rc.Target.Name
	return []findings.Finding{{
		ID:         "TRG-POD-PENDING-TAINT-MISMATCH",
		RuleID:     "TRG-POD-PENDING-TAINT-MISMATCH",
		Title:      "Pod is unschedulable: no node tolerates its required taints",
		Summary:    "All nodes have taints this pod does not tolerate. Add the appropriate tolerations to the pod spec.",
		Category:   findings.CategoryScheduling,
		Severity:   findings.SeverityMedium,
		Confidence: findings.ConfidenceHigh,
		Target:     rc.Target,
		Evidence: []findings.Evidence{
			{Kind: findings.EvidenceKindField, Source: "pod.status.phase", Value: "Pending"},
			{Kind: findings.EvidenceKindEvent, Source: "FailedScheduling", Value: truncate(msg, 200)},
		},
		Remediation: findings.Remediation{
			Explanation: "Kubernetes taints prevent the scheduler from placing the pod on any available node.",
			NextCommands: []string{
				fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
				"kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints",
			},
			SuggestedFix: "Add spec.tolerations to the pod/deployment spec matching the node taints. " +
				"Example: tolerations: [{key: 'gpu', operator: 'Exists', effect: 'NoSchedule'}]",
			DocsLinks: []string{"https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/"},
		},
	}}, nil
}

// ----- TRG-POD-PENDING-SELECTOR-MISMATCH ---------------------------------

type podPendingSelector struct{}

func (r *podPendingSelector) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-PENDING-SELECTOR-MISMATCH",
		Title:    "Pod is Pending due to node selector or affinity mismatch",
		Category: findings.CategoryScheduling,
		Severity: findings.SeverityMedium,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `No node satisfies the pod's nodeSelector, nodeAffinity, or podAffinity rules.
The pod will wait indefinitely until a matching node is available or the
scheduling constraints are relaxed.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/",
		},
		Priority: 72,
	}
}

func (r *podPendingSelector) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil || !found {
		return nil, err
	}
	if pod.Status.Phase != corev1.PodPending {
		return nil, nil
	}

	events, _ := rc.Cache.ListEventsFor(ctx, "Pod", rc.Target.Namespace, rc.Target.Name)
	msg, matched := schedulingEventMsg(events, func(m string) bool {
		m = strings.ToLower(m)
		return strings.Contains(m, "node affinity") ||
			strings.Contains(m, "nodeselector") ||
			strings.Contains(m, "node selector") ||
			strings.Contains(m, "match pod's node affinity") ||
			strings.Contains(m, "didn't match")
	})
	if !matched {
		return nil, nil
	}

	ns, name := rc.Target.Namespace, rc.Target.Name
	return []findings.Finding{{
		ID:         "TRG-POD-PENDING-SELECTOR-MISMATCH",
		RuleID:     "TRG-POD-PENDING-SELECTOR-MISMATCH",
		Title:      "Pod is unschedulable: no node matches its nodeSelector/affinity",
		Summary:    "The pod's scheduling constraints (nodeSelector or nodeAffinity) don't match any available node.",
		Category:   findings.CategoryScheduling,
		Severity:   findings.SeverityMedium,
		Confidence: findings.ConfidenceHigh,
		Target:     rc.Target,
		Evidence: []findings.Evidence{
			{Kind: findings.EvidenceKindField, Source: "pod.status.phase", Value: "Pending"},
			{Kind: findings.EvidenceKindEvent, Source: "FailedScheduling", Value: truncate(msg, 200)},
		},
		Remediation: findings.Remediation{
			Explanation: "The pod's nodeSelector or nodeAffinity requirements don't match any node's labels.",
			NextCommands: []string{
				fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
				"kubectl get nodes --show-labels",
			},
			SuggestedFix: "Either label a node to match the pod's selector, or relax/remove the nodeSelector/affinity in the pod spec.",
			DocsLinks:    []string{"https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/"},
		},
	}}, nil
}

// ----- TRG-POD-PENDING-PVC -----------------------------------------------

type podPendingPVC struct{}

func (r *podPendingPVC) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-PENDING-PVC-UNBOUND",
		Title:    "Pod is Pending because a referenced PVC is not bound",
		Category: findings.CategoryStorage,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `A PersistentVolumeClaim (PVC) referenced by this pod is in Pending state — it
has not been bound to a PersistentVolume. The pod cannot start until the PVC
is bound.

Common causes:
- No PersistentVolume matches the PVC's storageClass, accessMode, or capacity.
- The StorageClass does not exist.
- Dynamic provisioning failed (CSI driver error, quota, permissions).`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/storage/persistent-volumes/",
		},
		Priority: 78,
	}
}

func (r *podPendingPVC) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil || !found {
		return nil, err
	}
	if pod.Status.Phase != corev1.PodPending {
		return nil, nil
	}

	ns := rc.Target.Namespace
	var out []findings.Finding

	for _, v := range pod.Spec.Volumes {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		pvcName := v.PersistentVolumeClaim.ClaimName
		pvc, pvcFound, pvcErr := rc.Cache.GetPVC(ctx, ns, pvcName)
		if pvcErr != nil {
			if kube.IsForbidden(pvcErr) {
				continue
			}
			continue
		}
		if !pvcFound {
			// PVC itself missing
			out = append(out, findings.Finding{
				ID:         "TRG-POD-PENDING-PVC-UNBOUND",
				RuleID:     "TRG-POD-PENDING-PVC-UNBOUND",
				Title:      fmt.Sprintf("PVC %q referenced by pod does not exist", pvcName),
				Summary:    fmt.Sprintf("Pod mounts volume %q via PVC %q, but that PVC does not exist in namespace %q.", v.Name, pvcName, ns),
				Category:   findings.CategoryStorage,
				Severity:   findings.SeverityHigh,
				Confidence: findings.ConfidenceHigh,
				Target:     rc.Target,
				Evidence: []findings.Evidence{
					{Kind: findings.EvidenceKindField, Source: fmt.Sprintf("spec.volumes[%s].persistentVolumeClaim.claimName", v.Name), Value: pvcName},
					{Kind: findings.EvidenceKindComputed, Value: fmt.Sprintf("kubectl get pvc -n %s %s → NotFound", ns, pvcName)},
				},
				Remediation: findings.Remediation{
					Explanation:  "The PVC must exist and be Bound before the pod can start.",
					NextCommands: []string{fmt.Sprintf("kubectl get pvc -n %s", ns)},
					SuggestedFix: fmt.Sprintf("Create PVC %q or correct the volume reference in the pod spec.", pvcName),
				},
			})
			continue
		}

		if pvc.Status.Phase != corev1.ClaimBound {
			scName := ""
			if pvc.Spec.StorageClassName != nil {
				scName = *pvc.Spec.StorageClassName
			}
			out = append(out, findings.Finding{
				ID:         "TRG-POD-PENDING-PVC-UNBOUND",
				RuleID:     "TRG-POD-PENDING-PVC-UNBOUND",
				Title:      fmt.Sprintf("PVC %q is in %q state (not Bound)", pvcName, pvc.Status.Phase),
				Summary:    fmt.Sprintf("PVC %q is %q. Pod cannot start until it is Bound.", pvcName, pvc.Status.Phase),
				Category:   findings.CategoryStorage,
				Severity:   findings.SeverityHigh,
				Confidence: findings.ConfidenceHigh,
				Target:     rc.Target,
				Evidence: []findings.Evidence{
					{Kind: findings.EvidenceKindField, Source: fmt.Sprintf("pvc/%s.status.phase", pvcName), Value: string(pvc.Status.Phase)},
					{Kind: findings.EvidenceKindField, Source: fmt.Sprintf("pvc/%s.spec.storageClassName", pvcName), Value: scName},
				},
				Remediation: findings.Remediation{
					Explanation: "The PVC needs a matching PersistentVolume or a StorageClass with dynamic provisioning.",
					NextCommands: []string{
						fmt.Sprintf("kubectl describe pvc -n %s %s", ns, pvcName),
						fmt.Sprintf("kubectl get pv | grep %s", pvcName),
						"kubectl get storageclass",
					},
					SuggestedFix: "Check that the StorageClass exists and the provisioner is functioning. " +
						"Inspect PVC events for provisioning errors.",
				},
			})
		}
	}
	return out, nil
}

// ----- shared scheduler helpers -------------------------------------------

// schedulingEventMsg returns the first FailedScheduling event message
// that satisfies predicate, and true if any match was found.
func schedulingEventMsg(events []eventsv1.Event, predicate func(string) bool) (string, bool) {
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		if e.Reason == "FailedScheduling" && predicate(e.Note) {
			return e.Note, true
		}
	}
	return "", false
}

func podResourceSummary(pod *corev1.Pod) string {
	var parts []string
	for _, c := range pod.Spec.Containers {
		req := c.Resources.Requests
		cpu := req.Cpu()
		mem := req.Memory()
		if !cpu.IsZero() || !mem.IsZero() {
			parts = append(parts, fmt.Sprintf("%s: cpu=%s memory=%s", c.Name, cpu.String(), mem.String()))
		}
	}
	return strings.Join(parts, "; ")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

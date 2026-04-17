package rules

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/OWNER/triage/internal/findings"
)

func init() {
	Register(&clusterNodeNotReady{})
	Register(&clusterNodePressure{})
}

// ----- TRG-CLUSTER-NODE-NOT-READY ----------------------------------------

type clusterNodeNotReady struct{}

func (r *clusterNodeNotReady) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-CLUSTER-NODE-NOT-READY",
		Title:    "One or more cluster nodes are NotReady",
		Category: findings.CategoryResourcePressure,
		Severity: findings.SeverityCritical,
		Scopes:   []findings.TargetKind{findings.TargetKindCluster, findings.TargetKindNamespace},
		Description: `At least one node has condition Ready=False or Ready=Unknown, meaning the
kubelet on that node is not communicating with the control plane. Pods
scheduled to that node will not start and existing pods may become Terminating.

Common causes:
- Node lost network connectivity.
- kubelet process crashed or is OOMKilled on the node.
- Disk is full on the node (log or container image partition).
- Node was forcibly shut down (spot instance reclaimed, maintenance).`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/architecture/nodes/#node-status",
		},
		Priority: 99,
	}
}

func (r *clusterNodeNotReady) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	nodes, err := rc.Cache.ListNodes(ctx)
	if err != nil {
		return nil, nil // RBAC or cluster error — degrade gracefully
	}

	var notReady []corev1.Node
	for _, n := range nodes {
		if !nodeReady(n) {
			notReady = append(notReady, n)
		}
	}
	if len(notReady) == 0 {
		return nil, nil
	}

	names := make([]string, len(notReady))
	for i, n := range notReady {
		names[i] = n.Name
	}

	severity := findings.SeverityHigh
	if len(notReady) > len(nodes)/2 {
		severity = findings.SeverityCritical
	}

	ev := []findings.Evidence{
		{Kind: findings.EvidenceKindComputed, Value: fmt.Sprintf("%d/%d nodes NotReady: %s", len(notReady), len(nodes), strings.Join(names, ", "))},
	}
	for _, n := range notReady {
		for _, c := range n.Status.Conditions {
			if c.Type == corev1.NodeReady {
				ev = append(ev, findings.Evidence{
					Kind:  findings.EvidenceKindField,
					Source: fmt.Sprintf("node/%s.status.conditions[Ready]", n.Name),
					Value: fmt.Sprintf("status=%s reason=%s message=%s", c.Status, c.Reason, truncate(c.Message, 100)),
				})
			}
		}
	}

	return []findings.Finding{{
		ID:         "TRG-CLUSTER-NODE-NOT-READY",
		RuleID:     "TRG-CLUSTER-NODE-NOT-READY",
		Title:      fmt.Sprintf("%d cluster node(s) are NotReady", len(notReady)),
		Summary:    fmt.Sprintf("%d of %d nodes are NotReady: %s. New pods will not be scheduled to these nodes.", len(notReady), len(nodes), strings.Join(names, ", ")),
		Category:   findings.CategoryResourcePressure,
		Severity:   severity,
		Confidence: findings.ConfidenceHigh,
		Target:     rc.Target,
		Evidence:   ev,
		Remediation: findings.Remediation{
			Explanation: "Nodes are not communicating with the control plane. Check node health and kubelet logs.",
			NextCommands: []string{
				"kubectl get nodes",
				fmt.Sprintf("kubectl describe node %s", strings.Join(names, " ")),
			},
			SuggestedFix: "SSH to the affected node and check: `systemctl status kubelet`, disk usage (`df -h`), and memory (`free -h`). " +
				"If the node is in a cloud environment, check whether it has been reclaimed or terminated.",
		},
	}}, nil
}

// ----- TRG-CLUSTER-NODE-PRESSURE -----------------------------------------

type clusterNodePressure struct{}

func (r *clusterNodePressure) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-CLUSTER-NODE-PRESSURE",
		Title:    "One or more nodes have Memory, Disk, or PID pressure",
		Category: findings.CategoryResourcePressure,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindCluster, findings.TargetKindNamespace},
		Description: `One or more nodes report pressure conditions: MemoryPressure, DiskPressure,
or PIDPressure. Under pressure, the kubelet will evict pods (starting with
Best-Effort, then Burstable) until pressure is relieved.

Pressure conditions are early warnings — address them before pods start being
evicted.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/concepts/scheduling-eviction/node-pressure-eviction/",
		},
		Priority: 90,
	}
}

func (r *clusterNodePressure) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	nodes, err := rc.Cache.ListNodes(ctx)
	if err != nil {
		return nil, nil
	}

	pressureConditions := []corev1.NodeConditionType{
		corev1.NodeMemoryPressure,
		corev1.NodeDiskPressure,
		corev1.NodePIDPressure,
	}

	type pressureEntry struct {
		node      string
		condition string
		message   string
	}
	var pressured []pressureEntry

	for _, n := range nodes {
		for _, ct := range pressureConditions {
			for _, c := range n.Status.Conditions {
				if c.Type == ct && c.Status == corev1.ConditionTrue {
					pressured = append(pressured, pressureEntry{n.Name, string(ct), c.Message})
				}
			}
		}
	}

	if len(pressured) == 0 {
		return nil, nil
	}

	ev := make([]findings.Evidence, len(pressured))
	for i, p := range pressured {
		ev[i] = findings.Evidence{
			Kind:  findings.EvidenceKindField,
			Source: fmt.Sprintf("node/%s.status.conditions[%s]", p.node, p.condition),
			Value: truncate(p.message, 150),
		}
	}

	nodeNames := make([]string, len(pressured))
	for i, p := range pressured {
		nodeNames[i] = fmt.Sprintf("%s(%s)", p.node, p.condition)
	}

	return []findings.Finding{{
		ID:         "TRG-CLUSTER-NODE-PRESSURE",
		RuleID:     "TRG-CLUSTER-NODE-PRESSURE",
		Title:      fmt.Sprintf("%d node pressure condition(s) detected", len(pressured)),
		Summary:    fmt.Sprintf("Nodes with pressure: %s. Evictions may be imminent.", strings.Join(nodeNames, ", ")),
		Category:   findings.CategoryResourcePressure,
		Severity:   findings.SeverityHigh,
		Confidence: findings.ConfidenceHigh,
		Target:     rc.Target,
		Evidence:   ev,
		Remediation: findings.Remediation{
			Explanation: "Node resource pressure causes the kubelet to evict pods. Address the root resource exhaustion.",
			NextCommands: []string{
				"kubectl get nodes",
				"kubectl top nodes",
				"kubectl describe nodes | grep -A10 'Conditions:'",
			},
			SuggestedFix: "For MemoryPressure: identify high-memory pods and reduce their limits or add nodes. " +
				"For DiskPressure: clear container images (`crictl rmi --prune`), logs, or add disk. " +
				"For PIDPressure: find processes spawning many child PIDs.",
		},
	}}, nil
}

// ----- node helpers -------------------------------------------------------

func nodeReady(n corev1.Node) bool {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

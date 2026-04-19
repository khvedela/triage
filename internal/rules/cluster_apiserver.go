package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/khvedela/triage/internal/findings"
)

func init() { Register(&clusterAPIServerLatency{}) }

// TRG-CLUSTER-APISERVER-LATENCY detects events in the kube-system namespace
// (or all namespaces) that indicate elevated API server latency or unavailability.
// The primary signals are Warning events with reasons such as:
//   - "SlowReadResponse", "SlowWriteResponse" (kube-apiserver)
//   - "FailedToCreateEndpoint" sourced from EndpointSlice/Endpoint controllers
//   - "LeaderElection" loss or "NodeNotSchedulable" bursts
//   - "Timeout", "context deadline exceeded", "etcd" errors in event notes
type clusterAPIServerLatency struct{}

func (r *clusterAPIServerLatency) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-CLUSTER-APISERVER-LATENCY",
		Title:    "API server latency or availability events detected",
		Category: findings.CategoryResourcePressure,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindCluster, findings.TargetKindNamespace},
		Description: `Warning events have been detected that indicate the Kubernetes API server is
experiencing elevated latency or transient unavailability. These events may
originate from:

- The API server itself (SlowReadResponse, SlowWriteResponse)
- The etcd backend (timeout, leader election)
- Controllers failing to reach the API (FailedToCreateEndpoint, context deadline exceeded)

API server latency causes cascading failures: controllers fall behind, pod
readiness state becomes stale, and deployments stall. It is usually caused by:
- etcd disk I/O saturation
- Control-plane node resource exhaustion (CPU/memory)
- Large object counts (too many secrets/configmaps/pods in cluster)
- Network issues between control-plane and worker nodes`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/tasks/debug/debug-cluster/",
			"https://kubernetes.io/docs/concepts/cluster-administration/logging/",
		},
		Priority: 88,
	}
}

// apiserverLatencySignals are case-insensitive substrings searched in event
// Reason or Note fields that indicate API server / etcd trouble.
var apiserverLatencySignals = []string{
	"slowreadresponse",
	"slowwriteresponse",
	"timeout",
	"context deadline exceeded",
	"etcd",
	"leader election",
	"api server",
	"apiserver",
	"failed to create endpoint",
	"connection refused",
	"unable to reach apiserver",
	"request timed out",
	"rpc error",
	"transport: error",
}

// apiserverLatencyReasons are exact Reason matches (case-insensitive).
var apiserverLatencyReasons = []string{
	"slowreadresponse",
	"slowwriteresponse",
	"leaderelection",
	"failedtocreateroute",
	"failedtocreateendpoint",
}

func (r *clusterAPIServerLatency) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	// Check kube-system events first (most API server events land there),
	// then the target namespace if different.
	namespacesToCheck := []string{"kube-system"}
	if rc.Target.Namespace != "" && rc.Target.Namespace != "kube-system" {
		namespacesToCheck = append(namespacesToCheck, rc.Target.Namespace)
	}
	// For cluster target, also check "" (all) — but limit to avoid noise.
	if rc.Target.Namespace == "" {
		namespacesToCheck = []string{"kube-system"}
	}

	type matchedEvent struct {
		ns     string
		reason string
		note   string
		signal string
	}
	var matched []matchedEvent

	seen := map[string]bool{}
	for _, ns := range namespacesToCheck {
		events, err := rc.Cache.ListEventsInNamespace(ctx, ns)
		if err != nil {
			continue
		}
		for _, ev := range events {
			if ev.Type != "Warning" {
				continue
			}
			signal := latencySignal(ev.Reason, ev.Note)
			if signal == "" {
				continue
			}
			key := ev.Reason + "|" + truncate(ev.Note, 80)
			if seen[key] {
				continue
			}
			seen[key] = true
			matched = append(matched, matchedEvent{
				ns:     ns,
				reason: ev.Reason,
				note:   ev.Note,
				signal: signal,
			})
			if len(matched) >= 10 {
				break
			}
		}
	}

	if len(matched) == 0 {
		return nil, nil
	}

	ev := make([]findings.Evidence, 0, len(matched))
	for _, m := range matched {
		ev = append(ev, findings.Evidence{
			Kind:  findings.EvidenceKindEvent,
			Value: fmt.Sprintf("[%s] reason=%s: %s", m.ns, m.reason, truncate(m.note, 150)),
		})
	}

	// Deduplicate signal names for the title.
	signalSeen := map[string]bool{}
	var signalNames []string
	for _, m := range matched {
		if !signalSeen[m.signal] {
			signalSeen[m.signal] = true
			signalNames = append(signalNames, m.signal)
		}
	}

	return []findings.Finding{{
		ID:         "TRG-CLUSTER-APISERVER-LATENCY",
		RuleID:     "TRG-CLUSTER-APISERVER-LATENCY",
		Title:      fmt.Sprintf("API server latency events detected (%d events, signals: %s)", len(matched), strings.Join(signalNames, ", ")),
		Summary: fmt.Sprintf(
			"%d Warning event(s) indicate elevated API server latency or availability issues. "+
				"Signals found: %s. This may be causing cascading failures in controllers and workloads.",
			len(matched), strings.Join(signalNames, ", "),
		),
		Category:   findings.CategoryResourcePressure,
		Severity:   findings.SeverityHigh,
		Confidence: findings.ConfidenceMedium,
		Target:     rc.Target,
		Evidence:   ev,
		Remediation: findings.Remediation{
			Explanation: "API server latency events suggest the control plane is under stress. " +
				"This can cause pods to stay in Terminating/Pending, controllers to fall behind, and deployments to stall.",
			NextCommands: []string{
				"kubectl get events -n kube-system --sort-by='.lastTimestamp' | grep -i 'warning'",
				"kubectl top nodes",
				"kubectl get componentstatuses",
				"kubectl get --raw='/readyz?verbose'",
			},
			SuggestedFix: "Check etcd disk I/O and latency (etcd metrics: etcd_disk_wal_fsync_duration_seconds). " +
				"Check control-plane node CPU and memory. " +
				"If the cluster is large, review object count (secrets, configmaps, pods) and enable --watch-cache-sizes.",
			DocsLinks: []string{
				"https://kubernetes.io/docs/tasks/debug/debug-cluster/",
			},
		},
	}}, nil
}

// latencySignal returns a short label if the event reason or note matches a
// known API server / etcd latency signal, otherwise "".
func latencySignal(reason, note string) string {
	reasonLow := strings.ToLower(reason)
	noteLow := strings.ToLower(note)

	for _, r := range apiserverLatencyReasons {
		if reasonLow == r {
			return reason
		}
	}
	for _, sig := range apiserverLatencySignals {
		if strings.Contains(noteLow, sig) || strings.Contains(reasonLow, sig) {
			// Return a canonical label.
			switch {
			case strings.Contains(sig, "etcd"):
				return "etcd"
			case strings.Contains(sig, "timeout") || strings.Contains(sig, "deadline"):
				return "timeout"
			case strings.Contains(sig, "slow"):
				return "slow-response"
			case strings.Contains(sig, "leader"):
				return "leader-election"
			case strings.Contains(sig, "endpoint"):
				return "endpoint-failure"
			default:
				return sig
			}
		}
	}
	return ""
}

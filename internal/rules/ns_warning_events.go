package rules

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/OWNER/triage/internal/findings"
)

func init() { Register(&nsWarningEvents{}) }

type nsWarningEvents struct{}

func (r *nsWarningEvents) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-NS-WARNING-EVENTS",
		Title:    "Namespace has recent Warning events",
		Category: findings.CategoryRuntime,
		Severity: findings.SeverityMedium,
		Scopes:   []findings.TargetKind{findings.TargetKindNamespace},
		Description: `The namespace has recent Kubernetes Warning events across its workloads.
This rule aggregates events by reason and provides a summary — it does not
diagnose individual pods, but gives a namespace-level picture of what is
failing.

Use this as a quick scan before running more specific "triage pod" commands.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/reference/kubectl/cheatsheet/#viewing-and-finding-resources",
		},
		Priority: 60,
	}
}

func (r *nsWarningEvents) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	if rc.Target.Kind != findings.TargetKindNamespace {
		return nil, nil
	}
	ns := rc.Target.Name

	events, err := rc.Cache.ListEventsInNamespace(ctx, ns)
	if err != nil {
		return nil, nil
	}

	// Aggregate Warning events by reason.
	type reasonEntry struct {
		count   int
		samples []string
		objects []string
	}
	byReason := map[string]*reasonEntry{}

	for _, e := range events {
		if e.Type != "Warning" {
			continue
		}
		key := e.Reason
		entry := byReason[key]
		if entry == nil {
			entry = &reasonEntry{}
			byReason[key] = entry
		}
		entry.count++
		if len(entry.samples) < 2 {
			entry.samples = append(entry.samples, truncate(e.Note, 120))
		}
		obj := fmt.Sprintf("%s/%s", strings.ToLower(e.Regarding.Kind), e.Regarding.Name)
		if !containsStr(entry.objects, obj) && len(entry.objects) < 5 {
			entry.objects = append(entry.objects, obj)
		}
	}

	if len(byReason) == 0 {
		return nil, nil
	}

	// Sort reasons by count descending.
	reasons := make([]string, 0, len(byReason))
	for r := range byReason {
		reasons = append(reasons, r)
	}
	sort.Slice(reasons, func(i, j int) bool {
		return byReason[reasons[i]].count > byReason[reasons[j]].count
	})

	ev := make([]findings.Evidence, 0, len(reasons))
	for _, reason := range reasons {
		entry := byReason[reason]
		detail := fmt.Sprintf("reason=%s count=%d objects=[%s]", reason, entry.count, strings.Join(entry.objects, ","))
		if len(entry.samples) > 0 {
			detail += " sample: " + entry.samples[0]
		}
		ev = append(ev, findings.Evidence{
			Kind:  findings.EvidenceKindEvent,
			Value: truncate(detail, 300),
		})
	}

	total := 0
	for _, e := range byReason {
		total += e.count
	}

	return []findings.Finding{{
		ID:         "TRG-NS-WARNING-EVENTS",
		RuleID:     "TRG-NS-WARNING-EVENTS",
		Title:      fmt.Sprintf("Namespace %q has %d Warning events across %d reason(s)", ns, total, len(byReason)),
		Summary:    fmt.Sprintf("Recent Warning events in namespace %q: %s", ns, strings.Join(reasons, ", ")),
		Category:   findings.CategoryRuntime,
		Severity:   findings.SeverityMedium,
		Confidence: findings.ConfidenceMedium,
		Target:     rc.Target,
		Evidence:   ev,
		Remediation: findings.Remediation{
			Explanation: "Run per-workload triage commands for the objects with the most warnings.",
			NextCommands: []string{
				fmt.Sprintf("kubectl get events -n %s --field-selector type=Warning --sort-by='.lastTimestamp'", ns),
				fmt.Sprintf("triage pod <pod-name> -n %s", ns),
			},
		},
	}}, nil
}

func containsStr(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

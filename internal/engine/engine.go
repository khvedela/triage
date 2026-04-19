// Package engine orchestrates a kubediag run: it resolves the target, drives
// the collector to fetch related resources into a request-scoped cache, runs
// all scope-matching rules, then ranks and deduplicates the resulting
// findings into a Report.
package engine

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"

	"github.com/khvedela/kubediag/internal/findings"
	"github.com/khvedela/kubediag/internal/kube"
	"github.com/khvedela/kubediag/internal/rules"
)

// Options are the per-run engine knobs.
type Options struct {
	MaxFindings    int
	SeverityMin    findings.Severity
	ConfidenceMin  findings.Confidence
	IncludeEvents  bool
	IncludeRelated bool
	DisabledRules  []string // rule IDs to skip
	EnabledRules   []string // if non-empty, only these rule IDs run
	Logger         logr.Logger
	Now            func() time.Time
}

func (o *Options) defaults() {
	if o.Logger.GetSink() == nil {
		o.Logger = logr.Discard()
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.MaxFindings <= 0 {
		o.MaxFindings = 20
	}
	if o.SeverityMin == "" {
		o.SeverityMin = findings.SeverityInfo
	}
	if o.ConfidenceMin == "" {
		o.ConfidenceMin = findings.ConfidenceLow
	}
}

// Run performs one diagnosis pass against the given target and returns a Report.
// The returned error indicates an engine-level failure (not cluster access
// denials, which surface as findings) such that rendering a partial report
// is not meaningful.
func Run(ctx context.Context, client kube.Interface, target findings.Target, opts Options) (findings.Report, error) {
	opts.defaults()

	if target.Kind == "" {
		return findings.Report{}, errors.New("engine.Run: target.Kind is empty")
	}

	started := opts.Now()

	cache := kube.NewResourceCache(client, opts.Logger)

	// Collect related objects before running rules so rules hit a warm cache.
	collector := &Collector{Client: client, Cache: cache, Logger: opts.Logger}
	if err := collector.Prefetch(ctx, target, opts.IncludeEvents, opts.IncludeRelated); err != nil {
		// Prefetch errors are not fatal — rules can still run with whatever
		// landed in the cache. But we note them on the report.
		opts.Logger.V(1).Info("prefetch errors; continuing with partial data", "err", err)
	}

	selected := selectRules(target.Kind, opts.EnabledRules, opts.DisabledRules)
	opts.Logger.V(1).Info("selected rules", "count", len(selected), "target", target.String())

	rc := &rules.Context{
		Target: target,
		Cache:  cache,
		Logger: opts.Logger,
		Now:    opts.Now,
	}

	all := make([]findings.Finding, 0, 16)
	notes := make([]string, 0)
	for _, r := range selected {
		meta := r.Meta()
		log := opts.Logger.WithValues("rule", meta.ID)
		out, err := r.Evaluate(ctx, rc)
		if err != nil {
			// Per-rule errors are logged and reported as notes, but never
			// fail the whole run.
			log.V(1).Info("rule error", "err", err)
			notes = append(notes, fmt.Sprintf("rule %s errored: %v", meta.ID, err))
			continue
		}
		for i := range out {
			stampFinding(&out[i], meta, target, opts.Now())
		}
		all = append(all, out...)
	}

	filtered := filter(all, opts.SeverityMin, opts.ConfidenceMin)
	ranked := Rank(filtered)
	deduped := dedupe(ranked)
	if len(deduped) > opts.MaxFindings {
		deduped = deduped[:opts.MaxFindings]
	}

	end := opts.Now()
	return findings.Report{
		Target:      target,
		GeneratedAt: end,
		DurationMs:  end.Sub(started).Milliseconds(),
		Findings:    deduped,
		Notes:       notes,
	}, nil
}

// selectRules picks rules whose scope includes the target kind and respects
// the enabled/disabled lists.
func selectRules(kind findings.TargetKind, enabled, disabled []string) []rules.Rule {
	all := rules.All()
	skip := make(map[string]struct{}, len(disabled))
	for _, id := range disabled {
		skip[id] = struct{}{}
	}
	var allow map[string]struct{}
	if len(enabled) > 0 {
		allow = make(map[string]struct{}, len(enabled))
		for _, id := range enabled {
			allow[id] = struct{}{}
		}
	}

	out := make([]rules.Rule, 0, len(all))
	for _, r := range all {
		m := r.Meta()
		if !m.AppliesTo(kind) {
			continue
		}
		if _, denied := skip[m.ID]; denied {
			continue
		}
		if allow != nil {
			if _, ok := allow[m.ID]; !ok {
				continue
			}
		}
		out = append(out, r)
	}
	// Stable ordering: by Priority descending, then by ID.
	sort.SliceStable(out, func(i, j int) bool {
		mi, mj := out[i].Meta(), out[j].Meta()
		if mi.Priority != mj.Priority {
			return mi.Priority > mj.Priority
		}
		return mi.ID < mj.ID
	})
	return out
}

// stampFinding fills in fields that rules commonly leave empty: Target,
// CreatedAt, Scope, ID, RuleID. Rules can override any of these by setting
// them explicitly before returning.
func stampFinding(f *findings.Finding, meta findings.RuleMeta, t findings.Target, now time.Time) {
	if f.RuleID == "" {
		f.RuleID = meta.ID
	}
	if f.ID == "" {
		f.ID = meta.ID
	}
	if f.Category == "" {
		f.Category = meta.Category
	}
	if f.Severity == "" {
		f.Severity = meta.Severity
	}
	if f.Scope == "" {
		f.Scope = t.Kind
	}
	// A finding's Target defaults to the engine's current target, but rules
	// may override to point at an owned object (e.g. a specific pod inside
	// a deployment diagnosis).
	if f.Target == (findings.Target{}) {
		f.Target = t
	}
	if f.CreatedAt.IsZero() {
		f.CreatedAt = now
	}
}

func filter(in []findings.Finding, minSev findings.Severity, minConf findings.Confidence) []findings.Finding {
	out := in[:0]
	for _, f := range in {
		if !f.Severity.AtLeast(minSev) {
			continue
		}
		if !f.Confidence.AtLeast(minConf) {
			continue
		}
		out = append(out, f)
	}
	return out
}

func dedupe(in []findings.Finding) []findings.Finding {
	// Collapse findings with identical (RuleID, Target); merge evidence.
	seen := map[string]int{} // key → index in out
	out := make([]findings.Finding, 0, len(in))
	for _, f := range in {
		key := f.RuleID + "|" + f.Target.String()
		if idx, ok := seen[key]; ok {
			out[idx].Evidence = append(out[idx].Evidence, f.Evidence...)
			continue
		}
		seen[key] = len(out)
		out = append(out, f)
	}
	return out
}

package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"

	"github.com/khvedela/kubediag/internal/findings"
)

// TextRenderer renders a Report to an ANSI terminal.
type TextRenderer struct {
	opts RenderOptions
}

// NewTextRenderer builds a text renderer.
func NewTextRenderer(opts RenderOptions) *TextRenderer {
	return &TextRenderer{opts: opts}
}

// Render writes the report to w.
func (t *TextRenderer) Render(w io.Writer, r findings.Report) error {
	c := newColorizer(t.opts.Color)

	// Header
	fmt.Fprintf(w, "%s %s\n", c.bold("▶"), c.bold(targetHeader(r.Target)))
	fmt.Fprintf(w, "  generated: %s  (%dms)\n",
		r.GeneratedAt.Format("15:04:05 MST"),
		r.DurationMs)

	// Overall status
	if len(r.Findings) == 0 {
		fmt.Fprintf(w, "\n  %s no findings.\n", c.green("✓"))
		return nil
	}
	fmt.Fprintf(w, "  overall: %s\n", c.severity(r.HighestSeverity()))

	// Counts by severity
	counts := countBySeverity(r.Findings)
	fmt.Fprintf(w, "  findings: %d", len(r.Findings))
	if s := summarizeCounts(counts, c); s != "" {
		fmt.Fprintf(w, " (%s)", s)
	}
	fmt.Fprintln(w)

	max := t.opts.MaxFindings
	if max <= 0 || max > len(r.Findings) {
		max = len(r.Findings)
	}

	for i := 0; i < max; i++ {
		t.renderFinding(w, c, &r.Findings[i])
	}
	if max < len(r.Findings) {
		fmt.Fprintf(w, "\n  ... %d more finding(s) omitted (use --max-findings to see more)\n", len(r.Findings)-max)
	}
	if len(r.Notes) > 0 {
		fmt.Fprintf(w, "\n%s\n", c.dim("notes:"))
		for _, n := range r.Notes {
			fmt.Fprintf(w, "  %s %s\n", c.dim("·"), c.dim(n))
		}
	}
	return nil
}

func (t *TextRenderer) renderFinding(w io.Writer, c *colorizer, f *findings.Finding) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s  %s  %s\n",
		c.severityBadge(f.Severity),
		c.confidence(f.Confidence),
		c.bold(f.RuleID),
	)
	fmt.Fprintf(w, "   %s\n", c.bold(f.Title))
	if f.Summary != "" {
		for _, line := range wrap(f.Summary, 88) {
			fmt.Fprintf(w, "   %s\n", line)
		}
	}
	if len(f.Evidence) > 0 {
		fmt.Fprintf(w, "\n   %s\n", c.dim("Evidence:"))
		for _, e := range f.Evidence {
			fmt.Fprintf(w, "     %s %s\n", c.dim("•"), renderEvidence(e))
		}
	}
	if r := f.Remediation; len(r.NextCommands) > 0 {
		fmt.Fprintf(w, "\n   %s\n", c.dim("Next commands:"))
		for _, cmd := range r.NextCommands {
			fmt.Fprintf(w, "     %s %s\n", c.dim("$"), c.cyan(cmd))
		}
	}
	if fix := strings.TrimSpace(f.Remediation.SuggestedFix); fix != "" {
		fmt.Fprintf(w, "\n   %s\n", c.dim("Suggested fix:"))
		for _, line := range wrap(fix, 88) {
			fmt.Fprintf(w, "     %s\n", line)
		}
	}
	if len(f.Related) > 0 {
		fmt.Fprintf(w, "\n   %s\n", c.dim("Related:"))
		for _, rr := range f.Related {
			fmt.Fprintf(w, "     %s %s\n", c.dim("↳"), rr.String())
		}
	}
	if len(f.Remediation.DocsLinks) > 0 {
		fmt.Fprintf(w, "\n   %s %s\n", c.dim("Docs:"), c.dim(strings.Join(f.Remediation.DocsLinks, "  ")))
	}
}

func renderEvidence(e findings.Evidence) string {
	switch e.Kind {
	case findings.EvidenceKindEvent:
		return fmt.Sprintf("Event: %s", e.Value)
	case findings.EvidenceKindLog:
		return fmt.Sprintf("Log: %s", e.Value)
	case findings.EvidenceKindField:
		if e.Source != "" {
			return fmt.Sprintf("%s = %s", e.Source, e.Value)
		}
		return e.Value
	case findings.EvidenceKindComputed:
		return e.Value
	}
	return e.Value
}

func targetHeader(t findings.Target) string {
	switch t.Kind {
	case findings.TargetKindCluster:
		return "Cluster"
	case findings.TargetKindNamespace:
		return fmt.Sprintf("Namespace %s", t.Name)
	case findings.TargetKindDeployment:
		return fmt.Sprintf("Deployment %s/%s", t.Namespace, t.Name)
	case findings.TargetKindPod:
		return fmt.Sprintf("Pod %s/%s", t.Namespace, t.Name)
	}
	return t.String()
}

func countBySeverity(in []findings.Finding) map[findings.Severity]int {
	out := map[findings.Severity]int{}
	for _, f := range in {
		out[f.Severity]++
	}
	return out
}

func summarizeCounts(counts map[findings.Severity]int, c *colorizer) string {
	order := []findings.Severity{
		findings.SeverityCritical,
		findings.SeverityHigh,
		findings.SeverityMedium,
		findings.SeverityLow,
		findings.SeverityInfo,
	}
	parts := make([]string, 0, len(order))
	for _, s := range order {
		if n := counts[s]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, c.severityShort(s)))
		}
	}
	return strings.Join(parts, ", ")
}

// wrap breaks a long string at word boundaries to fit width.
func wrap(s string, width int) []string {
	words := strings.Fields(strings.ReplaceAll(s, "\n", " "))
	var lines []string
	var cur string
	for _, w := range words {
		if cur == "" {
			cur = w
			continue
		}
		if len(cur)+1+len(w) > width {
			lines = append(lines, cur)
			cur = w
		} else {
			cur += " " + w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// -----------------------------------------------------------------------------
// Colorizer
// -----------------------------------------------------------------------------

type colorizer struct{ enabled bool }

func newColorizer(enabled bool) *colorizer {
	if !enabled {
		color.NoColor = true
	}
	return &colorizer{enabled: enabled}
}

func (c *colorizer) bold(s string) string  { return color.New(color.Bold).Sprint(s) }
func (c *colorizer) dim(s string) string   { return color.New(color.Faint).Sprint(s) }
func (c *colorizer) green(s string) string { return color.New(color.FgGreen).Sprint(s) }
func (c *colorizer) cyan(s string) string  { return color.New(color.FgCyan).Sprint(s) }

func (c *colorizer) severity(s findings.Severity) string {
	switch s {
	case findings.SeverityCritical:
		return color.New(color.FgHiRed, color.Bold).Sprint("CRITICAL")
	case findings.SeverityHigh:
		return color.New(color.FgRed, color.Bold).Sprint("HIGH")
	case findings.SeverityMedium:
		return color.New(color.FgYellow, color.Bold).Sprint("MEDIUM")
	case findings.SeverityLow:
		return color.New(color.FgBlue).Sprint("LOW")
	case findings.SeverityInfo:
		return color.New(color.FgCyan).Sprint("INFO")
	}
	return string(s)
}

func (c *colorizer) severityBadge(s findings.Severity) string {
	switch s {
	case findings.SeverityCritical:
		return color.New(color.FgHiRed, color.Bold).Sprint("ⓧ CRITICAL")
	case findings.SeverityHigh:
		return color.New(color.FgRed, color.Bold).Sprint("● HIGH    ")
	case findings.SeverityMedium:
		return color.New(color.FgYellow, color.Bold).Sprint("● MEDIUM  ")
	case findings.SeverityLow:
		return color.New(color.FgBlue).Sprint("● LOW     ")
	case findings.SeverityInfo:
		return color.New(color.FgCyan).Sprint("ⓘ INFO    ")
	}
	return string(s)
}

func (c *colorizer) severityShort(s findings.Severity) string {
	switch s {
	case findings.SeverityCritical:
		return color.New(color.FgHiRed).Sprint("critical")
	case findings.SeverityHigh:
		return color.New(color.FgRed).Sprint("high")
	case findings.SeverityMedium:
		return color.New(color.FgYellow).Sprint("medium")
	case findings.SeverityLow:
		return color.New(color.FgBlue).Sprint("low")
	case findings.SeverityInfo:
		return color.New(color.FgCyan).Sprint("info")
	}
	return string(s)
}

func (c *colorizer) confidence(conf findings.Confidence) string {
	label := fmt.Sprintf("[%s confidence]", conf)
	switch conf {
	case findings.ConfidenceHigh:
		return color.New(color.Faint, color.FgWhite).Sprint(label)
	case findings.ConfidenceMedium:
		return color.New(color.Faint).Sprint(label)
	case findings.ConfidenceLow:
		return color.New(color.Faint, color.Italic).Sprint(label)
	}
	return label
}

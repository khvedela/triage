package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/OWNER/triage/internal/findings"
)

// MarkdownRenderer emits an embeddable markdown report. Useful for pasting
// into incident tickets or saving via `triage report namespace <ns>`.
type MarkdownRenderer struct {
	opts RenderOptions
}

// NewMarkdownRenderer builds a markdown renderer.
func NewMarkdownRenderer(opts RenderOptions) *MarkdownRenderer { return &MarkdownRenderer{opts: opts} }

// Render writes a markdown document to w.
func (m *MarkdownRenderer) Render(w io.Writer, r findings.Report) error {
	fmt.Fprintf(w, "# Triage report — %s\n\n", targetHeader(r.Target))
	fmt.Fprintf(w, "_Generated at %s (%dms)_\n\n", r.GeneratedAt.Format("2006-01-02 15:04:05 MST"), r.DurationMs)

	if len(r.Findings) == 0 {
		fmt.Fprintln(w, "✅ No findings.")
		return nil
	}
	fmt.Fprintf(w, "**Overall severity:** `%s`  \n", r.HighestSeverity())
	fmt.Fprintf(w, "**Findings:** %d\n\n", len(r.Findings))

	fmt.Fprintln(w, "| # | Severity | Confidence | Rule | Title |")
	fmt.Fprintln(w, "|---|----------|-----------|------|-------|")
	for i, f := range r.Findings {
		fmt.Fprintf(w, "| %d | %s | %s | `%s` | %s |\n",
			i+1, f.Severity, f.Confidence, f.RuleID, escapeMD(f.Title))
	}
	fmt.Fprintln(w)

	for i := range r.Findings {
		m.renderFinding(w, i+1, &r.Findings[i])
	}
	if len(r.Notes) > 0 {
		fmt.Fprintln(w, "\n## Notes")
		for _, n := range r.Notes {
			fmt.Fprintf(w, "- %s\n", n)
		}
	}
	return nil
}

func (m *MarkdownRenderer) renderFinding(w io.Writer, idx int, f *findings.Finding) {
	fmt.Fprintf(w, "## %d. %s\n\n", idx, escapeMD(f.Title))
	fmt.Fprintf(w, "- **Rule:** `%s`\n", f.RuleID)
	fmt.Fprintf(w, "- **Severity:** `%s`\n", f.Severity)
	fmt.Fprintf(w, "- **Confidence:** `%s`\n", f.Confidence)
	fmt.Fprintf(w, "- **Category:** `%s`\n", f.Category)
	fmt.Fprintf(w, "- **Target:** `%s`\n\n", f.Target.String())

	if f.Summary != "" {
		fmt.Fprintln(w, f.Summary)
		fmt.Fprintln(w)
	}
	if len(f.Evidence) > 0 {
		fmt.Fprintln(w, "### Evidence")
		for _, e := range f.Evidence {
			fmt.Fprintf(w, "- **%s** %s\n", e.Kind, escapeMD(renderEvidence(e)))
		}
		fmt.Fprintln(w)
	}
	if cmds := f.Remediation.NextCommands; len(cmds) > 0 {
		fmt.Fprintln(w, "### Next commands")
		fmt.Fprintln(w, "```sh")
		for _, c := range cmds {
			fmt.Fprintln(w, c)
		}
		fmt.Fprintln(w, "```")
		fmt.Fprintln(w)
	}
	if fix := strings.TrimSpace(f.Remediation.SuggestedFix); fix != "" {
		fmt.Fprintln(w, "### Suggested fix")
		fmt.Fprintln(w, fix)
		fmt.Fprintln(w)
	}
	if len(f.Related) > 0 {
		fmt.Fprintln(w, "### Related")
		for _, r := range f.Related {
			fmt.Fprintf(w, "- `%s`\n", r.String())
		}
		fmt.Fprintln(w)
	}
	if links := f.Remediation.DocsLinks; len(links) > 0 {
		fmt.Fprintln(w, "### Docs")
		for _, l := range links {
			fmt.Fprintf(w, "- %s\n", l)
		}
		fmt.Fprintln(w)
	}
}

func escapeMD(s string) string {
	return strings.NewReplacer("|", "\\|", "`", "'").Replace(s)
}

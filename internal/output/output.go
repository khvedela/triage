// Package output renders Reports for humans and machines.
//
// Three formats are supported, each as a plain struct (not an interface —
// we'll add the interface when a fourth format materializes):
//   - text: ANSI-colored terminal output, the default
//   - json: machine-readable JSON
//   - markdown: embeddable markdown report
package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/OWNER/triage/internal/findings"
)

// Format enumerates supported output formats.
type Format string

const (
	FormatText     Format = "text"
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

// ParseFormat validates and returns the Format for a string, error on unknown.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "text", "":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	case "markdown", "md":
		return FormatMarkdown, nil
	}
	return "", fmt.Errorf("unknown output format %q (want one of: text, json, markdown)", s)
}

// RenderOptions controls renderer behavior.
type RenderOptions struct {
	Color       bool
	Verbose     bool
	MaxFindings int
}

// Render dispatches to the renderer for f.
func Render(w io.Writer, r findings.Report, f Format, opts RenderOptions) error {
	switch f {
	case FormatText:
		return NewTextRenderer(opts).Render(w, r)
	case FormatJSON:
		return NewJSONRenderer(opts).Render(w, r)
	case FormatMarkdown:
		return NewMarkdownRenderer(opts).Render(w, r)
	}
	return fmt.Errorf("output.Render: unknown format %q", f)
}

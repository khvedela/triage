package output

import (
	"encoding/json"
	"io"

	"github.com/OWNER/triage/internal/findings"
)

// JSONRenderer emits the report as JSON. Output schema is part of the public
// contract: machine consumers parse this; changes require a minor bump.
type JSONRenderer struct {
	opts RenderOptions
}

// NewJSONRenderer builds a JSON renderer.
func NewJSONRenderer(opts RenderOptions) *JSONRenderer { return &JSONRenderer{opts: opts} }

// Render writes JSON-encoded report to w.
func (j *JSONRenderer) Render(w io.Writer, r findings.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

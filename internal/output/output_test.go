package output_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/khvedela/triage/internal/findings"
	"github.com/khvedela/triage/internal/output"
)

var update = flag.Bool("update", false, "regenerate golden files")

// fixedReport returns a deterministic Report for golden testing.
func fixedReport() findings.Report {
	target := findings.Target{Kind: findings.TargetKindPod, Namespace: "default", Name: "crashloop-demo"}
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	return findings.Report{
		Target:      target,
		GeneratedAt: ts,
		DurationMs:  42,
		Findings: []findings.Finding{
			{
				ID:         "TRG-POD-CRASHLOOPBACKOFF",
				RuleID:     "TRG-POD-CRASHLOOPBACKOFF",
				Title:      `Container "app" is in CrashLoopBackOff (5 restarts)`,
				Summary:    `Container "app" has crashed 5 times. The kubelet is backing off restarts.`,
				Category:   findings.CategoryRuntime,
				Severity:   findings.SeverityCritical,
				Confidence: findings.ConfidenceHigh,
				Target:     target,
				Evidence: []findings.Evidence{
					{Kind: findings.EvidenceKindField, Source: "pod.status.containerStatuses[app].state.waiting.reason", Value: "CrashLoopBackOff"},
					{Kind: findings.EvidenceKindField, Source: "pod.status.containerStatuses[app].restartCount", Value: "5"},
				},
				Remediation: findings.Remediation{
					Explanation: "Check the container logs for the exit reason.",
					NextCommands: []string{
						"kubectl logs -n default crashloop-demo -c app --previous",
						"kubectl describe pod -n default crashloop-demo",
					},
				},
			},
			{
				ID:         "TRG-POD-OOMKILLED",
				RuleID:     "TRG-POD-OOMKILLED",
				Title:      `Container "app" was OOMKilled`,
				Summary:    `Container "app" was killed by the kernel OOM killer.`,
				Category:   findings.CategoryResourcePressure,
				Severity:   findings.SeverityHigh,
				Confidence: findings.ConfidenceHigh,
				Target:     target,
				Evidence: []findings.Evidence{
					{Kind: findings.EvidenceKindField, Source: "pod.status.containerStatuses[app].lastState.terminated.reason", Value: "OOMKilled"},
				},
				Remediation: findings.Remediation{
					Explanation:  "The container exceeded its memory limit.",
					NextCommands: []string{"kubectl top pod -n default crashloop-demo --containers"},
					SuggestedFix: "Increase resources.limits.memory.",
				},
			},
		},
	}
}

func goldenPath(name string) string {
	return filepath.Join("testdata", name)
}

func runGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := goldenPath(name)
	if *update {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(path, got, 0644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file %s (run with -update to create): %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("golden mismatch for %s:\ngot:\n%s\nwant:\n%s", name, got, want)
	}
}

func TestJSONRenderer_ValidJSON(t *testing.T) {
	r := fixedReport()
	var buf bytes.Buffer
	err := output.NewJSONRenderer(output.RenderOptions{}).Render(&buf, r)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !json.Valid(buf.Bytes()) {
		t.Errorf("output is not valid JSON:\n%s", buf.String())
	}
}

func TestJSONRenderer_ContainsFindingIDs(t *testing.T) {
	r := fixedReport()
	var buf bytes.Buffer
	_ = output.NewJSONRenderer(output.RenderOptions{}).Render(&buf, r)
	got := buf.String()
	for _, id := range []string{"TRG-POD-CRASHLOOPBACKOFF", "TRG-POD-OOMKILLED"} {
		if !strings.Contains(got, id) {
			t.Errorf("JSON output missing rule ID %q", id)
		}
	}
}

func TestJSONRenderer_Golden(t *testing.T) {
	r := fixedReport()
	var buf bytes.Buffer
	_ = output.NewJSONRenderer(output.RenderOptions{}).Render(&buf, r)
	runGolden(t, "report.json", buf.Bytes())
}

func TestMarkdownRenderer_ContainsFindingIDs(t *testing.T) {
	r := fixedReport()
	var buf bytes.Buffer
	err := output.NewMarkdownRenderer(output.RenderOptions{}).Render(&buf, r)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	got := buf.String()
	for _, id := range []string{"TRG-POD-CRASHLOOPBACKOFF", "TRG-POD-OOMKILLED"} {
		if !strings.Contains(got, id) {
			t.Errorf("markdown output missing rule ID %q", id)
		}
	}
}

func TestMarkdownRenderer_Golden(t *testing.T) {
	r := fixedReport()
	var buf bytes.Buffer
	_ = output.NewMarkdownRenderer(output.RenderOptions{}).Render(&buf, r)
	runGolden(t, "report.md", buf.Bytes())
}

func TestTextRenderer_ContainsFindingIDs(t *testing.T) {
	r := fixedReport()
	var buf bytes.Buffer
	err := output.NewTextRenderer(output.RenderOptions{Color: false}).Render(&buf, r)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	got := buf.String()
	for _, id := range []string{"TRG-POD-CRASHLOOPBACKOFF", "TRG-POD-OOMKILLED"} {
		if !strings.Contains(got, id) {
			t.Errorf("text output missing rule ID %q", id)
		}
	}
}

func TestTextRenderer_Golden(t *testing.T) {
	r := fixedReport()
	var buf bytes.Buffer
	_ = output.NewTextRenderer(output.RenderOptions{Color: false}).Render(&buf, r)
	runGolden(t, "report.txt", buf.Bytes())
}

func TestTextRenderer_NoFindings(t *testing.T) {
	r := findings.Report{
		Target:      findings.Target{Kind: findings.TargetKindPod, Namespace: "default", Name: "ok-pod"},
		GeneratedAt: time.Now(),
		Findings:    nil,
	}
	var buf bytes.Buffer
	err := output.NewTextRenderer(output.RenderOptions{Color: false}).Render(&buf, r)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !strings.Contains(buf.String(), "no findings") {
		t.Errorf("expected 'no findings' in output:\n%s", buf.String())
	}
}

func TestParseFormat(t *testing.T) {
	cases := []struct {
		in   string
		want output.Format
		err  bool
	}{
		{"text", output.FormatText, false},
		{"json", output.FormatJSON, false},
		{"markdown", output.FormatMarkdown, false},
		{"md", output.FormatMarkdown, false},
		{"", output.FormatText, false},
		{"XML", "", true},
	}
	for _, tc := range cases {
		got, err := output.ParseFormat(tc.in)
		if tc.err && err == nil {
			t.Errorf("ParseFormat(%q): expected error", tc.in)
		}
		if !tc.err && err != nil {
			t.Errorf("ParseFormat(%q): unexpected error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("ParseFormat(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

package engine_test

import (
	"fmt"
	"testing"

	"github.com/khvedela/kubediag/internal/engine"
	"github.com/khvedela/kubediag/internal/findings"
)

func makeF(ruleID string, sev findings.Severity, conf findings.Confidence) findings.Finding {
	return findings.Finding{
		RuleID:     ruleID,
		ID:         ruleID,
		Severity:   sev,
		Confidence: conf,
	}
}

func TestRank_OrderBySeverityThenConfidence(t *testing.T) {
	in := []findings.Finding{
		makeF("LOW-LOW", findings.SeverityLow, findings.ConfidenceLow),
		makeF("CRIT-HIGH", findings.SeverityCritical, findings.ConfidenceHigh),
		makeF("HIGH-MED", findings.SeverityHigh, findings.ConfidenceMedium),
		makeF("CRIT-MED", findings.SeverityCritical, findings.ConfidenceMedium),
		makeF("INFO-HIGH", findings.SeverityInfo, findings.ConfidenceHigh),
	}

	out := engine.Rank(in)

	want := []string{"CRIT-HIGH", "CRIT-MED", "HIGH-MED", "LOW-LOW", "INFO-HIGH"}
	for i, w := range want {
		if out[i].RuleID != w {
			t.Errorf("position %d: want %s, got %s", i, w, out[i].RuleID)
		}
	}
}

func TestRank_TiesBrokenByRuleID(t *testing.T) {
	in := []findings.Finding{
		makeF("Z-RULE", findings.SeverityHigh, findings.ConfidenceHigh),
		makeF("A-RULE", findings.SeverityHigh, findings.ConfidenceHigh),
		makeF("M-RULE", findings.SeverityHigh, findings.ConfidenceHigh),
	}
	out := engine.Rank(in)
	if out[0].RuleID != "A-RULE" || out[1].RuleID != "M-RULE" || out[2].RuleID != "Z-RULE" {
		t.Errorf("tie-breaking by rule ID failed: %v", []string{out[0].RuleID, out[1].RuleID, out[2].RuleID})
	}
}

func TestRank_EmptyInput(t *testing.T) {
	out := engine.Rank(nil)
	if len(out) != 0 {
		t.Errorf("expected empty slice, got %d", len(out))
	}
}

func TestRank_DoesNotMutateInput(t *testing.T) {
	in := []findings.Finding{
		makeF("B", findings.SeverityHigh, findings.ConfidenceHigh),
		makeF("A", findings.SeverityCritical, findings.ConfidenceHigh),
	}
	originalFirst := in[0].RuleID
	engine.Rank(in)
	if in[0].RuleID != originalFirst {
		t.Error("Rank mutated the input slice")
	}
}

func TestRank_StableForLargeInput(t *testing.T) {
	severities := []findings.Severity{
		findings.SeverityCritical, findings.SeverityHigh,
		findings.SeverityMedium, findings.SeverityLow, findings.SeverityInfo,
	}
	confidences := []findings.Confidence{
		findings.ConfidenceHigh, findings.ConfidenceMedium, findings.ConfidenceLow,
	}

	var in []findings.Finding
	for i, sev := range severities {
		for j, conf := range confidences {
			in = append(in, makeF(
				fmt.Sprintf("RULE-%d-%d", i, j),
				sev, conf,
			))
		}
	}

	out1 := engine.Rank(in)
	out2 := engine.Rank(in)
	for i := range out1 {
		if out1[i].RuleID != out2[i].RuleID {
			t.Errorf("rank is not stable at position %d: %s vs %s", i, out1[i].RuleID, out2[i].RuleID)
		}
	}
}

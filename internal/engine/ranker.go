package engine

import (
	"sort"

	"github.com/khvedela/kubediag/internal/findings"
)

// Rank sorts findings from most to least important and returns a new slice.
// Score formula: severity*1000 + confidence*100 + (100 - rulePriorityClamp).
// Ties are broken by rule ID for stable output.
func Rank(in []findings.Finding) []findings.Finding {
	out := make([]findings.Finding, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		si, sj := score(out[i]), score(out[j])
		if si != sj {
			return si > sj
		}
		return out[i].RuleID < out[j].RuleID
	})
	return out
}

func score(f findings.Finding) int {
	return f.Severity.Weight()*1000 + f.Confidence.Weight()*100
}

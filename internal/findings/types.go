// Package findings defines the domain model for triage: Targets, Rules,
// Findings, Evidence, and Remediation.
//
// These types are the contract between the engine, rules, and output
// renderers. They are deliberately in `internal/` for v1 — we promote to
// `pkg/` only after the schema has settled in real-world use.
package findings

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// -----------------------------------------------------------------------------
// Target: what we are diagnosing
// -----------------------------------------------------------------------------

// TargetKind enumerates the scopes kubediag can diagnose.
type TargetKind string

const (
	TargetKindPod        TargetKind = "Pod"
	TargetKindDeployment TargetKind = "Deployment"
	TargetKindNamespace  TargetKind = "Namespace"
	TargetKindCluster    TargetKind = "Cluster"
)

// Target identifies one subject of diagnosis.
type Target struct {
	Kind      TargetKind `json:"kind"`
	Namespace string     `json:"namespace,omitempty"`
	Name      string     `json:"name,omitempty"`
}

// String returns a compact human-readable form.
func (t Target) String() string {
	switch t.Kind {
	case TargetKindCluster:
		return "cluster"
	case TargetKindNamespace:
		return "namespace/" + t.Name
	default:
		if t.Namespace != "" {
			return fmt.Sprintf("%s/%s/%s", strings.ToLower(string(t.Kind)), t.Namespace, t.Name)
		}
		return fmt.Sprintf("%s/%s", strings.ToLower(string(t.Kind)), t.Name)
	}
}

// -----------------------------------------------------------------------------
// Severity & Confidence
// -----------------------------------------------------------------------------

// Severity classifies how bad a finding is.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Weight returns the ranking weight for this severity (higher = worse).
func (s Severity) Weight() int {
	switch s {
	case SeverityCritical:
		return 5
	case SeverityHigh:
		return 4
	case SeverityMedium:
		return 3
	case SeverityLow:
		return 2
	case SeverityInfo:
		return 1
	}
	return 0
}

// AtLeast reports whether s is at least as severe as other.
func (s Severity) AtLeast(other Severity) bool { return s.Weight() >= other.Weight() }

// ParseSeverity returns the Severity for a string, or an error if unrecognized.
func ParseSeverity(s string) (Severity, error) {
	switch strings.ToLower(s) {
	case "critical":
		return SeverityCritical, nil
	case "high":
		return SeverityHigh, nil
	case "medium":
		return SeverityMedium, nil
	case "low":
		return SeverityLow, nil
	case "info":
		return SeverityInfo, nil
	}
	return "", fmt.Errorf("unknown severity %q (want one of: critical, high, medium, low, info)", s)
}

// Confidence captures how sure a rule is that its finding is correct.
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// Weight returns the ranking weight for this confidence.
func (c Confidence) Weight() int {
	switch c {
	case ConfidenceHigh:
		return 3
	case ConfidenceMedium:
		return 2
	case ConfidenceLow:
		return 1
	}
	return 0
}

// AtLeast reports whether c is at least as confident as other.
func (c Confidence) AtLeast(other Confidence) bool { return c.Weight() >= other.Weight() }

// ParseConfidence returns the Confidence for a string, or an error.
func ParseConfidence(s string) (Confidence, error) {
	switch strings.ToLower(s) {
	case "high":
		return ConfidenceHigh, nil
	case "medium":
		return ConfidenceMedium, nil
	case "low":
		return ConfidenceLow, nil
	}
	return "", fmt.Errorf("unknown confidence %q (want one of: high, medium, low)", s)
}

// -----------------------------------------------------------------------------
// Category
// -----------------------------------------------------------------------------

// Category groups findings for human scanning and for filtering.
type Category string

const (
	CategoryScheduling        Category = "Scheduling"
	CategoryImage             Category = "Image"
	CategoryConfiguration     Category = "Configuration"
	CategoryProbes            Category = "Probes"
	CategoryNetworking        Category = "Networking"
	CategoryStorage           Category = "Storage"
	CategoryAccess            Category = "Access"
	CategoryRollout           Category = "Rollout"
	CategoryDNS               Category = "DNS"
	CategoryPolicy            Category = "Policy"
	CategoryResourcePressure  Category = "ResourcePressure"
	CategoryRuntime           Category = "Runtime"
	CategoryDependency        Category = "Dependency"
)

// -----------------------------------------------------------------------------
// Evidence
// -----------------------------------------------------------------------------

// EvidenceKind describes what kind of signal an Evidence item captures.
type EvidenceKind string

const (
	EvidenceKindField    EvidenceKind = "Field"    // a specific JSONPath-like field on a resource
	EvidenceKindEvent    EvidenceKind = "Event"    // a Kubernetes Event
	EvidenceKindLog      EvidenceKind = "Log"      // sampled log line(s) from a container
	EvidenceKindComputed EvidenceKind = "Computed" // a derived fact (e.g. "3 of 4 replicas unavailable")
)

// Evidence is a single supporting data point for a finding.
type Evidence struct {
	Kind    EvidenceKind      `json:"kind"`
	Source  string            `json:"source,omitempty"`
	Value   string            `json:"value,omitempty"`
	Context map[string]string `json:"context,omitempty"`
}

// -----------------------------------------------------------------------------
// Remediation
// -----------------------------------------------------------------------------

// Remediation tells the user what to do next.
type Remediation struct {
	Explanation  string   `json:"explanation,omitempty"`
	NextCommands []string `json:"nextCommands,omitempty"`
	SuggestedFix string   `json:"suggestedFix,omitempty"`
	DocsLinks    []string `json:"docsLinks,omitempty"`
}

// -----------------------------------------------------------------------------
// ResourceRef
// -----------------------------------------------------------------------------

// ResourceRef identifies a related Kubernetes object.
type ResourceRef struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name,omitempty"`
}

// String formats the ref like "v1/Pod/namespace/name".
func (r ResourceRef) String() string {
	if r.Namespace == "" {
		return fmt.Sprintf("%s/%s/%s", r.APIVersion, r.Kind, r.Name)
	}
	return fmt.Sprintf("%s/%s/%s/%s", r.APIVersion, r.Kind, r.Namespace, r.Name)
}

// -----------------------------------------------------------------------------
// Finding
// -----------------------------------------------------------------------------

// Finding is the output unit of the diagnosis engine: one root cause candidate
// with enough evidence and remediation to act on.
type Finding struct {
	ID          string        `json:"id"`
	RuleID      string        `json:"ruleId"`
	Title       string        `json:"title"`
	Summary     string        `json:"summary"`
	Category    Category      `json:"category"`
	Severity    Severity      `json:"severity"`
	Confidence  Confidence    `json:"confidence"`
	Scope       TargetKind    `json:"scope"`
	Target      Target        `json:"target"`
	Related     []ResourceRef `json:"related,omitempty"`
	Evidence    []Evidence    `json:"evidence,omitempty"`
	Remediation Remediation   `json:"remediation"`
	CreatedAt   time.Time     `json:"createdAt"`
}

// -----------------------------------------------------------------------------
// RuleMeta
// -----------------------------------------------------------------------------

// RuleMeta is the static metadata a rule publishes about itself.
type RuleMeta struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Category    Category     `json:"category"`
	Severity    Severity     `json:"severity"`
	Scopes      []TargetKind `json:"scopes"`
	Description string       `json:"description"`
	DocsLinks   []string     `json:"docsLinks,omitempty"`
	Priority    int          `json:"priority"`
}

// AppliesTo reports whether this rule is relevant for the given target kind.
func (m RuleMeta) AppliesTo(k TargetKind) bool {
	for _, s := range m.Scopes {
		if s == k {
			return true
		}
	}
	return false
}

// -----------------------------------------------------------------------------
// Report — the top-level serializable output
// -----------------------------------------------------------------------------

// Report is the full output of one kubediag run.
type Report struct {
	Target      Target    `json:"target"`
	GeneratedAt time.Time `json:"generatedAt"`
	DurationMs  int64     `json:"durationMs"`
	Findings    []Finding `json:"findings"`
	// Notes holds non-fatal informational messages (e.g., "skipped rule X
	// because RBAC denied access to Y"). These are separate from Findings
	// because they are about the *run*, not about the *cluster*.
	Notes []string `json:"notes,omitempty"`
}

// HighestSeverity returns the severity of the worst finding, or SeverityInfo
// when the report has no findings.
func (r Report) HighestSeverity() Severity {
	worst := SeverityInfo
	for _, f := range r.Findings {
		if f.Severity.Weight() > worst.Weight() {
			worst = f.Severity
		}
	}
	return worst
}

// MarshalJSON ensures stable field ordering by delegating to the struct layout.
func (r Report) MarshalJSON() ([]byte, error) {
	type alias Report
	return json.Marshal(alias(r))
}

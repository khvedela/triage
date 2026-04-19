// Package cli holds CLI-layer helpers shared across commands: exit codes,
// option plumbing, and color/TTY detection.
package cli

// Exit codes are a public contract for CI pipelines that shell out to triage.
const (
	// ExitOK — kubediag ran successfully and produced no findings at or above
	// the configured severity threshold.
	ExitOK = 0
	// ExitFindings — kubediag ran successfully and produced at least one
	// finding at or above the configured severity threshold.
	ExitFindings = 1
	// ExitUsage — the invocation itself was invalid (bad flags, missing args).
	ExitUsage = 2
	// ExitClusterError — unable to reach or authenticate with the cluster,
	// or a required API call failed in a way that prevents meaningful
	// diagnosis. RBAC denials do not raise this — they produce an
	// informational finding instead.
	ExitClusterError = 3
	// ExitInternal — an unexpected error in kubediag itself.
	ExitInternal = 10
)

package cli

import (
	"os"

	"github.com/mattn/go-isatty"
)

// ResolveColor decides whether ANSI color should be emitted, consolidating
// all the inputs that can disable it. This function is the single source of
// truth; no renderer should re-check flags/env directly.
//
// Precedence (later wins when in conflict):
//  1. TTY detection on stdout
//  2. NO_COLOR env var (https://no-color.org) → off
//  3. FORCE_COLOR env var → on
//  4. `color` config value: "auto" defers, "always" → on, "never" → off
//  5. `--no-color` flag → off
func ResolveColor(cfgMode string, noColorFlag bool) bool {
	if noColorFlag {
		return false
	}
	switch cfgMode {
	case "always":
		return true
	case "never":
		return false
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	if _, ok := os.LookupEnv("FORCE_COLOR"); ok {
		return true
	}
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

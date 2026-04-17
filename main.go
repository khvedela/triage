// Package main is the entry point for the triage CLI.
//
// triage is a kubectl-native diagnostic tool that surfaces probable root
// causes for broken Kubernetes workloads, with evidence and the exact next
// commands to run.
package main

import (
	"os"

	"github.com/OWNER/triage/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}

// Package cmd implements the triage command tree.
//
// Root wires global flags, loads configuration, sets up the logger, and
// attaches a resolved *cli.Options to each subcommand's context via
// PersistentPreRunE.
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/OWNER/triage/internal/cli"
	"github.com/OWNER/triage/internal/config"
	"github.com/OWNER/triage/internal/logging"
)

// Version metadata. Overridden via -ldflags at build time.
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

// Execute runs the root command. Returns the process exit code; never calls os.Exit.
func Execute() int {
	root := NewRootCmd()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	root.SetContext(context.WithValue(ctx, rootStateKey{}, &rootState{}))
	if err := root.ExecuteContext(root.Context()); err != nil {
		return cli.ExitUsage
	}
	return readExitCode(root)
}

// rootState carries the chosen exit code from subcommands up to Execute.
type rootState struct{ exitCode int }
type rootStateKey struct{}

// setExitCode records the subcommand's exit code on the command tree.
func setExitCode(cmd *cobra.Command, code int) {
	st, ok := cmd.Root().Context().Value(rootStateKey{}).(*rootState)
	if ok && st != nil {
		st.exitCode = code
	}
}

func readExitCode(root *cobra.Command) int {
	if root.Context() == nil {
		return cli.ExitOK
	}
	st, ok := root.Context().Value(rootStateKey{}).(*rootState)
	if !ok || st == nil {
		return cli.ExitOK
	}
	return st.exitCode
}

// NewRootCmd builds the root command. Exposed for tests.
func NewRootCmd() *cobra.Command {
	v := viper.New()
	kubeFlags := genericclioptions.NewConfigFlags(true)

	// rootFlags holds the pflag destination variables.
	var (
		output         string
		verbose        bool
		debug          bool
		noColor        bool
		maxFindings    int
		severityMin    string
		confidenceMin  string
		includeEvents  bool
		includeRelated bool
		timeout        time.Duration
		configPath     string
	)

	root := &cobra.Command{
		Use:           rootUse(),
		Short:         "Diagnose broken Kubernetes workloads",
		Long:          rootLongDescription(),
		SilenceUsage:  true,
		SilenceErrors: false,
		Version:       fmt.Sprintf("%s (commit %s, built %s)", version, commit, buildDate),
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(v, config.LoadOptions{ConfigPath: configPath})
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			log := logging.New(logging.Options{
				Debug:   debug,
				Verbose: verbose,
				NoColor: noColor,
			})

			ns := derefStr(kubeFlags.Namespace)
			opts := &cli.Options{
				Config:         cfg,
				KubeFlags:      kubeFlags,
				Logger:         log,
				Color:          cli.ResolveColor(cfg.Color, noColor),
				Namespace:      ns,
				Output:         cfg.Output, // includes --output flag via viper binding
				Verbose:        verbose,
				Debug:          debug,
				NoColor:        noColor,
				Timeout:        cfg.Timeout,
				MaxFindings:    cfg.MaxFindings,
				SeverityMin:    cfg.SeverityMin,
				ConfMin:        cfg.ConfidenceMin,
				IncludeEvents:  cfg.IncludeEvents,
				IncludeRelated: cfg.IncludeRelated,
				Deadline:       time.Now().Add(cfg.Timeout),
			}
			cmd.SetContext(cli.WithOptions(cmd.Context(), opts))
			return nil
		},
	}

	// Kube/kubectl-style flags (--kubeconfig, --context, --namespace/-n, etc.)
	kubeFlags.AddFlags(root.PersistentFlags())

	// triage-specific persistent flags — also bound into viper immediately so
	// flag values take precedence over config-file values.
	pf := root.PersistentFlags()

	pf.StringVarP(&output, "output", "o", "text", "output format: text, json, markdown")
	_ = v.BindPFlag("output", pf.Lookup("output"))

	pf.BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	pf.BoolVar(&debug, "debug", false, "enable debug logs on stderr")

	pf.BoolVar(&noColor, "no-color", false, "disable ANSI color output")

	pf.IntVar(&maxFindings, "max-findings", 20, "cap rendered findings")
	_ = v.BindPFlag("maxFindings", pf.Lookup("max-findings"))

	pf.StringVar(&severityMin, "severity-min", "info", "minimum severity (critical|high|medium|low|info)")
	_ = v.BindPFlag("severityMin", pf.Lookup("severity-min"))

	pf.StringVar(&confidenceMin, "confidence-min", "low", "minimum confidence (high|medium|low)")
	_ = v.BindPFlag("confidenceMin", pf.Lookup("confidence-min"))

	pf.BoolVar(&includeEvents, "include-events", true, "include related Events in evidence")
	_ = v.BindPFlag("includeEvents", pf.Lookup("include-events"))

	pf.BoolVar(&includeRelated, "include-related", true, "include related Services/PVCs in evidence")
	_ = v.BindPFlag("includeRelated", pf.Lookup("include-related"))

	pf.DurationVar(&timeout, "timeout", 15*time.Second, "overall cluster-call timeout")
	_ = v.BindPFlag("timeout", pf.Lookup("timeout"))

	pf.StringVar(&configPath, "config", "", "config file (default ~/.config/triage/config.yaml)")

	// Hide low-signal kubectl flags from --help to reduce noise.
	for _, name := range []string{
		"as", "as-group", "as-uid", "cache-dir",
		"certificate-authority", "client-certificate", "client-key",
		"insecure-skip-tls-verify", "password", "tls-server-name",
		"token", "username",
	} {
		_ = root.PersistentFlags().MarkHidden(name)
	}

	root.AddCommand(
		newPodCmd(),
		newDeploymentCmd(),
		newNamespaceCmd(),
		newClusterCmd(),
		newReportCmd(),
		newRulesCmd(),
		newConfigCmd(v),
		newVersionCmd(),
		newCompletionCmd(),
	)

	return root
}

// derefStr safely dereferences a *string from cli-runtime (which uses pointer fields).
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// rootUse returns "kubectl triage" when invoked as a kubectl plugin.
func rootUse() string {
	if isKubectlPluginMode() {
		return "kubectl triage"
	}
	return "triage"
}

func isKubectlPluginMode() bool {
	if os.Getenv("KREW_PLUGIN_NAME") != "" {
		return true
	}
	bin := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	return strings.HasPrefix(bin, "kubectl-")
}

func rootLongDescription() string {
	return `triage diagnoses broken Kubernetes workloads.

It cross-references pod status, events, owner refs, services, endpoints, PVCs,
and RBAC in one pass and returns ranked findings with evidence and the exact
next commands to run.

Examples:
  triage pod my-pod -n default
  triage deployment web -n prod
  triage namespace prod
  triage cluster
  kubectl triage pod my-pod -n default   # when installed as a plugin
`
}

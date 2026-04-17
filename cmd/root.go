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
	if err := root.ExecuteContext(ctx); err != nil {
		// Cobra already prints the error; here we distinguish usage errors
		// (bad flag/args) from actual runtime errors. Cobra doesn't give us
		// that cleanly, so we treat anything surfacing here as a usage error.
		return cli.ExitUsage
	}
	// Subcommands set their exit code explicitly via cmd.SetContext + a
	// dedicated "exit code" value key. The diagnosis commands write the code
	// on the shared state and we read it here.
	return readExitCode(root)
}

// rootState carries the chosen exit code from subcommands up to Execute.
type rootState struct {
	exitCode int
}

type rootStateKey struct{}

// setExitCode records the subcommand's exit code on the command tree.
func setExitCode(cmd *cobra.Command, code int) {
	root := cmd.Root()
	st, ok := root.Context().Value(rootStateKey{}).(*rootState)
	if !ok || st == nil {
		return
	}
	st.exitCode = code
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
	flags := rootFlags{}
	kubeFlags := genericclioptions.NewConfigFlags(true)
	v := viper.New()

	root := &cobra.Command{
		Use:           rootUse(),
		Short:         "Diagnose broken Kubernetes workloads",
		Long:          rootLongDescription(),
		SilenceUsage:  true,
		SilenceErrors: false,
		Version:       fmt.Sprintf("%s (commit %s, built %s)", version, commit, buildDate),
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Bind known flags into viper so config file / env vars can override defaults.
			bindFlag(v, cmd, "output", "output")
			bindFlag(v, cmd, "timeout", "timeout")
			bindFlag(v, cmd, "max-findings", "maxFindings")
			bindFlag(v, cmd, "severity-min", "severityMin")
			bindFlag(v, cmd, "confidence-min", "confidenceMin")
			bindFlag(v, cmd, "include-events", "includeEvents")
			bindFlag(v, cmd, "include-related", "includeRelated")

			cfg, err := config.Load(v, config.LoadOptions{ConfigPath: flags.configPath})
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			log := logging.New(logging.Options{
				Debug:   flags.debug,
				Verbose: flags.verbose,
				NoColor: flags.noColor,
			})
			colorOn := cli.ResolveColor(cfg.Color, flags.noColor)

			opts := &cli.Options{
				Config:         cfg,
				KubeFlags:      kubeFlags,
				Logger:         log,
				Color:          colorOn,
				Namespace:      strOrDefault(flags.namespace, *kubeFlags.Namespace),
				Output:         cfg.Output,
				Verbose:        flags.verbose,
				Debug:          flags.debug,
				NoColor:        flags.noColor,
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

	// Install rootState carrier on the root's context so subcommands can
	// communicate exit codes up to Execute.
	root.SetContext(context.WithValue(context.Background(), rootStateKey{}, &rootState{exitCode: cli.ExitOK}))

	// Kube/kubectl-style flags (--kubeconfig, --context, --namespace, --cluster, etc.)
	// cli-runtime's ConfigFlags owns --namespace/-n; we read via kubeFlags.Namespace.
	kubeFlags.AddFlags(root.PersistentFlags())

	// triage-specific flags
	pf := root.PersistentFlags()
	pf.StringVarP(&flags.output, "output", "o", "text", "output format: text, json, markdown")
	pf.BoolVarP(&flags.verbose, "verbose", "v", false, "verbose output")
	pf.BoolVar(&flags.debug, "debug", false, "enable debug logs on stderr")
	pf.BoolVar(&flags.noColor, "no-color", false, "disable ANSI color output")
	pf.IntVar(&flags.maxFindings, "max-findings", 20, "cap rendered findings")
	pf.StringVar(&flags.severityMin, "severity-min", "info", "minimum severity to render (critical|high|medium|low|info)")
	pf.StringVar(&flags.confidenceMin, "confidence-min", "low", "minimum confidence to render (high|medium|low)")
	pf.BoolVar(&flags.includeEvents, "include-events", true, "include related Events in evidence")
	pf.BoolVar(&flags.includeRelated, "include-related", true, "include related Services/PVCs in evidence")
	pf.DurationVar(&flags.timeout, "timeout", 15*time.Second, "overall cluster-call timeout")
	pf.StringVar(&flags.configPath, "config", "", "config file (default $XDG_CONFIG_HOME/triage/config.yaml)")

	// The --namespace flag is owned by cli-runtime; shadow it to also bind to pf.namespace.
	// This way our Options always see it.
	_ = root.PersistentFlags().MarkHidden("as") // cli-runtime adds a lot; hide rarely-used ones in --help
	_ = root.PersistentFlags().MarkHidden("as-group")
	_ = root.PersistentFlags().MarkHidden("as-uid")
	_ = root.PersistentFlags().MarkHidden("cache-dir")
	_ = root.PersistentFlags().MarkHidden("certificate-authority")
	_ = root.PersistentFlags().MarkHidden("client-certificate")
	_ = root.PersistentFlags().MarkHidden("client-key")
	_ = root.PersistentFlags().MarkHidden("insecure-skip-tls-verify")
	_ = root.PersistentFlags().MarkHidden("password")
	_ = root.PersistentFlags().MarkHidden("tls-server-name")
	_ = root.PersistentFlags().MarkHidden("token")
	_ = root.PersistentFlags().MarkHidden("username")

	// Subcommands
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

// rootFlags holds pflag-bound values for later consumption in PersistentPreRunE.
type rootFlags struct {
	namespace      string
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
}

func bindFlag(v *viper.Viper, cmd *cobra.Command, flagName, cfgKey string) {
	if f := cmd.PersistentFlags().Lookup(flagName); f != nil {
		_ = v.BindPFlag(cfgKey, f)
	}
}

func strOrDefault(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// -----------------------------------------------------------------------------
// Plugin mode (kubectl triage)
// -----------------------------------------------------------------------------

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
	bin := filepath.Base(os.Args[0])
	// strip platform executable suffixes
	bin = strings.TrimSuffix(bin, ".exe")
	return strings.HasPrefix(bin, "kubectl-")
}

func rootLongDescription() string {
	return `triage diagnoses broken Kubernetes workloads.

It inspects Pods, Deployments, Namespaces, and whole Clusters, and returns
ranked findings with evidence, suggested fixes, and the exact next commands
to run. Not a wrapper around kubectl describe — a rule-based diagnosis engine.

Examples:
  triage pod my-pod -n default
  triage deployment web -n prod
  triage namespace prod
  triage cluster
  kubectl triage pod my-pod -n default   # when installed as a plugin
`
}

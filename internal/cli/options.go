package cli

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/khvedela/triage/internal/config"
)

// Options is the per-invocation state threaded through every command.
// Built in root.PersistentPreRunE, attached to cmd.Context via WithOptions,
// and retrieved in subcommands via Get.
type Options struct {
	Config      config.Config
	KubeFlags   *genericclioptions.ConfigFlags
	Logger      logr.Logger
	Color       bool
	Namespace   string
	Output      string
	Verbose     bool
	Debug       bool
	NoColor     bool
	Timeout     time.Duration
	MaxFindings int
	SeverityMin string
	ConfMin     string

	IncludeEvents  bool
	IncludeRelated bool

	// Overall deadline derived from Timeout. Commands should pass this down
	// rather than starting new background contexts.
	Deadline time.Time
}

type optionsKey struct{}

// WithOptions returns a child context carrying the given options.
func WithOptions(ctx context.Context, o *Options) context.Context {
	return context.WithValue(ctx, optionsKey{}, o)
}

// Get retrieves the Options from a cobra command's context. Panics if not set;
// that always indicates a wiring bug in the command tree, not user error.
func Get(cmd *cobra.Command) *Options {
	v := cmd.Context().Value(optionsKey{})
	o, ok := v.(*Options)
	if !ok || o == nil {
		panic("cli.Get: Options not attached to command context (missing root PersistentPreRunE?)")
	}
	return o
}

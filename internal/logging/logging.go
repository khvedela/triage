// Package logging wires go-logr/logr onto zap with sensible CLI defaults.
//
// Production CLI logging rules of thumb:
//   - stderr, never stdout (stdout is for tool output).
//   - Info-and-above by default; Debug gated on --debug.
//   - Human-readable console encoder (not JSON) unless explicitly asked.
package logging

import (
	"io"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Options configures the logger.
type Options struct {
	Debug   bool      // include debug-level messages
	Verbose bool      // marginal extra info; does not imply Debug
	NoColor bool      // disable ANSI color in the encoder
	Out     io.Writer // defaults to stderr
}

// New builds a logr.Logger. It never fails.
func New(o Options) logr.Logger {
	out := o.Out
	if out == nil {
		out = os.Stderr
	}
	level := zapcore.InfoLevel
	if o.Debug {
		level = zapcore.DebugLevel
	}

	enc := zap.NewDevelopmentEncoderConfig()
	enc.TimeKey = ""     // CLI logs don't need per-line timestamps
	enc.CallerKey = ""   // noisy for a CLI
	enc.StacktraceKey = "" // avoid surprise stacktraces in user terminals
	if !o.NoColor {
		enc.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		enc.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(enc),
		zapcore.AddSync(out),
		level,
	)
	return zapr.NewLogger(zap.New(core))
}

// Discard returns a logger that drops all output. Useful for tests.
func Discard() logr.Logger { return logr.Discard() }

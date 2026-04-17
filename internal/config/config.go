// Package config loads and resolves triage configuration.
//
// Resolution precedence (highest to lowest):
//  1. Command-line flags (bound via pflag)
//  2. Environment variables (TRIAGE_*)
//  3. Config file ($XDG_CONFIG_HOME/triage/config.yaml)
//  4. Built-in defaults
//
// The Provenance map tracks where each value came from, for `triage config view`.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the resolved runtime configuration for a triage invocation.
type Config struct {
	Output         string        `mapstructure:"output"         yaml:"output"`
	Color          string        `mapstructure:"color"          yaml:"color"`
	SeverityMin    string        `mapstructure:"severityMin"    yaml:"severityMin"`
	ConfidenceMin  string        `mapstructure:"confidenceMin"  yaml:"confidenceMin"`
	MaxFindings    int           `mapstructure:"maxFindings"    yaml:"maxFindings"`
	IncludeEvents  bool          `mapstructure:"includeEvents"  yaml:"includeEvents"`
	IncludeRelated bool          `mapstructure:"includeRelated" yaml:"includeRelated"`
	Timeout        time.Duration `mapstructure:"timeout"        yaml:"timeout"`

	Rules      RulesConfig      `mapstructure:"rules"      yaml:"rules"`
	Namespaces NamespacesConfig `mapstructure:"namespaces" yaml:"namespaces"`

	// Provenance records where each key's value came from: "flag", "env",
	// "file:<path>", or "default". Populated by Load.
	Provenance map[string]string `mapstructure:"-" yaml:"-"`
}

// RulesConfig controls which rules run.
type RulesConfig struct {
	Disabled []string `mapstructure:"disabled" yaml:"disabled"`
	Enabled  []string `mapstructure:"enabled"  yaml:"enabled"` // empty = all (minus disabled)
}

// NamespacesConfig controls namespace-level filtering.
type NamespacesConfig struct {
	Exclude []string `mapstructure:"exclude" yaml:"exclude"`
}

// Defaults returns the built-in configuration.
func Defaults() Config {
	return Config{
		Output:         "text",
		Color:          "auto",
		SeverityMin:    "info",
		ConfidenceMin:  "low",
		MaxFindings:    20,
		IncludeEvents:  true,
		IncludeRelated: true,
		Timeout:        15 * time.Second,
		Rules:          RulesConfig{},
		Namespaces:     NamespacesConfig{Exclude: []string{"kube-system", "kube-public"}},
	}
}

// LoadOptions controls how the configuration is loaded.
type LoadOptions struct {
	// ConfigPath, if non-empty, overrides the default file lookup.
	ConfigPath string
	// EnvPrefix for env-var binding. Defaults to "TRIAGE".
	EnvPrefix string
}

// Load resolves configuration from defaults, optional file, environment,
// and the provided flag-bound viper instance. It never returns an error for
// a missing config file — that is the normal case.
func Load(v *viper.Viper, opts LoadOptions) (Config, error) {
	if v == nil {
		v = viper.New()
	}
	if opts.EnvPrefix == "" {
		opts.EnvPrefix = "TRIAGE"
	}

	// Defaults
	d := Defaults()
	v.SetDefault("output", d.Output)
	v.SetDefault("color", d.Color)
	v.SetDefault("severityMin", d.SeverityMin)
	v.SetDefault("confidenceMin", d.ConfidenceMin)
	v.SetDefault("maxFindings", d.MaxFindings)
	v.SetDefault("includeEvents", d.IncludeEvents)
	v.SetDefault("includeRelated", d.IncludeRelated)
	v.SetDefault("timeout", d.Timeout)
	v.SetDefault("rules.disabled", d.Rules.Disabled)
	v.SetDefault("rules.enabled", d.Rules.Enabled)
	v.SetDefault("namespaces.exclude", d.Namespaces.Exclude)

	// Env binding
	v.SetEnvPrefix(opts.EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// File binding (optional)
	configFile := opts.ConfigPath
	if configFile == "" {
		configFile = DefaultPath()
	}
	if configFile != "" {
		if _, err := os.Stat(configFile); err == nil {
			v.SetConfigFile(configFile)
			if err := v.ReadInConfig(); err != nil {
				return Config{}, fmt.Errorf("reading config %s: %w", configFile, err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("stat config %s: %w", configFile, err)
		}
	}

	var out Config
	if err := v.Unmarshal(&out); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	out.Provenance = buildProvenance(v, configFile)
	return out, nil
}

// DefaultPath returns the XDG-friendly default location for the config file.
// Returns "" if no reasonable default can be determined.
func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "triage", "config.yaml")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "triage", "config.yaml")
}

func buildProvenance(v *viper.Viper, configFile string) map[string]string {
	prov := map[string]string{}
	for _, key := range v.AllKeys() {
		switch {
		case v.InConfig(key) && configFile != "":
			prov[key] = "file:" + configFile
		case v.IsSet(key):
			prov[key] = "flag-or-env"
		default:
			prov[key] = "default"
		}
	}
	return prov
}

// Template returns a commented YAML template for `triage config init`.
func Template() string {
	return `# triage configuration
# Location: $XDG_CONFIG_HOME/triage/config.yaml (default: ~/.config/triage/config.yaml)
# Every key is also overridable via flag or env var (TRIAGE_<UPPER_SNAKE_KEY>).

# Default output format: text | json | markdown
output: text

# ANSI color mode: auto | always | never
color: auto

# Minimum severity to render: critical | high | medium | low | info
severityMin: info

# Minimum confidence to render: high | medium | low
confidenceMin: low

# Cap the number of findings shown in text output (JSON/markdown are not capped).
maxFindings: 20

# Include related Events in output.
includeEvents: true

# Include related resources (Service, Endpoints, PVC, etc) in output.
includeRelated: true

# Overall Kubernetes API call timeout.
timeout: 15s

rules:
  # Disable specific rules by ID. Use ` + "`triage rules list`" + ` to see all IDs.
  disabled: []
  # If non-empty, only these rule IDs will run.
  enabled: []

namespaces:
  # Namespaces excluded from ` + "`triage cluster`" + ` and ` + "`triage namespace`" + ` scans
  # when no target is explicitly specified.
  exclude:
    - kube-system
    - kube-public
`
}

# Configuration

`triage` can be configured via a YAML file, environment variables, or CLI flags. Precedence (highest to lowest): flag → env → config file → default.

## Config file location

Default: `$XDG_CONFIG_HOME/triage/config.yaml` (usually `~/.config/triage/config.yaml`).

Run `triage config path` to print the resolved path.  
Run `triage config init` to write a commented template.  
Run `triage config view` to see the current resolved config with provenance.

## Reference

```yaml
# Output format: text | json | markdown
output: text

# Color mode: auto | always | never
# auto = enable when stdout is a TTY and NO_COLOR is unset
color: auto

# Minimum severity to display: critical | high | medium | low | info
severityMin: info

# Minimum confidence to display: high | medium | low
confidenceMin: low

# Maximum number of findings to render (0 = unlimited)
maxFindings: 20

# Include related events in output (applies to pod/deployment targets)
includeEvents: true

# Include related resources (services, pvcs, endpoints) in output
includeRelated: true

# Overall Kubernetes API timeout
timeout: 15s

rules:
  # Rule IDs to disable entirely
  disabled: []
  # Rule IDs to enable exclusively (empty = all, minus disabled)
  enabled: []

namespaces:
  # Namespaces to exclude from namespace/cluster scans
  exclude: [kube-system, kube-public]
```

## Environment variable mapping

| Env var | Config key |
|---------|-----------|
| `TRIAGE_OUTPUT` | `output` |
| `TRIAGE_COLOR` | `color` |
| `TRIAGE_SEVERITY_MIN` | `severityMin` |
| `TRIAGE_CONFIDENCE_MIN` | `confidenceMin` |
| `TRIAGE_MAX_FINDINGS` | `maxFindings` |
| `TRIAGE_TIMEOUT` | `timeout` |
| `NO_COLOR` | disables color (standard convention) |

## CLI flags

All flags are defined on the root command and inherited by all subcommands. Run `triage --help` for the full list.

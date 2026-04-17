---
title: Configuration
description: Config file shape, precedence, environment variables, and inherited CLI flags.
slug: /configuration
---

import DocHero from "@site/src/components/docs/DocHero";
import InfoGrid from "@site/src/components/docs/InfoGrid";
import CommandBlock from "@site/src/components/docs/CommandBlock";

<DocHero
  eyebrow="Configuration"
  title="One config model, four layers of precedence."
  lede="triage configuration is intentionally shallow. The important behavior is precedence: flags override environment variables, environment variables override the config file, and the config file overrides defaults. That makes the tool predictable in local shells, CI jobs, and incident automation."
  meta={["Flags > env > file > defaults", "~/.config/triage/config.yaml", "Same config across scopes"]} />

<InfoGrid
  columns={2}
  items={[
    {
      label: "Defaults",
      title: "Start from safe general behavior",
      body: "Text output, low confidence threshold, and a modest finding cap are meant to be useful without any configuration."
    },
    {
      label: "File config",
      title: "Use a config file for workstation preferences",
      body: "Good for persistent output preferences, namespace exclusions, or rule enable/disable lists."
    },
    {
      label: "Environment",
      title: "Use env vars for automation and wrappers",
      body: "Good when the invoking environment wants to shape output or thresholds without editing files."
    },
    {
      label: "Flags",
      title: "Use flags for incident-specific overrides",
      body: "Best when the diagnosis needs to be narrower, quieter, or differently formatted right now."
    }
  ]} />

<CommandBlock
  eyebrow="Find the active config"
  title="Inspect the resolved path and the resolved values."
  description="These commands are the fastest way to debug configuration behavior before you start guessing about precedence."
  command={[
    "triage config path",
    "triage config view"
  ]}
  caption="`config view` includes provenance, which makes it obvious whether a value came from a flag, env var, file, or default." />

## Reference config

```yaml
# Output format: text | json | markdown
output: text

# Color mode: auto | always | never
color: auto

# Minimum severity to display: critical | high | medium | low | info
severityMin: info

# Minimum confidence to display: high | medium | low
confidenceMin: low

# Maximum number of findings to render (0 = unlimited)
maxFindings: 20

# Include related events in output
includeEvents: true

# Include related resources (services, pvcs, endpoints) in output
includeRelated: true

# Overall Kubernetes API timeout
timeout: 15s

rules:
  disabled: []
  enabled: []

namespaces:
  exclude: [kube-system, kube-public]
```

## High-signal knobs

| Key | What it changes |
| --- | --- |
| `output` | Selects terminal text, JSON, or markdown as the renderer. |
| `severityMin` | Filters out lower-severity findings early in the output. |
| `confidenceMin` | Filters out lower-confidence diagnoses when you want a stricter signal. |
| `maxFindings` | Caps output size on dense namespace and cluster scans. |
| `includeEvents` | Controls whether event context is folded into evidence. |
| `includeRelated` | Controls whether adjacent resources are included when relevant. |
| `rules.disabled` / `rules.enabled` | Lets you trim or constrain the active rule set. |
| `namespaces.exclude` | Prevents noisy namespaces from dominating broad scans. |

## Environment variable mapping

| Env var | Config key |
| --- | --- |
| `TRIAGE_OUTPUT` | `output` |
| `TRIAGE_COLOR` | `color` |
| `TRIAGE_SEVERITY_MIN` | `severityMin` |
| `TRIAGE_CONFIDENCE_MIN` | `confidenceMin` |
| `TRIAGE_MAX_FINDINGS` | `maxFindings` |
| `TRIAGE_TIMEOUT` | `timeout` |
| `NO_COLOR` | Disables color by convention |

## When to use flags instead

Use flags when the value is specific to one run, such as:

- raising `--severity-min` during a noisy incident
- switching to `-o json` for one automation step
- lowering `--max-findings` when only the top-ranked items matter
- pointing at a temporary config file with `--config`

The fewer persistent surprises the tool has, the more trustworthy it is under pressure. That is why triage keeps the config surface small and the precedence model explicit.

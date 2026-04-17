# Commands

`triage` keeps the CLI surface intentionally small: one command per diagnosis scope, plus reporting, rule introspection, config helpers, and shell completion.

## Root command

```text
triage [command]
```

Available commands:

| Command | Purpose |
| --- | --- |
| `triage pod <name>` | Diagnose a single pod |
| `triage deployment <name>` | Diagnose a deployment and include owned pod findings |
| `triage namespace <name>` | Diagnose every workload in a namespace |
| `triage cluster` | Diagnose cluster-wide conditions such as node health |
| `triage report namespace <name>` | Generate a full markdown diagnostic report |
| `triage rules list` | List built-in rules |
| `triage rules explain <rule-id>` | Show full rule documentation |
| `triage config view` | Print resolved configuration with provenance |
| `triage config init` | Write a commented config template |
| `triage config path` | Print the default config path |
| `triage completion bash|zsh|fish|powershell` | Generate shell completion |
| `triage version` | Print version information |

## Global flags

| Flag | Description |
| --- | --- |
| `-o, --output` | Output format: `text`, `json`, `markdown` |
| `-n, --namespace` | Namespace scope for pod and deployment diagnoses |
| `--severity-min` | Minimum severity to render |
| `--confidence-min` | Minimum confidence to render |
| `--include-events` | Include related events in evidence |
| `--include-related` | Include related resources such as Services and PVCs |
| `--max-findings` | Cap the number of findings rendered |
| `--timeout` | Overall cluster-call timeout |
| `--config` | Explicit config file path |
| `--debug` | Enable debug logs on stderr |
| `--no-color` | Disable ANSI color output |

Kubernetes client flags such as `--context`, `--kubeconfig`, `--cluster`, and `--user` are also supported.

## Scope commands

### `triage pod <name>`

```sh
triage pod my-pod -n default
```

Diagnose a single pod and rank the most likely root cause first.

### `triage deployment <name>`

```sh
triage deployment web -n prod
```

Diagnose a deployment and include rollout-level plus owned pod-level findings.

### `triage namespace <name>`

```sh
triage namespace prod
```

Survey a namespace for workload, event, service, and endpoint issues.

### `triage cluster`

```sh
triage cluster
```

Run cluster-scoped checks such as node readiness and resource pressure.

## Reports

### `triage report namespace <name>`

```sh
triage report namespace prod > triage-report.md
```

Forces markdown output and a larger finding cap for report generation.

## Rules

### `triage rules list`

```sh
triage rules list
triage rules list --category Runtime
triage rules list --severity high
```

### `triage rules explain <rule-id>`

```sh
triage rules explain TRG-POD-OOMKILLED
```

Print the rule description, scopes, severity, and linked upstream docs.

## Config

### `triage config view`

Show the resolved config and provenance.

### `triage config init`

Write the default commented config template to `~/.config/triage/config.yaml`.

### `triage config path`

Print the default config path without writing anything.

## Completion

```sh
triage completion bash
triage completion zsh
triage completion fish
triage completion powershell
```

## Version

```sh
triage version
```

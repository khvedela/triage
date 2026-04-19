# Commands

`triage` keeps the CLI surface intentionally small: one command per diagnosis scope, plus reporting, rule introspection, config helpers, and shell completion.

## Root command

```text
kubediag [command]
```

Available commands:

| Command | Purpose |
| --- | --- |
| `kubediag pod <name>` | Diagnose a single pod |
| `kubediag deployment <name>` | Diagnose a deployment and include owned pod findings |
| `kubediag namespace <name>` | Diagnose every workload in a namespace |
| `kubediag cluster` | Diagnose cluster-wide conditions such as node health |
| `kubediag report namespace <name>` | Generate a full markdown diagnostic report |
| `kubediag rules list` | List built-in rules |
| `kubediag rules explain <rule-id>` | Show full rule documentation |
| `kubediag config view` | Print resolved configuration with provenance |
| `kubediag config init` | Write a commented config template |
| `kubediag config path` | Print the default config path |
| `kubediag completion bash|zsh|fish|powershell` | Generate shell completion |
| `kubediag version` | Print version information |

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

### `kubediag pod <name>`

```sh
kubediag pod my-pod -n default
```

Diagnose a single pod and rank the most likely root cause first.

### `kubediag deployment <name>`

```sh
kubediag deployment web -n prod
```

Diagnose a deployment and include rollout-level plus owned pod-level findings.

### `kubediag namespace <name>`

```sh
kubediag namespace prod
```

Survey a namespace for workload, event, service, and endpoint issues.

### `kubediag cluster`

```sh
kubediag cluster
```

Run cluster-scoped checks such as node readiness and resource pressure.

## Reports

### `kubediag report namespace <name>`

```sh
kubediag report namespace prod > triage-report.md
```

Forces markdown output and a larger finding cap for report generation.

## Rules

### `kubediag rules list`

```sh
kubediag rules list
kubediag rules list --category Runtime
kubediag rules list --severity high
```

### `kubediag rules explain <rule-id>`

```sh
kubediag rules explain TRG-POD-OOMKILLED
```

Print the rule description, scopes, severity, and linked upstream docs.

## Config

### `kubediag config view`

Show the resolved config and provenance.

### `kubediag config init`

Write the default commented config template to `~/.config/triage/config.yaml`.

### `kubediag config path`

Print the default config path without writing anything.

## Completion

```sh
kubediag completion bash
kubediag completion zsh
kubediag completion fish
kubediag completion powershell
```

## Version

```sh
kubediag version
```

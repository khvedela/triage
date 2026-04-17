# triage

> A kubectl-native diagnostic CLI that turns broken Kubernetes workload symptoms into ranked root-cause findings, evidence, and the exact next command to run.

[![CI](https://github.com/OWNER/triage/actions/workflows/ci.yml/badge.svg)](https://github.com/OWNER/triage/actions/workflows/ci.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Report](https://goreportcard.com/badge/github.com/OWNER/triage)](https://goreportcard.com/report/github.com/OWNER/triage)

**`triage` is not another wrapper around `kubectl describe`.** It's a rule-based diagnosis engine that cross-references pod status, events, owner refs, services, endpoints, PVCs, and RBAC in one pass and tells you *what is broken*, *why*, and *what to do next* — in under two seconds.

---

## Why `triage`?

When a pod breaks, most of us end up running the same forensic sequence:

```
kubectl describe pod ...
kubectl get events ...
kubectl logs ...
kubectl get svc,endpoints ...
# ... then Google the error message
```

`triage` automates that entire loop. One command, one ranked diagnosis.

|                         | `kubectl describe` | `triage`                              |
| ----------------------- | ------------------ | ------------------------------------- |
| Shows raw status        | ✅                 | ✅ (as evidence)                      |
| Identifies root cause   | ❌                 | ✅ (ranked by confidence × severity)  |
| Cross-references events | ❌                 | ✅                                    |
| Walks owner references  | ❌                 | ✅                                    |
| Checks Service/Endpoint | ❌                 | ✅                                    |
| Detects RBAC gaps       | ❌                 | ✅                                    |
| Suggests next commands  | ❌                 | ✅                                    |
| Machine-readable        | ❌                 | ✅ (`-o json`, `-o markdown`)         |

---

## Install

### Homebrew *(coming soon)*

```sh
brew install OWNER/tap/triage
```

### Krew *(Kubernetes plugin manager — coming soon)*

```sh
kubectl krew install triage
```

### Binary download

Grab a prebuilt binary from [Releases](https://github.com/OWNER/triage/releases) and drop it on your `$PATH`.

### From source

```sh
go install github.com/OWNER/triage@latest
```

### As a kubectl plugin

Symlink the binary to `kubectl-triage` somewhere on your `$PATH`:

```sh
ln -s $(which triage) ~/.local/bin/kubectl-triage
kubectl triage pod my-pod -n default
```

---

## Quick start

```sh
# Diagnose a single pod
triage pod my-pod -n default

# Diagnose a deployment (surfaces deployment-level and pod-level findings)
triage deployment web -n prod

# Survey a whole namespace
triage namespace prod

# Cluster-wide check
triage cluster

# Machine-readable output
triage pod my-pod -o json | jq '.findings[0]'

# Generate a markdown incident report
triage report namespace prod > triage-report.md
```

### Example output

```
▶ Pod default/my-api-7f9b-xk2m2      Phase: Running     Ready: 0/1

ⓧ CRITICAL  [high confidence]  TRG-POD-CRASHLOOPBACKOFF
  Container `api` is in CrashLoopBackOff (5 restarts in the last 3m)

  Evidence:
    • pod.status.containerStatuses[0].lastState.terminated.reason = "Error"
    • pod.status.containerStatuses[0].lastState.terminated.exitCode = 1
    • Event (Warning, BackOff, 30s ago): "Back-off restarting failed container"
    • Last 3 log lines from container `api`:
        panic: open /etc/config/app.yaml: no such file or directory

  Next commands:
    $ kubectl logs -n default my-api-7f9b-xk2m2 -c api --previous
    $ kubectl describe configmap -n default app-config
    $ kubectl get pod -n default my-api-7f9b-xk2m2 -o yaml

  Suggested fix:
    The container is panicking because /etc/config/app.yaml is missing. The
    referenced ConfigMap `app-config` either doesn't exist, lacks the key
    `app.yaml`, or isn't mounted at /etc/config.

  Docs: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-states
```

---

## Commands

| Command                           | Purpose                                                   |
| --------------------------------- | --------------------------------------------------------- |
| `triage pod <name>`               | Diagnose a single pod                                     |
| `triage deployment <name>`        | Diagnose a deployment and all pods under it               |
| `triage namespace <ns>`           | Diagnose every workload in a namespace                    |
| `triage cluster`                  | Cluster-wide checks (node conditions, recent warnings)    |
| `triage report namespace <ns>`    | Full markdown diagnostic report                           |
| `triage rules list`               | List all built-in rules                                   |
| `triage rules explain <rule-id>`  | Full docs for a specific rule                             |
| `triage config view`              | Show resolved configuration with provenance               |
| `triage config init`              | Write a commented config template                         |
| `triage completion {shell}`       | Shell completion script                                   |
| `triage version`                  | Print version info                                        |

See [docs/commands.md](docs/commands.md) for full reference.

---

## Architecture

```
┌──────────────────┐   ┌──────────────────┐   ┌────────────────┐
│  CLI (Cobra)     │──▶│  engine (orches) │──▶│  rules         │
└──────────────────┘   └────┬─────────────┘   └────────────────┘
                            │  collects via
                            ▼
                       ┌──────────────┐
                       │  kube client │   client-go, informer-less
                       │  + cache     │     one-shot queries, dedup
                       └──────────────┘
                            │
                            ▼ produces
                       ┌──────────────┐      ┌─────────────┐
                       │  Findings    │────▶│  renderers  │ text/json/md
                       └──────────────┘      └─────────────┘
```

See [docs/architecture.md](docs/architecture.md) for the full design.

---

## Rules

`triage` ships with a built-in rule set covering the most common failure modes. Each rule has a stable ID, a category, and documented evidence + remediation.

Sample categories:

- **Scheduling** — pending pods: insufficient resources, taint mismatch, selector mismatch, unbound PVC
- **Image / Registry** — ImagePullBackOff, auth failures, manifest not found
- **Configuration** — missing ConfigMap/Secret, bad env ref, bad command
- **Probes / Health** — failing readiness/liveness/startup probes
- **Crash / Runtime** — CrashLoopBackOff, OOMKilled, init failures
- **Networking** — service with no endpoints, selector/port mismatch
- **Rollout / Controller** — stuck deployment rollouts
- **Resource Pressure** — node NotReady, Memory/Disk/PID pressure

Full list: [docs/rules.md](docs/rules.md). Want to add a rule? See [CONTRIBUTING.md](CONTRIBUTING.md).

---

## Configuration

Optional config at `~/.config/triage/config.yaml`:

```yaml
output: text                     # text | json | markdown
color: auto                      # auto | always | never
severityMin: info                # critical | high | medium | low | info
confidenceMin: low               # high | medium | low
maxFindings: 20
includeEvents: true
includeRelated: true
timeout: 15s
rules:
  disabled: []
  enabled: []                    # empty = all (minus disabled)
namespaces:
  exclude: [kube-system, kube-public]
```

All keys are also overridable via environment variables prefixed `TRIAGE_` (e.g. `TRIAGE_OUTPUT=json`). See [docs/configuration.md](docs/configuration.md).

---

## Roadmap

Near-term (v0.x):

- Full coverage of the rule set in [docs/rules.md](docs/rules.md)
- Namespace and cluster-scope aggregations
- `--diff-time` — compare cluster state at two points in time
- Prometheus integration — incorporate metrics as evidence

Longer-term:

- YAML rule packs (declarative rules without recompile)
- CRD-aware rules (Istio, cert-manager, Argo Rollouts)
- Optional LLM-assisted explainer (pluggable, off by default)

See [docs/roadmap.md](docs/roadmap.md).

---

## Contributing

Contributions welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for how to build, test, and add rules. Security issues: see [SECURITY.md](SECURITY.md).

## License

Apache 2.0 — see [LICENSE](LICENSE).

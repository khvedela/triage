# triage

> A kubectl-native diagnostic CLI that turns broken Kubernetes workload symptoms into ranked root-cause findings, evidence, and the exact next command to run.

[![CI](https://github.com/khvedela/triage/actions/workflows/ci.yml/badge.svg)](https://github.com/khvedela/triage/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/khvedela/triage)](https://github.com/khvedela/triage/releases/latest)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Report](https://goreportcard.com/badge/github.com/khvedela/triage)](https://goreportcard.com/report/github.com/khvedela/triage)

**`triage` is not another wrapper around `kubectl describe`.** It's a rule-based diagnosis engine that cross-references pod status, events, owner refs, services, endpoints, PVCs, and RBAC in one pass and tells you *what is broken*, *why*, and *what to do next* — in under two seconds.

Docs website: <https://khvedela.github.io/triage/>  
Interactive sandbox: <https://khvedela.github.io/triage/sandbox>

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

### Linux

**Quick install** (x86_64):
```sh
curl -sL https://github.com/khvedela/triage/releases/download/v0.2.0/triage_linux_amd64.tar.gz | tar xz -C /usr/local/bin
triage --help
```

**Arm64**:
```sh
curl -sL https://github.com/khvedela/triage/releases/download/v0.2.0/triage_linux_arm64.tar.gz | tar xz -C /usr/local/bin
```

### macOS

**Homebrew**:
```sh
brew install khvedela/triage/triage
```

Or manually: grab a binary from [Releases](https://github.com/khvedela/triage/releases).

### Krew (all platforms)

```sh
kubectl krew install triage
```

Manifest is in-repo; public Krew index publication pending.

### From source

```sh
go install github.com/khvedela/triage@latest
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

# Generate a markdown incident report for a namespace
triage report namespace prod > triage-report.md

# Generate a full cluster report
triage report cluster > cluster-report.md
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
| `triage cluster`                  | Cluster-wide checks (node conditions, quota, events)      |
| `triage report namespace <ns>`    | Full markdown diagnostic report for a namespace           |
| `triage report cluster`           | Full markdown diagnostic report for the cluster           |
| `triage rules list`               | List all built-in rules                                   |
| `triage rules explain <rule-id>`  | Full docs for a specific rule                             |
| `triage config view`              | Show resolved configuration with provenance               |
| `triage config init`              | Write a commented config template                         |
| `triage completion {shell}`       | Shell completion script                                   |
| `triage version`                  | Print version info                                        |

See [docs/commands.md](docs/commands.md) for the repo copy, or browse the hosted docs at <https://khvedela.github.io/triage/docs/commands>.

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

`triage` ships with 28 built-in rules covering the most common failure modes. Each rule has a stable ID, a category, and documented evidence + remediation.

| Category | Rules |
| --- | --- |
| **Scheduling** | Pending: insufficient resources, taint mismatch, selector mismatch, unbound PVC |
| **Image / Registry** | ImagePullBackOff, auth failures, manifest not found |
| **Configuration** | Missing ConfigMap/Secret, bad env key ref, bad command |
| **Runtime / Crash** | CrashLoopBackOff, OOMKilled, init failures, immediate exec failure |
| **Probes / Health** | Failing readiness, liveness, startup probes |
| **Networking** | Service with no endpoints, selector mismatch, targetPort mismatch |
| **Rollout** | Stuck deployment rollout, unavailable replicas |
| **Resource Pressure** | Node NotReady, Memory/Disk/PID pressure, quota exhausted |
| **Cluster** | API server latency events |

Full list: [docs/rules.md](docs/rules.md). Hosted reference: <https://khvedela.github.io/triage/docs/rules>.  
Want to add a rule? See [CONTRIBUTING.md](CONTRIBUTING.md).

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

All keys are also overridable via environment variables prefixed `TRIAGE_` (e.g. `TRIAGE_OUTPUT=json`). See [docs/configuration.md](docs/configuration.md) or <https://khvedela.github.io/triage/docs/configuration>.

---

## Roadmap

**v0.2.0** (current): Rule set expansion — exec-format errors, service port mismatches, bad env key refs, quota exhaustion, API server latency events, `triage report cluster`, richer readiness probe sampling.

**v0.3.0**: YAML rule packs — declarative rules with CEL expressions, loadable without recompile.

**v0.4.0**: Interactive mode — `triage watch pod <name>`, `--since` flag, fzf-style namespace picker.

**v1.0.0**: Stable public API, `pkg/` promotion, Homebrew tap, container image.

See [docs/roadmap.md](docs/roadmap.md) or <https://khvedela.github.io/triage/docs/roadmap>.

---

## Contributing

Contributions welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for how to build, test, and add rules. Security issues: see [SECURITY.md](SECURITY.md). For the presentable docs and sandbox experience, use <https://khvedela.github.io/triage/>.

## License

Apache 2.0 — see [LICENSE](LICENSE).

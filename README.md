# kubediag

> A kubectl-native diagnostic CLI that turns broken Kubernetes workload symptoms into ranked root-cause findings, evidence, and the exact next command to run.

[![CI](https://github.com/khvedela/kubediag/actions/workflows/ci.yml/badge.svg)](https://github.com/khvedela/kubediag/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/khvedela/kubediag)](https://github.com/khvedela/kubediag/releases/latest)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Report](https://goreportcard.com/badge/github.com/khvedela/kubediag)](https://goreportcard.com/report/github.com/khvedela/kubediag)

**`kubediag` is not another wrapper around `kubectl describe`.** It's a rule-based diagnosis engine that cross-references pod status, events, owner refs, services, endpoints, PVCs, and RBAC in one pass and tells you *what is broken*, *why*, and *what to do next* — in under two seconds.

Docs website: <https://khvedela.github.io/kubediag/>  
Interactive sandbox: <https://khvedela.github.io/kubediag/sandbox>

---

## Why `kubediag`?

When a pod breaks, most of us end up running the same forensic sequence:

```
kubectl describe pod ...
kubectl get events ...
kubectl logs ...
kubectl get svc,endpoints ...
# ... then Google the error message
```

`kubediag` automates that entire loop. One command, one ranked diagnosis.

|                         | `kubectl describe` | `kubediag`                            |
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
curl -sL https://github.com/khvedela/kubediag/releases/download/v0.2.0/kubediag_linux_amd64.tar.gz | tar xz -C /usr/local/bin
kubediag --help
```

**Arm64**:
```sh
curl -sL https://github.com/khvedela/kubediag/releases/download/v0.2.0/kubediag_linux_arm64.tar.gz | tar xz -C /usr/local/bin
```

### macOS

**Homebrew**:
```sh
brew install khvedela/kubediag/kubediag
```

Or manually: grab a binary from [Releases](https://github.com/khvedela/kubediag/releases).

### Krew (all platforms)

```sh
kubectl krew install kubediag
```

Manifest is in-repo; public Krew index publication pending.

### From source

```sh
go install github.com/khvedela/kubediag@latest
```

### As a kubectl plugin

Symlink the binary to `kubectl-kubediag` somewhere on your `$PATH`:

```sh
ln -s $(which kubediag) ~/.local/bin/kubectl-kubediag
kubectl kubediag pod my-pod -n default
```

---

## Quick start

```sh
# Diagnose a single pod
kubediag pod my-pod -n default

# Diagnose a deployment (surfaces deployment-level and pod-level findings)
kubediag deployment web -n prod

# Survey a whole namespace
kubediag namespace prod

# Cluster-wide check
kubediag cluster

# Machine-readable output
kubediag pod my-pod -o json | jq '.findings[0]'

# Generate a markdown incident report for a namespace
kubediag report namespace prod > kubediag-report.md

# Generate a full cluster report
kubediag report cluster > cluster-report.md
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
| `kubediag pod <name>`               | Diagnose a single pod                                     |
| `kubediag deployment <name>`        | Diagnose a deployment and all pods under it               |
| `kubediag namespace <ns>`           | Diagnose every workload in a namespace                    |
| `kubediag cluster`                  | Cluster-wide checks (node conditions, quota, events)      |
| `kubediag report namespace <ns>`    | Full markdown diagnostic report for a namespace           |
| `kubediag report cluster`           | Full markdown diagnostic report for the cluster           |
| `kubediag rules list`               | List all built-in rules                                   |
| `kubediag rules explain <rule-id>`  | Full docs for a specific rule                             |
| `kubediag config view`              | Show resolved configuration with provenance               |
| `kubediag config init`              | Write a commented config template                         |
| `kubediag completion {shell}`       | Shell completion script                                   |
| `kubediag version`                  | Print version info                                        |

See [docs/commands.md](docs/commands.md) for the repo copy, or browse the hosted docs at <https://khvedela.github.io/kubediag/docs/commands>.

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

See [docs/architecture.md](docs/architecture.md) for full design.

---

## Rules

`kubediag` ships with 28 built-in rules covering the most common failure modes. Each rule has a stable ID, a category, and documented evidence + remediation.

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

Full list: [docs/rules.md](docs/rules.md). Hosted reference: <https://khvedela.github.io/kubediag/docs/rules>.  
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

All keys are also overridable via environment variables prefixed `KUBEDIAG_` (e.g. `KUBEDIAG_OUTPUT=json`). See [docs/configuration.md](docs/configuration.md) or <https://khvedela.github.io/kubediag/docs/configuration>.

---

## Roadmap

**v0.2.0** (current): Rule set expansion — exec-format errors, service port mismatches, bad env key refs, quota exhaustion, API server latency events, `kubediag report cluster`, richer readiness probe sampling.

**v0.3.0**: YAML rule packs — declarative rules with CEL expressions, loadable without recompile.

**v0.4.0**: Interactive mode — `kubediag watch pod <name>`, `--since` flag, fzf-style namespace picker.

**v1.0.0**: Stable public API, `pkg/` promotion, Homebrew tap, container image.

See [docs/roadmap.md](docs/roadmap.md) or <https://khvedela.github.io/kubediag/docs/roadmap>.

---

## Contributing

Contributions welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for how to build, test, and add rules. Security issues: see [SECURITY.md](SECURITY.md). For the presentable docs and sandbox experience, use <https://khvedela.github.io/kubediag/>.

## License

Apache 2.0 — see [LICENSE](LICENSE).

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

_No changes yet._

## [0.2.0] — 2026-04-19

### Added

- **TRG-POD-EXIT-IMMEDIATE** — detects containers that exit immediately because the binary cannot be executed: exit code 126 (not executable), exit code 127 (binary not found in PATH), and `exec format error` (image built for wrong CPU architecture). Covers both regular and init containers.
- **TRG-SVC-PORT-MISMATCH** — fires when a Service's `targetPort` does not match any `containerPort` declared by the pods the Service selects. Catches typos and stale port numbers after application changes.
- **TRG-POD-BAD-ENV-REF** — fires when an env var using `configMapKeyRef` or `secretKeyRef` references a key that does not exist in an otherwise-present ConfigMap or Secret. Respects the `optional: true` flag.
- **TRG-CLUSTER-QUOTA-EXHAUSTED** — detects namespace ResourceQuotas at ≥95% utilisation (High) or 100% (Critical). Fires for namespace and cluster targets.
- **TRG-CLUSTER-APISERVER-LATENCY** — detects Warning events in `kube-system` whose reason or message matches known API server / etcd latency signals (`SlowReadResponse`, `SlowWriteResponse`, `context deadline exceeded`, etcd errors, etc.).
- `kubediag report cluster` — full markdown cluster report now implemented (was a stub returning "not yet implemented").
- Markdown report (`kubediag report namespace`, `kubediag report cluster`) now emits a **table of contents** with anchor links and structured `## Contents / ## Summary / ## Findings` sections.
- `ResourceQuota` added to the `kube.Interface`, `ResourceCache`, and `FakeClient` — available to all rules and prefetched for namespace and cluster targets.
- Collector now prefetches `ResourceQuotas` for namespace targets and `kube-system` events for cluster targets.

### Improved

- **TRG-POD-READINESS-FAILING** — now samples up to 3 distinct recent `Unhealthy` event messages per container (previously only the single most-recent event). Distinct messages are deduplicated.

## [0.1.0] — 2026-01-01

### Added

- Initial project scaffold.
- Core CLI structure: `kubediag {pod, deployment, namespace, cluster, report, rules, config, version, completion}`.
- Rule engine with built-in first-party rule set (23 rules).
- Output renderers: `text`, `json`, `markdown`.
- Configuration via `~/.config/triage/config.yaml` and `KUBEDIAG_*` env vars.
- kubectl plugin support (`kubectl-kubediag` symlink).

[Unreleased]: https://github.com/khvedela/kubediag/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/khvedela/kubediag/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/khvedela/kubediag/releases/tag/v0.1.0

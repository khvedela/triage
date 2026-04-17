---
title: Roadmap
description: Near-term and long-term roadmap for triage.
slug: /roadmap
---

# Roadmap

## v0.1.0 — First release

- Core rule engine with ~23 rules covering crash/runtime, scheduling, image pull, networking, rollout, and RBAC
- Three output formats: text (ANSI), JSON, Markdown
- Standalone binary (`triage`) and kubectl plugin (`kubectl-triage`)
- `triage pod`, `triage deployment`, `triage namespace`, `triage cluster` commands
- `triage rules list` and `triage rules explain <id>` for rule introspection
- Request-scoped ResourceCache to avoid redundant Kubernetes API calls
- GoReleaser distribution (linux/darwin/windows, amd64/arm64)
- Krew manifest for kubectl plugin installation

## v0.2.0 — Rule set expansion

- `TRG-POD-EXIT-IMMEDIATE` — container exits non-zero immediately (exec format error, missing binary)
- `TRG-SVC-PORT-MISMATCH` — service targetPort not exposed by pod
- `TRG-POD-BAD-ENV-REF` — configMapKeyRef/secretKeyRef pointing at a missing key
- `TRG-POD-READINESS-FAILING` improvements: sample readiness event messages
- Additional cluster-level rules (quota exhaustion, API server latency events)
- `triage report namespace` — full markdown report with table of contents

## v0.3.0 — YAML rule packs (external rules)

- Rule pack format: YAML-defined rules with CEL expressions for field matching
- `triage rules load ./my-rules.yaml` for custom/org-specific rules
- Rule versioning and conflict resolution
- Rule pack repository (community-contributed rules)

## v0.4.0 — Interactive mode and watch

- `triage watch pod <name>` — re-run every N seconds, diff findings
- `--since` flag: filter events newer than a duration
- Interactive selector for `triage namespace` (fzf-style picker)

## v1.0.0 — Stable API

- Rule ID and finding schema stability guarantee
- `pkg/` promotion for embedding triage as a library
- Deprecation policy enforcement (old IDs kept as aliases for one minor)
- Comprehensive e2e test suite against kind clusters
- Homebrew tap and container image

## Out of scope (v1)

- Mutating operations (triage is read-only)
- Operator/CRD-specific rules (Crossplane, Istio, cert-manager) — v2 roadmap
- LLM-powered explanation — architecture leaves a pluggable explainer hook
- Log aggregation / indexing
- Persistent state or alerting

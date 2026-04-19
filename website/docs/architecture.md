---
title: Architecture
description: Internal design of the kubediag CLI, cache, rule engine, and renderers.
slug: /architecture
---

import DocHero from "@site/src/components/docs/DocHero";
import InfoGrid from "@site/src/components/docs/InfoGrid";
import CommandBlock from "@site/src/components/docs/CommandBlock";
import ArchitectureFlow from "@site/src/components/docs/ArchitectureFlow";

<DocHero
  eyebrow="Architecture"
  title="A one-shot diagnostic pipeline, optimized for fast operator feedback."
  lede="kubediag is not a long-running service and not an informer-based control plane. It is a focused CLI that builds one diagnosis context, evaluates compiled rules against that context, ranks the findings, and renders the result in the format the next consumer expects."
  meta={["One-shot CLI", "Request-scoped cache", "Compiled Go rules", "Shared finding model"]} />

<ArchitectureFlow />

<InfoGrid
  columns={3}
  items={[
    {
      label: "Why it stays fast",
      title: "No informers",
      body: "Informer startup latency is the wrong tradeoff for a tool that aims to answer one incident question in one pass."
    },
    {
      label: "Why it stays explicit",
      title: "Rules are compiled code",
      body: "The v1 rule surface favors explicit logic, stable IDs, and testability over dynamic runtime behavior."
    },
    {
      label: "Why it stays reusable",
      title: "One finding model",
      body: "Text, JSON, and markdown renderers all consume the same ranked findings and evidence structures."
    }
  ]} />

## Control flow

<CommandBlock
  eyebrow="End-to-end pass"
  title="What happens during `kubediag pod foo -n bar`."
  description="The product is easier to reason about if you think in stages: target resolution, cache warm-up, rule evaluation, ranking, and rendering."
  command={[
    "1. Parse flags and resolve the target scope.",
    "2. Build a Kubernetes client from the CLI runtime config.",
    "3. Warm the request-scoped cache for likely-needed resources.",
    "4. Run matching rules against the cached context.",
    "5. Rank, dedupe, and filter findings.",
    "6. Render text, JSON, or markdown to stdout."
  ]}
  caption="Rules never talk to the live client directly; they read through the cache so repeated resource access stays cheap and consistent." />

## Key packages

| Package | Responsibility |
| --- | --- |
| `internal/findings` | Domain model for targets, findings, evidence, remediation, and reports |
| `internal/kube` | Kubernetes client wrapper, request-scoped cache, RBAC access helpers, fake client |
| `internal/engine` | Orchestration, prefetch, ranking, dedupe, and filtering |
| `internal/rules` | Rule registry plus one file per rule implementation |
| `internal/output` | Text, JSON, and markdown renderers |
| `internal/config` | Viper-backed config loading and precedence handling |
| `internal/cli` | Bridge between Cobra commands and the engine |

## Design decisions that matter

### Rules as Go code

The current rule surface is compiled Go because the priority is correctness, stable identifiers, and testability. YAML rule packs are a roadmap item, not a present-tense architecture constraint.

### Meaningful exit codes

The CLI returns meaningful status codes so scripts can tell the difference between “no findings,” “findings present,” usage errors, cluster access problems, and internal failures.

### One binary, two names

The same binary behaves as `triage` and `kubectl-kubediag`. That reduces packaging complexity while preserving a kubectl-native experience.

## Testing strategy

- unit tests for rules using the fake Kubernetes client and request-scoped cache
- renderer golden tests for text, JSON, and markdown output
- ranking tests for score ordering stability
- optional envtest-backed end-to-end coverage for broader behavior validation

For contribution guidance and the rule-writing workflow, continue to [Contributing](./contributing.md).

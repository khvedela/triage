---
title: Commands
description: Reference for kubediag commands, subcommands, and high-signal global flags.
slug: /commands
---

import DocHero from "@site/src/components/docs/DocHero";
import InfoGrid from "@site/src/components/docs/InfoGrid";
import CommandBlock from "@site/src/components/docs/CommandBlock";

<DocHero
  eyebrow="Reference"
  title="A small command surface, grouped by operator task."
  lede="The CLI is intentionally compact. The important distinction is not command count, but which scope you are diagnosing, how much context you want to gather, and whether the next consumer is a human or another system."
  meta={["Diagnosis scopes", "Report generation", "Rule introspection", "Config helpers"]} />

<InfoGrid
  columns={2}
  items={[
    {
      label: "Diagnose",
      title: "Scope commands",
      body: "Use pod, deployment, namespace, and cluster commands to match the size of the problem you are investigating."
    },
    {
      label: "Inspect",
      title: "Rules and config",
      body: "Use rule and config commands when you need to understand why kubediag behaves a certain way or how it is currently configured."
    },
    {
      label: "Publish",
      title: "Report output",
      body: "Use report generation when the audience is an incident document, handoff, or postmortem rather than a live terminal."
    },
    {
      label: "Integrate",
      title: "Completions and JSON",
      body: "Use completion scripts and JSON output when you want kubediag to feel native in shells and automation pipelines."
    }
  ]} />

## Diagnosis scopes

<CommandBlock
  eyebrow="Pod"
  title="Diagnose one workload instance."
  description="Best when you already know which pod is failing and want the fastest path from symptom to evidence."
  command="kubediag pod my-pod -n default"
  caption="This is the most useful first command for learning how the tool thinks." />

<CommandBlock
  eyebrow="Deployment"
  title="Diagnose rollout and owned pods together."
  description="Best when availability is failing at the controller layer and you need rollout conditions plus pod-level context in one view."
  command="kubediag deployment web -n prod" />

<CommandBlock
  eyebrow="Namespace and cluster"
  title="Widen the diagnosis surface when the incident is systemic."
  description="Best when you suspect service selection issues, namespace-wide warning events, or cluster resource pressure."
  command={[
    "kubediag namespace prod",
    "kubediag cluster"
  ]} />

## Reporting and renderers

<CommandBlock
  eyebrow="Markdown report"
  title="Generate an artifact for incident sharing."
  description="The report command forces markdown output and raises the finding cap so the result works as a handoff or post-incident summary."
  command="kubediag report namespace prod > triage-report.md"
  caption="`kubediag report cluster` is reserved for a future release and currently returns a not-yet-implemented error." />

<CommandBlock
  eyebrow="Output formats"
  title="Pick the renderer that matches the next consumer."
  description="Use terminal text for live operations, JSON for tooling, and markdown for human-readable artifacts."
  command={[
    "kubediag pod my-pod -o text",
    "kubediag pod my-pod -o json",
    "kubediag namespace prod -o markdown"
  ]} />

## Rule and configuration helpers

<CommandBlock
  eyebrow="Rules"
  title="Inspect the built-in diagnosis surface."
  description="These commands are useful when you need to understand which rules exist or explain one rule in more detail."
  command={[
    "kubediag rules list",
    "kubediag rules list --category Runtime",
    "kubediag rules explain TRG-POD-OOMKILLED"
  ]} />

<CommandBlock
  eyebrow="Config"
  title="Understand the resolved runtime configuration."
  description="These commands help when config precedence or environment overrides are part of the debugging path."
  command={[
    "kubediag config view",
    "kubediag config init",
    "kubediag config path"
  ]} />

## Global flags that matter most

| Flag | Why you would care |
| --- | --- |
| `-o, --output` | Choose text, JSON, or markdown depending on whether the consumer is a human, report, or machine. |
| `-n, --namespace` | Required for pod and deployment diagnoses unless your kube context already resolves the right namespace. |
| `--severity-min` | Useful when you want to suppress lower-priority findings during noisy incidents. |
| `--confidence-min` | Useful when you want kubediag to report only higher-certainty diagnoses. |
| `--include-events` | Controls whether related Kubernetes Events are pulled into the evidence surface. |
| `--include-related` | Controls whether related Services, PVCs, and adjacent resources are included. |
| `--max-findings` | Caps output size for dense namespace- or cluster-level runs. |
| `--timeout` | Bounds total cluster-call time for the run. |
| `--config` | Points kubediag at a specific config file instead of the default path. |

Kubernetes client flags such as `--context`, `--kubeconfig`, `--cluster`, and `--user` are also available because kubediag uses the standard Kubernetes CLI runtime stack.

## Shell integration

<CommandBlock
  eyebrow="Completion"
  title="Make the small command surface feel native."
  description="Shell completion is worth enabling because the commands are few, but the value is in fast recall during incident response."
  command={[
    "kubediag completion bash",
    "kubediag completion zsh",
    "kubediag completion fish",
    "kubediag completion powershell"
  ]} />

<CommandBlock
  eyebrow="Version"
  title="Check the binary identity."
  description="Useful when validating a release artifact or comparing behavior across environments."
  command="kubediag version" />

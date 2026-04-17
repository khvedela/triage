---
title: Quickstart
description: Run triage against a pod, deployment, namespace, or cluster and inspect the outputs.
slug: /getting-started/quickstart
---

import DocHero from "@site/src/components/docs/DocHero";
import InfoGrid from "@site/src/components/docs/InfoGrid";
import CommandBlock from "@site/src/components/docs/CommandBlock";
import TerminalFrame from "@site/src/components/TerminalFrame";

<DocHero
  eyebrow="First run"
  title="Learn the incident loop, not just the command syntax."
  lede="The fastest way to understand triage is to run it against one broken workload, inspect the top-ranked finding, and follow the next commands it suggests. That loop is the product: symptom to evidence to action without manual archaeology."
  meta={["pod, deployment, namespace, cluster", "text, json, markdown", "stable rule IDs"]} />

<InfoGrid
  columns={3}
  items={[
    {
      label: "Start narrow",
      title: "Run against one pod first",
      body: "A single-pod diagnosis makes the ranking model and evidence format obvious before you move into deployment or namespace scope."
    },
    {
      label: "Read the top finding",
      title: "Use severity and confidence as the entrypoint",
      body: "The first finding should tell you what triage believes is driving the incident and how certain it is."
    },
    {
      label: "Confirm quickly",
      title: "Paste the suggested next commands",
      body: "triage shortens the gap between diagnosis and confirmation by embedding the most useful kubectl follow-ups directly in the output."
    }
  ]} />

<CommandBlock
  eyebrow="Pod diagnosis"
  title="Start with the smallest useful scope."
  description="For a broken workload, the pod view is the quickest way to see how triage structures a diagnosis."
  command="triage pod my-pod -n default"
  caption="If the problem is rollout- or service-shaped rather than pod-shaped, move up to deployment or namespace scope after this first run." />

<TerminalFrame label="triage:text">
{`▶ Pod default/my-api-7f9b-xk2m2      Phase: Running     Ready: 0/1

ⓧ CRITICAL  [high confidence]  TRG-POD-CRASHLOOPBACKOFF
  Container "api" is in CrashLoopBackOff (5 restarts in the last 3m)

  Evidence:
    • pod.status.containerStatuses[0].lastState.terminated.reason = "Error"
    • pod.status.containerStatuses[0].lastState.terminated.exitCode = 1
    • Event (Warning, BackOff, 30s ago): "Back-off restarting failed container"

  Next commands:
    $ kubectl logs -n default my-api-7f9b-xk2m2 -c api --previous
    $ kubectl describe pod -n default my-api-7f9b-xk2m2`}
</TerminalFrame>

## Expand scope when the incident shape changes

<CommandBlock
  eyebrow="Deployment scope"
  title="Use deployment diagnosis when the failure is about rollout progress."
  description="This view combines deployment-level findings and the pod-level signals underneath them."
  command="triage deployment web -n prod" />

<CommandBlock
  eyebrow="Namespace and cluster scope"
  title="Use wider scopes for fleet-wide health signals."
  description="Namespace and cluster modes surface warning events, service issues, and node pressure patterns that are easy to miss when starting from one pod."
  command={[
    "triage namespace prod",
    "triage cluster"
  ]} />

## Switch renderer based on audience

<CommandBlock
  eyebrow="Machine-readable"
  title="Choose the renderer that matches the next consumer."
  description="Use JSON for automation, markdown for reports, and terminal text for live incident response."
  command={[
    "triage pod my-pod -o json",
    "triage namespace prod -o markdown",
    "triage report namespace prod > triage-report.md"
  ]} />

## Inspect and explain rules

<CommandBlock
  eyebrow="Rule introspection"
  title="Use stable rule IDs as a reference surface."
  description="Rules are public identifiers, which makes them useful in alerts, runbooks, and postmortems."
  command={[
    "triage rules list",
    "triage rules explain TRG-POD-CRASHLOOPBACKOFF"
  ]} />

## Keep the feedback loop tight

1. Run `triage` on the narrowest scope that still contains the incident.
2. Read the highest-ranked finding and the evidence it cites.
3. Paste the suggested next commands to confirm or falsify the diagnosis.
4. Apply the fix and rerun the same command to see whether the finding clears.

If you want a guided browser version of that flow, use the [interactive sandbox](/sandbox).

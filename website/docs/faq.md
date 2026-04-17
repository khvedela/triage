---
title: FAQ
description: Common questions about triage scope, safety, installation, and rule behavior.
slug: /faq
---

# FAQ

## Does triage mutate cluster resources?

No. `triage` is read-only. It inspects cluster state and emits findings, evidence, and follow-up commands.

## Is triage a kubectl plugin or a standalone CLI?

Both. The same binary works as `triage` and as `kubectl-triage`.

## Does triage replace kubectl?

No. It narrows the search space and tells you which `kubectl` commands are most useful next.

## How does triage rank findings?

Findings are ranked by severity and confidence, with ties broken consistently by rule metadata. The goal is to show the most incident-driving diagnosis first.

## Does triage support machine-readable output?

Yes. Use `-o json` for automation and `-o markdown` or `triage report namespace` for report writing.

## Why are there rule IDs?

Rule IDs are public, stable identifiers that teams can reference in runbooks, alerts, and automation.

## What happens if RBAC blocks some resources?

triage emits an informational access finding explaining that the diagnosis may be incomplete rather than fabricating certainty.

## Can I use triage in CI or incident bots?

Yes. The JSON output, stable IDs, and meaningful exit codes are designed for that style of integration.

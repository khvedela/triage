---
title: Install
description: Install triage from signed release binaries, source, or as a kubectl plugin.
slug: /getting-started/install
---

import DocHero from "@site/src/components/docs/DocHero";
import InfoGrid from "@site/src/components/docs/InfoGrid";
import CommandBlock from "@site/src/components/docs/CommandBlock";

<DocHero
  eyebrow="Getting started"
  title="Install triage for fast incident diagnosis."
  lede="The release flow is intentionally simple: grab a signed binary, install from source, or expose the same binary as a kubectl plugin. The right choice depends on how much control your environment expects over provenance and update cadence."
  meta={["Signed release assets", "SBOMs published", "Plugin-compatible binary"]} />

<InfoGrid
  columns={3}
  items={[
    {
      label: "Fastest path",
      title: "Binary download",
      body: "Best when you want a known release artifact with checksums, signatures, and no build step in the critical path."
    },
    {
      label: "Developer path",
      title: "Go install",
      body: "Best when you already manage Go tooling locally and want the public module path as the source of truth."
    },
    {
      label: "kubectl workflow",
      title: "Plugin mode",
      body: "Best when operators already live inside kubectl and want triage to feel like part of the native CLI surface."
    }
  ]} />

<CommandBlock
  eyebrow="Binary install"
  title="Use GitHub Releases when provenance matters."
  description="GitHub Releases is the canonical distribution channel. Each release includes platform archives plus checksum, signature, and certificate material."
  command={[
    "curl -L https://github.com/khvedela/triage/releases/latest/download/triage_darwin_arm64.tar.gz | tar xz",
    "chmod +x triage",
    "mv triage /usr/local/bin/triage"
  ]}
  caption="Swap the archive name for the platform you actually run. Published artifacts currently cover linux amd64/arm64, darwin amd64/arm64, and windows amd64." />

<CommandBlock
  eyebrow="Source install"
  title="Use the public module path when source-based tooling is standard."
  description="This is the cleanest option for local development machines and internal environments that already trust Go modules."
  command="go install github.com/khvedela/triage@latest"
  caption="The binary lands in your Go bin directory and carries the same CLI behavior as the release build." />

<CommandBlock
  eyebrow="kubectl plugin mode"
  title="Expose the same binary as kubectl-triage."
  description="triage does not need a second build target to behave like a kubectl plugin. The binary checks how it was invoked and switches its command surface accordingly."
  command={[
    "ln -s \"$(which triage)\" ~/.local/bin/kubectl-triage",
    "kubectl triage pod my-pod -n default"
  ]}
  caption="This is useful for teams that want a kubectl-native incident flow without maintaining a separate plugin implementation." />

## Verify release integrity

Each release publishes:

- `checksums.txt`
- `checksums.txt.sig`
- `checksums.txt.pem`
- per-archive SBOM files

Use those artifacts with your normal verification flow. The important point is not the specific verification tool, but that binary distribution, provenance, and dependency disclosure are already part of the release surface.

## Homebrew and Krew status

Homebrew tap publication and Krew index publication are planned, but not live yet. The repository already carries the release structure and Krew manifest needed to support those channels later, so binary install and `go install` are the production-ready paths today.

## Choose the next step

Continue to [Quickstart](./quickstart.md) to see how the first diagnosis flow should feel once the binary is installed.

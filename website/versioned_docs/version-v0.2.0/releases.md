---
title: Releases
description: Artifact layout, signatures, SBOMs, and release distribution notes.
slug: /releases
---

import DocHero from "@site/src/components/docs/DocHero";
import InfoGrid from "@site/src/components/docs/InfoGrid";
import CommandBlock from "@site/src/components/docs/CommandBlock";

<DocHero
  eyebrow="Distribution"
  title="GitHub-native releases with trust material built in."
  lede="kubediag releases are cut by GitHub Actions and GoReleaser when a semantic version tag is pushed. The important outcome is not only the binary archive, but the supporting material around it: checksums, signatures, certificates, and SBOMs."
  meta={["GitHub Actions", "GoReleaser", "Checksums + signatures", "SBOMs per archive"]} />

<InfoGrid
  columns={3}
  items={[
    {
      label: "Artifacts",
      title: "Platform archives",
      body: "Each release ships archives for Linux amd64/arm64, macOS amd64/arm64, and Windows amd64."
    },
    {
      label: "Trust",
      title: "Checksums and signatures",
      body: "Checksum and cosign artifacts make the release verifiable in environments that care about provenance and supply-chain policy."
    },
    {
      label: "Inventory",
      title: "SBOMs",
      body: "Per-archive SBOM output gives downstream consumers dependency visibility without running a second scanning pipeline."
    }
  ]} />

## Latest release

**v0.2.0** — Rule set expansion. Five new diagnostic rules, `kubediag report cluster`, markdown table of contents, and richer readiness probe sampling. See the [changelog](https://github.com/khvedela/kubediag/blob/main/CHANGELOG.md) for the full list.

## Release history

| Version | Highlights |
| --- | --- |
| **v0.2.0** | TRG-POD-EXIT-IMMEDIATE, TRG-SVC-PORT-MISMATCH, TRG-POD-BAD-ENV-REF, TRG-CLUSTER-QUOTA-EXHAUSTED, TRG-CLUSTER-APISERVER-LATENCY, `report cluster`, markdown ToC |
| **v0.1.0** | Initial release — 23 rules, text/JSON/markdown output, kubectl plugin |

## Artifact names

```text
kubediag_linux_amd64.tar.gz
kubediag_linux_arm64.tar.gz
kubediag_darwin_amd64.tar.gz
kubediag_darwin_arm64.tar.gz
kubediag_windows_amd64.zip
checksums.txt
checksums.txt.sig
checksums.txt.pem
```

## Verification flow

<CommandBlock
  eyebrow="Release verification"
  title="Treat GitHub Releases as the canonical artifact source."
  description="The release feed is the source of truth for binary download, checksum verification, signature validation, and future packaging integrations."
  command="https://github.com/khvedela/kubediag/releases"
  caption="Whether you mirror artifacts internally or install directly from GitHub, the verification story should anchor on this release feed." />

## Packaging status

| Channel | Status |
| --- | --- |
| GitHub Releases | Available |
| Prebuilt binary archives | Available |
| Checksums, signatures, certificates | Available |
| SBOMs | Available |
| Homebrew tap | Planned |
| Krew index publication | Planned |

## Why this matters operationally

For infrastructure tooling, install instructions are not enough. Teams also need to know:

- where the artifact came from
- how to verify it
- whether dependency inventory is published
- which distribution channel is actually authoritative

That is why the docs site treats release trust as a first-class product surface rather than burying it under a short download paragraph.

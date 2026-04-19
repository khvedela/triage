---
title: Contributing
description: Build, test, and contribute rules to triage.
slug: /contributing
---

# Contributing

Contributions are welcome. `triage` is built around a rule registry, request-scoped Kubernetes cache, and renderer pipeline designed to keep new diagnostics straightforward to add.

## Development setup

Prerequisites:

- Go 1.22 or newer
- `make`
- `golangci-lint`
- A Kubernetes cluster for manual validation (`kind`, `k3d`, or `minikube` all work)

Clone and build:

```sh
git clone https://github.com/khvedela/triage
cd triage
make build
make test
make lint
```

## Repository layout

| Path | Responsibility |
| --- | --- |
| `cmd/` | Cobra commands and CLI entrypoints |
| `internal/engine` | Orchestration, ranking, dedupe, filtering |
| `internal/kube` | Kubernetes client wrapper and request-scoped cache |
| `internal/rules` | Rule implementations |
| `internal/findings` | Shared domain types |
| `internal/output` | Text, JSON, and Markdown renderers |
| `website/` | Docusaurus site and interactive sandbox |

## Adding a new rule

1. Pick a stable rule ID, such as `TRG-POD-BAD-COMMAND`.
2. Implement the rule in `internal/rules/`.
3. Add a focused unit test.
4. Regenerate docs with `make docgen` if the rule registry changed.
5. Add or update an example manifest when the failure mode is reproducible.

Rule guidance:

- Evidence should be explicit and defensible.
- Remediation should be actionable.
- Rule IDs are public API and should not be renamed casually.
- RBAC limitations must degrade gracefully without false positives.

## Tests

```sh
make test
make test-short
make test-e2e
make cover
```

Renderer golden files regenerate with:

```sh
go test ./internal/output/... -update
```

## Docs site workflow

The public docs site lives in `website/`.

```sh
cd website
npm install
npm run start
```

Build the static site:

```sh
npm run build
```

## Pull requests

1. Create a branch.
2. Add code, tests, and doc updates.
3. Run `make lint test`.
4. Sign off commits with `git commit -s`.
5. Open a pull request describing the problem and validation steps.

## Security and conduct

- Security issues: see [SECURITY.md](https://github.com/khvedela/triage/blob/main/SECURITY.md)
- Contributor conduct: see [CODE_OF_CONDUCT.md](https://github.com/khvedela/triage/blob/main/CODE_OF_CONDUCT.md)

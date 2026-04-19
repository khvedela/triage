# Contributing to `triage`

Thanks for your interest! This document covers how to build, test, and add new rules.

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). By participating, you agree to uphold it.

## Developer Certificate of Origin (DCO)

We use [DCO](https://developercertificate.org/) instead of a CLA. Sign off your commits:

```sh
git commit -s -m "rules: add TRG-POD-FOO"
```

This appends a `Signed-off-by:` trailer asserting that you have the right to submit your contribution under the project's license.

---

## Development setup

Prerequisites:

- Go ≥ 1.22
- `make`
- `golangci-lint` ([install](https://golangci-lint.run/usage/install/))
- A Kubernetes cluster for manual testing (kind / k3d / minikube all work)

Clone and build:

```sh
git clone https://github.com/khvedela/kubediag
cd kubediag
make build       # builds ./bin/kubediag
make test        # unit tests
make lint        # golangci-lint
```

Run against your current kubeconfig context:

```sh
./bin/kubediag pod <some-pod> -n <ns> --debug
```

---

## Repository layout

See [docs/architecture.md](docs/architecture.md) for an overview. High-level:

```
cmd/               Cobra commands (thin)
internal/engine    Orchestration: target → collector → rules → ranker
internal/kube      client-go wrapper + request-scoped cache + RBAC probe
internal/rules     One file per rule; registered via init()
internal/findings  Domain types: Finding, Evidence, Remediation, Target
internal/output    Renderers: text, json, markdown
internal/config    Viper-backed configuration
test/e2e           envtest-based end-to-end tests
```

---

## Adding a new rule

Rules are the heart of the project. Adding a good one is ~50 lines of code + a unit test.

1. Pick a stable ID: `TRG-<SCOPE>-<SHORT-SLUG>` (e.g., `TRG-POD-BAD-COMMAND`).
2. Create `internal/rules/pod_bad_command.go`:

```go
package rules

import (
    "context"
    "fmt"

    "github.com/khvedela/kubediag/internal/findings"
)

func init() { Register(&podBadCommand{}) }

type podBadCommand struct{}

func (r *podBadCommand) Meta() findings.RuleMeta {
    return findings.RuleMeta{
        ID:          "TRG-POD-BAD-COMMAND",
        Title:       "Container exited with exec/command failure",
        Category:    findings.CategoryConfiguration,
        Severity:    findings.SeverityHigh,
        Scopes:      []findings.TargetKind{findings.TargetKindPod},
        Description: "Container's ENTRYPOINT/args likely refer to a binary that isn't in the image or isn't executable.",
        DocsLinks:   []string{"https://kubernetes.io/docs/concepts/containers/images/"},
        Priority:    10,
    }
}

func (r *podBadCommand) Evaluate(ctx context.Context, rc *RuleContext) ([]findings.Finding, error) {
    pod, ok, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
    if err != nil || !ok {
        return nil, err
    }
    // ... pattern-match on containerStatuses[*].lastState.terminated
    // ... return []findings.Finding{...}
    _ = fmt.Sprintf // placeholder
    return nil, nil
}
```

3. Unit test in `internal/rules/pod_bad_command_test.go`:

```go
func TestPodBadCommand(t *testing.T) {
    cases := []struct {
        name    string
        pod     *corev1.Pod
        wantIDs []string
    }{
        { /* ... */ },
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) { /* ... */ })
    }
}
```

4. Run `make docgen` — regenerates `docs/rules.md` from the registry.
5. Add a positive-case fixture manifest to `examples/` if the failure mode is reproducible.

### Rule design principles

- **Be specific about confidence.** `High` confidence means: *no other rule could plausibly fire for this exact signal*. If a symptom could have multiple causes, use `Medium` or `Low`.
- **Evidence is contract-grade.** Every finding must cite at least one `Evidence`. Ambiguous "something is wrong" findings without evidence are not accepted.
- **Remediation must be actionable.** `NextCommands` should be pasteable as-is. `SuggestedFix` should tell the user *what to change*, not "check your config".
- **Degrade gracefully on RBAC.** If your rule needs a resource and `cache.Get*` returns `forbidden`, emit no false-positives. The `TRG-ACCESS-INSUFFICIENT-READ` meta-finding already signals to the user why diagnosis was incomplete.

### Rule ID stability & deprecation

Rule IDs are a **public API**. Users grep for them, automate against them, and reference them in runbooks. The policy:

1. Never rename a rule ID without keeping the old ID as an alias for at least one minor version.
2. Retire an alias only after it has carried a deprecation notice for a full minor version.
3. Document retired IDs in [docs/rules.md](docs/rules.md) under a "Retired" section.

---

## Running tests

```sh
make test        # unit tests with race detector
make test-short  # skip slow/envtest tests
make test-e2e    # full envtest-backed e2e (downloads kube-apiserver binary on first run)
make cover       # coverage report → coverage.html
```

Golden-test outputs for renderers regenerate with:

```sh
go test ./internal/output/... -update
```

Review the diffs; commit when you're satisfied.

---

## Submitting a pull request

1. Fork and create a feature branch.
2. Make your change, add tests, update `docs/rules.md` via `make docgen` if applicable.
3. Run `make lint test`.
4. Sign off your commits (`git commit -s`).
5. Open a PR. Describe the problem being solved and how you verified the fix.

We aim to respond to PRs within a week. Please be patient with reviewers.

## Docs site

The GitHub Pages site lives under `website/`.

```sh
cd website
npm install
npm run start
```

Build the static output with:

```sh
npm run build
```

## Releasing (maintainers)

1. Update `CHANGELOG.md`.
2. Tag: `git tag -a vX.Y.Z -m "vX.Y.Z" && git push --tags`.
3. GitHub Actions runs `goreleaser` and publishes artifacts + SBOM + signatures.

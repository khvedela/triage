# Architecture

## Overview

```
┌──────────────────┐   ┌──────────────────┐   ┌────────────────┐
│  cmd / Cobra CLI │──▶│  engine (orches) │──▶│  rules         │
└──────────────────┘   └────┬─────────────┘   └────────────────┘
                            │  collects via
                            ▼
                       ┌──────────────┐
                       │  kube client │   (client-go, no informers)
                       │  + cache     │
                       └──────────────┘
                            │
                            ▼ produces
                       ┌──────────────┐      ┌─────────────┐
                       │  Findings    │─────▶│  renderers  │ text/json/md
                       └──────────────┘      └─────────────┘
```

## Control flow

For `triage pod foo -n bar`:

1. `cmd/pod.go` parses flags and builds `Target{Kind:Pod, Namespace:"bar", Name:"foo"}`.
2. `internal/cli.RunDiagnosis()` creates a `kube.Client` from the kubeconfig flags and calls `engine.Run()`.
3. The engine selects rules whose `Scopes` include `Pod`, then calls `Collector.Prefetch()` to warm the cache for likely-needed resources.
4. Each rule's `Evaluate(ctx, rc)` is called sequentially. Rules read from `rc.Cache` — never the live Kubernetes client directly.
5. Findings are collected, stamped with timestamps and defaults, then passed through `engine.Rank()` and `engine.Dedupe()`.
6. The renderer (text/json/markdown) writes the final output to stdout.

## Key packages

### `internal/findings`

Domain model: `Target`, `Finding`, `Evidence`, `Remediation`, `Report`. All downstream packages import this. Zero Kubernetes dependencies.

### `internal/kube`

Thin wrapper around client-go:
- `Interface` — 21-method interface for the Kubernetes client (testable via `FakeClient`).
- `ResourceCache` — request-scoped cache keyed by `(kind, namespace, name)`. All `Get*` and `List*` methods memoize results. Eliminates redundant API calls when multiple rules need the same resource.
- `access.go` — `CanI()` using `SelfSubjectAccessReview`.
- `errors.go` — `IsForbidden()`, `IsNotFound()` helpers.
- `fake.go` — `FakeClient` for unit tests.

### `internal/engine`

- `engine.go` — `Run()` orchestrates: build cache → prefetch → evaluate rules → rank → dedupe → filter.
- `collector.go` — `Collector.Prefetch()` advisory warm-up based on target kind.
- `ranker.go` — `Rank()`: score = `severity×1000 + confidence×100`, ties broken by rule ID.

### `internal/rules`

- `registry.go` — global rule registry (`Register`, `All`, `Get`). Rules self-register via `init()`.
- One file per rule, named `<category>_<what>.go`. Each file ends with `func init() { Register(&myRule{}) }`.
- `Context` struct — carries `Target`, `Cache`, `Logger`, `Now`. Passed to every rule.

### `internal/output`

- `text.go` — ANSI-colored terminal renderer.
- `json.go` — JSON renderer (schema-stable).
- `markdown.go` — Markdown renderer for `triage report`.
- `output.go` — `ParseFormat()`, `Render()` dispatcher, `RenderOptions`.

### `internal/config`

Viper-backed config loader. Handles flag → env → file → default precedence.

### `internal/cli`

- `exitcodes.go` — exit code constants (0=ok, 1=findings, 2=usage, 3=cluster, 10=internal).
- `color.go` — `ResolveColor()`: single authority for color on/off.
- `options.go` — `Options` struct, context-key injection.
- `run.go` — `RunDiagnosis()` — the bridge between Cobra commands and the engine.

## Rule writing guide

See [CONTRIBUTING.md](../CONTRIBUTING.md) for a full guide. Quick summary:

```go
func init() { Register(&myRule{}) }

type myRule struct{}

func (r *myRule) Meta() findings.RuleMeta {
    return findings.RuleMeta{
        ID:       "TRG-POD-MY-RULE",
        Title:    "Short headline",
        Category: findings.CategoryRuntime,
        Severity: findings.SeverityHigh,
        Scopes:   []findings.TargetKind{findings.TargetKindPod},
        Description: `Multiline description shown by triage rules explain.`,
        Priority: 80,
    }
}

func (r *myRule) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
    pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
    if err != nil || !found {
        return nil, err
    }
    // ... pattern match → return findings
}
```

## Testing

- **Unit tests**: `internal/rules/*_test.go` — use `kube.FakeClient` and `kube.NewResourceCache`.
- **Renderer tests**: `internal/output/output_test.go` — golden files in `internal/output/testdata/`. Regenerate with `go test ./internal/output/... -update`.
- **Engine tests**: `internal/engine/ranker_test.go` — test ranking stability and score ordering.
- **E2E** (optional, slow): `test/e2e/` using `envtest`. Run with `make test-e2e`.

## Design decisions

**No informers.** triage is a one-shot CLI. Informers have startup latency (~1s) which is unacceptable for a diagnostic tool targeting sub-2s response times. Plain `Get`/`List` calls with the request-scoped cache are sufficient.

**Rules as Go code, not YAML.** For v1, rules are statically compiled Go. YAML rule packs are roadmapped for v0.3.0. The `Rule` interface makes this extensible without changing the engine.

**Single binary, two names.** The same binary behaves as `kubectl-triage` when `os.Args[0]` starts with `kubectl-` or `KREW_PLUGIN_NAME` is set. No conditional compilation.

**Color from one place.** `internal/cli.ResolveColor()` is the only function that decides color on/off. It reads isatty, `NO_COLOR`, `FORCE_COLOR`, config, and `--no-color`. All renderers receive a bool.

**Exit codes are meaningful.** `0` = no findings at or above threshold. `1` = findings present. `2` = CLI usage error. `3` = cluster access error. `10` = internal error. Scripts can `if triage pod foo; then ...` reliably.

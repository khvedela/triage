## Summary

<!-- What does this PR do? One paragraph max. -->

## Type of change

- [ ] Bug fix (non-breaking)
- [ ] New feature (non-breaking)
- [ ] Breaking change (rule ID rename, flag removal, schema change)
- [ ] New diagnostic rule
- [ ] Documentation / tooling

## New or modified rules

<!-- If adding/modifying a rule, fill in: -->
| Rule ID | What it detects | Severity | Confidence |
|---------|-----------------|----------|------------|
| | | | |

## Testing

- [ ] Unit tests added / updated (`go test ./...` passes)
- [ ] Golden files regenerated if renderers changed (`go test ./internal/output/... -update`)
- [ ] Manual test against a real or kind cluster (describe briefly)

## Checklist

- [ ] `make lint` clean
- [ ] `make vet` clean
- [ ] No new exported symbols in `internal/` that aren't necessary
- [ ] Rule ID is stable (not a rename of an existing ID without an alias)
- [ ] CHANGELOG.md updated (for user-facing changes)

## Breaking changes

<!-- If this is a breaking change, describe the migration path. -->

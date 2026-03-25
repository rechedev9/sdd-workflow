# Proposal: Model Routing por Fase

**Change**: model-routing
**Date**: 2026-03-25
**Status**: proposed

## Intent

Allow SDD users to configure which LLM model (opus, sonnet, haiku) each pipeline phase uses, reducing cost by routing mechanical phases to cheaper models while preserving reasoning quality for complex phases.

## Scope

### In Scope
- New `models` section in `openspec/config.yaml` with `default` + per-phase overrides
- `ModelFor(phase)` method on `Config` struct
- `<!-- sdd:model=X -->` directive injected at top of assembled context output
- Model name validation (allowlist: opus, sonnet, haiku)
- Model field in `PhaseAssembled` event payload

### Out of Scope
- Automatic model selection based on phase complexity
- Token budget enforcement per model
- Dashboard model-per-phase display (deferred)
- Compact mode directive stripping (deferred)

## Approach

**Approach A: HTML comment directive** (selected from exploration)

1. **Config extension**: Add `Models` struct to `config/types.go` — `Default string` + `Phases map[string]string`
2. **Resolution method**: `Config.ModelFor(phase) string` returns phase-specific model or default or empty string
3. **Injection point**: `context.Assemble()` prepends `<!-- sdd:model=X -->` before assembled content (both cached and fresh paths)
4. **Validation**: `config.Load()` validates model names against `validModels` set
5. **Telemetry**: Include resolved model in `PhaseAssembled` event payload

## Config Format

```yaml
models:
  default: sonnet
  phases:
    propose: opus
    spec: opus
    design: opus
    review: opus
```

Omitting `models` entirely = no directive emitted (backward compatible).

## Files to Modify

| File | Change |
|------|--------|
| `internal/config/types.go` | Add `Models` struct, add `Models` field to `Config` |
| `internal/config/config.go` | Add `ModelFor()` method, add validation in `Load()` |
| `internal/context/context.go` | Prepend model directive in `Assemble()` |
| `internal/events/broker.go` | Add `Model` field to `PhaseAssembledPayload` |
| `openspec/config.yaml` | Add example `models` section |

## Risks

| Risk | Mitigation |
|------|------------|
| Model names change (new models released) | Allowlist is a simple string set — trivial to extend |
| Cache serves stale directive if model config changes | Directive prepended outside cache; cache content unchanged |
| Consumers ignore directive | Directive is advisory; system works without it |

## Rollback Plan

Revert the 4 modified files. No schema migration, no state change, no external dependency. Config is additive — removing `models:` section restores prior behavior.

## Success Criteria

- `sdd context model-routing propose` output starts with `<!-- sdd:model=opus -->`
- `sdd context model-routing apply` output starts with `<!-- sdd:model=sonnet -->` (or no directive if default matches)
- Config without `models` section produces no directive (zero regression)
- `make check` passes

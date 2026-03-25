# Exploration: Model Routing por Fase

**Date**: 2026-03-25T18:50:00+01:00
**Detail Level**: standard
**Change Name**: model-routing

## Current State

The CLI assembles context per-phase via `context.Assemble()` and writes to stdout. Config is parsed from `openspec/config.yaml` into `config.Config` struct and injected into every assembler via `phase.AssemblerParams.Config`. There is **no model selection mechanism** — all phases implicitly use whatever model the caller (Claude Code) happens to be running.

The `commands/*.md` slash commands invoke skills which call `sdd context <name>`, but never pass a model directive. The Agent tool in Claude Code accepts a `model` parameter — the missing piece is telling it *which* model to use per phase.

## Relevant Files

| File Path | Purpose | Lines | Complexity | Test Coverage |
|-----------|---------|-------|------------|---------------|
| `internal/config/types.go` | Config struct definitions | 36 | low | yes |
| `internal/config/config.go` | Load/Save/Detect config | 142 | medium | yes |
| `internal/context/context.go` | Central assembler dispatch + caching | 301 | medium | yes |
| `internal/phase/phase.go` | Phase descriptor + AssemblerParams | 130 | low | yes |
| `internal/cli/cmd_context.go` | `sdd context` command handler | ~139 | low | yes |
| `openspec/config.yaml` | Live project config | 11 | low | N/A |

## Dependency Map

```
config.Config (types.go)
  ← loaded by config.Load() (config.go)
  ← injected into phase.AssemblerParams (phase.go:24-33)
  ← consumed by context.Assemble() (context.go:64)
  ← each phase assembler (explore.go, propose.go, etc.)
  ← called from cmd_context.go → writes to stdout
  ← consumed by commands/*.md → Claude reads output
```

## Data Flow

1. `sdd context model-routing` invoked
2. `cmd_context.go` loads `config.yaml` → `config.Config`
3. Creates `AssemblerParams{Config: cfg, ...}`
4. Calls `context.Assemble(stdout, phase, params)`
5. Assembler writes sections: `--- SKILL ---`, `--- PROJECT ---`, `--- CHANGE ---`, etc.
6. Output goes to stdout → Claude Code consumes it
7. **Gap**: No model directive in output; no way for caller to know which model to use

## Risk Assessment

| Dimension | Level | Notes |
|-----------|-------|-------|
| Blast radius | low | Config struct + 1 injection point in Assemble() |
| Type safety | low | New struct fields, strongly typed |
| Test coverage | low | Config load/save tests exist; add model directive test |
| Coupling | low | Config flows one-way into assemblers; no reverse dep |
| Complexity | low | ~30 lines of new code total |
| Data integrity | low | Config is read-only during assembly |
| Breaking changes | low | New optional YAML fields; existing configs work unchanged |
| Security surface | low | Model names are validated against allowlist |

## Approach Comparison

| Approach | Pros | Cons | Effort | Risk |
|----------|------|------|--------|------|
| **A: HTML comment directive** `<!-- sdd:model=X -->` injected at top of assembled context by `Assemble()` | Minimal code change (1 injection point). Preserved through cache. Invisible to non-LLM consumers. | Implicit — requires `commands/*.md` to parse it. | Low | Low |
| **B: Explicit `--- MODEL ---` section** | Consistent with existing section pattern. Easy to grep. | One more section in every output. May confuse users. | Low | Low |
| **C: JSON field in `--json` output** | Machine-parseable. Clean separation of data vs content. | Only works with `--json` flag. Doesn't help plain output. | Low | Medium |
| **D: `sdd model <name> [phase]` query command** | Clean API. Caller asks for model before assembling. | Extra CLI call per phase. Two round-trips. | Medium | Low |

## Recommendation

**Approach A** (HTML comment directive) with validation.

Design:
1. Add `Models` struct to `config/types.go` with `Default` + per-phase map
2. Add `ModelFor(phase string) string` method on `Config`
3. Inject `<!-- sdd:model=X -->` as first line of `Assemble()` output
4. Validate model names: only `opus`, `sonnet`, `haiku` allowed
5. Include model in `PhaseAssembled` event payload for telemetry
6. Cache invalidation: model config change does NOT bust cache (directive is prepended to cached output too)

Config format:
```yaml
models:
  default: sonnet
  phases:
    propose: opus
    spec: opus
    design: opus
    review: opus
```

Estimated changes: ~60 lines production code, ~40 lines test code, 4 files modified.

## Open Questions (DEFERRED)

- Should `sdd doctor` warn if model config references unknown phase names?
- Should the dashboard show model-per-phase in the pipeline progress table?
- Should compact mode strip the model directive (save tokens)?

---
id: 0002
title: Pipeline phases as DAG with parallel spec+design
status: implemented
change: concurrency-performance
date: 2026-03-21
supersedes: ~
superseded-by: ~
---

## Decision

The 10-phase pipeline is a directed acyclic graph (DAG), not a strict linear sequence. Specifically, `spec` and `design` are independent branches that both require `propose` to complete, but can run in parallel with each other. `tasks` requires both `spec` AND `design` to complete before it can start.

```
explore → propose → spec ──┐
                    design ─┴→ tasks → apply → review → verify → clean → archive
```

All other transitions are strictly sequential.

## Context

Before this decision, all phases ran strictly in sequence. `spec` and `design` are semantically independent — one produces requirements, the other produces architecture. Running them sequentially wastes wall-clock time and burns an extra context assembly round-trip.

The DAG model was chosen to capture the natural dependency structure of the pipeline, not to maximize parallelism for its own sake.

## Alternatives Considered

- **Fully linear pipeline** — rejected because spec and design have no causal dependency on each other; sequencing them is artificial.
- **Full parallelism after propose** — rejected because `tasks` is causally dependent on both; the DAG models the real dependency, no more.
- **User-configurable DAG** — rejected as over-engineering; the phase dependency structure is fixed and known.

## Consequences

**Positive:**
- `sdd context <name>` detects the `spec+design` window via `ReadyPhases()` and calls `AssembleConcurrent`, assembling both contexts in parallel goroutines.
- Reduces time-to-context for the spec+design phase window.
- DAG structure is self-documenting in the phase registry.

**Negative:**
- Slightly more complex state machine (`ReadyPhases()` must return multiple phases).
- Concurrent assembly output must be written in deterministic order to avoid non-reproducible context.

## References

- Change: `openspec/changes/archive/2026-03-21-...-concurrency-performance/`
- Code: `internal/state/state.go` — `ReadyPhases()`, `internal/context/context.go` — `AssembleConcurrent`
- Docs: `docs/architecture.md` — "State Machine", "AssembleConcurrent"

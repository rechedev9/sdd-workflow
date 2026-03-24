---
id: 0001
title: Go owns state, Claude owns content
status: implemented
change: phase-interface
date: 2026-03-21
supersedes: ~
superseded-by: ~
---

## Decision

The CLI binary (Go) is responsible for all state persistence, phase transitions, caching, and quality gates. The LLM (Claude) is responsible exclusively for generating phase artifacts. Neither crosses into the other's domain.

In practice: Claude writes output to `.pending/{phase}.md`. Go promotes it to the final location and advances `state.json`. Go never generates artifact content; Claude never mutates state.

## Context

A pipeline with 10 phases and multiple quality gates needs reliable state tracking that survives crashes, network failures, and partial runs. LLMs are stateless — they cannot reliably track which phases are complete or enforce transition prerequisites. A deterministic Go binary can.

The alternative of letting Claude manage its own pipeline state leads to drift, inconsistency, and difficult recovery when sessions are interrupted.

## Alternatives Considered

- **Claude manages its own state via memory files** — rejected because memory is lossy across sessions and cannot enforce atomic transitions.
- **External process manager (e.g. Temporal, Prefect)** — rejected because it adds infrastructure overhead for a single-user CLI tool.
- **Hybrid: Go validates, Claude writes state.json** — rejected because it creates split ownership and ambiguous failure modes.

## Consequences

**Positive:**
- Crash recovery is deterministic (`Recover()` rebuilds from artifact presence on disk).
- Phase prerequisites are mechanically enforced — impossible to run `apply` without `tasks` completed.
- Zero LLM tokens spent on state management.
- Go binary is the integration test surface for the entire pipeline.

**Negative:**
- Two moving parts (binary + LLM) must stay in sync on the phase contract.
- Adding a new phase requires Go code changes (phase registry) + a new SKILL.md.

## References

- Change: `openspec/changes/archive/2026-03-21-...-phase-interface/`
- Code: `internal/state/state.go`, `internal/artifacts/promote.go`
- Docs: `docs/architecture.md` — "Key Boundary: Go Owns State, Claude Owns Content"

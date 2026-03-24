# Architecture Decision Records

Navigable index of architectural decisions made in the sdd-cli project.
Each entry links to a full ADR with context, alternatives, and consequences.

Status values: `proposal` | `accepted` | `implemented` | `superseded`

| ID | Title | Status | Change | Date |
|----|-------|--------|--------|------|
| [0001](0001-go-owns-state-claude-owns-content.md) | Go owns state, Claude owns content | implemented | phase-interface | 2026-03-21 |
| [0002](0002-pipeline-phases-dag-parallel-spec-design.md) | Pipeline phases as DAG with parallel spec+design | implemented | concurrency-performance | 2026-03-21 |
| [0003](0003-content-hash-cache-per-phase-ttl.md) | Content-hash caching with per-phase TTLs | implemented | concurrency-performance | 2026-03-21 |
| [0004](0004-context-cascade-cumulative-summary.md) | Context cascade: cumulative summary forwarded downstream | implemented | context-cascade | 2026-03-21 |
| [0005](0005-dashboard-offscreen-canvas-hud-dirty-flag.md) | Dashboard: offscreen canvas + HUD dirty flag | implemented | dashboard-visual-polish | 2026-03-23 |

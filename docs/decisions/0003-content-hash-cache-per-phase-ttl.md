---
id: 0003
title: Content-hash caching with per-phase TTLs
status: implemented
change: concurrency-performance
date: 2026-03-21
supersedes: ~
superseded-by: ~
---

## Decision

Context assembly results are cached on disk using a content-hash key. The hash covers: format version prefix + SKILL.md bytes + sorted artifact bytes for all phase inputs. Cache entries also have a per-phase TTL (0–4h) as a secondary invalidation mechanism.

Cache files per phase:
```
.cache/
  <phase>.hash    — "{sha256_hex}|{unix_timestamp}"
  <phase>.ctx     — raw assembled context bytes
  metrics.json    — cumulative token / cache-hit counters
```

## Context

Context assembly is the main token-cost driver in the pipeline. For early phases (`explore`, `propose`), the assembled context can be several thousand tokens. Running `sdd context` multiple times in the same session — which happens during review/iteration — would re-pay this cost on every invocation.

The content hash means the cache is automatically invalidated when any input changes (SKILL.md edited, artifact updated). TTLs provide a safety net for time-sensitive phases where staleness matters more than cost.

## Alternatives Considered

- **No caching (always reassemble)** — rejected because it burns tokens on every context call, including no-op re-runs.
- **Session-level in-memory cache** — rejected because the CLI is invoked as a subprocess; there is no persistent process to hold memory.
- **Timestamp-only TTL without content hash** — rejected because it invalidates valid cache entries on artifact touches without content changes (e.g., `touch` or git operations).
- **LLM-side caching only (prompt caching)** — rejected because it doesn't avoid the cost of transmitting context to the LLM in the first place.

## Consequences

**Positive:**
- Zero token cost on cache hits — context is served from disk.
- SKILL.md edits automatically invalidate affected phase caches.
- Metrics JSON tracks hit rates, helping identify phases with low cache utilization.

**Negative:**
- Cache files must be excluded from git (`.gitignore`) to avoid noise.
- Per-phase TTL constants are hardcoded; adjusting them requires a code change.
- Cache can go stale in adversarial cases (clock skew, system time changes).

## References

- Change: `openspec/changes/archive/2026-03-21-...-concurrency-performance/`
- Code: `internal/context/cache.go` — `tryCachedContext`, `saveContextCache`
- Docs: `docs/architecture.md` — "Cache Architecture"

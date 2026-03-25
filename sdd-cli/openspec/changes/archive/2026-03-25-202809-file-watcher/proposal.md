# Proposal: File Watcher

**Change**: file-watcher
**Date**: 2026-03-25
**Status**: proposed

## Intent

Add a long-running `sdd watch <name>` command that uses kernel-level filesystem notifications (fsnotify) with debounced reassembly, so artifact edits automatically re-run `context.Assemble` and print fresh context to stdout. This eliminates the current workaround of wrapping `sdd context` with external tools like `watchexec`.

## Scope

### In Scope
- New `sdd watch <name>` command — long-running, signal-aware (SIGINT/SIGTERM)
- New `internal/watch` package: fsnotify event loop, 300ms debounced timer, path filtering
- Recursive watch on `openspec/changes/{name}/` subtree
- Filter `.cache/` and `.pending/` paths to prevent feedback loops
- Re-read `state.json` on each debounce fire to track phase transitions
- Full context re-emit to stdout on each trigger (same output as `sdd context`)
- JSON startup message consistent with dashboard pattern
- `--debounce` flag (milliseconds, default 300)
- `WatchReassembled` event type on the event broker (telemetry hook)
- fsnotify v1.8.x as new external dependency in `go.mod`

### Out of Scope
- Daemon mode / Unix socket RPC (roadmap 6.3 — separate future change)
- Dashboard integration for watch events (deferred; MVP logs via slog)
- `sdd context --watch` alias (deferred; discoverability concern, not MVP)
- Polling fallback for platforms without fsnotify support (deferred)
- Rate limiting beyond debounce (max-reassemblies-per-minute guard)
- Watching files outside the change directory (e.g., source code tree)

## Approach

**Approach A: Standalone `sdd watch <name>` command** (selected from exploration)

Rationale: clean separation from the one-shot `sdd context`; long-running commands deserve their own entry point (same pattern as `sdd dashboard`). Mixing one-shot and long-running in the same command via `--watch` flag would complicate `runContext` and confuse users.

### Architecture

```
cmd_watch.go (CLI glue)
    ├── flag parsing: name, --debounce, verbosity
    ├── loadChangeState / loadConfig / newBroker (reuse from commands.go)
    ├── signal.NotifyContext (SIGINT, SIGTERM)
    ├── JSON startup message to stdout
    └── watch.Run(ctx, opts) — blocks until ctx cancelled

internal/watch/watcher.go (core logic)
    ├── fsnotify.NewWatcher()
    ├── enumerate + Add() all subdirs of changeDir (recursive setup)
    ├── event loop goroutine:
    │   ├── filter: skip .cache/, .pending/ paths
    │   ├── filter: skip non-Write/Create/Remove/Rename ops
    │   ├── on Create of directory → watcher.Add(path)
    │   └── debounce: mutex-guarded timer.Reset(300ms)
    ├── debounce fires → reassemble():
    │   ├── state.Load(state.json)
    │   ├── st.ReadyPhases()
    │   ├── context.Assemble(stdout, phase, params)
    │   └── broker.Emit(WatchReassembled)
    └── ctx.Done() → watcher.Close()
```

### Data Flow

1. User runs `sdd watch foo` — CLI wires params, starts fsnotify watcher on `openspec/changes/foo/`
2. User (or Claude) edits an artifact (e.g., `proposal.md` promoted by `sdd write`)
3. fsnotify delivers Write/Create event
4. Path filter rejects `.cache/` and `.pending/` events; accepts artifact events
5. Debounce timer resets to 300ms from now (mutex-guarded)
6. Timer fires: `reassemble()` re-reads `state.json`, resolves current phase, calls `context.Assemble` to stdout
7. Cache layer inside `Assemble` may short-circuit (content-hash match) or do full assembly
8. Broker emits `WatchReassembled` event for telemetry subscribers
9. Loop continues until SIGINT/SIGTERM cancels context

### Startup JSON

```json
{"command":"watch","status":"watching","change":"foo","phase":"propose","dir":"/path/to/openspec/changes/foo"}
```

## Key Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | Standalone command, not `--watch` flag | Long-running commands need their own dispatch; avoids overloading `runContext` |
| 2 | fsnotify, not stdlib polling | Kernel-level notifications: O(1) per event vs O(files) per poll interval; no CGO |
| 3 | 300ms default debounce | Matches editor save patterns (VS Code 250-500ms); configurable via `--debounce` |
| 4 | Filter `.cache/` and `.pending/` | Prevents feedback loop: cache writes after assembly would re-trigger assembly |
| 5 | Re-read `state.json` per debounce fire | Phase may advance between triggers (user ran `sdd write`); always current |
| 6 | Full context re-emit, not diffs | Consumers (sub-agents) expect complete context; matches `sdd context` contract |
| 7 | New `internal/watch` package | Clean boundary: no `internal/cli` imports; reusable for future daemon (6.3) |
| 8 | `WatchReassembled` event in broker | Consistent with existing telemetry pattern; dashboard integration deferred |

## Affected Areas

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/watch/watcher.go` | NEW | Core debounce loop, fsnotify setup, path filtering, reassembly invocation |
| `internal/watch/watcher_test.go` | NEW | Unit tests: debounce timing, path filter, reassembly count |
| `internal/cli/cmd_watch.go` | NEW | CLI glue: flag parsing, signal handling, JSON startup, delegates to `watch.Run` |
| `internal/cli/cli.go` | MODIFY | Add `"watch"` case to `Run()` switch; add help text entry and `printHelp` line |
| `go.mod` / `go.sum` | MODIFY | Add `github.com/fsnotify/fsnotify v1.8.x` |
| `internal/events/broker.go` | MODIFY | Add `WatchReassembled` event type constant + `WatchReassembledPayload` struct |

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Feedback loop: cache writes re-trigger watcher | High | High | Path filter rejects events under `.cache/` and `.pending/` before debounce |
| fsnotify non-recursive: misses new subdirectories | Medium | Medium | Enumerate all subdirs at startup; on Create events for dirs, call `watcher.Add()` dynamically |
| Rapid multi-file saves thrash assembly | High | Medium | 300ms debounced timer with mutex-guarded reset prevents thrashing |
| Concurrent `sdd watch` + `sdd context` write to same cache | Low | Low | Cache writes already atomic (`fsutil.AtomicWrite`); content-hash guards correctness |
| SIGINT mid-assembly produces partial stdout | Low | Low | Context consumers are tolerant of partial output; next debounce fire overwrites |
| `state.json` corruption under concurrent read+write | Low | Medium | `state.Load` is a pure read (`os.ReadFile` + `json.Unmarshal`); atomic writes by `sdd write` prevent partial reads |
| New dependency (fsnotify) increases binary size | Certain | Low | fsnotify is ~2K LOC pure Go; negligible compared to modernc.org/sqlite already in go.mod |

## Rollback Plan

1. Remove `internal/watch/` directory (new package)
2. Remove `internal/cli/cmd_watch.go` (new file)
3. Revert `internal/cli/cli.go` (remove `"watch"` case + help text)
4. Revert `internal/events/broker.go` (remove `WatchReassembled` constant + payload)
5. Run `go mod tidy` to remove fsnotify from `go.mod` / `go.sum`

No schema migrations, no state.json changes, no config format changes. Fully additive; removal restores prior behavior with zero side effects.

## Dependencies

| Dependency | Type | Version | Justification |
|------------|------|---------|---------------|
| `github.com/fsnotify/fsnotify` | External (new) | v1.8.x | Kernel-level inotify/kqueue/ReadDirectoryChangesW abstraction; pure Go, no CGO; canonical Go ecosystem solution |
| `internal/context` | Internal (existing) | — | `Assemble()` is the reassembly entry point; fully re-invokable |
| `internal/events` | Internal (existing) | — | Broker for `WatchReassembled` event emission |
| `internal/state` | Internal (existing) | — | `Load()` + `ReadyPhases()` for phase resolution per trigger |
| `internal/cli/commands.go` | Internal (existing) | — | `loadChangeState`, `loadConfig`, `newBroker`, `getProjectRoot` |

## Success Criteria

- `sdd watch foo` prints JSON startup message and blocks until SIGINT
- Editing `openspec/changes/foo/proposal.md` triggers context reassembly within ~300ms
- Editing `openspec/changes/foo/.cache/propose.ctx` does NOT trigger reassembly (filter works)
- Running `sdd write foo propose` (which changes `state.json` + promotes artifact) triggers reassembly for the new current phase
- Multiple rapid file saves within 300ms produce exactly one reassembly (debounce works)
- `sdd watch foo --debounce 500` uses 500ms debounce interval
- `make check` passes (build, lint, test)
- `sdd watch nonexistent` exits with error JSON (change not found)
- Ctrl-C cleanly shuts down the watcher (no goroutine leak, no orphan fsnotify fd)

## Open Questions

1. **Separator between reassemblies**: Should the watcher emit a visual separator (e.g., `--- [reassembled at 14:32:01] ---`) between context outputs on stdout, or rely on the consumer to detect the model directive / section headers? Leaning toward a separator on stderr (not stdout) to keep stdout clean for piping.

2. **`--phase` flag**: Should `sdd watch foo --phase propose` lock the watcher to a specific phase, or always auto-resolve via `ReadyPhases()`? Auto-resolve is more useful (tracks pipeline progression), but explicit phase lock could help debugging. Deferred to design phase.

3. **Error recovery**: If `context.Assemble` returns an error mid-watch (e.g., missing required artifact), should the watcher log and continue waiting, or exit? Recommend: log to stderr and continue — the user may be about to create the missing artifact.

4. **Reusability for daemon mode (6.3)**: `internal/watch.Run` could accept a callback `func(ctx context.Context) error` instead of hardcoding `context.Assemble`, making it reusable as the file-watching primitive for the future daemon. Worth considering in design phase, low-cost to parameterize.

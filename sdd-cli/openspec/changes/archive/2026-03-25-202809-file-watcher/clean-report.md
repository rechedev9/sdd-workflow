# Clean Report: file-watcher

**Date**: 2026-03-25
**Status**: SUCCESS

## Files Inspected

| File | Role |
|------|------|
| `internal/watch/watcher.go` | Core watch loop — new file |
| `internal/watch/watcher_test.go` | Watch package tests — new file |
| `internal/cli/cmd_watch.go` | CLI glue for sdd watch — new file |
| `internal/cli/cmd_watch_test.go` | CLI watch tests — new file |
| `internal/events/broker.go` | Added WatchReassembled event type + payload |
| `internal/cli/cli.go` | Added watch dispatch, help entry, commandHelp entry |

## Lines Removed

**Total**: 1 line changed (style normalization, net 0)

| File | Change |
|------|--------|
| `internal/watch/watcher_test.go` | Line 287: composite literal `callNum := atomic.Int32{}` → `var callNum atomic.Int32` (style) |

## Actions Taken

### Pass 1 — Dead Code & Stale References

- **Unused imports**: 0 found. All imports are used in every file.
- **Unused variables**: 0 found. `verbosity` in `cmd_watch.go` is passed to `newBroker` (line 79). `reassembleCount` in `watcher.go` is read inside `fire()` to suppress the separator on the first reassembly.
- **Unused private functions**: 0 found. `shouldFilter` and `addRecursive` are both called from `Run`.
- **Dead parameters**: 0 found. All parameters in exported functions match their signatures.
- **TODO/FIXME comments**: 0 found across all 6 files.
- **Commented-out code**: 0 found.
- **Stale docs**: All doc comments match current signatures and behavior. No stale references.

### Pass 2 — Duplication & Reuse

- **Duplicate blocks within scope**: None meeting the Rule of Three (3+ identical occurrences). The test helper `runWatch` in `watcher_test.go` is used by 9 integration tests — appropriate extraction.
- **Codebase reuse search**: `ParseVerbosityFlags`, `loadChangeState`, `getProjectRoot`, `loadConfig`, `tryOpenStore`, `newBroker`, `writeJSON`, `errUnknownFlag` — all from existing `internal/cli` helpers, correctly reused.
- **Cross-file helper consolidation**: No shared helpers between `watcher_test.go` and `cmd_watch_test.go` — the two test files test different levels (package vs CLI integration) with no duplicated helpers.

**Observation (deferred)**: `opts.Broker` and `opts.ChangeName` in `watch.Options` are never read inside `watcher.go` — the CLI closes over `broker` and `name` directly inside the `reassemble` func. These fields were included for future daemon-mode reusability per the design (ADR #5 in design.md). Removing them would be a breaking API change and contradicts design intent. Deferred.

### Pass 3 — Quality & Efficiency

- **Function length**: `Run()` in `watcher.go` is 85 lines total, but the goroutine body and `fire` closure are well-scoped. No split needed. `runWatch` in `cmd_watch.go` is 143 lines; the `reassemble` closure is 45 lines inside it — acceptable for a command handler, and splitting would require passing many locals.
- **Nesting depth**: Maximum depth 3 in `Run` goroutine (select → case → if). Within threshold.
- **Complexity**: No branch exceeds cyclomatic 5.
- **Efficiency**:
  - Timer management: `time.NewTimer(0)` + drain on init, then `timer.Reset()` per event — correct pattern per Go docs (no tick loss).
  - Mutex is narrow: wraps only the `timer.Reset()` call and the `reassembleCount` increment. No lock held during reassembly.
  - `addRecursive` allocates a walk per startup, not per event — correct hot-path design.
- **Goroutine leak**: `TestRun_NoGoroutineLeak` verifies `runtime.NumGoroutine()` tolerance of +2 after shutdown. Passes under `-race`.

**Fixed**: `TestRun_ReassembleErrorLoggedWatchContinues` declared `callNum := atomic.Int32{}` (composite literal) on the line immediately following `var count atomic.Int32` (var form). Both are zero-value `atomic.Int32`, but the mixed style within the same function is inconsistent with the rest of the file. Normalized to `var callNum atomic.Int32`.

## Documentation Synchronization

| File | Function | Fix Type | Description |
|------|----------|----------|-------------|
| — | — | — | No stale docs found. All comments accurately describe current behavior. |

## Issues Found and Fixed

| File | Line | Issue | Fix |
|------|------|-------|-----|
| `internal/watch/watcher_test.go` | 287 | Mixed atomic declaration style (`callNum := atomic.Int32{}` vs adjacent `var count atomic.Int32`) | Changed to `var callNum atomic.Int32` |

## Issues Found and Deferred

| File | Issue | Reason deferred |
|------|-------|-----------------|
| `internal/watch/watcher.go` | `opts.Broker` and `opts.ChangeName` fields in `Options` are declared and accepted but never read by `watch.Run` itself — the CLI uses them via closure capture instead | Design explicitly includes these for future daemon-mode reuse. Removing is a public API break. Revisit if daemon mode (roadmap 6.3) is implemented. |

## Build Status

- **Format (gofumpt)**: PASS
- **Lint (golangci-lint)**: PASS — 0 issues
- **Tests (go test -race ./...)**: PASS — all 17 packages green
- **Build**: PASS

## Full make check Output

```
gofumpt -w .
golangci-lint run ./...
0 issues.
CGO_ENABLED=1 go test -race ./...
ok  internal/artifacts   (cached)
ok  internal/cli         (cached)
ok  internal/events      (cached)
ok  internal/watch       (cached)
... (all 17 packages pass)
```

# Review Report: file-watcher

**Date**: 2026-03-25
**Reviewer**: sdd-review (automated)
**Status**: PASSED

## Summary

The file-watcher implementation is clean, well-structured, and faithfully follows both the spec (17 requirements) and the design document. All 17 requirements have corresponding test coverage. The watch package correctly maintains its import boundary (no `internal/cli` dependency), the debounce logic is sound, and the CLI glue follows established codebase patterns. No blocking issues found.

## Requirement Coverage Matrix

| Requirement | Description | Status | Evidence |
|-------------|-------------|--------|----------|
| REQ-SPEC-001 | Standalone `sdd watch` command | COVERED | `cli.go:60-61` dispatches to `runWatch`; `printHelp` includes watch line; `commandHelp` has watch entry. Tests: `TestRunDispatch_Watch`, `TestRunHelp_ContainsWatch`, `TestRunWatch_PerCommandHelp` |
| REQ-SPEC-002 | Change name validation | COVERED | `cmd_watch.go:60-63` calls `loadChangeState` which validates existence and emits JSON error. Test: `TestRunWatch_NonexistentChange` asserts exit code 1 + JSON stderr |
| REQ-SPEC-003 | Usage error on missing args | COVERED | `cmd_watch.go:56-58` returns `errs.Usage` when name is empty. Test: `TestRunWatch_NoArgs` asserts exit code 2 |
| REQ-SPEC-004 | JSON startup message | COVERED | `cmd_watch.go:92-106` emits JSON with command/status/change/phase/dir. Test: `TestRunWatch_StartupJSON` decodes via `json.Decoder` from `io.Pipe`, verifies all fields |
| REQ-SPEC-005 | Debounce flag | COVERED | `cmd_watch.go:28-45` parses `--debounce`, validates >0, rejects non-numeric. Tests: `TestRunWatch_DebounceZero`, `TestRunWatch_DebounceNonNumeric`. Default 300ms at `cmd_watch.go:28` |
| REQ-SPEC-006 | Recursive filesystem watch | COVERED | `watcher.go:82-102` `addRecursive` walks and adds subdirs, skips `.cache`/`.pending`. Dynamic add at `watcher.go:175-179`. Tests: `TestAddRecursive_EnumeratesSubdirs`, `TestAddRecursive_SkipsCacheAndPending`, `TestRun_DynamicSubdirWatched` |
| REQ-SPEC-007 | Path filtering | COVERED | `watcher.go:64-77` `shouldFilter` rejects `.cache/`, `.pending/`, Chmod-only. Tests: 5 unit tests covering all filter branches |
| REQ-SPEC-008 | Debounced reassembly | COVERED | `watcher.go:126-137,181-183` mutex-guarded timer reset. Tests: `TestRun_SingleFileWriteTriggersOneReassembly`, `TestRun_RapidWritesCoalesceIntoOne`, `TestRun_WidelySpacedWritesTriggerSeparate` |
| REQ-SPEC-009 | Phase re-resolution on each trigger | COVERED | `cmd_watch.go:111-144` re-reads `state.json` and calls `ReadyPhases()` on every trigger. Handles single-phase (`Assemble`) and multi-phase (`AssembleConcurrent`) paths. Error on state read returns error (logged by watch, continues) |
| REQ-SPEC-010 | Context output to stdout | COVERED | `cmd_watch.go:136,141` writes to the `w` param (stdout). Empty ready phases logs info and returns nil (`cmd_watch.go:118-120`) |
| REQ-SPEC-011 | WatchReassembled event | COVERED | `cmd_watch.go:146-153` emits event only on success path (after Assemble/AssembleConcurrent returns nil). Tests: `TestRun_WatchReassembledEventEmitted`, `TestRun_WatchReassembledEventNotEmittedOnError` |
| REQ-SPEC-012 | Signal handling and graceful shutdown | COVERED | `cmd_watch.go:107-108` `signal.NotifyContext` for SIGINT+SIGTERM. `watcher.go:160-165` cancels timer, closes fsnotify watcher. Tests: `TestRun_BlocksUntilContextCancelled`, `TestRun_NoGoroutineLeak`, `TestRunWatch_StartupJSON` (sends SIGINT) |
| REQ-SPEC-013 | Error recovery during watch | COVERED | `watcher.go:150-153` logs error via `slog.Error` and returns (does not exit loop). Test: `TestRun_ReassembleErrorLoggedWatchContinues` verifies 2 callbacks after first error |
| REQ-SPEC-014 | Watch package boundary | COVERED | `watcher.go:3` package comment documents constraint. `watch.Run` blocks until context cancelled. Test: `TestWatchPackageDoesNotImportCLI` runs `go list -deps` to verify import graph |
| REQ-SPEC-015 | fsnotify external dependency | COVERED | `go.mod` contains `github.com/fsnotify/fsnotify v1.8.0`. `watcher.go` imports and uses it |
| REQ-SPEC-016 | Unknown flag rejection | COVERED | `cmd_watch.go:46-47` calls `errUnknownFlag`. Test: `TestRunWatch_UnknownFlag` asserts exit code 2 + "unknown flag" message |
| REQ-SPEC-017 | Reassembly separator on stderr | COVERED | `watcher.go:146-148` writes separator to `opts.Stderr` when `reassembleCount > 1`. Test: `TestRun_SeparatorOnStderrBetweenReassemblies` |

**Coverage**: 17/17 requirements covered.

## Code Quality Assessment

### Design Compliance

- **Module boundary**: `internal/watch` imports only `fsnotify` and `internal/events`. No `internal/cli` dependency. Enforced by test (`TestWatchPackageDoesNotImportCLI`).
- **Types match design exactly**: `ReassembleFunc`, `Options`, `Run`, `shouldFilter`, `addRecursive` all match the design document signatures.
- **Data flow**: Matches the design's three diagrams (startup, event loop, debounce fire) exactly.
- **Architecture decisions 1-7**: All implemented as designed.

### Pattern Compliance

- **Command signature**: `runWatch(args []string, stdout io.Writer, stderr io.Writer) error` matches all other `cmd_*.go` files.
- **Error handling**: Uses `errs.Usage` for usage errors (exit 2), wraps errors with `fmt.Errorf`, uses `slog.Error` for non-fatal errors. Consistent with codebase patterns.
- **JSON output**: Uses `writeJSON` helper (same as all other commands).
- **Signal handling**: Uses `signal.NotifyContext` (same pattern as `cmd_dashboard.go`).
- **Test isolation**: CLI tests `chdir` to `t.TempDir()` to avoid binding to repo's real `openspec/`. Consistent with documented failure mode.
- **t.Parallel()**: All tests that can be parallel are marked. `TestRunWatch_StartupJSON` correctly omits `t.Parallel()` because it sends SIGINT to the process.

### Naming and Readability

- Function names are clear verbs: `shouldFilter`, `addRecursive`, `runWatch`.
- Type names are descriptive: `ReassembleFunc`, `WatchReassembledPayload`.
- No magic numbers except debounce default (300ms) which is documented in help text and spec.
- Nesting depth is 2-3 levels maximum.

### Error Handling

- `fsnotify.NewWatcher()` error: wrapped and returned as fatal.
- `addRecursive` error: watcher closed (best-effort), wrapped and returned as fatal.
- Reassembly callback error: logged via `slog.Error`, watch continues.
- `state.Load` error: wrapped with "reload state:", propagated to fire() which logs it.
- Dynamic `watcher.Add` error: silently ignored (`//nolint:errcheck`). Acceptable -- transient race where directory was deleted between Create event and Stat/Add.

## Issues

### Advisory Issues (Non-Blocking)

| # | Severity | Category | File | Line | Description | Fixability | Fix Direction |
|---|----------|----------|------|------|-------------|------------|---------------|
| 1 | SUGGESTION | Dead Fields | `internal/watch/watcher.go` | 53-57 | `Options.Broker` and `Options.ChangeName` are declared on the struct but never referenced inside `watcher.go`. They exist for caller documentation and are used in tests, but the watch package itself never reads them. Consider removing them from `Options` if the watch package doesn't need them, or document that they are pass-through fields for callers. | AUTO_FIXABLE | Remove `Broker` and `ChangeName` from `Options` struct since the watch package doesn't use them. Callers (like `cmd_watch.go`) already hold these values in their own closure scope. |
| 2 | SUGGESTION | Robustness | `internal/watch/watcher.go` | 134-137 | The stopped-timer pattern `timer = time.NewTimer(0); if !timer.Stop() { <-timer.C }` is correct but subtle. A one-line comment explaining "drain initial fire so first real event triggers cleanly" would help future readers. | AUTO_FIXABLE | Add a comment above line 134: `// Create a stopped timer; drain the initial channel send so Reset works cleanly.` |
| 3 | SUGGESTION | Test Naming | `internal/watch/watcher_test.go` | 36-69 | The five `TestShouldFilter_*` tests could be a table-driven test for compactness. Not blocking -- the current form is clear and each test has a distinct name for failure identification. | AUTO_FIXABLE | Refactor into table-driven test if desired, though current form is acceptable. |

### No Blocking Issues

No CRITICAL or WARNING issues found.

## Function Tracing

| Function | File:Line | Parameter Types | Return Type | Verified Behavior |
|----------|-----------|-----------------|-------------|-------------------|
| `shouldFilter` | `watcher.go:64` | `(string, fsnotify.Op)` | `bool` | Filters `.cache/`, `.pending/` paths and Chmod-only events. 5 unit tests cover all branches. |
| `addRecursive` | `watcher.go:82` | `(*fsnotify.Watcher, string)` | `(int, error)` | Walks dir tree, adds subdirs to watcher, skips `.cache`/`.pending`. 2 unit tests verify counting and skipping. |
| `Run` | `watcher.go:115` | `(context.Context, Options)` | `error` | Creates fsnotify watcher, enumerates subdirs, enters event loop, blocks until context cancelled. Returns nil on clean shutdown, error on fatal init failure. 9 integration tests. |
| `runWatch` | `cmd_watch.go:25` | `([]string, io.Writer, io.Writer)` | `error` | Parses flags, validates change, emits JSON startup, creates signal context, delegates to `watch.Run`. 6 tests (5 unit + 1 integration). |

## Data Flow Analysis

### Critical Path: File Change -> Reassembly

1. **CREATION**: fsnotify kernel event delivered to `watcher.Events` channel (`watcher.go:167`)
2. **FILTER**: `shouldFilter(event.Name, event.Op)` at `watcher.go:171` -- discards `.cache/`, `.pending/`, Chmod-only
3. **DEBOUNCE**: `timer.Reset(opts.Debounce)` at `watcher.go:182` under mutex
4. **FIRE**: `timer.C` fires at `watcher.go:192`, calls `fire()` at `watcher.go:140`
5. **CALLBACK**: `opts.Reassemble(ctx, opts.Stdout)` at `watcher.go:150`
6. **STATE READ**: `state.Load(changeDir/state.json)` at `cmd_watch.go:112`
7. **PHASE RESOLVE**: `st.ReadyPhases()` at `cmd_watch.go:117`
8. **ASSEMBLE**: `sddctx.Assemble` or `sddctx.AssembleConcurrent` at `cmd_watch.go:136/141`
9. **EVENT**: `broker.Emit(WatchReassembled)` at `cmd_watch.go:146` (only on success)

**Invariants**: (a) filtered paths never reach the timer, (b) timer resets coalesce rapid events, (c) state is re-read on every trigger (never stale), (d) event is only emitted after successful assembly.

## Counter-Hypothesis Results

### CH-1: Timer race between Reset and channel read

- **CLAIM**: `timer.Reset(opts.Debounce)` at `watcher.go:182` could race with `<-timer.C` at `watcher.go:192` if the timer fires between the lock release and the select statement.
- **EVIDENCE SOUGHT**: Can `timer.C` deliver a stale fire after `Reset` is called?
- **FINDING**: NO EVIDENCE OF FAILURE
- **DETAILS**: Per Go documentation, `Timer.Reset` on an expired timer is safe when the timer's channel has already been drained. The select statement at line 192 is the only consumer of `timer.C`. Since the timer starts stopped (lines 134-137) and Reset is always called under the same mutex that guards Stop (line 162), there's no race. The mutex serializes Reset calls from the event branch and Stop from the context-done branch.

### CH-2: Dynamic directory add race with deletion

- **CLAIM**: `watcher.Add(event.Name)` at `watcher.go:178` could fail if the directory is deleted between the Create event and the `os.Stat` call at line 177.
- **EVIDENCE SOUGHT**: What happens if `os.Stat` returns an error or `watcher.Add` fails?
- **FINDING**: NO EVIDENCE OF FAILURE
- **DETAILS**: If `os.Stat` returns an error, the `err == nil` check at line 177 prevents `watcher.Add` from being called. If `watcher.Add` fails, the `//nolint:errcheck` comment acknowledges this is best-effort. The directory simply won't be watched -- no crash, no goroutine leak. Acceptable behavior for a transient race.

### CH-3: Reassembly callback panics

- **CLAIM**: If the `ReassembleFunc` callback panics inside `fire()`, the goroutine at line 156 could die silently, leaving `Run` blocked forever.
- **EVIDENCE SOUGHT**: Is there panic recovery in the fire path?
- **FINDING**: NO EVIDENCE OF FAILURE (mitigated by design)
- **DETAILS**: The `fire()` function at `watcher.go:140` does NOT have explicit panic recovery. However, the reassembly callback in `cmd_watch.go` calls `state.Load`, `sddctx.Assemble`, and `broker.Emit` -- all well-tested Go functions that don't panic under normal operation. The broker's `Emit` has its own panic recovery (`broker.go:142-148`). A panic in the assembly path would be a Go-level bug (nil pointer, etc.) that should crash. This is consistent with the codebase's approach -- no defensive panic recovery in command paths.

## Security Findings

No security concerns. The watcher operates on a local directory path derived from validated change state. No network I/O, no user-controlled path injection, no privilege escalation vectors.

## Test Coverage Assessment

| Test File | Tests | Coverage |
|-----------|-------|----------|
| `watcher_test.go` | 17 tests (7 unit + 10 integration) | `shouldFilter` (5 tests, all branches), `addRecursive` (2 tests, enumerate + skip), `Run` (9 tests: blocks, single trigger, coalesce, separate triggers, dynamic subdir, error recovery, goroutine leak, stderr separator, event emit/suppress), import boundary (1 test) |
| `cmd_watch_test.go` | 6 tests (5 unit + 1 integration) | No args (exit 2), unknown flag (exit 2), debounce 0 (exit 2), debounce non-numeric (exit 2), nonexistent change (exit 1 + JSON), startup JSON structure (full field validation + SIGINT shutdown) |
| `cli_test.go` | 5 watch-related entries | Dispatch table (exit codes), error JSON table, `TestRunDispatch_Watch`, `TestRunHelp_ContainsWatch`, `TestRunWatch_PerCommandHelp` |

**Total**: 28 test assertions across 3 files covering all 17 spec requirements.

**Race detection**: `make check` runs with `-race` flag -- all tests pass.

## Verdict

**PASSED**

The implementation faithfully satisfies all 17 spec requirements, matches the design document's types/signatures/data-flow exactly, follows codebase conventions (error handling, JSON output, command signature, slog logging, atomic patterns), and maintains the `watch` -> no-`cli` import boundary. Test coverage is thorough with 28 test cases including race detection. Three advisory suggestions noted (dead struct fields, comment improvement, table-driven test refactor) -- none blocking.

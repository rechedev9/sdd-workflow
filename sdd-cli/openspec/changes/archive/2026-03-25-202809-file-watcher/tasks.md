# Apply Report: file-watcher

**Phase**: 1-5 (all)
**Date**: 2026-03-25
**Status**: SUCCESS
**Tasks Completed**: 22/22

## Tasks

### Phase 1: External Dependency + Event Types
- [x] 1.1 Add fsnotify v1.8.0 to go.mod
- [x] 1.2 Add WatchReassembled event type + WatchReassembledPayload to broker.go
- [x] 1.3 Run go mod tidy to populate go.sum

### Phase 2: watch Package -- Core Logic
- [x] 2.1 Create internal/watch/watcher.go with package, types, helpers (shouldFilter, addRecursive)
- [x] 2.2 Implement Run() function with fsnotify, debounce timer, event loop
- [x] 2.3 Verify build passes and no internal/cli import

### Phase 3: watch Package -- Tests
- [x] 3.1 Unit tests for helpers (6 tests)
- [x] 3.2 Integration tests for Run (11 tests)

### Phase 4: CLI Glue
- [x] 4.1 Create internal/cli/cmd_watch.go with runWatch
- [x] 4.2 Add "watch" case to cli.go Run() switch + printHelp()
- [x] 4.3 Add "watch" entry to commandHelp map

### Phase 5: CLI Tests + Final Verification
- [x] 5.1 Unit tests for cmd_watch (5 tests)
- [x] 5.2 Dispatch and help tests in cli_test.go (3 tests)
- [x] 5.3 Integration test for startup JSON (io.Pipe + SIGINT)
- [x] 5.4 make check passes (fmt + lint + test with -race)

## Files Created

| File | Purpose |
|------|---------|
| `internal/watch/watcher.go` | Core watch loop: fsnotify setup, recursive dir enumeration, path filtering, debounce timer, reassembly invocation via callback |
| `internal/watch/watcher_test.go` | Unit tests (7) + integration tests (10) for watch package |
| `internal/cli/cmd_watch.go` | CLI glue: flag parsing (--debounce), validation, JSON startup message, signal handling, delegates to watch.Run |
| `internal/cli/cmd_watch_test.go` | Unit tests (5) + integration test (1) for CLI watch command |

## Files Modified

| File | Changes |
|------|---------|
| `go.mod` | Added github.com/fsnotify/fsnotify v1.8.0 as direct dependency |
| `go.sum` | Updated by go mod tidy with fsnotify checksums |
| `internal/events/broker.go` | Added WatchReassembled EventType constant + WatchReassembledPayload struct |
| `internal/cli/cli.go` | Added "watch" case to Run() switch, watch line to printHelp(), watch entry to commandHelp map |
| `internal/cli/cli_test.go` | Added watch to TestRunSubcommands table, TestRunErrorsWriteJSON table, TestRunDispatch_Watch, TestRunHelp_ContainsWatch, TestRunWatch_PerCommandHelp |

## Build Health

| Check | Result |
|-------|--------|
| Type/Compile | PASS |
| Lint | PASS (0 issues) |
| Tests | PASS (all packages, with -race) |
| Format | PASS (gofumpt) |

## Deviations

- Design specified `time.AfterFunc` for debounce but Architecture Decision #3 in design.md explicitly chose `sync.Mutex`-guarded `time.Timer` with `Reset()` instead. Implementation follows the design's final decision.
- `TestRunWatch_StartupJSON` uses `io.Pipe` + `SIGINT` signal to test startup JSON race-free, rather than the simpler context-cancellation approach described in tasks.md, because `runWatch` creates its own `signal.NotifyContext` internally.
- Added `TestShouldFilter_WriteAndChmod` as an additional edge case test not listed in the original test plan (Write|Chmod combined op should NOT be filtered).
- Added `TestRunWatch_PerCommandHelp` to verify `sdd watch --help` works (commandHelp entry).

## Manual Review Needed

None.

# Technical Design: File Watcher

**Change**: file-watcher
**Date**: 2026-03-25T21:00:00Z
**Status**: draft
**Depends On**: proposal.md

---

## Technical Approach

The file watcher adds a long-running `sdd watch <name>` command that monitors a change directory for artifact modifications and re-runs context assembly on each debounced trigger. The design follows two established patterns in the codebase: (1) the `cmd_dashboard.go` long-running command pattern (`signal.NotifyContext`, JSON startup message, blocking `Run` call), and (2) the `context.Assemble` / `AssembleConcurrent` API for context emission.

The core watch logic lives in a new `internal/watch` package, cleanly separated from `internal/cli`. The CLI layer (`cmd_watch.go`) handles flag parsing, state validation, and signal setup, then delegates to `watch.Run(ctx, opts)` which blocks until cancellation. This boundary makes the watch primitive reusable for the future daemon mode (roadmap 6.3) without any CLI coupling.

The debounce mechanism uses a `sync.Mutex`-guarded `time.Timer` that resets on each qualifying fsnotify event. When the timer fires, a reassembly callback re-reads `state.json`, resolves ready phases, and calls `context.Assemble` or `context.AssembleConcurrent`. Path filtering rejects `.cache/` and `.pending/` events before they reach the debounce timer, preventing feedback loops.

## Architecture Decisions

| # | Decision | Choice | Alternatives Considered | Rationale |
|---|----------|--------|-------------------------|-----------|
| 1 | Watch package location | `internal/watch/` as standalone package | Embed in `internal/cli/`, embed in `internal/context/` | Package boundary rule (REQ-SPEC-014): watch must not import cli. Standalone package enables reuse by future daemon. Matches existing pattern where each domain (`dashboard`, `verify`, `events`) is its own package. |
| 2 | Reassembly invocation | Callback function `ReassembleFunc` passed via options struct | Hardcode `context.Assemble` import in watch package, interface-based dependency injection | Callback avoids import cycle (`watch` -> `context` -> `phase`; `cli` -> `watch` already depends on `cli` -> `context`). Simpler than a full interface — there is only one operation. Also enables reuse: daemon can pass a different callback. |
| 3 | Debounce implementation | `sync.Mutex`-guarded `time.Timer` with `Reset()` | Channel-based debounce (event -> goroutine -> time.After), `time.AfterFunc` with atomic swap | Mutex+Timer is the simplest correct approach. Channel-based debounce requires careful goroutine lifecycle management. `time.AfterFunc` creates a new goroutine per call; `Timer.Reset` reuses one. Consistent with Go stdlib patterns. |
| 4 | Options struct vs functional options | Plain struct `watch.Options` | `watch.WithDebounce(d)` functional options pattern | The project uses plain structs everywhere (`phase.AssemblerParams`, `config.Config`). Functional options would be the only instance in the codebase. Consistency wins. |
| 5 | Stderr separator between reassemblies | `fmt.Fprintf(stderr, "--- reassembled at %s ---\n", time.Now().Format("15:04:05"))` | slog.Info structured log line, no separator | Spec REQ-SPEC-017 requires a human-readable separator on stderr. A plain `fmt.Fprintf` is simpler than slog for a fixed-format separator and keeps it visually distinct from structured log lines. |
| 6 | Dynamic subdirectory watching | Enumerate at startup + `watcher.Add` on Create events for directories | Polling with `filepath.WalkDir` on timer, single top-level watch (fsnotify non-recursive) | fsnotify is non-recursive by design. Polling defeats the purpose of kernel notifications. Enumerate+dynamic-add is the canonical fsnotify pattern and matches REQ-SPEC-006. |
| 7 | Event type for broker | New `WatchReassembled` constant + `WatchReassembledPayload` struct in `internal/events/broker.go` | Reuse `PhaseAssembled` event, new events file | Spec REQ-SPEC-011 requires a distinct event type. Adding to `broker.go` follows the existing pattern where all event types and payloads are defined in the same file. |

## Data Flow

### Watch Startup

```
User runs: sdd watch foo --debounce 500
  │
  ├─ cmd_watch.go: runWatch()
  │   ├─ ParseVerbosityFlags(args)
  │   ├─ Parse --debounce flag (default 300)
  │   ├─ Validate: debounce > 0, no unknown flags
  │   ├─ loadChangeState(stderr, "watch", name)
  │   │   ├─ resolveChangeDir("foo")
  │   │   └─ state.Load(changeDir + "/state.json")
  │   ├─ getProjectRoot(stderr, "watch")
  │   ├─ loadConfig(stderr, "watch", projectRoot)
  │   ├─ tryOpenStore(projectRoot) + newBroker(verbosity, db)
  │   ├─ Build watch.Options{...}
  │   ├─ Emit JSON startup message to stdout:
  │   │   {"command":"watch","status":"watching","change":"foo",
  ���   │    "phase":"propose","dir":"/abs/path/to/openspec/changes/foo"}
  │   ├─ signal.NotifyContext(ctx, SIGINT, SIGTERM)
  │   └─ watch.Run(ctx, opts) ← blocks
  │
  └─ Returns nil on clean shutdown, error on fatal failure
```

### Event Loop (inside watch.Run)

```
watch.Run(ctx, opts)
  │
  ├─ fsnotify.NewWatcher()
  ├─ filepath.WalkDir(changeDir) → watcher.Add(dir) for each subdir
  │   └─ Skip: .cache/, .pending/ subdirectories
  │
  ├─ goroutine: event loop
  │   │
  │   ├─ ← watcher.Events channel
  │   │   ├─ Filter: skip if path contains /.cache/ or /.pending/
  │   │   ├─ Filter: skip Chmod-only events (no data change)
  ���   │   ├─ If Create + IsDir → watcher.Add(path) (dynamic watch)
  │   │   └─ mu.Lock(); timer.Reset(debounce); mu.Unlock()
  │   │
  │   ├─ ← watcher.Errors channel
  │   │   └─ slog.Error("fsnotify error", "err", err)
  │   │
  │   └─ ← ctx.Done()
  │       ├─ timer.Stop()
  │       ├─ watcher.Close()
  │       └─ return nil
  │
  └─ (blocks until goroutine returns)
```

### Debounce Fire (reassembly)

```
Timer fires after debounce interval
  │
  ├─ reassembleCount++ (for separator logic)
  ├─ If reassembleCount > 1:
  │   └─ fmt.Fprintf(stderr, "--- reassembled at %s ---\n", now)
  │
  ├─ state.Load(changeDir + "/state.json")
  │   └─ On error: slog.Error(...); return (continue watching)
  │
  ├─ st.ReadyPhases()
  │   └─ If empty: slog.Info("no phases ready"); return
  │
  ├─ start := time.Now()
  │
  ├─ If len(ready) > 1:
  │   └─ context.AssembleConcurrent(stdout, ready, params)
  │ Else:
  │   └─ context.Assemble(stdout, ready[0], params)
  │
  ├─ On error: slog.Error(...); return (continue watching)
  │
  └─ broker.Emit(Event{
        Type: WatchReassembled,
        Payload: WatchReassembledPayload{
          Change:     name,
          Phase:      phase,
          DurationMs: time.Since(start).Milliseconds(),
        },
     })
```

## File Changes

| # | File Path (absolute) | Action | Description |
|---|----------------------|--------|-------------|
| 1 | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher.go` | create | Core watch loop: fsnotify setup, recursive dir enumeration, path filtering, debounce timer, reassembly invocation via callback |
| 2 | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | create | Unit tests: debounce coalescing, path filter, dynamic dir add, shutdown cleanup, error recovery |
| 3 | `/home/reche/projects/SDDworkflow/sdd-cli/internal/cli/cmd_watch.go` | create | CLI glue: flag parsing (--debounce), validation, JSON startup message, signal handling, delegates to watch.Run |
| 4 | `/home/reche/projects/SDDworkflow/sdd-cli/internal/cli/cli.go` | modify | Add `"watch"` case to `Run()` switch; add `watch` line to `printHelp()` Pipeline commands section |
| 5 | `/home/reche/projects/SDDworkflow/sdd-cli/internal/cli/help.go` | modify | Add `"watch"` entry to `commandHelp` map (per-command `--help` text) — note: `commandHelp` is in `cli.go`; this is the same file as #4 |
| 6 | `/home/reche/projects/SDDworkflow/sdd-cli/internal/events/broker.go` | modify | Add `WatchReassembled` event type constant + `WatchReassembledPayload` struct |
| 7 | `/home/reche/projects/SDDworkflow/sdd-cli/go.mod` | modify | Add `github.com/fsnotify/fsnotify v1.8.0` to require block |
| 8 | `/home/reche/projects/SDDworkflow/sdd-cli/go.sum` | modify | Updated by `go mod tidy` after adding fsnotify |

**Summary**: 3 files created, 5 files modified, 0 files deleted

## Interfaces and Contracts

### Types

```go
// Package watch implements a debounced filesystem watcher that monitors
// a change directory and invokes a callback when artifacts change.
// Import constraint: this package MUST NOT import internal/cli.
package watch

import (
	"context"
	"io"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
)

// ReassembleFunc is the callback invoked on each debounce fire.
// The watcher passes the current stdout writer; the callback is
// responsible for re-reading state, resolving phases, and calling
// context.Assemble or context.AssembleConcurrent.
//
// If the callback returns an error, the watcher logs it and continues.
// The callback receives the same context as Run — it should respect
// cancellation for long-running assemblies.
type ReassembleFunc func(ctx context.Context, stdout io.Writer) error

// Options configures a watch.Run invocation.
// All fields are required unless noted.
type Options struct {
	// ChangeDir is the absolute path to openspec/changes/{name}/.
	ChangeDir string

	// Debounce is the duration to wait after the last filesystem event
	// before invoking the reassembly callback. Must be positive.
	Debounce time.Duration

	// Stdout receives assembled context output on each reassembly.
	Stdout io.Writer

	// Stderr receives log messages, errors, and reassembly separators.
	Stderr io.Writer

	// Reassemble is called on each debounce fire. Must not be nil.
	Reassemble ReassembleFunc

	// Broker emits WatchReassembled events. May be nil (no events emitted).
	Broker *events.Broker

	// ChangeName is used in event payloads. Required if Broker is non-nil.
	ChangeName string
}
```

```go
// Run starts the filesystem watcher and blocks until ctx is cancelled.
// Returns nil on clean shutdown (context cancellation).
// Returns an error only for fatal failures (e.g., fsnotify init failure).
//
// Lifecycle:
//   1. Create fsnotify watcher
//   2. Enumerate all subdirs of opts.ChangeDir (skip .cache/, .pending/)
//   3. Add each to watcher
//   4. Enter event loop goroutine
//   5. Block until ctx.Done()
//   6. Close fsnotify watcher, return nil
func Run(ctx context.Context, opts Options) error
```

```go
// shouldFilter reports whether an fsnotify event path should be
// discarded (not trigger debounce). Filters:
//   - paths containing "/.cache/"
//   - paths containing "/.pending/"
//   - Chmod-only events (Op == fsnotify.Chmod with no other bits)
func shouldFilter(eventPath string, op fsnotify.Op) bool
```

```go
// addRecursive walks dir and adds all subdirectories to w,
// skipping .cache/ and .pending/ directories.
// Returns the count of directories added.
func addRecursive(w *fsnotify.Watcher, dir string) (int, error)
```

### Event Types (addition to internal/events/broker.go)

```go
// In the const block:
WatchReassembled EventType = "WatchReassembled"

// WatchReassembledPayload is emitted after the watch loop completes
// a successful context reassembly.
type WatchReassembledPayload struct {
	Change     string `json:"change"`
	Phase      string `json:"phase"`       // phase(s) assembled, e.g. "propose" or "spec+design"
	DurationMs int64  `json:"duration_ms"`
}
```

### CLI Command (cmd_watch.go)

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	sddctx "github.com/rechedev9/shenronSDD/sdd-cli/internal/context"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/watch"
)

// runWatch implements the "sdd watch <name>" command.
// Long-running: blocks until SIGINT/SIGTERM.
func runWatch(args []string, stdout io.Writer, stderr io.Writer) error
```

The `runWatch` function follows this contract:

1. Parse verbosity flags, then iterate remaining args for `--debounce <ms>` and positional args.
2. Reject unknown flags via `errUnknownFlag()`.
3. Require exactly one positional arg (change name); return `errs.Usage` if missing.
4. Validate debounce: parse as int, reject <= 0 with `errs.Usage` containing "debounce".
5. Call `loadChangeState(stderr, "watch", name)` to validate change exists.
6. Call `getProjectRoot`, `loadConfig`, `tryOpenStore`, `newBroker`.
7. Build `sddctx.Params` (same as `runContext`).
8. Emit JSON startup message to stdout via `writeJSON`.
9. Create `signal.NotifyContext` for `os.Interrupt` + `syscall.SIGTERM`.
10. Build `watch.Options` with a `ReassembleFunc` closure that:
    - Re-reads `state.json` via `state.Load`
    - Calls `st.ReadyPhases()`
    - Calls `sddctx.Assemble` or `sddctx.AssembleConcurrent`
    - Emits `WatchReassembled` event on success
11. Call `watch.Run(ctx, opts)` — blocks.
12. Return nil on clean exit.

### Reassembly Callback (closure in cmd_watch.go)

```go
// Built inside runWatch as a closure over changeDir, params, broker, etc.
reassemble := func(ctx context.Context, stdout io.Writer) error {
	// Re-read state on every trigger (phase may have advanced).
	st, err := state.Load(filepath.Join(changeDir, "state.json"))
	if err != nil {
		return fmt.Errorf("reload state: %w", err)
	}

	ready := st.ReadyPhases()
	if len(ready) == 0 {
		slog.Info("watch: no phases ready (pipeline complete or blocked)")
		return nil
	}

	start := time.Now()

	// Update params for current state.
	p.ChangeName = st.Name
	p.Description = st.Description

	var phase string
	if len(ready) > 1 {
		names := make([]string, len(ready))
		for i, r := range ready {
			names[i] = string(r)
		}
		phase = strings.Join(names, "+")
		if err := sddctx.AssembleConcurrent(stdout, ready, p); err != nil {
			return err
		}
	} else {
		phase = string(ready[0])
		if err := sddctx.Assemble(stdout, ready[0], p); err != nil {
			return err
		}
	}

	broker.Emit(events.Event{
		Type: events.WatchReassembled,
		Payload: events.WatchReassembledPayload{
			Change:     name,
			Phase:      phase,
			DurationMs: time.Since(start).Milliseconds(),
		},
	})

	return nil
}
```

## Testing Strategy

| # | What to Test | Type | File Path | Maps to Requirement |
|---|-------------|------|-----------|---------------------|
| 1 | `watch.Run` blocks until context cancelled | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-014 |
| 2 | `shouldFilter` rejects `.cache/` paths | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-007 |
| 3 | `shouldFilter` rejects `.pending/` paths | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-007 |
| 4 | `shouldFilter` accepts artifact paths | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-007 |
| 5 | `shouldFilter` rejects Chmod-only events | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-007 |
| 6 | `addRecursive` enumerates subdirs, skips .cache/.pending | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-006 |
| 7 | Single file write triggers exactly one reassembly | integration | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-008 |
| 8 | Rapid multi-file writes coalesce into one reassembly | integration | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-008 |
| 9 | Widely spaced writes trigger separate reassemblies | integration | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-008 |
| 10 | New subdirectory is dynamically watched | integration | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-006 |
| 11 | Reassemble error logged, watch continues | integration | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-013 |
| 12 | No goroutine leak after shutdown | integration | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-012 |
| 13 | Separator on stderr after second reassembly | integration | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-017 |
| 14 | `runWatch` no args returns usage error (exit 2) | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/cli/cmd_watch_test.go` | REQ-SPEC-003 |
| 15 | `runWatch` unknown flag returns usage error (exit 2) | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/cli/cmd_watch_test.go` | REQ-SPEC-016 |
| 16 | `runWatch` --debounce 0 returns usage error | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/cli/cmd_watch_test.go` | REQ-SPEC-005 |
| 17 | `runWatch` --debounce abc returns usage error | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/cli/cmd_watch_test.go` | REQ-SPEC-005 |
| 18 | `runWatch` nonexistent change returns error JSON | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/cli/cmd_watch_test.go` | REQ-SPEC-002 |
| 19 | `Run()` switch dispatches `"watch"` to `runWatch` | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/cli/cli_test.go` | REQ-SPEC-001 |
| 20 | Help output contains `watch` line | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/cli/cli_test.go` | REQ-SPEC-001 |
| 21 | JSON startup message structure | integration | `/home/reche/projects/SDDworkflow/sdd-cli/internal/cli/cmd_watch_test.go` | REQ-SPEC-004 |
| 22 | WatchReassembled event emitted after successful reassembly | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-011 |
| 23 | WatchReassembled event NOT emitted on reassembly error | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-011 |
| 24 | `watch` package import graph does not contain `internal/cli` | unit | `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/watcher_test.go` | REQ-SPEC-014 |

### Test Dependencies

- **Mocks needed**: `ReassembleFunc` — tests pass a counting/recording callback instead of real `context.Assemble`. No interface mocking required; the callback type is sufficient.
- **Fixtures needed**: Temporary directory tree with `state.json` and subdirectories for fsnotify tests. Created via `t.TempDir()` + `os.MkdirAll` + `os.WriteFile`.
- **Infrastructure**: No external services. fsnotify tests use real filesystem events on a temp directory. Tests that verify debounce timing should use short debounce intervals (10-50ms) to avoid slow tests, with appropriate tolerance windows.
- **Goroutine leak detection**: `runtime.NumGoroutine()` before/after `watch.Run` with tolerance for background GC goroutines (allow +/-2).
- **Test isolation**: CLI tests that call `runWatch` with invalid args should `chdir` to `t.TempDir()` first to avoid binding to the repo's real `openspec/` (per CLAUDE.md failure mode).

## Migration and Rollout

No migration or rollout steps required. This is a purely additive change:

- New command (`sdd watch`) — existing commands are unaffected.
- New package (`internal/watch/`) — no existing code modified except glue.
- New dependency (`fsnotify`) — added to `go.mod`; no binary size concern.
- New event type (`WatchReassembled`) — existing subscribers are unaffected (they only process events they subscribe to).

### Rollback Steps

1. Delete `/home/reche/projects/SDDworkflow/sdd-cli/internal/watch/` directory
2. Delete `/home/reche/projects/SDDworkflow/sdd-cli/internal/cli/cmd_watch.go`
3. Revert `cli.go`: remove `"watch"` case from `Run()` switch, remove `watch` from `printHelp()`, remove `"watch"` from `commandHelp` map
4. Revert `broker.go`: remove `WatchReassembled` constant + `WatchReassembledPayload` struct
5. Run `go mod tidy` to drop `fsnotify` from `go.mod`/`go.sum`

## Open Questions

- **Resolved: --phase flag**: Per spec review, the watcher auto-resolves phases via `ReadyPhases()` on every trigger (REQ-SPEC-009). No `--phase` flag for MVP. This is the more useful behavior since it tracks pipeline progression automatically.

- **Resolved: Error recovery**: Per REQ-SPEC-013, the watcher logs assembly errors to stderr via `slog.Error` and continues watching. No exit on transient failures.

- **Resolved: Separator format**: Per REQ-SPEC-017, the separator is `--- reassembled at HH:MM:SS ---` on stderr only. Stdout remains a clean pipe-friendly stream.

- **Resolved: Reusability for daemon**: The `ReassembleFunc` callback design decouples the watch loop from the specific reassembly logic. Future daemon mode can pass a different callback (e.g., one that writes to a socket instead of stdout).

---

**Next Step**: After both design and specs are complete, run `sdd-tasks` to create the implementation checklist.

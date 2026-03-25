# Delta Spec: File Watcher

**Change**: file-watcher
**Date**: 2026-03-25T20:30:00Z
**Status**: draft
**Depends On**: proposal.md

---

## Context

The SDD CLI is currently a one-shot tool: every command runs, produces output, and exits. There is no long-running watch mode. The `sdd watch <name>` command introduces a persistent filesystem watcher that monitors `openspec/changes/{name}/` for artifact changes and automatically re-runs context assembly, emitting fresh output to stdout. This eliminates the external `watchexec` workaround documented in the roadmap.

This spec covers four domains: the CLI command surface (`watch-cli`), the core watch loop (`watch-loop`), the event integration (`watch-events`), and the dependency addition (`watch-deps`).

No existing specs exist in `openspec/specs/` so all requirements are ADDED.

---

## ADDED Requirements

### REQ-SPEC-001: Standalone `sdd watch` Command

The CLI **MUST** expose a `watch` subcommand that accepts a change name as its first positional argument and blocks until interrupted.

The `watch` subcommand **MUST** follow the same `(rest []string, stdout, stderr io.Writer) error` dispatch pattern as all other CLI commands.

The `watch` subcommand **MUST** be registered in the `Run()` switch in `internal/cli/cli.go` and listed in `printHelp()`.

#### Scenario: Watch command dispatches to runWatch handler `code-based` `critical`

- **WHEN** the CLI receives args `["watch", "foo"]`
- **THEN** `Run()` dispatches to `runWatch` without returning a usage error

#### Scenario: Watch command appears in help output `code-based` `standard`

- **WHEN** the CLI receives args `["help"]`
- **THEN** stdout contains a line matching `watch` in the "Pipeline commands" section

---

### REQ-SPEC-002: Change Name Validation

The `watch` command **MUST** validate that the named change exists by loading `state.json` from `openspec/changes/{name}/`.

If the change does not exist, the command **MUST** return a JSON error to stderr with `"command":"watch"` and exit code 1.

#### Scenario: Watch nonexistent change returns error JSON `code-based` `critical`

- **WHEN** the CLI receives args `["watch", "nonexistent"]`
- **THEN** stderr contains JSON with `"command":"watch"` and `"error"` containing a message about the change not being found, and the process exits with code 1

#### Scenario: Watch valid change does not error on startup `code-based` `critical`

- **GIVEN** `openspec/changes/foo/state.json` exists with phase `"propose"` completed
- **WHEN** the CLI receives args `["watch", "foo"]`
- **THEN** the command does not return an error during initialization

---

### REQ-SPEC-003: Usage Error on Missing Arguments

The `watch` command **MUST** return exit code 2 if no change name is provided.

#### Scenario: Watch with no args returns usage error `code-based` `critical`

- **WHEN** the CLI receives args `["watch"]`
- **THEN** the command returns an `errs.Usage` error and the process exits with code 2

---

### REQ-SPEC-004: JSON Startup Message

Upon successful initialization, the `watch` command **MUST** emit a single JSON object to stdout before entering the watch loop. The JSON **MUST** contain the fields: `command` (value `"watch"`), `status` (value `"watching"`), `change` (the change name), `phase` (the current phase from `state.json`), and `dir` (the absolute path to the change directory).

#### Scenario: Startup message structure `code-based` `critical`

- **GIVEN** `openspec/changes/foo/state.json` has current phase `"propose"`
- **WHEN** `sdd watch foo` starts
- **THEN** the first line of stdout is a JSON object: `{"command":"watch","status":"watching","change":"foo","phase":"propose","dir":"/abs/path/to/openspec/changes/foo"}`

---

### REQ-SPEC-005: Debounce Flag

The `watch` command **MUST** accept a `--debounce` flag with a value in milliseconds. The default **MUST** be `300`.

The debounce value **MUST** be a positive integer. If the value is zero or negative, the command **MUST** return a usage error (exit 2).

#### Scenario: Default debounce is 300ms `code-based` `critical`

- **WHEN** `sdd watch foo` is invoked without `--debounce`
- **THEN** the watcher uses a 300-millisecond debounce interval

#### Scenario: Custom debounce value is respected `code-based` `critical`

- **WHEN** `sdd watch foo --debounce 500` is invoked
- **THEN** the watcher uses a 500-millisecond debounce interval

#### Scenario: Invalid debounce value returns usage error `code-based` `critical`

- **WHEN** `sdd watch foo --debounce 0` is invoked
- **THEN** the command returns an `errs.Usage` error with message containing "debounce" and the process exits with code 2

#### Scenario: Non-numeric debounce value returns usage error `code-based` `critical`

- **WHEN** `sdd watch foo --debounce abc` is invoked
- **THEN** the command returns an `errs.Usage` error with message containing "debounce" and the process exits with code 2

---

### REQ-SPEC-006: Recursive Filesystem Watch

The watcher **MUST** monitor all directories under `openspec/changes/{name}/` recursively using kernel-level filesystem notifications (fsnotify).

At startup, the watcher **MUST** enumerate all existing subdirectories of the change directory and add each to the fsnotify watcher.

When a new subdirectory is created at runtime, the watcher **MUST** dynamically add it to the watch set.

#### Scenario: All existing subdirectories are watched at startup `code-based` `critical`

- **GIVEN** `openspec/changes/foo/` contains subdirectories `specs/` and `specs/watch-cli/`
- **WHEN** `sdd watch foo` starts
- **THEN** fsnotify is watching `openspec/changes/foo/`, `openspec/changes/foo/specs/`, and `openspec/changes/foo/specs/watch-cli/`

#### Scenario: Newly created subdirectory is dynamically watched `code-based` `critical`

- **GIVEN** `sdd watch foo` is running
- **WHEN** a new directory `openspec/changes/foo/newdir/` is created
- **THEN** the watcher adds `openspec/changes/foo/newdir/` to the fsnotify watch set, and subsequent file changes in `newdir/` trigger the debounce timer

---

### REQ-SPEC-007: Path Filtering

The watcher **MUST NOT** trigger reassembly for events under `.cache/` or `.pending/` subdirectories of the change directory.

Events on paths matching `{changeDir}/.cache/**` or `{changeDir}/.pending/**` **MUST** be silently discarded before reaching the debounce timer.

The watcher **SHOULD NOT** add `.cache/` or `.pending/` subdirectories to the fsnotify watch set at startup or dynamically.

#### Scenario: Cache write does not trigger reassembly `code-based` `critical`

- **GIVEN** `sdd watch foo` is running and idle
- **WHEN** a file `openspec/changes/foo/.cache/propose.ctx` is written
- **THEN** no reassembly occurs (the debounce timer is not started or reset)

#### Scenario: Pending directory write does not trigger reassembly `code-based` `critical`

- **GIVEN** `sdd watch foo` is running and idle
- **WHEN** a file `openspec/changes/foo/.pending/spec.md` is written
- **THEN** no reassembly occurs

#### Scenario: Artifact write does trigger reassembly `code-based` `critical`

- **GIVEN** `sdd watch foo` is running and idle
- **WHEN** a file `openspec/changes/foo/proposal.md` is written
- **THEN** the debounce timer is started (or reset), and reassembly occurs after the debounce interval elapses

---

### REQ-SPEC-008: Debounced Reassembly

When a non-filtered filesystem event is received, the watcher **MUST** start or reset a debounce timer. The watcher **MUST NOT** invoke `context.Assemble` until the debounce timer fires (no events received for the configured debounce interval).

Multiple events within the debounce window **MUST** coalesce into a single reassembly invocation.

The debounce timer **MUST** be safe for concurrent access (events arrive from the fsnotify goroutine; the timer callback runs on a separate goroutine).

#### Scenario: Single file change triggers one reassembly `code-based` `critical`

- **GIVEN** `sdd watch foo` is running with default 300ms debounce
- **WHEN** `openspec/changes/foo/proposal.md` is modified once
- **THEN** exactly one `context.Assemble` call occurs approximately 300ms after the modification

#### Scenario: Rapid multi-file saves coalesce into one reassembly `code-based` `critical`

- **GIVEN** `sdd watch foo` is running with default 300ms debounce
- **WHEN** 5 files under `openspec/changes/foo/` are modified within 100ms of each other
- **THEN** exactly one `context.Assemble` call occurs approximately 300ms after the last modification

#### Scenario: Widely spaced saves trigger separate reassemblies `code-based` `critical`

- **GIVEN** `sdd watch foo` is running with default 300ms debounce
- **WHEN** `proposal.md` is modified at T=0, and `design.md` is modified at T=600ms
- **THEN** two separate `context.Assemble` calls occur: one at approximately T=300ms and one at approximately T=900ms

---

### REQ-SPEC-009: Phase Re-Resolution on Each Trigger

On each debounce fire, the watcher **MUST** re-read `state.json` from disk and call `ReadyPhases()` to determine the current phase(s) before invoking `context.Assemble`.

This ensures that if `sdd write foo propose` is run between triggers (advancing the state machine), the next reassembly targets the new current phase.

#### Scenario: Phase advances between triggers `code-based` `critical`

- **GIVEN** `sdd watch foo` is running, state.json has `"propose"` pending, and `ReadyPhases()` returns `["propose"]`
- **WHEN** `sdd write foo propose` completes (advancing state to `"spec"` pending), and then an artifact in the change directory is modified
- **THEN** the watcher re-reads `state.json`, `ReadyPhases()` returns `["spec","design"]`, and `context.AssembleConcurrent` is invoked for the `spec+design` parallel window

#### Scenario: state.json read failure logs error and continues `code-based` `critical`

- **GIVEN** `sdd watch foo` is running
- **WHEN** `state.json` is temporarily unreadable (e.g., mid-write) and the debounce timer fires
- **THEN** the watcher logs an error to stderr via `slog.Error` and continues watching (does not exit)

---

### REQ-SPEC-010: Context Output to Stdout

On each successful reassembly, the watcher **MUST** write the assembled context to stdout, using the same format as `sdd context <name>`.

If `ReadyPhases()` returns multiple phases (spec+design parallel window), the watcher **MUST** use `context.AssembleConcurrent`.

If `ReadyPhases()` returns zero phases (pipeline complete or blocked), the watcher **SHOULD** log a message to stderr via `slog.Info` and skip reassembly without exiting.

#### Scenario: Reassembly output matches sdd context format `code-based` `critical`

- **GIVEN** `sdd watch foo` is running and the current phase is `"propose"`
- **WHEN** a reassembly is triggered
- **THEN** stdout receives the same assembled context that `sdd context foo propose` would produce, including the `<!-- sdd:model=... -->` directive

#### Scenario: No ready phases logs info and skips reassembly `code-based` `standard`

- **GIVEN** `sdd watch foo` is running and all phases are completed (pipeline complete)
- **WHEN** a file modification triggers the debounce timer
- **THEN** `slog.Info` emits a message to stderr containing "no phases ready" and no context is written to stdout

---

### REQ-SPEC-011: WatchReassembled Event

After each successful reassembly, the watcher **MUST** emit a `WatchReassembled` event through the event broker.

The `WatchReassembled` event type **MUST** be defined as a constant in `internal/events/broker.go`.

A `WatchReassembledPayload` struct **MUST** be defined with at least the fields: `Change` (string), `Phase` (string), and `DurationMs` (int64).

#### Scenario: Event emitted after successful reassembly `code-based` `critical`

- **GIVEN** `sdd watch foo` is running with a broker and a subscriber on `WatchReassembled`
- **WHEN** a reassembly completes successfully for phase `"propose"`
- **THEN** the subscriber receives an `Event` with `Type == WatchReassembled` and `Payload.(WatchReassembledPayload)` containing `Change == "foo"`, `Phase == "propose"`, and `DurationMs > 0`

#### Scenario: Event not emitted on reassembly error `code-based` `critical`

- **GIVEN** `sdd watch foo` is running with a broker and a subscriber on `WatchReassembled`
- **WHEN** `context.Assemble` returns an error (e.g., missing artifact)
- **THEN** no `WatchReassembled` event is emitted, and the error is logged to stderr

---

### REQ-SPEC-012: Signal Handling and Graceful Shutdown

The `watch` command **MUST** handle SIGINT and SIGTERM signals using `signal.NotifyContext`, consistent with the `dashboard` command pattern.

Upon receiving a signal, the watcher **MUST** close the fsnotify watcher, drain pending events, and exit with code 0.

The shutdown **MUST NOT** leak goroutines or leave orphaned file descriptors.

#### Scenario: SIGINT triggers clean shutdown `code-based` `critical`

- **GIVEN** `sdd watch foo` is running
- **WHEN** the process receives SIGINT (Ctrl-C)
- **THEN** the fsnotify watcher is closed, the `watch.Run` function returns nil, and the process exits with code 0

#### Scenario: SIGTERM triggers clean shutdown `code-based` `critical`

- **GIVEN** `sdd watch foo` is running
- **WHEN** the process receives SIGTERM
- **THEN** the fsnotify watcher is closed, the `watch.Run` function returns nil, and the process exits with code 0

#### Scenario: No goroutine leak after shutdown `code-based` `critical`

- **GIVEN** `sdd watch foo` has been running for at least one reassembly cycle
- **WHEN** the context is cancelled and `watch.Run` returns
- **THEN** the number of goroutines returns to the pre-watch baseline (verified via `runtime.NumGoroutine()` in tests)

---

### REQ-SPEC-013: Error Recovery During Watch

If `context.Assemble` returns an error during a reassembly trigger, the watcher **MUST** log the error to stderr via `slog.Error` and continue watching. The watcher **MUST NOT** exit on assembly errors.

#### Scenario: Assembly error is logged and watch continues `code-based` `critical`

- **GIVEN** `sdd watch foo` is running
- **WHEN** `context.Assemble` returns an error (e.g., `"no assembler for phase: unknown"`)
- **THEN** stderr receives an `slog.Error` log line containing the error message, and the watcher remains active (subsequent file changes still trigger reassembly)

---

### REQ-SPEC-014: Watch Package Boundary

The watch logic **MUST** reside in `internal/watch/` as a standalone package.

The `watch` package **MUST NOT** import `internal/cli`. The dependency direction is: `cli` -> `watch`, not the reverse.

The `watch.Run` function **MUST** accept a `context.Context` for cancellation and an options struct (or functional options) containing: the change directory path, the debounce duration, an `io.Writer` for context output, an `io.Writer` for error/log output, and a reassembly callback or the necessary parameters to invoke `context.Assemble`.

This decoupling **SHOULD** enable reuse by the future daemon mode (roadmap 6.3) without modification.

#### Scenario: watch.Run blocks until context is cancelled `code-based` `critical`

- **WHEN** `watch.Run(ctx, opts)` is called with a valid change directory
- **THEN** the function blocks until `ctx` is cancelled, then returns nil

#### Scenario: watch package does not import cli `code-based` `critical`

- **WHEN** the import graph of `internal/watch` is analyzed
- **THEN** no import path contains `internal/cli`

---

### REQ-SPEC-015: fsnotify External Dependency

The project **MUST** add `github.com/fsnotify/fsnotify` (v1.8.x) to `go.mod`.

The dependency **MUST** be pure Go (no CGO requirement from fsnotify itself; the project's existing `CGO_ENABLED=1` setting is unaffected).

#### Scenario: fsnotify appears in go.mod after implementation `code-based` `critical`

- **WHEN** `go list -m github.com/fsnotify/fsnotify` is run in the `sdd-cli/` directory
- **THEN** the output shows `github.com/fsnotify/fsnotify v1.8.x` (where x is a valid minor version)

---

### REQ-SPEC-016: Unknown Flag Rejection

The `watch` command **MUST** reject unknown flags with an `errs.Usage` error, consistent with the existing `errUnknownFlag` helper used by other commands.

#### Scenario: Unknown flag returns usage error `code-based` `critical`

- **WHEN** `sdd watch foo --invalid-flag` is invoked
- **THEN** the command returns an `errs.Usage` error containing "unknown flag" and exits with code 2

---

### REQ-SPEC-017: Reassembly Separator on Stderr

The watcher **SHOULD** emit a human-readable separator line to stderr before each reassembly output (after the first), to help users visually distinguish successive context dumps in terminal output.

The separator **MUST NOT** be written to stdout, to preserve stdout as a clean pipe-friendly stream.

#### Scenario: Separator appears on stderr between reassemblies `code-based` `standard`

- **GIVEN** `sdd watch foo` is running
- **WHEN** a second reassembly is triggered
- **THEN** stderr contains a line matching the pattern `--- reassembled at HH:MM:SS ---` before the second context output, and stdout does not contain this line

---

## Acceptance Criteria Summary

| Requirement ID   | Type  | Priority | Scenarios |
|------------------|-------|----------|-----------|
| REQ-SPEC-001     | ADDED | MUST     | 2         |
| REQ-SPEC-002     | ADDED | MUST     | 2         |
| REQ-SPEC-003     | ADDED | MUST     | 1         |
| REQ-SPEC-004     | ADDED | MUST     | 1         |
| REQ-SPEC-005     | ADDED | MUST     | 4         |
| REQ-SPEC-006     | ADDED | MUST     | 2         |
| REQ-SPEC-007     | ADDED | MUST     | 3         |
| REQ-SPEC-008     | ADDED | MUST     | 3         |
| REQ-SPEC-009     | ADDED | MUST     | 2         |
| REQ-SPEC-010     | ADDED | MUST     | 2         |
| REQ-SPEC-011     | ADDED | MUST     | 2         |
| REQ-SPEC-012     | ADDED | MUST     | 3         |
| REQ-SPEC-013     | ADDED | MUST     | 1         |
| REQ-SPEC-014     | ADDED | MUST     | 2         |
| REQ-SPEC-015     | ADDED | MUST     | 1         |
| REQ-SPEC-016     | ADDED | MUST     | 1         |
| REQ-SPEC-017     | ADDED | SHOULD   | 1         |

**Total Requirements**: 17
**Total Scenarios**: 33

## Eval Definitions

| Scenario | Eval Type | Criticality | Threshold |
|----------|-----------|-------------|-----------|
| REQ-SPEC-001 > Watch command dispatches to runWatch handler | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-001 > Watch command appears in help output | code-based | standard | pass@3 >= 0.90 |
| REQ-SPEC-002 > Watch nonexistent change returns error JSON | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-002 > Watch valid change does not error on startup | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-003 > Watch with no args returns usage error | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-004 > Startup message structure | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-005 > Default debounce is 300ms | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-005 > Custom debounce value is respected | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-005 > Invalid debounce value returns usage error | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-005 > Non-numeric debounce value returns usage error | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-006 > All existing subdirectories are watched at startup | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-006 > Newly created subdirectory is dynamically watched | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-007 > Cache write does not trigger reassembly | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-007 > Pending directory write does not trigger reassembly | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-007 > Artifact write does trigger reassembly | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-008 > Single file change triggers one reassembly | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-008 > Rapid multi-file saves coalesce into one reassembly | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-008 > Widely spaced saves trigger separate reassemblies | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-009 > Phase advances between triggers | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-009 > state.json read failure logs error and continues | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-010 > Reassembly output matches sdd context format | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-010 > No ready phases logs info and skips reassembly | code-based | standard | pass@3 >= 0.90 |
| REQ-SPEC-011 > Event emitted after successful reassembly | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-011 > Event not emitted on reassembly error | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-012 > SIGINT triggers clean shutdown | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-012 > SIGTERM triggers clean shutdown | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-012 > No goroutine leak after shutdown | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-013 > Assembly error is logged and watch continues | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-014 > watch.Run blocks until context is cancelled | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-014 > watch package does not import cli | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-015 > fsnotify appears in go.mod after implementation | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-016 > Unknown flag returns usage error | code-based | critical | pass^3 = 1.00 |
| REQ-SPEC-017 > Separator appears on stderr between reassemblies | code-based | standard | pass@3 >= 0.90 |

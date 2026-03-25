# Exploration: file-watcher

**Change:** Debounced recursive file watcher that re-runs context assembly when artifacts change (sdd --watch mode)

---

## Current State

The SDD CLI is a single-shot tool: every command runs, produces output, and exits. There is no persistent watch loop. The roadmap (docs/roadmap.md §6.1) explicitly names this as "Not implemented" and identifies the upstream inspiration as `canvas internal/watch/`.

Today, the workaround documented in the roadmap is `watchexec -e md -- sdd context foo` — a third-party tool that wraps `sdd context`. The proposed native implementation would eliminate the external dependency and give the CLI first-class watch semantics: track the `openspec/changes/{name}/` tree, debounce rapid multi-file saves, re-assemble context, and print to stdout.

Go 1.26 (the project's runtime, confirmed via `go version`) does **not** include a native filesystem notification API in the standard library (`os`, `os/signal`, `io/fs`). The stdlib offers `filepath.WalkDir` and `os.Stat` for polling but no kernel-level inotify/kqueue/FSEvents abstraction. The canonical third-party library for this in the Go ecosystem is `github.com/fsnotify/fsnotify` (v1.8.x), which wraps inotify (Linux), kqueue (macOS/BSD), and ReadDirectoryChangesW (Windows). It has no CGO dependency and is a single external module.

Context assembly (`internal/context`) is already fully re-invokable as a library function: `context.Assemble(w io.Writer, ph state.Phase, p *Params) error`. The `sdd context` command in `internal/cli/cmd_context.go` is a thin wrapper that resolves paths, loads config/state, wires a broker, and calls `Assemble`. That same code path can be called in a loop from a watcher goroutine with no modification.

The event broker (`internal/events/broker.go`) is a panic-safe, mutex-guarded pub/sub bus already used for metrics, cache persistence, and error logging. It is nil-safe and concurrency-safe. It is a natural fit for emitting `WatchReassembled` events (for telemetry/dashboard) but is not strictly required for the MVP.

The `sdd dashboard` command (`internal/cli/cmd_dashboard.go`) provides the only existing long-running command pattern: `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` + a goroutine that blocks until context cancellation. This is the correct pattern for `sdd watch`.

---

## Relevant Files

| Path | Purpose | Lines | Complexity | Test Coverage |
|------|---------|-------|-----------|---------------|
| `internal/cli/cli.go` | Command dispatch switch; where `"watch"` case would be added | 394 | Low | `cli_test.go`, `integration_test.go` |
| `internal/cli/cmd_context.go` | `runContext` — the function that wires params and calls `context.Assemble`; re-used by watch | 139 | Low | `cmd_context_test.go`, `cmd_context_happy_test.go` |
| `internal/cli/commands.go` | Shared helpers: `loadChangeState`, `getProjectRoot`, `loadConfig`, `newBroker`, `writeJSON` | 606 | Medium | `commands_test.go` |
| `internal/cli/cmd_dashboard.go` | Long-running command pattern: `signal.NotifyContext` + goroutine | 74 | Low | None (happy path only) |
| `internal/context/context.go` | `Assemble(w, ph, p)` and `AssembleConcurrent` — the callable entry points | 309 | Medium | `context_test.go` (extensive) |
| `internal/context/cache.go` | Content-hash caching; `tryCachedContext`, `saveContextCache` | 398 | Medium | `cache_test.go` (extensive) |
| `internal/context/subscribers.go` | `RegisterSubscribers` — wires metrics, cache persistence, error logging to broker | 56 | Low | `subscribers_test.go` |
| `internal/events/broker.go` | Thread-safe pub/sub; `Emit`, `Subscribe`, `Handler`; nil-safe | 140 | Low | `broker_test.go`, `chaos_test.go` |
| `internal/phase/phase.go` | `AssemblerParams` struct; `Assembler` func type | 129 | Low | `registry_test.go` |
| `internal/phase/registry.go` | `DefaultRegistry.Get/All/AllNames` — phase descriptors with `CacheInputs` | 103 | Low | `registry_test.go` |
| `internal/state/state.go` | `Load`, `ReadyPhases`, `Advance` | 223 | Medium | `state_test.go`, `fuzz_test.go` |
| `internal/state/types.go` | `Phase`, `PhaseStatus`, `State` struct | 86 | Low | (types) |
| `internal/csync/lazyslice.go` | Bounded goroutine pool for concurrent artifact loading | 118 | Low | `lazyslice_test.go` |
| `internal/sddlog/sddlog.go` | `slog`-based structured logging; `SDD_LOG` / `SDD_LOG_FILE` env vars | 44 | Low | `sddlog_test.go` |
| `internal/fsutil/atomic.go` | `AtomicWrite` — used by cache; relevant if watcher writes anything | 34 | Low | `atomic_test.go` |
| `cmd/sdd/main.go` | Entry point; panic recovery; `sddlog.Init`; `cli.Run` | 108 | Low | (binary) |
| `docs/roadmap.md` | §6.1 specifies debounce pattern + use case | — | — | — |

---

## Dependency Map

```
sdd watch <name>
    └── internal/cli/cmd_watch.go  (new)
            ├── getProjectRoot()            ← internal/cli/commands.go
            ├── loadChangeState()           ← internal/cli/commands.go
            ├── loadConfig()                ← internal/cli/commands.go
            ├── newBroker()                 ← internal/cli/commands.go
            │       ├── events.NewBroker()  ← internal/events/broker.go
            │       └── context.RegisterSubscribers()  ← internal/context/subscribers.go
            └── internal/watch/watcher.go  (new)
                    ├── github.com/fsnotify/fsnotify  (new external dep)
                    ├── context.Assemble()  ← internal/context/context.go
                    │       ├── phase.DefaultRegistry.Get()  ← internal/phase/registry.go
                    │       ├── tryCachedContext()           ← internal/context/cache.go
                    │       └── desc.Assemble()              ← per-phase assembler
                    └── state.Load()        ← internal/state/state.go
```

The new `internal/watch` package has no imports from `internal/cli`, keeping the boundary clean. `cmd_watch.go` owns CLI concerns (flag parsing, signal handling, JSON output), and `internal/watch` owns debounce + re-assembly logic.

**Import constraints (must be respected):**
- `internal/phase` must NOT import `internal/state`, `internal/context`, or `internal/cli` (enforced by existing import cycle rules)
- `internal/watch` should import `internal/context`, `internal/events`, `internal/state` — no cycles
- `internal/cli/cmd_watch.go` imports `internal/watch` — same pattern as `cmd_dashboard.go` → `internal/dashboard`

---

## Data Flow

**Normal watch cycle:**

```
fsnotify event (Create/Write/Remove/Rename)
    │
    ▼
Watcher.onEvent(event)
    │  mutex.Lock()
    ▼
debounce timer reset (time.AfterFunc / timer.Reset)
    │  mutex.Unlock()
    ▼
[debounce interval elapses — e.g. 300ms]
    │
    ▼
Watcher.reassemble()
    ├── state.Load(changeDir/state.json)        ← read current phase
    ├── st.ReadyPhases()                        ← determine what to assemble
    ├── context.Assemble(w, phase, params)      ← may hit cache (hash match)
    │       ├── tryCachedContext() → HIT?       → write cached bytes → emit CacheHit
    │       └── miss → assemble → emit PhaseAssembled (Cached:false)
    └── emit WatchReassembled event             ← broker (new event type)

Output: assembled context written to stdout (same as `sdd context`)
```

**Files watched** for `sdd watch <name>`:
- `openspec/changes/{name}/` — entire subtree (recursive watch via fsnotify)
- This covers all artifact files: `exploration.md`, `proposal.md`, `design.md`, `specs/*.md`, `tasks.md`, `review-report.md`, etc.
- `.cache/` and `.pending/` subdirs are under the watched root but writes to them should be filtered (otherwise cache writes trigger re-assembly loops)

**Filter rule:** Ignore events where path contains `/.cache/` or `/.pending/` to break the feedback loop. The cache subscriber in `context/subscribers.go` writes to `{changeDir}/.cache/{phase}.ctx` — if not filtered, a reassembly triggers a cache write which triggers another reassembly.

**Debounce mechanics (mutex-guarded timer reset, as roadmap specifies):**

```go
type Watcher struct {
    mu       sync.Mutex
    timer    *time.Timer
    debounce time.Duration
    reassemble func()
}

func (w *Watcher) onEvent() {
    w.mu.Lock()
    defer w.mu.Unlock()
    if w.timer != nil {
        w.timer.Stop()
    }
    w.timer = time.AfterFunc(w.debounce, w.reassemble)
}
```

This is the exact pattern described in the roadmap: "mutex-guarded timer reset prevents thrashing on rapid multi-file saves."

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Feedback loop: cache writes trigger re-assembly loop | High | High | Filter events from `.cache/` and `.pending/` paths before debounce |
| fsnotify adds external dependency to zero-dep binary (go.mod) | Certain | Medium | Acceptable: fsnotify is the standard Go ecosystem solution; no CGO, single module. Evaluate stdlib polling as an alternative (see Approach Comparison). |
| Rapid multi-file saves cause thrashing | High (real editor behavior) | Medium | Debounce timer (300ms default) with mutex-guarded reset solves this |
| `sdd watch` and `sdd context` run concurrently, both write cache | Low | Low | Cache writes are already atomic (`fsutil.AtomicWrite`); last writer wins; content-hash guards correctness |
| Signal handling: SIGINT mid-assembly leaves partial stdout | Low | Low | `context.Assemble` writes to an `io.Writer`; cancel context before writing, or accept partial output (human-readable, not machine-consumed) |
| `state.json` changes while watching cause stale phase reads | Medium | Low | Reload `state.json` on each debounce fire; `state.Load` is pure read |
| Recursive watch on large repos picks up noise outside change dir | Low | Low | Watch root is scoped to `openspec/changes/{name}/` — small, bounded tree |
| fsnotify does not support recursive watch on all platforms | Medium | Medium | fsnotify v1.8.x requires `watcher.Add()` per subdirectory; `openspec/changes/{name}/` is shallow (max 2 levels: root + `specs/`); enumerate and add all subdirs at startup |
| golangci-lint `noctx` linter: watch loop must pass context to blocking calls | Certain | Low | Use `context.Context` parameter throughout; watcher exits when ctx is done |

---

## Approach Comparison

| Approach | Pros | Cons | Verdict |
|----------|------|------|---------|
| **A. `sdd watch <name>` — standalone command** | Clean separation; `sdd context` unchanged; clearly long-running; easy to document | Adds new case to `cli.go` switch; new `cmd_watch.go` + new `internal/watch/` package | Recommended |
| **B. `sdd context <name> --watch` — flag on existing command** | No new command name to learn; reuses existing entry point | `runContext` already has 5 flags; mixing one-shot + long-running in the same command is confusing; harder to test | Not recommended |
| **C. `sdd --watch <name>` — top-level flag before subcommand** | Looks like the roadmap spec literally | CLI dispatch in `cli.go` does `args[0]` as command; `--watch` at position 0 is non-standard; breaks pattern | Not recommended |
| **D. stdlib polling via `os.Stat` + `filepath.WalkDir`** | Zero new dependency; pure stdlib | Must poll every N ms across entire subtree; O(files) per poll; misses renames cleanly; 100–500ms minimum latency; busy CPU | Fallback only if fsnotify blocked |
| **E. `internal/watch` package with optional build tag for polling fallback** | Platform portability; fsnotify as primary, polling as fallback | Adds build complexity; two code paths to test | Overkill for MVP; revisit if needed |
| **F. Integrate with 6.3 daemon mode** | Single persistent process; cache stays warm | 6.3 daemon is unimplemented and higher complexity | Deferred; watch is useful standalone |

**Recommended approach: A — `sdd watch <name>` as a standalone command backed by a new `internal/watch` package.**

---

## Recommendation

Build `sdd watch <name>` as a new command with the following structure:

**New files:**
1. `internal/watch/watcher.go` — core debounce + fsnotify loop + `context.Assemble` invocation
2. `internal/cli/cmd_watch.go` — CLI glue: flag parsing, signal handling, JSON startup message

**Modified files:**
1. `internal/cli/cli.go` — add `"watch"` case to `Run()` switch + help text
2. `go.mod` / `go.sum` — add `github.com/fsnotify/fsnotify v1.8.x`
3. `internal/events/broker.go` — add `WatchReassembled` event type + payload (optional for MVP)

**New event type (optional MVP):**
```go
WatchReassembled EventType = "WatchReassembled"
type WatchReassembledPayload struct {
    Change     string
    Phase      string
    TriggerPath string
    DurationMs int64
    Cached     bool
}
```

**Debounce interval:** 300ms default, configurable via `--debounce` flag (milliseconds). This matches standard editor save-debounce patterns (VS Code uses 250–500ms).

**Watch scope:** `openspec/changes/{name}/` recursive. Filter out events under `.cache/` and `.pending/` subdirectories.

**fsnotify non-recursive workaround:** At startup, enumerate existing subdirectories (only `specs/` currently; at most a handful) and `watcher.Add()` each. On `Create` events for new directories, add them dynamically.

**Context destination:** stdout (same as `sdd context`). The watch loop continuously re-prints assembled context on each debounce fire. This is intentional: sub-agents tail the output or re-read it.

**Startup JSON output** (consistent with dashboard):
```json
{"command":"watch","status":"watching","change":"<name>","phase":"<phase>","dir":"<changeDir>"}
```

**Long-running pattern** (from `cmd_dashboard.go`):
```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
return watch.Run(ctx, params, stdout, stderr)
```

---

## Clarification Required (BLOCKING)

**None identified.** The feature is well-specified in the roadmap, all integration points are clear, and the existing codebase provides all necessary primitives. The only open question is whether to accept fsnotify as a new dependency — given the project already uses `modernc.org/sqlite` (a significantly heavier CGO-alternative dep), adding fsnotify (pure Go, widely used) is a reasonable trade-off. If the maintainer objects, stdlib polling is a viable fallback.

---

## Open Questions (DEFERRED)

1. **Debounce interval configurability:** Should `--debounce` be exposed as a flag, or hard-coded to 300ms? The roadmap does not specify. Recommend exposing it for power users but defaulting to 300ms.

2. **What should be written to stdout between reassemblies?** Options: (a) silence until next trigger, (b) a separator line `--- [reassembled at 14:32:01] ---`, (c) re-emit the full context each time. The roadmap says "context auto-reassembles → sub-agent picks up fresh context" — suggesting (c) full re-emit is the intended behavior.

3. **Should `--watch` be a flag on `sdd context` for discoverability?** A thin `sdd context <name> --watch` alias that delegates to `sdd watch <name>` could help users discover the feature. Not critical for MVP.

4. **Integration with daemon mode (6.3):** The daemon roadmap entry mentions "watches for artifact changes, pre-assembles likely next phases." `sdd watch` could become an early slice of that capability without requiring the full Unix socket RPC infrastructure. Whether to design `internal/watch` as a reusable component for the future daemon is worth considering but not blocking.

5. **`WatchReassembled` event integration with dashboard:** The dashboard polls SQLite every 3s. Watcher events could be stored in the same `phase_events` table (or a new `watch_events` table) to surface reassembly activity in the dashboard. Deferred; MVP can skip event emission and just write to stdout/slog.

6. **Platform support:** fsnotify supports Linux (inotify), macOS (kqueue), Windows (ReadDirectoryChangesW), BSD. The project currently targets Linux/WSL (confirmed by env). fsnotify works on all. Deferred testing on non-Linux platforms.

7. **Rate limiting in addition to debouncing:** Should there be a max-reassemblies-per-minute guard for pathological cases (e.g., a script writing files in a tight loop)? Probably no for MVP — debounce covers the common case.

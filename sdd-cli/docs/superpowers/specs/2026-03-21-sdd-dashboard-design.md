# sdd dashboard — Ops Dashboard Design

## Problem

Pipeline state, token usage, cache efficiency, and verify errors are only visible via CLI commands (`sdd health`, `sdd errors`, `sdd dump`). During active development with multiple changes, there's no persistent overview. You have to re-run commands to check status.

## Solution

`sdd dashboard` — a local web server that shows a live ops dashboard on a second screen. Dark theme, htmx auto-poll at 3s, Grafana-style grid layout. SQLite backs the metrics so history accumulates across sessions.

## Architecture

```
Event flow:
  assembler/verify → broker → SQLite subscriber → sdd.db
                                                      ↑
  browser ←─ htmx poll 3s ←─ Go HTTP server ←─ reads ┘

State flow:
  openspec/changes/*/state.json ←─ Go HTTP server reads directly
```

Two data sources:
1. **SQLite** (`openspec/.cache/sdd.db`) — phase_events and verify_events for metrics + errors. Written by broker subscribers regardless of whether dashboard is running.
2. **Filesystem** — `state.json` files for live pipeline status. State machine is the source of truth.

### errlog deprecation

The existing `internal/errlog/` package (writes to `openspec/.cache/errors.json`) is **superseded** by SQLite `verify_events`. Migration plan:
- `sdd errors` reads from SQLite if `sdd.db` exists, falls back to `errors.json` if not.
- The JSON-based errlog subscriber remains for backward compatibility but is not extended.
- No data migration — existing `errors.json` entries are not imported into SQLite.

## Packages

### `internal/store/` — SQLite persistence

**File:** `internal/store/store.go`

```go
type Store struct {
    db *sql.DB
}

func Open(path string) (*Store, error)   // opens DB, runs pragmas + migrations
func (s *Store) Close() error

// Inserts (called by broker subscribers)
func (s *Store) InsertPhaseEvent(ctx context.Context, e PhaseEvent) error
func (s *Store) InsertVerifyEvent(ctx context.Context, e VerifyEvent) error

// Queries (called by dashboard handlers)
func (s *Store) TokenSummary(ctx context.Context) (*TokenStats, error)
func (s *Store) PhaseTokensByChange(ctx context.Context) ([]ChangeTokens, error)
func (s *Store) RecentErrors(ctx context.Context, limit int) ([]ErrorRow, error)
```

**SQLite pragmas** (applied on Open):
```sql
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA temp_store=MEMORY;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
```

**Schema** (inline migrations, `CREATE TABLE IF NOT EXISTS`):
```sql
CREATE TABLE IF NOT EXISTS phase_events (
    id INTEGER PRIMARY KEY,
    timestamp TEXT NOT NULL,
    change TEXT NOT NULL,
    phase TEXT NOT NULL,
    bytes INTEGER NOT NULL,
    tokens INTEGER NOT NULL,
    cached BOOLEAN NOT NULL,
    duration_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS verify_events (
    id INTEGER PRIMARY KEY,
    timestamp TEXT NOT NULL,
    change TEXT NOT NULL,
    command_name TEXT NOT NULL,
    command TEXT NOT NULL,
    exit_code INTEGER NOT NULL,
    error_lines TEXT,
    fingerprint TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_phase_events_ts ON phase_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_verify_events_ts ON verify_events(timestamp);

PRAGMA user_version = 1;  -- schema version, checked on Open
```

**DB path:** `openspec/.cache/sdd.db`

**Types:**
```go
type PhaseEvent struct {
    Timestamp  time.Time
    Change     string
    Phase      string
    Bytes      int
    Tokens     int
    Cached     bool
    DurationMs int64
}

type VerifyEvent struct {
    Timestamp   time.Time
    Change      string
    CommandName string
    Command     string
    ExitCode    int
    ErrorLines  []string
    Fingerprint string
}

// TokenStats — aggregated from phase_events (SQLite only)
type TokenStats struct {
    TotalTokens int
    CacheHitPct float64
    ErrorCount  int
}

// ChangeTokens — per-change token total from phase_events
type ChangeTokens struct {
    Change string
    Tokens int
}

// ErrorRow — single verify failure
type ErrorRow struct {
    Timestamp   string
    CommandName string
    Command     string
    ExitCode    int
    Change      string
    Fingerprint string
    FirstLine   string // first error line for diagnostics
}
```

**File:** `internal/store/store_test.go`
- Table-driven tests
- SQLite in `t.TempDir()` (not in-memory, tests real file I/O)
- Roundtrip: insert + query
- TokenStats aggregation correctness (cache hit %, total tokens)
- RecentErrors ordering (newest first)
- Empty DB returns zero values, not errors

### `internal/dashboard/` — HTTP server + templates

**File:** `internal/dashboard/server.go`

```go
// Consumer-defined interfaces — dashboard declares what it needs from SQLite
type MetricsReader interface {
    TokenSummary(ctx context.Context) (*store.TokenStats, error)
    PhaseTokensByChange(ctx context.Context) ([]store.ChangeTokens, error)
    RecentErrors(ctx context.Context, limit int) ([]store.ErrorRow, error)
}

type Server struct {
    metrics    MetricsReader
    changesDir string       // path to openspec/changes/ for state.json reads
    templates  *template.Template
    httpServer *http.Server // for graceful shutdown
}

func New(m MetricsReader, changesDir string) *Server
func (s *Server) ListenAndServe(ctx context.Context, addr string) error
```

`ListenAndServe` accepts a `context.Context` for graceful shutdown. When the context is cancelled (e.g., SIGINT/SIGTERM), calls `httpServer.Shutdown(shutdownCtx)` with a 3-second deadline, then `returns`.

**Routes:**
| Route | Method | Returns | Poll |
|-------|--------|---------|------|
| `GET /` | Full page | base.html with htmx | Once |
| `GET /fragments/kpi` | HTML fragment | KPI cards | every 3s |
| `GET /fragments/pipelines` | HTML fragment | Pipeline table | every 3s |
| `GET /fragments/errors` | HTML fragment | Error log | every 3s |

**Handler data flow:**

The handlers merge data from two sources:

- `/fragments/kpi` — `ActiveChanges` from filesystem scan (count dirs with `state.json`), `TotalTokens` + `CacheHitPct` + `ErrorCount` from `MetricsReader.TokenSummary()`.
- `/fragments/pipelines` — reads each `state.json` for phase status (CurrentPhase, Completed, Total, Status), joins with `MetricsReader.PhaseTokensByChange()` for per-change token counts.
- `/fragments/errors` — purely from `MetricsReader.RecentErrors(ctx, 20)`.

This means `ActiveChanges` and pipeline phase status come from the filesystem (source of truth), while token metrics and error history come from SQLite.

**Templates** (embedded via `embed.FS`):

`internal/dashboard/templates/base.html`
- Full page skeleton
- Dark theme CSS inline (no external stylesheet)
- htmx JS embedded in `embed.FS` (not CDN — works offline)
- Three `<div>` slots with `hx-get="/fragments/..."` and `hx-trigger="load, every 3s"`

`internal/dashboard/templates/kpi.html`
- 4 KPI cards: Active Changes, Total Tokens, Cache %, Errors
- Color-coded left borders (cyan, green, purple, red)

`internal/dashboard/templates/pipelines.html`
- Table: Change | Phase | Progress | Tokens | Status
- Progress bar via inline CSS width percentage
- Status dot: green (ok), yellow (stale), red (failed verify)

`internal/dashboard/templates/errors.html`
- Recent verify failures, newest first
- Timestamp, command, exit code, change name, fingerprint prefix, first error line
- Red for errors, yellow for warnings

`internal/dashboard/static/htmx.min.js`
- htmx 2.0.4 minified (~14KB), embedded via `embed.FS`
- Served at `/static/htmx.min.js`

**File:** `internal/dashboard/server_test.go`
- `httptest.NewServer` tests for each fragment endpoint
- Fake `MetricsReader` implementation
- Verify HTML contains expected elements
- Test empty-state rendering (no changes, no errors)

### Broker Wiring

**File:** `internal/store/subscribers.go` — subscriber registration lives in the store package

```go
// RegisterSubscribers wires SQLite event subscribers to the broker.
// Safe to call with nil broker or nil store (no-op).
func RegisterSubscribers(broker *events.Broker, s *Store) {
    if broker == nil || s == nil {
        return
    }

    broker.Subscribe(events.PhaseAssembled, func(e events.Event) {
        p, ok := e.Payload.(events.PhaseAssembledPayload)
        if !ok { return }
        _ = s.InsertPhaseEvent(context.Background(), PhaseEvent{
            Timestamp:  time.Now().UTC(),
            Change:     filepath.Base(p.ChangeDir),
            Phase:      p.Phase,
            Bytes:      p.Bytes,
            Tokens:     p.Tokens,
            Cached:     p.Cached,
            DurationMs: p.DurationMs,
        })
    })

    broker.Subscribe(events.VerifyFailed, func(e events.Event) {
        p, ok := e.Payload.(events.VerifyFailedPayload)
        if !ok { return }
        for _, cmd := range p.Results {
            _ = s.InsertVerifyEvent(context.Background(), VerifyEvent{
                Timestamp:   time.Now().UTC(),
                Change:      p.Change,
                CommandName: cmd.Name,
                Command:     cmd.Command,
                ExitCode:    cmd.ExitCode,
                ErrorLines:  cmd.ErrorLines,
                Fingerprint: errlog.Fingerprint(cmd.Command, cmd.ErrorLines),
            })
        }
    })
}
```

**Key design:** Subscribers live in `internal/store/`, not `internal/context/`. This avoids pulling SQLite dependencies into the context package. The `newBroker()` function in `commands.go` calls `store.RegisterSubscribers(broker, db)` when the store is available.

### CLI Command

**File:** `internal/cli/commands.go` — new `runDashboard`

```go
func runDashboard(args []string, stdout io.Writer, stderr io.Writer) error {
    port := "8811"
    for i, arg := range args {
        switch {
        case (arg == "--port" || arg == "-p") && i+1 < len(args):
            port = args[i+1]
        }
    }

    // Validate port
    p, err := strconv.Atoi(port)
    if err != nil || p < 1024 || p > 65535 {
        return errs.Usage(fmt.Sprintf("invalid port: %s (must be 1024-65535)", port))
    }

    cwd, _ := os.Getwd()
    dbPath := filepath.Join(cwd, "openspec", ".cache", "sdd.db")
    changesDir := filepath.Join(cwd, "openspec", "changes")

    db, err := store.Open(dbPath)
    if err != nil {
        return errs.WriteError(stderr, "dashboard", fmt.Errorf("open store: %w", err))
    }
    defer db.Close()

    srv := dashboard.New(db, changesDir)
    addr := "127.0.0.1:" + port

    // JSON output
    out := struct {
        Command string `json:"command"`
        Status  string `json:"status"`
        URL     string `json:"url"`
    }{
        Command: "dashboard",
        Status:  "running",
        URL:     "http://" + addr,
    }
    data, _ := json.MarshalIndent(out, "", "  ")
    fmt.Fprintln(stdout, string(data))

    // Signal handling for graceful shutdown
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    slog.Info("dashboard started", "url", "http://"+addr)
    return srv.ListenAndServe(ctx, addr)
}
```

**Binds to `127.0.0.1` only** — local access, no network exposure.

**File:** `internal/cli/cli.go`
- Add `case "dashboard":` to switch
- Add to `printHelp()` and `commandHelp` map

**File:** `internal/cli/completion.go`
- Add `"dashboard"` to commands slice

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| htmx auto-poll, not SSE/WebSocket | Simpler, no goroutine management, 3s is fast enough for monitoring |
| SQLite via broker subscriber | History accumulates regardless of dashboard running |
| State from filesystem, metrics from SQLite | State machine is source of truth; no duplication |
| Consumer-defined MetricsReader interface | Dashboard testable with fakes; follows Go patterns |
| `modernc.org/sqlite` (pure Go) | CGO_ENABLED=0, single binary |
| Inline CSS + embedded htmx JS | No external assets, works offline, single binary |
| `internal/store/` not `internal/db/` | Follows Go project layout patterns |
| WAL mode + standard pragmas | Concurrent reads from dashboard while CLI writes |
| Store subscribers in `internal/store/` | Avoids pulling SQLite deps into `internal/context/` |
| Bind 127.0.0.1 only | Local dashboard, no network exposure |
| Graceful shutdown via context | Clean SQLite close, proper HTTP drain |
| `time.Time` for timestamps internally | Idiomatic Go; marshal to RFC3339 at storage boundary |
| PRAGMA user_version for schema | Simpler than a schema_version table |
| Indexes on timestamp columns | Dashboard polls every 3s; ORDER BY timestamp needs index |
| errlog superseded, not removed | Backward compat; `sdd errors` falls back to JSON if no DB |

## Dependencies

One new: `modernc.org/sqlite` (pure Go SQLite driver)

## Files

| File | Action |
|------|--------|
| `internal/store/store.go` | **New** — Open, Close, pragmas, migrations, Insert*, Query* |
| `internal/store/subscribers.go` | **New** — RegisterSubscribers for PhaseAssembled + VerifyFailed |
| `internal/store/store_test.go` | **New** — table-driven roundtrip tests |
| `internal/dashboard/server.go` | **New** — HTTP server, handlers, MetricsReader interface |
| `internal/dashboard/server_test.go` | **New** — httptest handler tests with fake store |
| `internal/dashboard/templates/base.html` | **New** — page skeleton, dark theme, htmx |
| `internal/dashboard/templates/kpi.html` | **New** — KPI cards fragment |
| `internal/dashboard/templates/pipelines.html` | **New** — pipeline table fragment |
| `internal/dashboard/templates/errors.html` | **New** — error log fragment |
| `internal/dashboard/static/htmx.min.js` | **New** — htmx 2.0.4 embedded |
| `internal/cli/commands.go` | Add `runDashboard` with signal handling |
| `internal/cli/cli.go` | Route `dashboard` command + help |
| `internal/cli/completion.go` | Add "dashboard" to commands list |
| `go.mod` | Add `modernc.org/sqlite` |

## Verification

```bash
# Unit tests
go test ./internal/store/... ./internal/dashboard/...

# Build
go build ./...

# Lint
golangci-lint run ./...

# Manual test
sdd dashboard --port 8811
# Open http://localhost:8811 in browser
# Run sdd context <name> in another terminal
# Verify dashboard updates within 3s
# Ctrl+C → verify graceful shutdown message
```

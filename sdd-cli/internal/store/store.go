package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database for SDD telemetry.
type Store struct {
	db *sql.DB
}

// PhaseEvent records a single phase execution.
type PhaseEvent struct {
	Timestamp  time.Time
	Change     string
	Phase      string
	Bytes      int
	Tokens     int
	Cached     bool
	DurationMs int64
}

// VerifyEvent records a single verify command execution.
type VerifyEvent struct {
	Timestamp   time.Time
	Change      string
	CommandName string
	Command     string
	ExitCode    int
	ErrorLines  []string
	Fingerprint string
}

// TokenStats summarises token usage across all phase events.
type TokenStats struct {
	TotalTokens int
	CacheHitPct float64
	ErrorCount  int
}

// ChangeTokens is a per-change token total.
type ChangeTokens struct {
	Change string
	Tokens int
}

// ErrorRow is a single row from verify_events for display.
type ErrorRow struct {
	Timestamp   string
	CommandName string
	Command     string
	ExitCode    int
	Change      string
	Fingerprint string
	FirstLine   string
}

// Open creates the parent directory if needed, opens the SQLite database,
// applies WAL pragmas, and runs schema migrations.
func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("store: mkdir %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}

	// WAL pragmas — must be executed outside a transaction.
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("store: pragma %q: %w", p, err)
		}
	}

	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate creates tables and indexes if they don't already exist.
func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS phase_events (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp  TEXT    NOT NULL,
			change     TEXT    NOT NULL,
			phase      TEXT    NOT NULL,
			bytes      INTEGER NOT NULL,
			tokens     INTEGER NOT NULL,
			cached     INTEGER NOT NULL,
			duration_ms INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS verify_events (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp    TEXT    NOT NULL,
			change       TEXT    NOT NULL,
			command_name TEXT    NOT NULL,
			command      TEXT    NOT NULL,
			exit_code    INTEGER NOT NULL,
			error_lines  TEXT    NOT NULL,
			fingerprint  TEXT    NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_phase_events_timestamp ON phase_events(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_verify_events_timestamp ON verify_events(timestamp)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: migrate: %w", err)
		}
	}
	return nil
}

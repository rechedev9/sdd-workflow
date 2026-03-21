package store

import (
	"context"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(dir + "/test.db")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpen_CreatesSchema(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Query sqlite_master for our tables.
	rows, err := s.db.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer rows.Close()

	tables := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		tables[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	if !tables["phase_events"] {
		t.Error("missing table phase_events")
	}
	if !tables["verify_events"] {
		t.Error("missing table verify_events")
	}
}

func TestOpen_WALMode(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	var mode string
	if err := s.db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}
}

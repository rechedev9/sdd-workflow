package sddlog

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInit_Default(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cleanup := Init(&buf)
	if cleanup == nil {
		t.Fatal("Init returned nil cleanup")
	}
	cleanup() // should not panic
}

func TestInit_JSONMode(t *testing.T) {
	// Uses t.Setenv — must not be parallel.
	t.Setenv("SDD_LOG", "json")

	var buf bytes.Buffer
	cleanup := Init(&buf)
	if cleanup == nil {
		t.Fatal("Init returned nil cleanup")
	}
	cleanup()
}

func TestInit_LogFile(t *testing.T) {
	// Uses t.Setenv — must not be parallel.
	path := filepath.Join(t.TempDir(), "sdd.log")
	t.Setenv("SDD_LOG_FILE", path)

	var buf bytes.Buffer
	cleanup := Init(&buf)
	if cleanup == nil {
		t.Fatal("Init returned nil cleanup")
	}
	cleanup()

	// File should have been created.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected log file to exist: %v", err)
	}
}

func TestInit_LogFileInvalidPath(t *testing.T) {
	// Uses t.Setenv — must not be parallel.
	// Invalid path → OpenFile fails → falls back to stderr only.
	t.Setenv("SDD_LOG_FILE", "/nonexistent/dir/sdd.log")

	var buf bytes.Buffer
	cleanup := Init(&buf)
	if cleanup == nil {
		t.Fatal("Init returned nil cleanup")
	}
	cleanup()
}

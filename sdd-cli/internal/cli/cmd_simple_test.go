package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunList_UnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runList([]string{"--bad-flag"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunDump_NoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runDump(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunQuickstart_NoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runQuickstart(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunQuickstart_UnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runQuickstart([]string{"--badflg"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunDashboard_UnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runDashboard([]string{"--bad-flag"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunDashboard_InvalidPort(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	// Invalid port causes early return before any blocking I/O.
	err := runDashboard([]string{"--port", "999"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunDashboard_PortAboveMax(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runDashboard([]string{"--port", "65536"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for port > 65535")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunDashboard_InvalidPortNaN(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runDashboard([]string{"--port", "notanumber"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for non-numeric port")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunDashboard_PortMissingValue(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runDashboard([]string{"--port"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when --port has no value")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunQuickstart_SpecMissingValue(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runQuickstart([]string{"name", "desc", "--spec"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when --spec has no value")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunDashboard_StoreOpenError(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Skip("cannot chdir:", err)
	}
	// Block store.Open by placing a file at openspec/.cache so MkdirAll fails.
	cacheParent := filepath.Join(dir, "openspec")
	if err := os.MkdirAll(cacheParent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheParent, ".cache"), []byte("block"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := runDashboard([]string{"--port", "8811"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when store cannot be opened")
	}
}

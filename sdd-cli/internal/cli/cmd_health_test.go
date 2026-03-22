package cli

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestRunHealth_NoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runHealth(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunHealth_InvalidName(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	// A name with a path separator is rejected by validateChangeName.
	err := runHealth([]string{"../bad"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestRunHealth_ChangeNotFound(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	// A valid-looking name that doesn't exist under cwd/openspec/changes/.
	err := runHealth([]string{"no-such-change-xyz"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for nonexistent change")
	}
}

func TestRunDump_InvalidName(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runDump([]string{"../bad"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestRunDump_ChangeNotFound(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runDump([]string{"no-such-change-xyz"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for nonexistent change")
	}
}

func TestRunQuickstart_InvalidName(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runQuickstart([]string{"bad/name", "desc", "--spec", "/dev/null"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunQuickstart_SpecNotFound(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	dir := t.TempDir()
	specPath := filepath.Join(dir, "does-not-exist.md")
	err := runQuickstart([]string{"valid-name", "desc", "--spec", specPath}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing spec file")
	}
}

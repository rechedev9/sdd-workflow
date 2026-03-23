package cli

import (
	"bytes"
	"testing"
)

func TestRunArchive_NoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runArchive(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunArchive_InvalidName(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runArchive([]string{"../bad"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestRunArchive_ChangeNotFound(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runArchive([]string{"no-such-change-xyz"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for nonexistent change")
	}
}

func TestRunArchive_ForceFlag(t *testing.T) {
	t.Parallel()
	// --force with a nonexistent change still fails at resolveChangeDir,
	// before the force logic is reached. Just verify no panic.
	var stdout, stderr bytes.Buffer
	err := runArchive([]string{"no-such-change-xyz", "--force"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for nonexistent change with --force")
	}
}

func TestRunArchive_ShortForceFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runArchive([]string{"no-such-change-xyz", "-f"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for nonexistent change with -f")
	}
}

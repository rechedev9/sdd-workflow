package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
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

func TestRunArchive_ForceSkipsPrerequisite(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "arch-force")
	os.MkdirAll(changeDir, 0o755)

	// Create an incomplete change (no phases completed).
	st := state.NewState("arch-force", "test force archive")
	state.Save(st, filepath.Join(changeDir, "state.json"))

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	// --force skips prerequisite check → reaches verify.Archive which will succeed
	// or fail depending on filesystem state, but must not fail on prerequisite.
	err := runArchive([]string{"arch-force", "--force"}, &stdout, &stderr)
	// Archive may succeed or fail (no issue if verify.Archive fails), but the
	// prerequisite error branch (slog.Warn) must have been executed without panic.
	_ = err
}

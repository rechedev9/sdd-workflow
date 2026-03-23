package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func TestRunDiff_UnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runDiff([]string{"some-change", "--unknown"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunDiff_NoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runDiff(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunDiff_InvalidName(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runDiff([]string{"../bad"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestRunDiff_ChangeNotFound(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runDiff([]string{"no-such-change-xyz"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for nonexistent change")
	}
}

func TestRunDiff_GitError(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	// Create a change with a BaseRef so we get past the empty-check.
	changeDir := filepath.Join(dir, "openspec", "changes", "feat-giterr")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	st := state.NewState("feat-giterr", "test change")
	st.BaseRef = "abc1234" // non-empty so we proceed to gitDiffFiles
	if err := state.Save(st, filepath.Join(changeDir, "state.json")); err != nil {
		t.Fatal(err)
	}

	// dir is not a git repo → gitDiffFiles returns an error.
	var stdout, stderr bytes.Buffer
	err := runDiff([]string{"feat-giterr"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when git diff fails")
	}
}

func TestRunDiff_NoBaseRef(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	// Create openspec/changes/<name>/state.json with no BaseRef.
	changeDir := filepath.Join(dir, "openspec", "changes", "feat-noref")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	st := state.NewState("feat-noref", "test change")
	// BaseRef is empty by default from NewState.
	if err := state.Save(st, filepath.Join(changeDir, "state.json")); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := runDiff([]string{"feat-noref"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when BaseRef is empty")
	}
}

func TestRunDiff_Success(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })

	// Init a minimal git repo in dir so gitDiffFiles works.
	for _, args := range [][]string{
		{"init", dir},
		{"-C", dir, "config", "user.email", "test@test.com"},
		{"-C", dir, "config", "user.name", "Test"},
		{"-C", dir, "commit", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		if err := cmd.Run(); err != nil {
			t.Skipf("git setup failed: %v", err)
		}
	}

	sha, err := gitHeadSHA(dir)
	if err != nil {
		t.Skip("cannot get HEAD SHA:", err)
	}

	changeDir := filepath.Join(dir, "openspec", "changes", "feat-diffok")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	st := state.NewState("feat-diffok", "test diff")
	st.BaseRef = sha
	if err := state.Save(st, filepath.Join(changeDir, "state.json")); err != nil {
		t.Fatal(err)
	}

	os.Chdir(dir)
	var stdout, stderr bytes.Buffer
	if err := runDiff([]string{"feat-diffok"}, &stdout, &stderr); err != nil {
		t.Fatalf("runDiff: %v", err)
	}
	if stdout.Len() == 0 {
		t.Error("expected JSON output")
	}
}

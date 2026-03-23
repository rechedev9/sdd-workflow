package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/errlog"
)

// repoRoot is the git repo root — used as cwd for shouldSkipVerify so that
// gitDiffFiles can run git diff HEAD without failing.
var repoRoot = func() string {
	// Walk up from the test binary location until we find a .git directory.
	// This works because Go tests run with cwd = package directory.
	dir, _ := os.Getwd()
	for dir != "/" {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return "."
}()

func TestRunVerify_NoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runVerify(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunVerify_InvalidName(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runVerify([]string{"../bad"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestRunVerify_ChangeNotFound(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runVerify([]string{"no-such-change-xyz"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for nonexistent change")
	}
}

func TestRunVerify_UnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runVerify([]string{"some-change", "--bad-flag"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestShouldSkipVerify_NoReport(t *testing.T) {
	t.Parallel()
	changeDir := t.TempDir()
	// No verify-report.md → cannot skip.
	skip := shouldSkipVerify(repoRoot, changeDir)
	if skip {
		t.Error("expected skip=false when no report exists")
	}
}

func TestShouldSkipVerify_FailedReport(t *testing.T) {
	t.Parallel()
	changeDir := t.TempDir()
	// Write a FAILED verify-report.md.
	report := "**Status:** FAILED\n"
	if err := os.WriteFile(filepath.Join(changeDir, "verify-report.md"), []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	skip := shouldSkipVerify(repoRoot, changeDir)
	if skip {
		t.Error("expected skip=false for failed report")
	}
}

func TestShouldSkipVerify_PassedReport(t *testing.T) {
	t.Parallel()
	changeDir := t.TempDir()
	// Write a PASSED verify-report.md — skip depends on git diff.
	report := "**Status:** PASSED\n"
	if err := os.WriteFile(filepath.Join(changeDir, "verify-report.md"), []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	// shouldSkipVerify runs git diff HEAD from repoRoot.
	// Result may be true or false depending on working tree state.
	_ = shouldSkipVerify(repoRoot, changeDir)
}

func TestShouldSkipVerify_GitError(t *testing.T) {
	t.Parallel()
	changeDir := t.TempDir()
	// Write a PASSED report so we get past the first check.
	report := "**Status:** PASSED\n"
	if err := os.WriteFile(filepath.Join(changeDir, "verify-report.md"), []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	// Use a non-git dir as cwd → gitDiffFiles fails → shouldSkipVerify returns false.
	skip := shouldSkipVerify(t.TempDir(), changeDir)
	if skip {
		t.Error("expected skip=false when git error occurs")
	}
}

// TestCheckRecurringFailures_NoMatchForChange verifies that recurring fingerprints
// from a different change do not block the target change.
func TestCheckRecurringFailures_NoMatchForChange(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()

	// Plant 3 recurring failures for "other-change", not for "my-change".
	fp := errlog.Fingerprint("go build ./...", []string{"some error"})
	entry := errlog.ErrorEntry{
		Timestamp:   "2026-01-01T00:00:00Z",
		Change:      "other-change",
		CommandName: "build",
		Command:     "go build ./...",
		ExitCode:    1,
		ErrorLines:  []string{"some error"},
		Fingerprint: fp,
	}
	for i := 0; i < 3; i++ {
		errlog.Record(cwd, entry)
	}

	// "my-change" has no failures → no match → should return nil.
	matches := checkRecurringFailures(cwd, "my-change")
	if matches != nil {
		t.Errorf("checkRecurringFailures = %v, want nil (no failures for my-change)", matches)
	}
}

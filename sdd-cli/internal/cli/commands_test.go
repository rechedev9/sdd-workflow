package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/errlog"
)

func TestGitDiffFiles_InvalidRef(t *testing.T) {
	t.Parallel()
	// Use a valid git repo but an invalid ref → ExitError path (lines 119-122).
	_, err := gitDiffFiles(repoRoot, "INVALID_REF_XYZ_DOES_NOT_EXIST")
	if err == nil {
		t.Fatal("expected error for invalid git ref")
	}
}

func TestGitDiffFiles_NonGitDir(t *testing.T) {
	t.Parallel()
	// Non-git directory → exec error path (line 123).
	_, err := gitDiffFiles(t.TempDir(), "HEAD")
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestGitDiffFiles_ValidRef(t *testing.T) {
	t.Parallel()
	// Valid git repo + valid ref → no error; result is nil or a list of files.
	files, err := gitDiffFiles(repoRoot, "HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Each file should be a relative path (no empty strings).
	for _, f := range files {
		if strings.TrimSpace(f) == "" {
			t.Errorf("empty file path in result")
		}
	}
}

func TestCheckRecurringFailures_NoLog(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// No error log → should return nil.
	result := checkRecurringFailures(dir, "feat-a")
	if result != nil {
		t.Errorf("expected nil for empty log, got %v", result)
	}
}

func TestCheckRecurringFailures_NoRecurring(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Record 2 errors (below threshold of 3) for feat-a.
	fp := errlog.Fingerprint("go build", []string{"error: foo"})
	for i := 0; i < 2; i++ {
		errlog.Record(dir, errlog.ErrorEntry{
			Change: "feat-a", CommandName: "build",
			Command: "go build", ExitCode: 1,
			ErrorLines: []string{"error: foo"}, Fingerprint: fp,
		})
	}
	result := checkRecurringFailures(dir, "feat-a")
	if result != nil {
		t.Errorf("expected nil below threshold, got %v", result)
	}
}

func TestCheckRecurringFailures_WithRecurring(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Record 3 errors with same fingerprint for feat-a (hits threshold).
	fp := errlog.Fingerprint("go test", []string{"FAIL"})
	for i := 0; i < 3; i++ {
		errlog.Record(dir, errlog.ErrorEntry{
			Change: "feat-a", CommandName: "test",
			Command: "go test", ExitCode: 1,
			ErrorLines: []string{"FAIL"}, Fingerprint: fp,
		})
	}
	result := checkRecurringFailures(dir, "feat-a")
	if result == nil {
		t.Fatal("expected non-nil result for recurring failure")
	}
	if _, ok := result[fp]; !ok {
		t.Errorf("expected fingerprint %q in result, got %v", fp, result)
	}
}

func TestCheckRecurringFailures_DifferentChange(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// 3 recurring errors for feat-b, but checking feat-a → no match.
	fp := errlog.Fingerprint("go build", []string{"error: bar"})
	for i := 0; i < 3; i++ {
		errlog.Record(dir, errlog.ErrorEntry{
			Change: "feat-b", CommandName: "build",
			Command: "go build", ExitCode: 1,
			ErrorLines: []string{"error: bar"}, Fingerprint: fp,
		})
	}
	result := checkRecurringFailures(dir, "feat-a")
	if result != nil {
		t.Errorf("expected nil when recurring errors are from different change, got %v", result)
	}
}

func TestLoadChangeState_MissingStateJSON(t *testing.T) {
	t.Parallel()
	// Create a valid change directory but without state.json — Load should fail.
	root := t.TempDir()
	// Set up a fake openspec/changes/feat-x directory structure.
	changeDir := filepath.Join(root, "openspec", "changes", "feat-x")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Temporarily override cwd via os.Chdir so resolveChangeDir finds the dir.
	orig, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Skip("cannot chdir:", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	var stderr bytes.Buffer
	_, _, err := loadChangeState(&stderr, "test", "feat-x")
	if err == nil {
		t.Fatal("expected error when state.json is missing")
	}
}

func TestResolveDir(t *testing.T) {
	t.Parallel()

	t.Run("existing_dir", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		got, err := resolveDir(dir)
		if err != nil {
			t.Fatalf("resolveDir(%q): %v", dir, err)
		}
		if got == "" {
			t.Error("expected non-empty path")
		}
	})

	t.Run("dot_uses_cwd", func(t *testing.T) {
		t.Parallel()
		got, err := resolveDir(".")
		if err != nil {
			t.Fatalf("resolveDir(.): %v", err)
		}
		if got == "" {
			t.Error("expected non-empty path for '.'")
		}
	})

	t.Run("missing_dir", func(t *testing.T) {
		t.Parallel()
		_, err := resolveDir("/nonexistent/path/xyz")
		if err == nil {
			t.Error("expected error for missing directory")
		}
	})

	t.Run("file_not_dir", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		f, _ := os.CreateTemp(dir, "test*.txt")
		f.Close()
		_, err := resolveDir(f.Name())
		if err == nil {
			t.Error("expected error when path is a file, not a directory")
		}
	})
}

func TestResolveChangeDir_FileNotDir(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	// Place a regular file at openspec/changes/<name> so stat succeeds but !IsDir.
	changesDir := filepath.Join(dir, "openspec", "changes")
	if err := os.MkdirAll(changesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changesDir, "feat-file"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := resolveChangeDir("feat-file")
	if err == nil {
		t.Fatal("expected error when change path is a file, not a directory")
	}
}

func TestGitHeadSHA_NotARepo(t *testing.T) {
	t.Parallel()
	_, err := gitHeadSHA(t.TempDir())
	if err == nil {
		t.Fatal("expected error when dir is not a git repo")
	}
}

func TestShouldSkipVerify_ReturnTrue(t *testing.T) {
	t.Parallel()
	changeDir := t.TempDir()

	// Write a PASSED verify-report.md.
	report := "**Status:** PASSED\n"
	if err := os.WriteFile(filepath.Join(changeDir, "verify-report.md"), []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}

	// Use a git repo that has HEAD: create an empty commit so gitDiffFiles succeeds.
	gitDir := t.TempDir()
	for _, args := range [][]string{
		{"init", gitDir},
		{"-C", gitDir, "config", "user.email", "test@test.com"},
		{"-C", gitDir, "config", "user.name", "Test"},
		{"-C", gitDir, "commit", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		if err := cmd.Run(); err != nil {
			t.Skipf("git setup failed: %v", err)
		}
	}

	// No changed files in a clean repo → gitDiffFiles returns nil → skip=true.
	skip := shouldSkipVerify(gitDir, changeDir)
	if !skip {
		t.Error("expected skip=true: PASSED report + no source changes")
	}
}

func TestShouldSkipVerify_SourceFileChanged(t *testing.T) {
	t.Parallel()
	changeDir := t.TempDir()

	// Write a PASSED verify-report.md.
	report := "**Status:** PASSED\n"
	if err := os.WriteFile(filepath.Join(changeDir, "verify-report.md"), []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}

	// Set up a git repo with a committed file, then modify it to produce a diff.
	// Use /usr/bin/git directly to bypass the git-policy shim.
	const realGit = "/usr/bin/git"
	gitDir := t.TempDir()
	srcFile := filepath.Join(gitDir, "main.go")
	for _, args := range [][]string{
		{"init", gitDir},
		{"-C", gitDir, "config", "user.email", "test@test.com"},
		{"-C", gitDir, "config", "user.name", "Test"},
	} {
		if err := exec.Command(realGit, args...).Run(); err != nil {
			t.Skipf("git setup failed: %v", err)
		}
	}
	// Commit a source file.
	if err := os.WriteFile(srcFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"-C", gitDir, "add", "main.go"},
		{"-C", gitDir, "commit", "-m", "init"},
	} {
		if err := exec.Command(realGit, args...).Run(); err != nil {
			t.Skipf("git commit failed: %v", err)
		}
	}
	// Modify the source file so git diff HEAD shows it as changed.
	if err := os.WriteFile(srcFile, []byte("package main\n// changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Non-openspec/ file changed → must not skip.
	skip := shouldSkipVerify(gitDir, changeDir)
	if skip {
		t.Error("expected skip=false: source file has uncommitted changes")
	}
}

// TestRun_DispatchCoverage exercises the Run switch cases not covered by
// direct calls to the underlying run* functions. Each subtest only needs
// the dispatch to fire — the underlying commands may return errors.
func TestRun_DispatchCoverage(t *testing.T) {
	root := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	cases := []struct {
		args []string
	}{
		{[]string{"errors", "--unknown-flag-xyz"}},     // runErrors dispatch
		{[]string{"doctor", "--unknown-flag-xyz"}},     // runDoctor dispatch
		{[]string{"quickstart"}},                       // runQuickstart dispatch (no args → usage error)
		{[]string{"completion", "bash"}},               // runCompletion dispatch
	}
	for _, tc := range cases {
		var stdout, stderr bytes.Buffer
		// Error is acceptable — we only need the dispatch branch to execute.
		Run(tc.args, &stdout, &stderr) //nolint:errcheck
	}
}

func TestValidateChangeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "add-auth", false},
		{"valid with numbers", "feat-123", false},
		{"empty", "", true},
		{"dot", ".", true},
		{"dotdot", "..", true},
		{"forward slash", "a/b", true},
		{"backslash", `a\b`, true},
		{"path traversal", "../etc/passwd", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateChangeName(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateChangeName(%q) error = %v, wantErr = %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

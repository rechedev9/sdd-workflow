package cli

import (
	"os"
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

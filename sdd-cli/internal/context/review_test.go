package context

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// findGitRoot walks up from cwd until it finds a .git directory.
func findGitRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Skip("cannot get cwd:", err)
	}
	for dir != "/" {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("no git root found")
	return ""
}

func TestGitDiff_InGitRepo(t *testing.T) {
	t.Parallel()
	root := findGitRoot(t)
	result, err := gitDiff(root)
	if err != nil {
		t.Fatalf("gitDiff: %v", err)
	}
	// Result should either be "(no changes)" or contain diff output.
	if result == "" {
		t.Error("expected non-empty result from gitDiff")
	}
}

func TestGitDiff_NoChanges(t *testing.T) {
	t.Parallel()
	// Use a fresh git repo with no changes.
	dir := t.TempDir()
	if err := initBareGitRepo(t, dir); err != nil {
		t.Skip("cannot init git repo:", err)
	}
	result, err := gitDiff(dir)
	if err != nil {
		t.Fatalf("gitDiff: %v", err)
	}
	if result != "(no changes)" {
		t.Errorf("expected '(no changes)', got %q", result)
	}
}

func TestGitDiff_ErrorOnNonGitDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := gitDiff(dir)
	if err == nil {
		t.Error("expected error for non-git directory")
	}
	if !strings.Contains(err.Error(), "git diff") {
		t.Errorf("expected 'git diff' in error, got %v", err)
	}
}

// runCmd runs a command in dir and returns any error.
func runCmd(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Run()
}

// initBareGitRepo initialises a minimal git repo in dir with an initial commit.
func initBareGitRepo(t *testing.T, dir string) error {
	t.Helper()
	if err := runCmd(dir, "git", "init", "-q"); err != nil {
		return err
	}
	if err := runCmd(dir, "git", "config", "user.email", "test@test.com"); err != nil {
		return err
	}
	if err := runCmd(dir, "git", "config", "user.name", "Test"); err != nil {
		return err
	}
	// Create an initial commit so HEAD exists.
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("test"), 0o644); err != nil {
		return err
	}
	if err := runCmd(dir, "git", "add", "README.md"); err != nil {
		return err
	}
	return runCmd(dir, "git", "commit", "-q", "-m", "init")
}

package context

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/csync"
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

func TestGitDiff_StagedChanges(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := initBareGitRepo(t, dir); err != nil {
		t.Skip("cannot init git repo:", err)
	}
	// Create and stage a new file.
	f := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(f, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runRealGit(dir, "add", "new.txt"); err != nil {
		t.Fatalf("git add: %v", err)
	}

	result, err := gitDiff(dir)
	if err != nil {
		t.Fatalf("gitDiff: %v", err)
	}
	if !strings.Contains(result, "STAGED") {
		t.Errorf("expected STAGED section, got %q", result)
	}
}

func TestGitDiff_UnstagedChanges(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := initBareGitRepo(t, dir); err != nil {
		t.Skip("cannot init git repo:", err)
	}
	// Modify an already-tracked file without staging.
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := gitDiff(dir)
	if err != nil {
		t.Fatalf("gitDiff: %v", err)
	}
	if !strings.Contains(result, "UNSTAGED") {
		t.Errorf("expected UNSTAGED section, got %q", result)
	}
}

func TestGitDiff_StagedAndUnstaged(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := initBareGitRepo(t, dir); err != nil {
		t.Skip("cannot init git repo:", err)
	}
	// Unstaged: modify tracked file.
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Staged: add a new file.
	f := filepath.Join(dir, "staged.txt")
	if err := os.WriteFile(f, []byte("staged"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runRealGit(dir, "add", "staged.txt"); err != nil {
		t.Fatalf("git add: %v", err)
	}

	result, err := gitDiff(dir)
	if err != nil {
		t.Fatalf("gitDiff: %v", err)
	}
	if !strings.Contains(result, "STAGED") {
		t.Errorf("expected STAGED section, got %q", result)
	}
	if !strings.Contains(result, "UNSTAGED") {
		t.Errorf("expected UNSTAGED section, got %q", result)
	}
}

// realGit returns a path to the real git binary, bypassing any shim.
func realGit() string {
	for _, p := range []string{"/usr/bin/git", "/usr/local/bin/git", "/bin/git"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "git"
}

// runRealGit runs git directly (bypassing any shim) in dir.
func runRealGit(dir string, args ...string) error {
	cmd := exec.Command(realGit(), args...)
	cmd.Dir = dir
	return cmd.Run()
}

// initBareGitRepo initialises a minimal git repo in dir with an initial commit.
// Uses the real git binary directly to bypass any project git shim.
func initBareGitRepo(t *testing.T, dir string) error {
	t.Helper()
	if err := runRealGit(dir, "init", "-q"); err != nil {
		return err
	}
	if err := runRealGit(dir, "config", "user.email", "test@test.com"); err != nil {
		return err
	}
	if err := runRealGit(dir, "config", "user.name", "Test"); err != nil {
		return err
	}
	// Create an initial commit so HEAD exists.
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("test"), 0o644); err != nil {
		return err
	}
	if err := runRealGit(dir, "add", "README.md"); err != nil {
		return err
	}
	return runRealGit(dir, "commit", "-q", "-m", "init")
}

func TestCheckSkillError_SkillFails(t *testing.T) {
	t.Parallel()
	// Build a LazySlice where loader 0 (skill) returns an error.
	sentinelErr := errors.New("skill missing")
	loaders := []func() ([]byte, error){
		func() ([]byte, error) { return nil, sentinelErr },
	}
	ls := csync.NewLazySlice(loaders)
	loadErr := ls.LoadAll()
	err := checkSkillError(ls, loadErr)
	if err == nil {
		t.Fatal("expected checkSkillError to return skill error")
	}
	if !errors.Is(err, sentinelErr) {
		t.Errorf("error = %v, want sentinel error", err)
	}
}

func TestCheckSkillError_NoLoadError(t *testing.T) {
	t.Parallel()
	// LoadAll succeeds — checkSkillError should always return nil.
	loaders := []func() ([]byte, error){
		func() ([]byte, error) { return []byte("skill"), nil },
	}
	ls := csync.NewLazySlice(loaders)
	loadErr := ls.LoadAll()
	if err := checkSkillError(ls, loadErr); err != nil {
		t.Errorf("expected nil when LoadAll succeeds, got %v", err)
	}
}

func TestCheckSkillError_OtherLoaderFails(t *testing.T) {
	t.Parallel()
	// Skill (index 0) succeeds but another loader fails.
	// checkSkillError should return nil (let caller handle the artifact error).
	loaders := []func() ([]byte, error){
		func() ([]byte, error) { return []byte("skill"), nil },
		func() ([]byte, error) { return nil, fmt.Errorf("artifact missing") },
	}
	ls := csync.NewLazySlice(loaders)
	loadErr := ls.LoadAll()
	if err := checkSkillError(ls, loadErr); err != nil {
		t.Errorf("expected nil when skill succeeds, got %v", err)
	}
}

func TestAssembleReview_MissingSpecs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// No specs/ directory — loadSpecsLoader will error, triggering the
	// errRequiredArtifact("review", "spec artifacts", ...) branch.
	var buf strings.Builder
	p := &Params{ChangeDir: dir, ProjectDir: dir}
	err := AssembleReview(&buf, p)
	if err == nil {
		t.Fatal("expected error when specs directory is missing")
	}
	if !strings.Contains(err.Error(), "spec artifacts") {
		t.Errorf("error = %q, want mention of 'spec artifacts'", err.Error())
	}
}

func TestAssembleReview_MissingDesign(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create specs/ with a dummy spec so loadSpecs succeeds.
	specsDir := filepath.Join(dir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "feat.md"), []byte("# spec"), 0o644); err != nil {
		t.Fatal(err)
	}
	// No design.md — artifactLoader("design.md") errors → errRequiredArtifact branch.
	var buf strings.Builder
	p := &Params{ChangeDir: dir, ProjectDir: dir}
	err := AssembleReview(&buf, p)
	if err == nil {
		t.Fatal("expected error when design.md is missing")
	}
	if !strings.Contains(err.Error(), "design artifact") {
		t.Errorf("error = %q, want mention of 'design artifact'", err.Error())
	}
}

func TestAssembleReview_MissingTasks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Specs and design present, tasks.md absent — triggers errRequiredArtifact for tasks.
	specsDir := filepath.Join(dir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(specsDir, "feat.md"), []byte("# spec"), 0o644)
	os.WriteFile(filepath.Join(dir, "design.md"), []byte("# design"), 0o644)

	var buf strings.Builder
	p := &Params{ChangeDir: dir, ProjectDir: dir}
	err := AssembleReview(&buf, p)
	if err == nil {
		t.Fatal("expected error when tasks.md is missing")
	}
	if !strings.Contains(err.Error(), "tasks artifact") {
		t.Errorf("error = %q, want mention of 'tasks artifact'", err.Error())
	}
}

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/config"
	sddctx "github.com/rechedev9/shenronSDD/sdd-cli/internal/context"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/store"
)

// newBroker creates and wires a broker with default subscribers.
// db may be nil — SQLite subscribers are skipped when nil.
func newBroker(verbosity int, db *store.Store) *events.Broker {
	broker := events.NewBroker()
	sddctx.RegisterSubscribers(broker, verbosity)
	store.RegisterSubscribers(broker, db)
	return broker
}

// tryOpenStore opens the SQLite store best-effort. Returns nil if unavailable.
func tryOpenStore(cwd string) *store.Store {
	path := openspecDB(cwd)
	db, err := store.Open(path)
	if err != nil {
		return nil
	}
	return db
}

// staleThreshold is the duration after which a change is considered abandoned.
// Changes inactive longer than this are flagged as stale.
const staleThreshold = 24 * time.Hour
const projectRootSearchDepth = 3

func resolveDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve directory: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", abs)
	}
	return abs, nil
}

func resolveProjectRoot(start string) (string, error) {
	abs, err := resolveDir(start)
	if err != nil {
		return "", err
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		if hasOpenspecRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	candidates, err := findDescendantProjectRoots(abs, projectRootSearchDepth)
	if err != nil {
		return "", err
	}
	switch len(candidates) {
	case 0:
		return "", fmt.Errorf("openspec/ not found from %s (run 'sdd init' first)", abs)
	case 1:
		return candidates[0], nil
	default:
		return "", fmt.Errorf("multiple candidate project roots found under %s: %s", abs, strings.Join(candidates, ", "))
	}
}

func findDescendantProjectRoots(start string, maxDepth int) ([]string, error) {
	candidates := make(map[string]struct{}, 4)
	err := filepath.WalkDir(start, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == start {
			return nil
		}
		rel, err := filepath.Rel(start, path)
		if err != nil {
			return err
		}
		depth := strings.Count(rel, string(filepath.Separator)) + 1
		if d.IsDir() {
			if depth > maxDepth {
				return filepath.SkipDir
			}
			switch d.Name() {
			case ".git":
				return filepath.SkipDir
			case "openspec":
				candidates[filepath.Dir(path)] = struct{}{}
				return filepath.SkipDir
			}
			return nil
		}
		if depth > maxDepth {
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan project roots: %w", err)
	}
	out := make([]string, 0, len(candidates))
	for dir := range candidates {
		out = append(out, dir)
	}
	slices.Sort(out)
	return out, nil
}

func hasOpenspecRoot(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "openspec"))
	return err == nil && info.IsDir()
}

// validateChangeName rejects names that contain path separators, special
// directory components, or characters invalid for git branch refs.
// "archive" is reserved: eachChangeDir silently skips it, so a change named
// "archive" would be invisible to list, doctor, and health commands.
func validateChangeName(name string) error {
	if name == "" {
		return fmt.Errorf("change name must not be empty")
	}
	if name == "." || name == ".." || name == "archive" {
		return fmt.Errorf("change name must not be %q", name)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("change name must not contain path separators: %q", name)
	}
	// Git ref-unsafe characters (git-check-ref-format rules).
	// Change names become branch refs via "sdd/<name>", so they must be ref-safe.
	if strings.ContainsAny(name, " ~^:?*[") {
		return fmt.Errorf("change name contains characters invalid for git branches: %q", name)
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("change name must not contain '..': %q", name)
	}
	if strings.Contains(name, "@{") {
		return fmt.Errorf("change name must not contain '@{': %q", name)
	}
	if strings.HasSuffix(name, ".lock") {
		return fmt.Errorf("change name must not end with '.lock': %q", name)
	}
	if strings.HasSuffix(name, ".") {
		return fmt.Errorf("change name must not end with '.': %q", name)
	}
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "-") {
		return fmt.Errorf("change name must not start with '.' or '-': %q", name)
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("change name contains control characters: %q", name)
		}
	}
	return nil
}

func resolveChangeDir(name string) (string, error) {
	if err := validateChangeName(name); err != nil {
		return "", err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	projectRoot, err := resolveProjectRoot(cwd)
	if err != nil {
		return "", err
	}
	changeDir := filepath.Join(openspecChanges(projectRoot), name)
	info, err := os.Stat(changeDir)
	if err != nil {
		return "", fmt.Errorf("change directory not found: %s", changeDir)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", changeDir)
	}
	return changeDir, nil
}

// getCWD returns the working directory, writing to stderr and returning an error on failure.
// Used by every CLI command that needs the project root.
func getCWD(stderr io.Writer, cmd string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errs.WriteError(stderr, cmd, fmt.Errorf("get working directory: %w", err))
	}
	return cwd, nil
}

func getProjectRoot(stderr io.Writer, cmd string) (string, error) {
	cwd, err := getCWD(stderr, cmd)
	if err != nil {
		return "", err
	}
	root, err := resolveProjectRoot(cwd)
	if err != nil {
		return "", errs.WriteError(stderr, cmd, err)
	}
	return root, nil
}

// openspecConfig returns the path to openspec/config.yaml in the project root.
// Used by commands that load config — centralises the magic string.
func openspecConfig(cwd string) string {
	return filepath.Join(cwd, "openspec", "config.yaml")
}

// openspecChanges returns the path to openspec/changes in the project root.
func openspecChanges(cwd string) string {
	return filepath.Join(cwd, "openspec", "changes")
}

// openspecDB returns the path to the SQLite store in openspec/.cache/sdd.db.
func openspecDB(cwd string) string {
	return filepath.Join(cwd, "openspec", ".cache", "sdd.db")
}

// loadConfig reads openspec/config.yaml from cwd.
// On error it writes to stderr and returns a wrapped error ready for return.
// Used by commands that need the project config after resolving cwd.
func loadConfig(stderr io.Writer, cmd, cwd string) (*config.Config, error) {
	cfg, err := config.Load(openspecConfig(cwd))
	if err != nil {
		return nil, errs.WriteError(stderr, cmd, fmt.Errorf("load config: %w", err))
	}
	return cfg, nil
}

// loadChangeState resolves the change directory for name and loads state.json.
// On error it writes to stderr and returns a wrapped error ready for return.
// Used by the 7+ commands that start with resolveChangeDir + state.Load.
func loadChangeState(stderr io.Writer, cmd, name string) (string, *state.State, error) {
	changeDir, err := resolveChangeDir(name)
	if err != nil {
		return "", nil, errs.WriteError(stderr, cmd, err)
	}
	st, err := state.Load(filepath.Join(changeDir, "state.json"))
	if err != nil {
		return "", nil, errs.WriteError(stderr, cmd, fmt.Errorf("load state: %w", err))
	}
	return changeDir, st, nil
}

// gitCmdTimeout is the maximum time allowed for a git subprocess.
const gitCmdTimeout = 30 * time.Second

// gitBin is the path to the real git binary, bypassing the PATH-based shim.
// The shim at ~/.local/bin/git blocks add/commit/push/checkout for safety;
// authorized commands (committer, ship) call the real binary directly.
var gitBin = detectGitBin()

// detectGitBin finds the real git binary, preferring well-known absolute paths
// that bypass any PATH-based shim. Falls back to "git" from PATH.
func detectGitBin() string {
	for _, p := range []string{
		"/usr/bin/git",          // Linux, WSL
		"/usr/local/bin/git",    // Homebrew (Intel Mac)
		"/opt/homebrew/bin/git", // Homebrew (Apple Silicon)
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "git" // fallback: PATH (may hit shim, but better than failing)
}

// gitHeadSHA returns the current HEAD SHA in dir.
func gitHeadSHA(dir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return string(bytes.TrimSpace(out)), nil
}

// gitRequireClean returns an error if the working tree has uncommitted changes.
func gitRequireClean(dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if len(bytes.TrimSpace(out)) > 0 {
		return fmt.Errorf("uncommitted changes detected")
	}
	return nil
}

// gitDiffFiles returns files changed between ref and the working tree.
func gitDiffFiles(dir, ref string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", ref)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("%w\n%s", err, string(bytes.TrimSpace(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("exec git: %w", err)
	}
	raw := string(bytes.TrimRight(out, "\n"))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// shouldSkipVerify returns true if verify can be skipped because:
// 1. verify-report.md exists and contains PASSED
// 2. No source files (excluding openspec/) changed since HEAD
// Returns false on any error — never skips when unsure.
func shouldSkipVerify(cwd, changeDir string) bool {
	// Check existing report is PASSED.
	reportPath := filepath.Join(changeDir, "verify-report.md")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return false // no report → can't skip
	}
	if !bytes.Contains(data, []byte("**Status:** PASSED")) {
		return false // last run failed → must re-verify
	}

	// Check for source file changes.
	files, err := gitDiffFiles(cwd, "HEAD")
	if err != nil {
		return false // git error → don't skip
	}

	// Filter out openspec/ files — those aren't source code.
	for _, f := range files {
		if !strings.HasPrefix(f, "openspec/") {
			return false // source file changed → must verify
		}
	}

	return true // no source changes + last verify passed → skip
}

// writeJSON marshals v as indented JSON and writes it to w followed by a newline.
// Uses json.Encoder to stream directly into w, avoiding the intermediate []byte→string copy
// that json.MarshalIndent + fmt.Fprintln would require.
func writeJSON(w io.Writer, v any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v) //nolint:errcheck // stdout write errors are not actionable
}

// gitCheckoutNewBranch creates and checks out a new branch in dir.
func gitCheckoutNewBranch(dir, branch string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitBin, "checkout", "-b", branch)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout -b %s: %s", branch, bytes.TrimSpace(out))
	}
	return nil
}

// gitCheckout switches to an existing branch in dir.
func gitCheckout(dir, branch string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitBin, "checkout", branch)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout %s: %s", branch, bytes.TrimSpace(out))
	}
	return nil
}

// gitPush pushes a branch to a remote, setting upstream tracking.
func gitPush(dir, remote, branch string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitBin, "push", "-u", remote, branch)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push -u %s %s: %s", remote, branch, bytes.TrimSpace(out))
	}
	return nil
}

// gitDeleteRemoteBranch removes a branch from the remote.
func gitDeleteRemoteBranch(dir, remote, branch string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitBin, "push", remote, "--delete", branch)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push --delete %s %s: %s", remote, branch, bytes.TrimSpace(out))
	}
	return nil
}

// gitDeleteLocalBranch removes a local branch.
func gitDeleteLocalBranch(dir, branch string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitBin, "branch", "-D", branch)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git branch -D %s: %s", branch, bytes.TrimSpace(out))
	}
	return nil
}

// gitRemoteBranchExists checks if a branch exists on the remote.
func gitRemoteBranchExists(dir, remote, branch string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitBin, "ls-remote", "--heads", remote, "refs/heads/"+branch)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git ls-remote: %w", err)
	}
	return len(bytes.TrimSpace(out)) > 0, nil
}

// ghAuthStatus checks that the GitHub CLI is installed and authenticated.
func ghAuthStatus(dir string) error {
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		return fmt.Errorf("gh CLI not found: install from https://cli.github.com")
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, ghPath, "auth", "status")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh not authenticated: %s", bytes.TrimSpace(out))
	}
	return nil
}

// ghCreatePR creates a pull request and returns the PR URL.
func ghCreatePR(dir, base, head, title, body string) (string, error) {
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		return "", fmt.Errorf("gh CLI not found")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ghPath, "pr", "create",
		"--base", base,
		"--head", head,
		"--title", title,
		"--body", body,
	)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr create: %s", bytes.TrimSpace(out))
	}
	// gh pr create outputs the PR URL on the last non-empty line.
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return lines[len(lines)-1], nil
}

// detectBaseBranch determines the repo's default branch (main, master, etc.)
// by querying gh CLI, then falling back to git remotes.
func detectBaseBranch(dir string) (string, error) {
	// Try gh first — most reliable.
	if ghPath, err := exec.LookPath("gh"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, ghPath, "repo", "view", "--json", "defaultBranchRef", "-q", ".defaultBranchRef.name")
		cmd.Dir = dir
		if out, err := cmd.Output(); err == nil {
			if branch := strings.TrimSpace(string(out)); branch != "" {
				return branch, nil
			}
		}
	}
	// Fallback: git symbolic-ref origin/HEAD.
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitBin, "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = dir
	if out, err := cmd.Output(); err == nil {
		ref := strings.TrimSpace(string(out))
		// ref is like "refs/remotes/origin/main"
		if parts := strings.SplitN(ref, "refs/remotes/origin/", 2); len(parts) == 2 && parts[1] != "" {
			return parts[1], nil
		}
	}
	// Fallback: check which common branch exists on the remote.
	for _, candidate := range []string{"main", "master"} {
		cmd := exec.CommandContext(context.Background(), gitBin, "rev-parse", "--verify", "refs/remotes/origin/"+candidate)
		cmd.Dir = dir
		if err := cmd.Run(); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("cannot detect default branch; set --base or configure origin/HEAD")
}

// verifyPRMerged checks that the PR for a shipped change was merged.
// Best-effort: if gh is unavailable or ship-report.md is missing, returns nil (skip).
func verifyPRMerged(cwd, changeDir string) error {
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		return nil // gh not available — skip check
	}
	// Read branch from ship-report.md.
	data, err := os.ReadFile(filepath.Join(changeDir, "ship-report.md"))
	if err != nil {
		return nil // no report — skip check
	}
	// Extract branch: look for "**Branch:** `sdd/foo`"
	var branch string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "**Branch:**") {
			// Extract content between backticks.
			if start := strings.Index(line, "`"); start >= 0 {
				if end := strings.Index(line[start+1:], "`"); end >= 0 {
					branch = line[start+1 : start+1+end]
				}
			}
			break
		}
	}
	if branch == "" {
		return nil // can't find branch — skip check
	}
	// Query gh for merge status.
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, ghPath, "pr", "view", branch, "--json", "mergedAt", "-q", ".mergedAt")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return nil // gh query failed — skip check (PR may have been deleted)
	}
	mergedAt := strings.TrimSpace(string(out))
	if mergedAt == "" || mergedAt == "null" {
		return fmt.Errorf("PR for branch %q has not been merged yet", branch)
	}
	return nil
}

// errUnknownFlag returns a usage error for an unrecognised CLI flag.
// Centralises the repeated errs.Usage(fmt.Sprintf("unknown flag: %s", flag)) pattern.
func errUnknownFlag(flag string) error {
	return errs.Usage("unknown flag: " + flag)
}

// eachChangeDir calls fn for each active change directory (skips non-dirs and "archive").
// Silently returns if changesDir cannot be read.
func eachChangeDir(changesDir string, fn func(changeDir string)) {
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		fn(filepath.Join(changesDir, e.Name()))
	}
}

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	sddctx "github.com/rechedev9/shenronSDD/sdd-cli/internal/context"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/store"
)

// newBroker creates and wires a broker with default subscribers.
// db may be nil — SQLite subscribers are skipped when nil.
func newBroker(stderr io.Writer, verbosity int, db *store.Store) *events.Broker {
	broker := events.NewBroker()
	sddctx.RegisterSubscribers(broker, stderr, verbosity)
	store.RegisterSubscribers(broker, db)
	return broker
}

// tryOpenStore opens the SQLite store best-effort. Returns nil if unavailable.
func tryOpenStore(cwd string) *store.Store {
	path := filepath.Join(cwd, "openspec", ".cache", "sdd.db")
	db, err := store.Open(path)
	if err != nil {
		return nil
	}
	return db
}

// staleThreshold is the duration after which a change is considered abandoned.
// Changes inactive longer than this are flagged as stale.
const staleThreshold = 24 * time.Hour

func resolveDir(dir string) (string, error) {
	abs, err := os.Getwd()
	if dir != "." {
		abs, err = filepath.Abs(dir)
	}
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

// validateChangeName rejects names that contain path separators or special
// directory components, preventing path traversal when used in filepath.Join.
func validateChangeName(name string) error {
	if name == "" {
		return fmt.Errorf("change name must not be empty")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("change name must not be %q", name)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("change name must not contain path separators: %q", name)
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
	changeDir := filepath.Join(cwd, "openspec", "changes", name)
	info, err := os.Stat(changeDir)
	if err != nil {
		return "", fmt.Errorf("change directory not found: %s", changeDir)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", changeDir)
	}
	return changeDir, nil
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
// Returns (false, nil) on any error — never skips when unsure.
func shouldSkipVerify(cwd, changeDir string) (bool, error) {
	// Check existing report is PASSED.
	reportPath := filepath.Join(changeDir, "verify-report.md")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return false, nil // no report → can't skip
	}
	if !bytes.Contains(data, []byte("**Status:** PASSED")) {
		return false, nil // last run failed → must re-verify
	}

	// Check for source file changes.
	files, err := gitDiffFiles(cwd, "HEAD")
	if err != nil {
		return false, nil // git error → don't skip
	}

	// Filter out openspec/ files — those aren't source code.
	for _, f := range files {
		if !strings.HasPrefix(f, "openspec/") {
			return false, nil // source file changed → must verify
		}
	}

	return true, nil // no source changes + last verify passed → skip
}

// writeJSON marshals v as indented JSON and writes it to w followed by a newline.
// Mirrors the repeated: data, _ := json.MarshalIndent(v, "", "  "); fmt.Fprintln(w, string(data))
func writeJSON(w io.Writer, v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Fprintln(w, string(data))
}

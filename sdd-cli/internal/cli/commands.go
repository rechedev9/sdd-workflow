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

func resolveDir(dir string) (string, error) {
	var abs string
	var err error
	if dir == "." {
		abs, err = os.Getwd()
	} else {
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

// getCWD returns the working directory, writing to stderr and returning an error on failure.
// Used by every CLI command that needs the project root.
func getCWD(stderr io.Writer, cmd string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errs.WriteError(stderr, cmd, fmt.Errorf("get working directory: %w", err))
	}
	return cwd, nil
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
// Uses json.Encoder to stream directly into w, avoiding the intermediate []byte→string copy
// that json.MarshalIndent + fmt.Fprintln would require.
func writeJSON(w io.Writer, v any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v) //nolint:errcheck // stdout write errors are not actionable
}

// errUnknownFlag returns a usage error for an unrecognised CLI flag.
// Centralises the repeated errs.Usage(fmt.Sprintf("unknown flag: %s", flag)) pattern.
func errUnknownFlag(flag string) error {
	return errs.Usage("unknown flag: " + flag)
}

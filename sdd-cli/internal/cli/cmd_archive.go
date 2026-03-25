package cli

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/verify"
)

func runArchive(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) < 1 {
		return errs.Usage("usage: sdd archive <name> [--force]")
	}

	name := args[0]
	force := false
	for _, arg := range args[1:] {
		switch arg {
		case "--force", "-f":
			force = true
		default:
			return errUnknownFlag(arg)
		}
	}

	changeDir, st, err := loadChangeState(stderr, "archive", name)
	if err != nil {
		return err
	}

	if err := st.CanTransition(state.PhaseArchive); err != nil {
		if !force {
			return errs.WriteError(stderr, "archive", fmt.Errorf("not ready to archive: %w", err))
		}
		slog.Warn("archive --force: skipping prerequisite check", "error", err)
	}

	// Verify PR was merged before archiving (best-effort).
	if st.Phases[state.PhaseShip] == state.StatusCompleted {
		cwd, err := getCWD(stderr, "archive")
		if err != nil {
			return err
		}
		if err := verifyPRMerged(cwd, changeDir); err != nil {
			if !force {
				return errs.WriteError(stderr, "archive", fmt.Errorf("PR not merged: %w", err))
			}
			slog.Warn("archive --force: skipping PR merge check", "error", err)
		}
	}

	// Execute archive.
	result, err := verify.Archive(changeDir)
	if err != nil {
		return errs.WriteError(stderr, "archive", err)
	}

	// JSON output.
	out := struct {
		Command      string `json:"command"`
		Status       string `json:"status"`
		Change       string `json:"change"`
		ArchivePath  string `json:"archive_path"`
		ManifestPath string `json:"manifest_path"`
	}{
		Command:      "archive",
		Status:       "success",
		Change:       name,
		ArchivePath:  result.ArchivePath,
		ManifestPath: result.ManifestPath,
	}

	writeJSON(stdout, out)
	return nil
}

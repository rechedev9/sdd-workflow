package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/verify"
)

func runShip(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) < 1 {
		return errs.Usage("usage: sdd ship <name> [--dry-run] [--title <title>]")
	}

	name := args[0]
	dryRun := false
	titleOverride := ""

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			dryRun = true
		case "--title":
			if i+1 >= len(args) {
				return errs.Usage("--title requires a value")
			}
			i++
			titleOverride = args[i]
		default:
			return errUnknownFlag(args[i])
		}
	}

	changeDir, st, err := loadChangeState(stderr, "ship", name)
	if err != nil {
		return err
	}

	if err := st.CanTransition(state.PhaseShip); err != nil {
		return errs.WriteError(stderr, "ship", fmt.Errorf("not ready to ship: %w", err))
	}

	if st.BaseRef == "" {
		return errs.WriteError(stderr, "ship", fmt.Errorf("no base_ref in state.json; cannot compute diff"))
	}

	projectRoot, err := getProjectRoot(stderr, "ship")
	if err != nil {
		return err
	}

	// Require clean working tree — uncommitted changes would not be pushed.
	if err := gitRequireClean(projectRoot); err != nil {
		return errs.WriteError(stderr, "ship",
			fmt.Errorf("working tree is not clean; commit or stash changes first: %w", err))
	}

	// Check gh CLI.
	if err := ghAuthStatus(projectRoot); err != nil {
		return errs.WriteError(stderr, "ship", err)
	}

	// Detect base branch (main, master, etc.).
	baseBranch, err := detectBaseBranch(projectRoot)
	if err != nil {
		return errs.WriteError(stderr, "ship", err)
	}

	branch := "sdd/" + name

	// Check remote branch does not already exist.
	exists, err := gitRemoteBranchExists(projectRoot, "origin", branch)
	if err != nil {
		return errs.WriteError(stderr, "ship", fmt.Errorf("check remote branch: %w", err))
	}
	if exists {
		return errs.WriteError(stderr, "ship", fmt.Errorf("remote branch %q already exists; delete it first", branch))
	}

	// Compute changed source files (exclude openspec/).
	allFiles, err := gitDiffFiles(projectRoot, st.BaseRef)
	if err != nil {
		return errs.WriteError(stderr, "ship", fmt.Errorf("compute diff: %w", err))
	}
	var sourceFiles []string
	for _, f := range allFiles {
		if !strings.HasPrefix(f, "openspec/") {
			sourceFiles = append(sourceFiles, f)
		}
	}

	// Build PR title.
	title := titleOverride
	if title == "" {
		title = fmt.Sprintf("feat(%s): %s", name, st.Description)
	}

	// Build PR body from artifacts.
	body := buildPRBody(changeDir, sourceFiles)

	if dryRun {
		out := struct {
			Command string   `json:"command"`
			Status  string   `json:"status"`
			DryRun  bool     `json:"dry_run"`
			Change  string   `json:"change"`
			Branch  string   `json:"branch"`
			Title   string   `json:"title"`
			Files   []string `json:"files"`
		}{
			Command: "ship",
			Status:  "dry_run",
			DryRun:  true,
			Change:  name,
			Branch:  branch,
			Title:   title,
			Files:   sourceFiles,
		}
		writeJSON(stdout, out)
		return nil
	}

	// Create and checkout branch.
	if err := gitCheckoutNewBranch(projectRoot, branch); err != nil {
		return errs.WriteError(stderr, "ship", err)
	}

	// Push branch.
	if err := gitPush(projectRoot, "origin", branch); err != nil {
		// Cleanup: return to master and delete local branch.
		gitCheckout(projectRoot, baseBranch)      //nolint:errcheck
		gitDeleteLocalBranch(projectRoot, branch) //nolint:errcheck
		return errs.WriteError(stderr, "ship", err)
	}

	// Create PR.
	prURL, err := ghCreatePR(projectRoot, baseBranch, branch, title, body)
	if err != nil {
		// Cleanup: delete remote branch, return to master, delete local branch.
		slog.Warn("ship: PR creation failed, cleaning up", "error", err)
		gitDeleteRemoteBranch(projectRoot, "origin", branch) //nolint:errcheck
		gitCheckout(projectRoot, baseBranch)                 //nolint:errcheck
		gitDeleteLocalBranch(projectRoot, branch)            //nolint:errcheck
		return errs.WriteError(stderr, "ship", err)
	}

	// Write ship report.
	result := &verify.ShipResult{
		Branch:  branch,
		PRURL:   prURL,
		Files:   sourceFiles,
		BaseSHA: st.BaseRef,
	}
	if err := verify.WriteShipReport(result, changeDir); err != nil {
		slog.Warn("ship: failed to write ship-report.md", "error", err)
	}

	// Advance state.
	if err := st.Advance(state.PhaseShip); err != nil {
		return errs.WriteError(stderr, "ship", fmt.Errorf("advance state: %w", err))
	}
	if err := state.Save(st, filepath.Join(changeDir, "state.json")); err != nil {
		return errs.WriteError(stderr, "ship", fmt.Errorf("save state: %w", err))
	}

	// Return to base branch.
	if err := gitCheckout(projectRoot, baseBranch); err != nil {
		slog.Warn("ship: failed to return to base branch", "error", err)
	}

	out := struct {
		Command    string `json:"command"`
		Status     string `json:"status"`
		Change     string `json:"change"`
		Branch     string `json:"branch"`
		PRURL      string `json:"pr_url"`
		FilesCount int    `json:"files_count"`
	}{
		Command:    "ship",
		Status:     "success",
		Change:     name,
		Branch:     branch,
		PRURL:      prURL,
		FilesCount: len(sourceFiles),
	}

	writeJSON(stdout, out)
	return nil
}

// buildPRBody assembles the PR description from change artifacts.
func buildPRBody(changeDir string, files []string) string {
	var b strings.Builder

	b.WriteString("## Summary\n\n")

	// Extract intent from proposal.
	if proposal, err := os.ReadFile(filepath.Join(changeDir, "proposal.md")); err == nil {
		if intent := extractSection(string(proposal), "Intent"); intent != "" {
			b.WriteString(intent)
			b.WriteString("\n\n")
		}
	}

	// File count.
	fmt.Fprintf(&b, "**Files changed:** %d\n\n", len(files))

	// Key design decisions.
	if design, err := os.ReadFile(filepath.Join(changeDir, "design.md")); err == nil {
		if decisions := extractSection(string(design), "Architecture Decisions"); decisions != "" {
			b.WriteString("## Key Decisions\n\n")
			b.WriteString(decisions)
			b.WriteString("\n\n")
		}
	}

	// Verify status.
	if report, err := os.ReadFile(filepath.Join(changeDir, "verify-report.md")); err == nil {
		if strings.Contains(string(report), "**Status:** PASSED") {
			b.WriteString("**Verify:** PASSED\n\n")
		}
	}

	b.WriteString("---\n")
	b.WriteString("_Shipped via `sdd ship`_\n")

	return b.String()
}

// extractSection returns the content of a markdown ## section.
func extractSection(md, heading string) string {
	marker := "## " + heading
	_, after, found := strings.Cut(md, marker)
	if !found {
		return ""
	}
	// Find next ## heading or end of content.
	if before, _, hasNext := strings.Cut(after, "\n## "); hasNext {
		after = before
	}
	return strings.TrimSpace(after)
}

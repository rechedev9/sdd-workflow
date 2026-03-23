package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	sddctx "github.com/rechedev9/shenronSDD/sdd-cli/internal/context"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func runHealth(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) < 1 {
		return errs.Usage("usage: sdd health <name>")
	}

	name := args[0]
	for _, arg := range args[1:] {
		return errUnknownFlag(arg)
	}

	changeDir, st, err := loadChangeState(stderr, "health", name)
	if err != nil {
		return err
	}

	// Count completed phases.
	allPhases := state.AllPhases()
	var completed int
	for _, p := range allPhases {
		if st.Phases[p] == state.StatusCompleted {
			completed++
		}
	}

	// Load pipeline metrics.
	pm := sddctx.LoadPipelineMetrics(changeDir)

	// Build warnings.
	warnings := make([]string, 0, 2)
	if st.IsStale(staleThreshold) {
		warnings = append(warnings, fmt.Sprintf("change inactive for %d hours", st.StaleHours()))
	}

	// Check if last verify failed.
	reportPath := filepath.Join(changeDir, "verify-report.md")
	if data, err := os.ReadFile(reportPath); err == nil {
		if bytes.Contains(data, []byte("**Status:** FAILED")) {
			warnings = append(warnings, "last verify FAILED")
		}
	}

	out := struct {
		Command      string   `json:"command"`
		Status       string   `json:"status"`
		Change       string   `json:"change"`
		CurrentPhase string   `json:"current_phase"`
		Completed    int      `json:"completed"`
		TotalPhases  int      `json:"total_phases"`
		CacheHits    int      `json:"cache_hits"`
		CacheMisses  int      `json:"cache_misses"`
		TotalTokens  int      `json:"total_tokens"`
		Stale        bool     `json:"stale"`
		StaleHours   int      `json:"stale_hours"`
		Warnings     []string `json:"warnings,omitempty"`
	}{
		Command:      "health",
		Status:       "success",
		Change:       st.Name,
		CurrentPhase: string(st.CurrentPhase),
		Completed:    completed,
		TotalPhases:  len(allPhases),
		CacheHits:    pm.CacheHits,
		CacheMisses:  pm.CacheMisses,
		TotalTokens:  pm.TotalTokens,
		Stale:        st.IsStale(staleThreshold),
		StaleHours:   st.StaleHours(),
		Warnings:     warnings,
	}

	writeJSON(stdout, out)
	return nil
}

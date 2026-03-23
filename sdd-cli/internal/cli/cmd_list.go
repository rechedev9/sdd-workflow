package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func runList(_ []string, stdout io.Writer, stderr io.Writer) error {
	cwd, err := getCWD(stderr, "list")
	if err != nil {
		return err
	}

	type changeInfo struct {
		Name         string `json:"name"`
		CurrentPhase string `json:"current_phase"`
		Description  string `json:"description"`
		IsComplete   bool   `json:"is_complete"`
		Stale        bool   `json:"stale,omitempty"`
	}

	changesDir := openspecChanges(cwd)
	if _, err := os.ReadDir(changesDir); err != nil && !os.IsNotExist(err) {
		return errs.WriteError(stderr, "list", fmt.Errorf("read changes directory: %w", err))
	}

	changes := make([]changeInfo, 0)
	eachChangeDir(changesDir, func(changeDir string) {
		st, err := state.Load(filepath.Join(changeDir, "state.json"))
		if err != nil {
			return // skip entries without valid state
		}
		changes = append(changes, changeInfo{
			Name:         st.Name,
			CurrentPhase: string(st.CurrentPhase),
			Description:  st.Description,
			IsComplete:   st.IsComplete(),
			Stale:        st.IsStale(staleThreshold),
		})
	})

	out := struct {
		Command string       `json:"command"`
		Status  string       `json:"status"`
		Count   int          `json:"count"`
		Changes []changeInfo `json:"changes"`
	}{
		Command: "list",
		Status:  "success",
		Count:   len(changes),
		Changes: changes,
	}

	writeJSON(stdout, out)
	return nil
}

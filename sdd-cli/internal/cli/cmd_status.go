package cli

import (
	"io"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func runStatus(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) < 1 {
		return errs.Usage("usage: sdd status <name>")
	}

	name := args[0]

	_, st, err := loadChangeState(stderr, "status", name)
	if err != nil {
		return err
	}

	// Build phase list with statuses.
	type phaseInfo struct {
		Phase  string `json:"phase"`
		Status string `json:"status"`
	}
	phases := make([]phaseInfo, 0, len(state.AllPhases()))
	completed := make([]string, 0, len(state.AllPhases()))
	for _, p := range state.AllPhases() {
		ps := st.Phases[p]
		phases = append(phases, phaseInfo{Phase: string(p), Status: string(ps)})
		if ps == state.StatusCompleted {
			completed = append(completed, string(p))
		}
	}

	out := struct {
		Command      string      `json:"command"`
		Status       string      `json:"status"`
		Change       string      `json:"change"`
		Description  string      `json:"description"`
		CurrentPhase string      `json:"current_phase"`
		Completed    []string    `json:"completed"`
		Phases       []phaseInfo `json:"phases"`
		IsComplete   bool        `json:"is_complete"`
		UpdatedAt    string      `json:"updated_at"`
		Stale        bool        `json:"stale,omitempty"`
		StaleHours   int         `json:"stale_hours,omitempty"`
	}{
		Command:      "status",
		Status:       "success",
		Change:       st.Name,
		Description:  st.Description,
		CurrentPhase: string(st.CurrentPhase),
		Completed:    completed,
		Phases:       phases,
		IsComplete:   st.IsComplete(),
		UpdatedAt:    st.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		Stale:        st.IsStale(staleThreshold),
		StaleHours:   st.StaleHours(),
	}

	writeJSON(stdout, out)
	return nil
}

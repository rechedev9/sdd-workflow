package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/artifacts"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func runWrite(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) < 2 {
		return errs.Usage("usage: sdd write <name> <phase> [--force]")
	}

	name := args[0]
	phaseStr := args[1]
	force := false
	for _, arg := range args[2:] {
		switch arg {
		case "--force", "-f":
			force = true
		}
	}
	phase, err := state.ResolvePhase(phaseStr)
	if err != nil {
		return errs.WriteError(stderr, "write", err)
	}

	changeDir, st, err := loadChangeState(stderr, "write", name)
	if err != nil {
		return err
	}
	statePath := filepath.Join(changeDir, "state.json")

	cwd, err := getCWD(stderr, "write")
	if err != nil {
		return err
	}
	db := tryOpenStore(cwd)
	if db != nil {
		defer db.Close()
	}
	broker := newBroker(0, db)
	prevPhase := string(st.CurrentPhase)

	// Promote pending artifact.
	promoted, err := artifacts.Promote(changeDir, phase, force)
	if err != nil {
		return errs.WriteError(stderr, "write", err)
	}

	broker.Emit(events.Event{
		Type: events.ArtifactPromoted,
		Payload: events.ArtifactPromotedPayload{
			Change:     name,
			Phase:      string(phase),
			PromotedTo: promoted,
		},
	})

	// Advance state.
	if err := st.Advance(phase); err != nil {
		return errs.WriteError(stderr, "write", fmt.Errorf("advance state: %w", err))
	}

	// Save state.
	if err := state.Save(st, statePath); err != nil {
		return errs.WriteError(stderr, "write", err)
	}

	broker.Emit(events.Event{
		Type: events.StateAdvanced,
		Payload: events.StateAdvancedPayload{
			Change:    name,
			FromPhase: prevPhase,
			ToPhase:   string(st.CurrentPhase),
		},
	})

	out := struct {
		Command      string `json:"command"`
		Status       string `json:"status"`
		Change       string `json:"change"`
		Phase        string `json:"phase"`
		PromotedTo   string `json:"promoted_to"`
		CurrentPhase string `json:"current_phase"`
	}{
		Command:      "write",
		Status:       "success",
		Change:       name,
		Phase:        string(phase),
		PromotedTo:   promoted,
		CurrentPhase: string(st.CurrentPhase),
	}
	writeJSON(stdout, out)
	return nil
}

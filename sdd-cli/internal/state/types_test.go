package state

import (
	"testing"
	"time"
)

func TestNewState(t *testing.T) {
	t.Parallel()
	s := NewState("add-auth", "Add authentication module")

	if s.Name != "add-auth" {
		t.Errorf("name = %q, want %q", s.Name, "add-auth")
	}
	if s.Description != "Add authentication module" {
		t.Errorf("description = %q, want %q", s.Description, "Add authentication module")
	}
	if s.CurrentPhase != PhaseExplore {
		t.Errorf("current phase = %q, want %q", s.CurrentPhase, PhaseExplore)
	}
	if s.CreatedAt.IsZero() {
		t.Error("created_at should not be zero")
	}

	allPhases := AllPhases()
	if len(s.Phases) != len(allPhases) {
		t.Errorf("phase count = %d, want %d", len(s.Phases), len(allPhases))
	}
	for _, p := range allPhases {
		status, ok := s.Phases[p]
		if !ok {
			t.Errorf("phase %q missing from state", p)
			continue
		}
		if status != StatusPending {
			t.Errorf("phase %q status = %q, want %q", p, status, StatusPending)
		}
	}
}

func TestIsStale(t *testing.T) {
	t.Parallel()

	t.Run("completed_never_stale", func(t *testing.T) {
		t.Parallel()
		s := NewState("feat", "desc")
		for p := range s.Phases {
			s.Phases[p] = StatusCompleted
		}
		s.UpdatedAt = time.Now().Add(-72 * time.Hour)
		if s.IsStale(time.Hour) {
			t.Error("completed state should never be stale")
		}
	})

	t.Run("not_stale_when_recent", func(t *testing.T) {
		t.Parallel()
		s := NewState("feat", "desc")
		s.UpdatedAt = time.Now().Add(-30 * time.Minute)
		if s.IsStale(time.Hour) {
			t.Error("state updated 30m ago should not be stale with 1h threshold")
		}
	})

	t.Run("stale_when_old", func(t *testing.T) {
		t.Parallel()
		s := NewState("feat", "desc")
		s.UpdatedAt = time.Now().Add(-2 * time.Hour)
		if !s.IsStale(time.Hour) {
			t.Error("state updated 2h ago should be stale with 1h threshold")
		}
	})
}

func TestStaleHours(t *testing.T) {
	t.Parallel()

	s := NewState("feat", "desc")
	s.UpdatedAt = time.Now().Add(-3*time.Hour - 45*time.Minute)
	h := s.StaleHours()
	if h != 3 {
		t.Errorf("StaleHours = %d, want 3", h)
	}
}

func TestAllPhasesOrder(t *testing.T) {
	t.Parallel()
	phases := AllPhases()
	expected := []Phase{
		PhaseExplore, PhasePropose, PhaseSpec, PhaseDesign,
		PhaseTasks, PhaseApply, PhaseReview, PhaseVerify,
		PhaseClean, PhaseShip, PhaseArchive,
	}
	if len(phases) != len(expected) {
		t.Fatalf("phase count = %d, want %d", len(phases), len(expected))
	}
	for i, p := range phases {
		if p != expected[i] {
			t.Errorf("phase[%d] = %q, want %q", i, p, expected[i])
		}
	}
}

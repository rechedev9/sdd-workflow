package context

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func TestAssembleConcurrent_Empty(t *testing.T) {
	t.Parallel()
	_, _, p := setupFixture(t)

	var buf bytes.Buffer
	err := AssembleConcurrent(&buf, nil, p)
	if err != nil {
		t.Fatalf("AssembleConcurrent with nil phases: %v", err)
	}
	if buf.Len() != 0 {
		t.Error("expected empty output for nil phases")
	}
}

func TestAssembleConcurrent_Single(t *testing.T) {
	t.Parallel()
	changeDir, _, p := setupFixture(t)
	os.WriteFile(filepath.Join(changeDir, "exploration.md"), []byte("# Explore\n\nSome exploration.\n"), 0o644)

	var buf bytes.Buffer
	err := AssembleConcurrent(&buf, []state.Phase{state.PhasePropose}, p)
	if err != nil {
		t.Fatalf("AssembleConcurrent single phase: %v", err)
	}
	if !strings.Contains(buf.String(), "sdd-propose") {
		t.Error("missing sdd-propose skill content")
	}
}

func TestAssembleConcurrent_Parallel(t *testing.T) {
	t.Parallel()
	changeDir, _, p := setupFixture(t)

	// Set up both spec and design prerequisites.
	os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte("# Proposal\n\nUse auth middleware.\n"), 0o644)
	specsDir := filepath.Join(changeDir, "specs")
	os.MkdirAll(specsDir, 0o755)
	os.WriteFile(filepath.Join(specsDir, "auth-spec.md"), []byte("# Auth Spec\n\nMUST validate tokens.\n"), 0o644)

	var buf bytes.Buffer
	err := AssembleConcurrent(&buf, []state.Phase{state.PhaseSpec, state.PhaseDesign}, p)
	if err != nil {
		t.Fatalf("AssembleConcurrent spec+design: %v", err)
	}

	out := buf.String()
	// Both assemblers should have run — output contains both skills.
	if !strings.Contains(out, "sdd-spec") {
		t.Error("missing sdd-spec skill content")
	}
	if !strings.Contains(out, "sdd-design") {
		t.Error("missing sdd-design skill content")
	}
	// Output must be in input order: spec before design.
	if strings.Index(out, "sdd-spec") > strings.Index(out, "sdd-design") {
		t.Error("sdd-spec must appear before sdd-design in output (deterministic order)")
	}
}

func TestAssembleConcurrent_PartialFailure(t *testing.T) {
	t.Parallel()
	changeDir, _, p := setupFixture(t)

	// Spec needs proposal.md — create it so spec works.
	os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte("# Proposal\n"), 0o644)
	specsDir := filepath.Join(changeDir, "specs")
	os.MkdirAll(specsDir, 0o755)
	os.WriteFile(filepath.Join(specsDir, "auth-spec.md"), []byte("# Spec\n"), 0o644)
	// Design needs proposal.md and specs/ — both present, so design succeeds too.
	// Let's test a real failure: run with a phase that has no assembler (verify/archive).
	// Actually, use a phase with missing required artifact — propose needs exploration.md.
	// Remove it to cause propose to fail.
	// Propose requires exploration.md but we didn't create it — so propose fails.

	// Run spec (succeeds) + propose (fails — no exploration.md).
	var buf bytes.Buffer
	err := AssembleConcurrent(&buf, []state.Phase{state.PhaseSpec, state.PhasePropose}, p)
	if err == nil {
		t.Fatal("expected error when one phase fails")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("expected 'failed' in error, got: %v", err)
	}
	// Despite propose failing, spec output should still be written.
	if !strings.Contains(buf.String(), "sdd-spec") {
		t.Error("expected partial output from succeeded phase (spec)")
	}
}

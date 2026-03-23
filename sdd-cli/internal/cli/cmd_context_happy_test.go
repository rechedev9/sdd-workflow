package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func TestRunContext_HappyPath_JSON(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "ctx-feat", "context test")
	writeConfig(t, root, "version: 0\nproject_name: test\n")

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runContext([]string{"ctx-feat", "--json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runContext: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout.String())
	}
	if out["command"] != "context" {
		t.Errorf("command = %v, want context", out["command"])
	}
	if out["change"] != "ctx-feat" {
		t.Errorf("change = %v, want ctx-feat", out["change"])
	}
	if out["status"] != "success" {
		t.Errorf("status = %v, want success", out["status"])
	}
	if _, ok := out["context"].(string); !ok {
		t.Errorf("context field missing or not a string")
	}
	if _, ok := out["bytes"].(float64); !ok {
		t.Errorf("bytes field missing or not a number")
	}
}

func TestRunContext_HappyPath_ExplicitPhase(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "ctx-phase", "context explicit phase test")
	writeConfig(t, root, "version: 0\nproject_name: test\n")

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	// Explicit phase arg — bypasses auto-resolve.
	var stdout, stderr bytes.Buffer
	err := runContext([]string{"ctx-phase", "explore"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runContext with explicit phase: %v\nstderr: %s", err, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Error("expected non-empty context output")
	}
}

func TestRunContext_NoPhasesReady(t *testing.T) {
	// Uses Chdir — must not be parallel.
	// Pipeline complete → no phases ready → error.
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "ctx-done")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, root, "version: 0\nproject_name: test\n")

	// Mark all phases completed.
	st := state.NewState("ctx-done", "complete pipeline")
	for _, p := range state.AllPhases() {
		st.Phases[p] = state.StatusCompleted
	}
	st.CurrentPhase = ""
	if err := state.Save(st, filepath.Join(changeDir, "state.json")); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runContext([]string{"ctx-done"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when no phases ready")
	}
}

func TestRunContext_ConcurrentPhases(t *testing.T) {
	// Uses Chdir — must not be parallel.
	// After propose completes, spec+design are both ready → AssembleConcurrent path.
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "ctx-para")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, root, "version: 0\nproject_name: test\n")

	// Mark explore+propose completed so spec+design become ready.
	st := state.NewState("ctx-para", "parallel phases test")
	st.Phases[state.PhaseExplore] = state.StatusCompleted
	st.Phases[state.PhasePropose] = state.StatusCompleted
	st.CurrentPhase = state.PhaseSpec
	if err := state.Save(st, filepath.Join(changeDir, "state.json")); err != nil {
		t.Fatal(err)
	}
	// Spec and design assemblers require proposal.md and specs/.
	if err := os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte("# Proposal\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	specsDir := filepath.Join(changeDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "spec.md"), []byte("# Spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runContext([]string{"ctx-para", "--json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runContext concurrent: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout.String())
	}
	if out["command"] != "context" {
		t.Errorf("command = %v, want context", out["command"])
	}
	// Phase should contain "+" for concurrent phases.
	phase, _ := out["phase"].(string)
	if len(phase) == 0 {
		t.Error("expected non-empty phase field")
	}
}

func TestRunContext_NoConfig(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "ctx-nocfg", "no config")
	// Deliberately no config.yaml.

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runContext([]string{"ctx-nocfg"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when config.yaml missing")
	}
}

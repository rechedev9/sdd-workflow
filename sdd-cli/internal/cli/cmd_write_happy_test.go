package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func TestRunWrite_HappyPath(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "write-feat", "write test")
	writeConfig(t, root, "version: 0\nproject_name: test\n")

	// Write a valid pending artifact for explore phase.
	changeDir := filepath.Join(root, "openspec", "changes", "write-feat")
	pendingDir := filepath.Join(changeDir, ".pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exploreContent := "# Exploration\n\n## Current State\n\nSome state.\n\n## Relevant Files\n\n- main.go\n"
	if err := os.WriteFile(filepath.Join(pendingDir, "explore.md"), []byte(exploreContent), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runWrite([]string{"write-feat", "explore"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runWrite: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout.String())
	}
	if out["command"] != "write" {
		t.Errorf("command = %v, want write", out["command"])
	}
	if out["status"] != "success" {
		t.Errorf("status = %v, want success", out["status"])
	}
	if out["change"] != "write-feat" {
		t.Errorf("change = %v, want write-feat", out["change"])
	}
	if out["phase"] != "explore" {
		t.Errorf("phase = %v, want explore", out["phase"])
	}
}

func TestRunWrite_ForceFlag(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "write-force", "write force test")
	writeConfig(t, root, "version: 0\nproject_name: test\n")

	// Write a pending artifact that would FAIL validation (missing required headings).
	changeDir := filepath.Join(root, "openspec", "changes", "write-force")
	pendingDir := filepath.Join(changeDir, ".pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Missing required headings — validation would reject this without --force.
	if err := os.WriteFile(filepath.Join(pendingDir, "explore.md"), []byte("# Exploration\n\nMinimal content.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	// Without --force should fail validation.
	var stdout, stderr bytes.Buffer
	err := runWrite([]string{"write-force", "explore"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected validation error without --force")
	}

	// With --force should succeed.
	stdout.Reset()
	stderr.Reset()
	err = runWrite([]string{"write-force", "explore", "--force"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runWrite --force: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout.String())
	}
	if out["status"] != "success" {
		t.Errorf("status = %v, want success", out["status"])
	}
}

func TestRunWrite_SpecDirectoryPending(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "write-spec", "write spec test")
	writeConfig(t, root, "version: 0\nproject_name: test\n")

	changeDir := filepath.Join(root, "openspec", "changes", "write-spec")
	st, err := state.Load(filepath.Join(changeDir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	st.Phases[state.PhaseExplore] = state.StatusCompleted
	st.Phases[state.PhasePropose] = state.StatusCompleted
	st.CurrentPhase = state.PhaseSpec
	if err := state.Save(st, filepath.Join(changeDir, "state.json")); err != nil {
		t.Fatal(err)
	}

	pendingSpecDir := filepath.Join(changeDir, ".pending", "specs", "watch-cli")
	if err := os.MkdirAll(pendingSpecDir, 0o755); err != nil {
		t.Fatal(err)
	}
	specContent := "# Watch CLI\n\n## Requirements\n- Expose watch command\n"
	if err := os.WriteFile(filepath.Join(pendingSpecDir, "spec.md"), []byte(specContent), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err = runWrite([]string{"write-spec", "spec"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runWrite spec: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout.String())
	}
	if out["status"] != "success" {
		t.Fatalf("status = %v, want success", out["status"])
	}
	if out["promoted_to"] != filepath.Join(changeDir, "specs") {
		t.Fatalf("promoted_to = %v, want %s", out["promoted_to"], filepath.Join(changeDir, "specs"))
	}
	got, err := os.ReadFile(filepath.Join(changeDir, "specs", "watch-cli", "spec.md"))
	if err != nil {
		t.Fatalf("read promoted spec: %v", err)
	}
	if string(got) != specContent {
		t.Fatalf("spec content = %q, want %q", got, specContent)
	}
}

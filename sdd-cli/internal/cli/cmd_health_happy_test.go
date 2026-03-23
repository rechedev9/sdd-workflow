package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func TestRunHealth_HappyPath(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "health-feat", "health test")

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runHealth([]string{"health-feat"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHealth: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout.String())
	}
	if out["command"] != "health" {
		t.Errorf("command = %v, want health", out["command"])
	}
	if out["change"] != "health-feat" {
		t.Errorf("change = %v, want health-feat", out["change"])
	}
	if out["status"] != "success" {
		t.Errorf("status = %v, want success", out["status"])
	}
	if out["current_phase"] != "explore" {
		t.Errorf("current_phase = %v, want explore", out["current_phase"])
	}
	total, ok := out["total_phases"].(float64)
	if !ok || total == 0 {
		t.Errorf("total_phases = %v, want > 0", out["total_phases"])
	}
}

func TestRunHealth_VerifyFailedWarning(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "health-fail", "verify fail test")

	// Plant a FAILED verify-report.md.
	changeDir := filepath.Join(root, "openspec", "changes", "health-fail")
	reportPath := filepath.Join(changeDir, "verify-report.md")
	failReport := "# Verify Report\n\n**Status:** FAILED (1 command(s) failed)\n"
	if err := os.WriteFile(reportPath, []byte(failReport), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runHealth([]string{"health-fail"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHealth: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout.String())
	}
	warnings, ok := out["warnings"].([]interface{})
	if !ok || len(warnings) == 0 {
		t.Errorf("expected warnings for FAILED verify, got %v", out["warnings"])
	}
}

func TestRunHealth_StaleWarning(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "health-stale")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Backdate UpdatedAt to 48h ago so IsStale(24h) returns true.
	st := state.NewState("health-stale", "stale test")
	st.UpdatedAt = time.Now().Add(-48 * time.Hour)
	if err := state.Save(st, filepath.Join(changeDir, "state.json")); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runHealth([]string{"health-stale"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runHealth: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout.String())
	}
	if out["stale"] != true {
		t.Errorf("stale = %v, want true", out["stale"])
	}
	warnings, ok := out["warnings"].([]interface{})
	if !ok || len(warnings) == 0 {
		t.Errorf("expected stale warning, got %v", out["warnings"])
	}
}

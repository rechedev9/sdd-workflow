package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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

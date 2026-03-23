package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/errlog"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func TestRunVerify_NoConfig(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "ver-nocfg", "no config test")
	// Deliberately no config.yaml.

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runVerify([]string{"ver-nocfg"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when config.yaml missing")
	}
}

func TestRunVerify_ForceFlag(t *testing.T) {
	// Uses Chdir — must not be parallel.
	// With no recurring errors, --force and no-force both proceed to config load.
	root := setupChange(t, "ver-force", "force flag test")
	// No config — still expect error from config load, but no recurring-failure block.

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runVerify([]string{"ver-force", "--force"}, &stdout, &stderr)
	// Expect config-load error (no config.yaml), not a recurring-failure error.
	if err == nil {
		t.Fatal("expected error from missing config")
	}
	if stderr.Len() == 0 {
		t.Error("expected error text on stderr")
	}
}


func TestRunVerify_RecurringFailuresBlocked(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "ver-recurring", "recurring failure test")
	writeConfig(t, root, "version: 0\nproject_name: test\n")

	// Plant recurring failures for "ver-recurring".
	fp := errlog.Fingerprint("go build ./...", []string{"compile error"})
	entry := errlog.ErrorEntry{
		Timestamp:   "2026-01-01T00:00:00Z",
		Change:      "ver-recurring",
		CommandName: "build",
		Command:     "go build ./...",
		ExitCode:    1,
		ErrorLines:  []string{"compile error"},
		Fingerprint: fp,
	}
	for i := 0; i < 3; i++ {
		errlog.Record(root, entry)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runVerify([]string{"ver-recurring"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error from recurring failures block")
	}
	if stderr.Len() == 0 {
		t.Error("expected warning on stderr about recurring failures")
	}

	// With --force it should bypass the recurring check and proceed to config-based run.
	// (Commands in config are empty strings; verify.Run will fail, but that's a different path.)
	stdout.Reset()
	stderr.Reset()
	// The run will succeed or fail based on actual commands — just verify it bypassed block.
	_ = runVerify([]string{"ver-recurring", "--force"}, &stdout, &stderr)
	// The recurring-failure message should NOT be in stderr when --force is used.
	if bytes.Contains(stderr.Bytes(), []byte("recurring failures detected")) {
		t.Error("--force should bypass recurring-failure warning")
	}
	_ = stdout
}

func TestRunVerify_SmartSkip(t *testing.T) {
	// Uses Chdir — must not be parallel.
	// Smart-skip fires when verify-report.md is PASSED and git diff shows no source changes.
	gitBin := "/usr/bin/git"
	if _, err := os.Stat(gitBin); err != nil {
		gitBin = "git"
	}

	root := t.TempDir()

	// Init a git repo with one committed file so git diff HEAD works.
	mustRun := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitBin, args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	mustRun("init")
	mustRun("config", "user.email", "test@test.com")
	mustRun("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustRun("add", "main.go")
	mustRun("commit", "-m", "initial")

	// Create change with state.json and a PASSED verify-report.md.
	changeDir := filepath.Join(root, "openspec", "changes", "sk-feat")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, root, "version: 0\nproject_name: test\n")
	st := state.NewState("sk-feat", "smart skip test")
	if err := state.Save(st, filepath.Join(changeDir, "state.json")); err != nil {
		t.Fatal(err)
	}
	report := "**Status:** PASSED\n\nAll commands succeeded.\n"
	if err := os.WriteFile(filepath.Join(changeDir, "verify-report.md"), []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runVerify([]string{"sk-feat"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runVerify smart-skip: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout.String())
	}
	if out["skipped"] != true {
		t.Errorf("skipped = %v, want true", out["skipped"])
	}
	if out["passed"] != true {
		t.Errorf("passed = %v, want true", out["passed"])
	}
}

func TestRunVerify_RecurringForceJSON(t *testing.T) {
	// Uses Chdir — must not be parallel.
	// This test confirms --force bypasses the block and reaches config+verify.Run.
	// We use a config with trivially-passing commands (echo) so the full path runs.
	root := setupChange(t, "ver-force-json", "force json test")
	writeConfig(t, root, "version: 0\nproject_name: test\ncommands:\n  build: \"echo ok\"\n  lint: \"echo ok\"\n  test: \"echo ok\"\n")

	// Plant recurring failures.
	fp := errlog.Fingerprint("go build ./...", []string{"err"})
	for i := 0; i < 3; i++ {
		errlog.Record(root, errlog.ErrorEntry{
			Timestamp:   "2026-01-01T00:00:00Z",
			Change:      "ver-force-json",
			CommandName: "build",
			Command:     "go build ./...",
			ExitCode:    1,
			ErrorLines:  []string{"err"},
			Fingerprint: fp,
		})
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runVerify([]string{"ver-force-json", "--force"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runVerify --force: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout.String())
	}
	if out["command"] != "verify" {
		t.Errorf("command = %v, want verify", out["command"])
	}
	if out["change"] != "ver-force-json" {
		t.Errorf("change = %v, want ver-force-json", out["change"])
	}
}

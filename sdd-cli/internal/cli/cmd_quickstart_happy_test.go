package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunQuickstart_HappyPath(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := t.TempDir()
	writeConfig(t, root, "version: 0\nproject_name: test\n")

	// Create a spec file.
	specPath := filepath.Join(root, "my-spec.md")
	if err := os.WriteFile(specPath, []byte("# Spec\n\n## Overview\n\nThis is the spec.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runQuickstart([]string{"qs-feat", "fast-forward test", "--spec", specPath}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runQuickstart: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout.String())
	}
	if out["command"] != "quickstart" {
		t.Errorf("command = %v, want quickstart", out["command"])
	}
	if out["status"] != "success" {
		t.Errorf("status = %v, want success", out["status"])
	}
	if out["change"] != "qs-feat" {
		t.Errorf("change = %v, want qs-feat", out["change"])
	}
	if out["current_phase"] != "apply" {
		t.Errorf("current_phase = %v, want apply", out["current_phase"])
	}

	// Verify artifacts were written.
	changeDir := filepath.Join(root, "openspec", "changes", "qs-feat")
	for _, f := range []string{"exploration.md", "proposal.md", "design.md", "tasks.md", "state.json"} {
		if _, err := os.Stat(filepath.Join(changeDir, f)); err != nil {
			t.Errorf("expected artifact %s: %v", f, err)
		}
	}
	// Spec file in specs/.
	specBaseName := filepath.Base(specPath)
	if _, err := os.Stat(filepath.Join(changeDir, "specs", specBaseName)); err != nil {
		t.Errorf("expected spec in specs/: %v", err)
	}
}

func TestRunQuickstart_MissingDescription(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runQuickstart([]string{"qs-feat", "--spec", "/some/spec.md"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when description missing")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunQuickstart_NoConfig(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := t.TempDir()
	// No config.yaml.

	specPath := filepath.Join(root, "spec.md")
	if err := os.WriteFile(specPath, []byte("# Spec"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runQuickstart([]string{"qs-nocfg", "test desc", "--spec", specPath}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when config missing")
	}
}

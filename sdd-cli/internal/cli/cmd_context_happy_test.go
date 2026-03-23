package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
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

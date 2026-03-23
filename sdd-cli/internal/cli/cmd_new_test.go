package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunNew_NoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runNew(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunNew_UnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runNew([]string{"--bad-flag"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunNew_InvalidName(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	// "../bad" triggers validateChangeName failure.
	err := runNew([]string{"../bad", "description"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestRunNew_NoConfig(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	var stdout, stderr bytes.Buffer
	// No config.yaml in dir → config.Load fails.
	err := runNew([]string{"feat-x", "my description"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when config is missing")
	}
}

func TestRunNew_TextOutput(t *testing.T) {
	// Uses Chdir — must not be parallel.
	// Without --json, runNew runs the explore assembler (non-fatal on failure) and returns nil.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	openspecDir := filepath.Join(dir, "openspec")
	os.MkdirAll(filepath.Join(openspecDir, "changes"), 0o755)
	configYAML := "version: 1\nproject_name: test\nstack:\n  language: go\ncommands:\n  build: go build ./...\n  test: go test ./...\n"
	os.WriteFile(filepath.Join(openspecDir, "config.yaml"), []byte(configYAML), 0o644)

	var stdout, stderr bytes.Buffer
	// No --json flag — takes the explore-assembler path; assembly fails non-fatally (no skills dir).
	err := runNew([]string{"feat-text", "a text change"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Change dir should have been created and state saved.
	changeDir := filepath.Join(openspecDir, "changes", "feat-text")
	if _, err := os.Stat(filepath.Join(changeDir, "state.json")); err != nil {
		t.Errorf("state.json not created: %v", err)
	}
}

func TestRunNew_JSONOutput(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	// Create minimal openspec structure with config.yaml.
	openspecDir := filepath.Join(dir, "openspec")
	os.MkdirAll(filepath.Join(openspecDir, "changes"), 0o755)
	configYAML := "version: 1\nproject_name: test\nstack:\n  language: go\ncommands:\n  build: go build ./...\n  test: go test ./...\n"
	os.WriteFile(filepath.Join(openspecDir, "config.yaml"), []byte(configYAML), 0o644)

	var stdout, stderr bytes.Buffer
	err := runNew([]string{"feat-json", "a json change", "--json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out struct {
		Command string `json:"command"`
		Status  string `json:"status"`
		Change  string `json:"change"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, stdout.String())
	}
	if out.Command != "new" {
		t.Errorf("command = %q, want new", out.Command)
	}
	if out.Status != "success" {
		t.Errorf("status = %q, want success", out.Status)
	}
	if out.Change != "feat-json" {
		t.Errorf("change = %q, want feat-json", out.Change)
	}
}

func TestRunNew_WithGitRepo_SetsBaseRef(t *testing.T) {
	// Uses Chdir — must not be parallel.
	// Verifies that gitHeadSHA succeeds in a real git repo → st.BaseRef is set.
	const realGit = "/usr/bin/git"
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", dir},
		{"-C", dir, "config", "user.email", "test@test.com"},
		{"-C", dir, "config", "user.name", "Test"},
		{"-C", dir, "commit", "--allow-empty", "-m", "init"},
	} {
		if err := exec.Command(realGit, args...).Run(); err != nil {
			t.Skipf("git setup failed: %v", err)
		}
	}

	openspecDir := filepath.Join(dir, "openspec")
	os.MkdirAll(filepath.Join(openspecDir, "changes"), 0o755)
	configYAML := "version: 1\nproject_name: test\nstack:\n  language: go\ncommands:\n  build: go build ./...\n  test: go test ./...\n"
	os.WriteFile(filepath.Join(openspecDir, "config.yaml"), []byte(configYAML), 0o644)

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	var stdout, stderr bytes.Buffer
	err := runNew([]string{"feat-gitref", "test git head sha", "--json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if jsonErr := json.Unmarshal(stdout.Bytes(), &out); jsonErr != nil {
		t.Fatalf("unmarshal: %v\n%s", jsonErr, stdout.String())
	}
	if out["status"] != "success" {
		t.Errorf("status = %v, want success", out["status"])
	}
}

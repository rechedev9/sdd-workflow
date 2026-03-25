package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunInit_UnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runInit([]string{"--unknown"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunInit_InvalidDir(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	// Passing a path that resolves but is not a directory.
	err := runInit([]string{"/dev/null"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for non-directory path")
	}
}

func TestRunInit_HappyPath(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	// Create a go.mod so config.Init can detect the stack.
	if err := os.WriteFile("go.mod", []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := runInit(nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runInit: %v", err)
	}
	if stdout.Len() == 0 {
		t.Error("expected JSON output on stdout")
	}
}

func TestRunInit_ForceOnExisting(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	// Create a go.mod so config.Init can detect the stack.
	if err := os.WriteFile("go.mod", []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	// Init first.
	if err := runInit(nil, &stdout, &stderr); err != nil {
		t.Fatalf("first init: %v", err)
	}
	// Second init without --force should fail.
	stdout.Reset()
	if err := runInit(nil, &stdout, &stderr); err == nil {
		t.Error("expected error for second init without --force")
	}
	// Third init with --force should succeed.
	stdout.Reset()
	if err := runInit([]string{"--force"}, &stdout, &stderr); err != nil {
		t.Fatalf("init --force: %v", err)
	}
}

func TestRunInit_NestedManifestFromContainerRoot(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	appDir := filepath.Join(dir, "sdd-cli")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := runInit(nil, &stdout, &stderr); err != nil {
		t.Fatalf("runInit nested manifest: %v\nstderr: %s", err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(appDir, "openspec", "config.yaml")); err != nil {
		t.Fatalf("nested config.yaml not created: %v", err)
	}
}

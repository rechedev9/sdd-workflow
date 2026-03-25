package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInitGoProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0o644)

	result, err := Init(dir, false)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Verify config.yaml exists.
	if _, err := os.Stat(result.ConfigPath); err != nil {
		t.Errorf("config.yaml should exist: %v", err)
	}

	// Verify directory structure.
	for _, d := range []string{"openspec", "openspec/changes", "openspec/changes/archive"} {
		path := filepath.Join(dir, d)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("directory %s should exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory", d)
		}
	}

	// Verify config content.
	cfg, err := Load(result.ConfigPath)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	if cfg.Stack.Language != "go" {
		t.Errorf("language = %q, want go", cfg.Stack.Language)
	}
}

func TestInitNodeProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0o644)

	result, err := Init(dir, false)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	cfg, err := Load(result.ConfigPath)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	if cfg.Stack.Language != "typescript" {
		t.Errorf("language = %q, want typescript", cfg.Stack.Language)
	}
}

func TestInitAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644)
	os.MkdirAll(filepath.Join(dir, "openspec"), 0o755)

	_, err := Init(dir, false)
	if err == nil {
		t.Fatal("expected error for existing openspec/")
	}
	if !errors.Is(err, ErrAlreadyInitialized) {
		t.Errorf("error = %v, want ErrAlreadyInitialized", err)
	}
}

func TestInitForce(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644)
	os.MkdirAll(filepath.Join(dir, "openspec"), 0o755)

	result, err := Init(dir, true)
	if err != nil {
		t.Fatalf("Init --force: %v", err)
	}
	if result.Config.Stack.Language != "go" {
		t.Errorf("language = %q, want go", result.Config.Stack.Language)
	}
}

func TestInitNoManifest(t *testing.T) {
	dir := t.TempDir()

	_, err := Init(dir, false)
	if err == nil {
		t.Fatal("expected error for no manifest")
	}
}

func TestInitNestedManifest(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "sdd-cli")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Init(dir, false)
	if err != nil {
		t.Fatalf("Init nested manifest: %v", err)
	}
	if got, want := result.ConfigPath, filepath.Join(appDir, "openspec", "config.yaml"); got != want {
		t.Fatalf("config path = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(appDir, "openspec", "changes")); err != nil {
		t.Fatalf("nested openspec not created: %v", err)
	}
}

func TestInit_MkdirFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644)
	// Place a file at the openspec path so MkdirAll can't create a directory there.
	os.WriteFile(filepath.Join(dir, "openspec"), []byte("block"), 0o644)
	_, err := Init(dir, false)
	if err == nil {
		t.Fatal("expected error when openspec path is a file")
	}
}

func TestInit_SaveFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644)
	// Create the openspec directory structure, then make openspec/ read-only
	// so config.yaml cannot be written.
	openspecDir := filepath.Join(dir, "openspec")
	os.MkdirAll(filepath.Join(openspecDir, "changes", "archive"), 0o755)
	// Make the openspec dir read-only so AtomicWrite (CreateTemp) fails.
	if err := os.Chmod(openspecDir, 0o555); err != nil {
		t.Skip("cannot chmod:", err)
	}
	t.Cleanup(func() { os.Chmod(openspecDir, 0o755) })

	_, err := Init(dir, true) // force=true skips the already-exists check
	if err == nil {
		t.Fatal("expected error when config.yaml cannot be written")
	}
}

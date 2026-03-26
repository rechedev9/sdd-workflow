package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectGo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0o644)

	cfg, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if cfg.Stack.Language != "go" {
		t.Errorf("language = %q, want go", cfg.Stack.Language)
	}
	if cfg.Commands.Test != "go test ./..." {
		t.Errorf("test cmd = %q, want %q", cfg.Commands.Test, "go test ./...")
	}
	if cfg.Commands.Lint != "golangci-lint run ./..." {
		t.Errorf("lint cmd = %q, want %q", cfg.Commands.Lint, "golangci-lint run ./...")
	}
	if len(cfg.Stack.Manifests) != 1 || cfg.Stack.Manifests[0] != "go.mod" {
		t.Errorf("manifests = %v, want [go.mod]", cfg.Stack.Manifests)
	}
}

func TestDetectNode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name":"test",
		"scripts":{
			"typecheck":"tsc --noEmit",
			"build":"vite build",
			"test":"vitest run",
			"lint":"eslint .",
			"format:check":"prettier --check ."
		}
	}`), 0o644)

	cfg, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if cfg.Stack.Language != "typescript" {
		t.Errorf("language = %q, want typescript", cfg.Stack.Language)
	}
	if cfg.Commands.Build != "npm run typecheck" {
		t.Errorf("build cmd = %q, want %q", cfg.Commands.Build, "npm run typecheck")
	}
	if cfg.Commands.Test != "npm test" {
		t.Errorf("test cmd = %q, want %q", cfg.Commands.Test, "npm test")
	}
	if cfg.Commands.Lint != "npm run lint" {
		t.Errorf("lint cmd = %q, want %q", cfg.Commands.Lint, "npm run lint")
	}
	if cfg.Commands.Format != "npm run format:check" {
		t.Errorf("format cmd = %q, want %q", cfg.Commands.Format, "npm run format:check")
	}
}

func TestDetectNodeWithoutTypecheckSkipsBuild(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name":"test",
		"scripts":{
			"build":"vite build",
			"lint":"eslint .",
			"test":"vitest run"
		}
	}`), 0o644)

	cfg, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if cfg.Commands.Build != "" {
		t.Errorf("build cmd = %q, want empty when only production build is defined", cfg.Commands.Build)
	}
}

func TestDetectPython(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname = \"test\"\n"), 0o644)

	cfg, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if cfg.Stack.Language != "python" {
		t.Errorf("language = %q, want python", cfg.Stack.Language)
	}
	if cfg.Commands.Test != "pytest" {
		t.Errorf("test cmd = %q, want %q", cfg.Commands.Test, "pytest")
	}
	if cfg.Commands.Lint != "ruff check ." {
		t.Errorf("lint cmd = %q, want %q", cfg.Commands.Lint, "ruff check .")
	}
}

func TestDetectRust(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"test\"\n"), 0o644)

	cfg, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if cfg.Stack.Language != "rust" {
		t.Errorf("language = %q, want rust", cfg.Stack.Language)
	}
	if cfg.Commands.Build != "cargo build" {
		t.Errorf("build cmd = %q, want %q", cfg.Commands.Build, "cargo build")
	}
}

func TestDetectJavaGradle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "build.gradle"), []byte(""), 0o644)

	cfg, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if cfg.Stack.Language != "java" {
		t.Errorf("language = %q, want java", cfg.Stack.Language)
	}
	if cfg.Stack.BuildTool != "gradle" {
		t.Errorf("build tool = %q, want gradle", cfg.Stack.BuildTool)
	}
}

func TestDetectJavaMaven(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0o644)

	cfg, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if cfg.Stack.Language != "java" {
		t.Errorf("language = %q, want java", cfg.Stack.Language)
	}
	if cfg.Stack.BuildTool != "maven" {
		t.Errorf("build tool = %q, want maven", cfg.Stack.BuildTool)
	}
}

func TestDetectNoManifest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	_, err := Detect(dir)
	if err == nil {
		t.Fatal("expected error for empty dir")
	}
	if !errors.Is(err, ErrNoManifest) {
		t.Errorf("error = %v, want ErrNoManifest", err)
	}
}

func TestDetectMonorepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Go + Node in same directory — Go wins (first in scan order).
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0o644)

	cfg, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if cfg.Stack.Language != "go" {
		t.Errorf("language = %q, want go (first match)", cfg.Stack.Language)
	}
	if len(cfg.Stack.Manifests) != 2 {
		t.Errorf("manifests count = %d, want 2", len(cfg.Stack.Manifests))
	}
}

func TestDetectProjectName(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644)

	cfg, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	expected := filepath.Base(dir)
	if cfg.ProjectName != expected {
		t.Errorf("project name = %q, want %q", cfg.ProjectName, expected)
	}
}

func TestDetectSkillsPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644)

	cfg, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if cfg.SkillsPath != "" {
		t.Errorf("skills_path should be empty (embedded prompts default), got %q", cfg.SkillsPath)
	}
}

func TestSaveLoad(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	original := &Config{
		ProjectName: "myproject",
		Stack: Stack{
			Language:  "go",
			BuildTool: "go",
			Manifests: []string{"go.mod"},
		},
		Commands: Commands{
			Build: "go build ./...",
			Test:  "go test ./...",
		},
		SkillsPath: "/home/test/.claude/skills/sdd",
	}

	if err := Save(original, path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ProjectName != original.ProjectName {
		t.Errorf("project name = %q, want %q", loaded.ProjectName, original.ProjectName)
	}
	if loaded.Stack.Language != original.Stack.Language {
		t.Errorf("language = %q, want %q", loaded.Stack.Language, original.Stack.Language)
	}
	if loaded.Commands.Test != original.Commands.Test {
		t.Errorf("test cmd = %q, want %q", loaded.Commands.Test, original.Commands.Test)
	}
}

func TestLoadNormalizesLegacyNodeCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"name":"web",
		"scripts":{
			"typecheck":"tsc --noEmit",
			"build":"vite build",
			"lint":"eslint .",
			"test":"vitest run"
		}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Join(dir, "openspec")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, "config.yaml")
	legacy := &Config{
		ProjectName: "web",
		Stack: Stack{
			Language:  "typescript",
			BuildTool: "npm",
			Manifests: []string{"package.json"},
		},
		Commands: Commands{
			Build:  "npm run build",
			Test:   "npm test",
			Lint:   "npm run lint",
			Format: "npm run format",
		},
	}
	if err := Save(legacy, path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Commands.Build != "npm run typecheck" {
		t.Errorf("build cmd = %q, want %q", loaded.Commands.Build, "npm run typecheck")
	}
	if loaded.Commands.Lint != "npm run lint" {
		t.Errorf("lint cmd = %q, want %q", loaded.Commands.Lint, "npm run lint")
	}
	if loaded.Commands.Test != "npm test" {
		t.Errorf("test cmd = %q, want %q", loaded.Commands.Test, "npm test")
	}
}

func TestSaveAtomicNoTmpLeftover(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{ProjectName: "test"}
	if err := Save(cfg, path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	tmp := path + ".tmp"
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error("temp file should not remain after save")
	}
}

func TestLoadMissing(t *testing.T) {
	t.Parallel()
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error loading missing file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(":\n  :\n    - :\n  invalid: [unclosed"), 0o644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadVersionMismatch_Warns(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// Write a config with a future version — Load should succeed with a slog warning.
	os.WriteFile(path, []byte("version: 999\nproject_name: test\n"), 0o644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Version != 999 {
		t.Errorf("version = %d, want 999", cfg.Version)
	}
}

func TestModelFor_PhaseOverride(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Models: Models{
			Default: "sonnet",
			Phases:  map[string]string{"propose": "opus", "spec": "opus"},
		},
	}
	if got := cfg.ModelFor("propose"); got != "opus" {
		t.Errorf("ModelFor(propose) = %q, want opus", got)
	}
	if got := cfg.ModelFor("apply"); got != "sonnet" {
		t.Errorf("ModelFor(apply) = %q, want sonnet (default)", got)
	}
}

func TestModelFor_DefaultOnly(t *testing.T) {
	t.Parallel()
	cfg := &Config{Models: Models{Default: "haiku"}}
	if got := cfg.ModelFor("explore"); got != "haiku" {
		t.Errorf("ModelFor(explore) = %q, want haiku", got)
	}
}

func TestModelFor_Empty(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	if got := cfg.ModelFor("explore"); got != "" {
		t.Errorf("ModelFor(explore) = %q, want empty", got)
	}
}

func TestValidateModels_Valid(t *testing.T) {
	t.Parallel()
	m := Models{
		Default: "sonnet",
		Phases:  map[string]string{"propose": "opus", "review": "haiku"},
	}
	if err := validateModels(m); err != nil {
		t.Fatalf("validateModels: unexpected error: %v", err)
	}
}

func TestValidateModels_InvalidDefault(t *testing.T) {
	t.Parallel()
	m := Models{Default: "gpt-4"}
	err := validateModels(m)
	if err == nil {
		t.Fatal("expected error for invalid default model")
	}
}

func TestValidateModels_InvalidPhaseModel(t *testing.T) {
	t.Parallel()
	m := Models{Phases: map[string]string{"propose": "gpt-4"}}
	err := validateModels(m)
	if err == nil {
		t.Fatal("expected error for invalid phase model")
	}
}

func TestValidateModels_UnknownPhase(t *testing.T) {
	t.Parallel()
	m := Models{Phases: map[string]string{"nonexistent": "opus"}}
	err := validateModels(m)
	if err == nil {
		t.Fatal("expected error for unknown phase name")
	}
}

func TestValidateModels_Empty(t *testing.T) {
	t.Parallel()
	if err := validateModels(Models{}); err != nil {
		t.Fatalf("empty models should be valid: %v", err)
	}
}

func TestLoadWithModels(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := `project_name: test
models:
  default: sonnet
  phases:
    propose: opus
    design: opus
`
	os.WriteFile(path, []byte(yaml), 0o644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Models.Default != "sonnet" {
		t.Errorf("models.default = %q, want sonnet", cfg.Models.Default)
	}
	if cfg.Models.Phases["propose"] != "opus" {
		t.Errorf("models.phases.propose = %q, want opus", cfg.Models.Phases["propose"])
	}
}

func TestLoadWithInvalidModel_Fails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := `project_name: test
models:
  default: gpt-4
`
	os.WriteFile(path, []byte(yaml), 0o644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid model in config")
	}
}

func TestSave_MkdirAllError(t *testing.T) {
	t.Parallel()
	// Create a file where the parent dir should be, so MkdirAll fails.
	root := t.TempDir()
	barrier := filepath.Join(root, "notadir")
	os.WriteFile(barrier, []byte("block"), 0o644)
	path := filepath.Join(barrier, "subdir", "config.yaml")

	cfg := &Config{ProjectName: "test"}
	err := Save(cfg, path)
	if err == nil {
		t.Fatal("expected error when MkdirAll fails")
	}
}

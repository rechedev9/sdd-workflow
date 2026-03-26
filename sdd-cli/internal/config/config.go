package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/fsutil"

	"gopkg.in/yaml.v3"
)

const ConfigVersion = 1

var ErrNoManifest = errors.New("no recognized project manifest found")

const nestedManifestSearchDepth = 3

var validModels = map[string]bool{
	"opus":   true,
	"sonnet": true,
	"haiku":  true,
}

var phaseNames = map[string]bool{
	"explore": true, "propose": true, "spec": true, "design": true,
	"tasks": true, "apply": true, "review": true, "verify": true,
	"clean": true, "ship": true, "archive": true,
}

// ModelFor returns the model configured for the given phase.
// Returns phase-specific override if set, else default, else "".
func (c *Config) ModelFor(phase string) string {
	if m, ok := c.Models.Phases[phase]; ok {
		return m
	}
	return c.Models.Default
}

func validateModels(m Models) error {
	if m.Default != "" && !validModels[m.Default] {
		return fmt.Errorf("unknown default model: %q (valid: opus, sonnet, haiku)", m.Default)
	}
	for phase, model := range m.Phases {
		if !validModels[model] {
			return fmt.Errorf("unknown model %q for phase %q (valid: opus, sonnet, haiku)", model, phase)
		}
		if !phaseNames[phase] {
			return fmt.Errorf("unknown phase %q in models.phases", phase)
		}
	}
	return nil
}

// manifestInfo maps manifest filenames to language/stack detection info.
type manifestInfo struct {
	Language  string
	BuildTool string
	BuildCmd  string
	TestCmd   string
	LintCmd   string
	FormatCmd string
}

// Ordered so the first match wins in monorepo scenarios.
var manifests = []struct {
	File string
	Info manifestInfo
}{
	{"go.mod", manifestInfo{
		Language: "go", BuildTool: "go",
		BuildCmd: "go build ./...", TestCmd: "go test ./...",
		LintCmd: "golangci-lint run ./...", FormatCmd: "gofumpt -w .",
	}},
	{"package.json", manifestInfo{
		Language: "typescript", BuildTool: "npm",
		BuildCmd: "", TestCmd: "",
		LintCmd: "", FormatCmd: "",
	}},
	{"pyproject.toml", manifestInfo{
		Language: "python", BuildTool: "pip",
		BuildCmd: "", TestCmd: "pytest",
		LintCmd: "ruff check .", FormatCmd: "ruff format .",
	}},
	{"Cargo.toml", manifestInfo{
		Language: "rust", BuildTool: "cargo",
		BuildCmd: "cargo build", TestCmd: "cargo test",
		LintCmd: "cargo clippy", FormatCmd: "cargo fmt",
	}},
	{"build.gradle", manifestInfo{
		Language: "java", BuildTool: "gradle",
		BuildCmd: "./gradlew build", TestCmd: "./gradlew test",
		LintCmd: "", FormatCmd: "",
	}},
	{"pom.xml", manifestInfo{
		Language: "java", BuildTool: "maven",
		BuildCmd: "mvn compile", TestCmd: "mvn test",
		LintCmd: "", FormatCmd: "",
	}},
}

// Detect scans projectDir for known manifest files and returns a Config.
func Detect(projectDir string) (*Config, error) {
	return detectConfig(projectDir)
}

// DetectRoot resolves the effective project root for init-style flows.
// It first checks startDir directly, then searches descendants up to a
// bounded depth. If exactly one descendant candidate is found, it wins.
// Multiple candidates are rejected to avoid guessing in container repos.
func DetectRoot(startDir string) (string, *Config, error) {
	cfg, err := detectConfig(startDir)
	if err == nil {
		return startDir, cfg, nil
	}
	if !errors.Is(err, ErrNoManifest) {
		return "", nil, err
	}
	noManifestErr := err

	candidates, err := findManifestDirs(startDir, nestedManifestSearchDepth)
	if err != nil {
		return "", nil, err
	}
	switch len(candidates) {
	case 0:
		return "", nil, noManifestErr
	case 1:
		cfg, derr := detectConfig(candidates[0])
		if derr != nil {
			return "", nil, derr
		}
		return candidates[0], cfg, nil
	default:
		return "", nil, fmt.Errorf("multiple candidate project roots found under %s: %s", startDir, strings.Join(candidates, ", "))
	}
}

func detectConfig(projectDir string) (*Config, error) {
	var found []string
	var primary *manifestInfo

	for _, m := range manifests {
		path := filepath.Join(projectDir, m.File)
		if _, err := os.Stat(path); err == nil {
			found = append(found, m.File)
			if primary == nil {
				info := m.Info
				primary = &info
			}
		}
	}

	if primary == nil {
		return nil, fmt.Errorf("%w: scanned %s", ErrNoManifest, projectDir)
	}

	name := filepath.Base(projectDir)

	cfg := &Config{
		Version:     ConfigVersion,
		ProjectName: name,
		Stack: Stack{
			Language:  primary.Language,
			BuildTool: primary.BuildTool,
			Manifests: found,
		},
		Commands: Commands{
			Build:  primary.BuildCmd,
			Test:   primary.TestCmd,
			Lint:   primary.LintCmd,
			Format: primary.FormatCmd,
		},
		SkillsPath: defaultSkillsPath(),
	}

	if primary.BuildTool == "npm" {
		cfg.Commands = detectNodeCommands(projectDir, cfg.Commands)
	}

	return cfg, nil
}

type packageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

func detectNodeCommands(projectDir string, fallback Commands) Commands {
	data, err := os.ReadFile(filepath.Join(projectDir, "package.json"))
	if err != nil {
		return fallback
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		slog.Warn("package.json parse failed; using fallback commands", "dir", projectDir, "error", err)
		return fallback
	}

	cmds := Commands{
		Build:  firstNodeScript(pkg.Scripts, "typecheck"),
		Test:   firstNodeScript(pkg.Scripts, "test"),
		Lint:   firstNodeScript(pkg.Scripts, "lint"),
		Format: firstNodeScript(pkg.Scripts, "format:check", "format"),
	}
	return cmds
}

func firstNodeScript(scripts map[string]string, names ...string) string {
	for _, name := range names {
		if strings.TrimSpace(scripts[name]) == "" {
			continue
		}
		if name == "test" {
			return "npm test"
		}
		return "npm run " + name
	}
	return ""
}

func findManifestDirs(startDir string, maxDepth int) ([]string, error) {
	candidates := make(map[string]struct{}, 8)
	err := filepath.WalkDir(startDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == startDir {
			return nil
		}

		rel, err := filepath.Rel(startDir, path)
		if err != nil {
			return err
		}
		depth := pathDepth(rel)

		if d.IsDir() {
			if depth > maxDepth {
				return filepath.SkipDir
			}
			switch d.Name() {
			case ".git", "openspec":
				return filepath.SkipDir
			}
			return nil
		}
		if depth > maxDepth {
			return nil
		}
		for _, m := range manifests {
			if d.Name() == m.File {
				candidates[filepath.Dir(path)] = struct{}{}
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan nested manifests: %w", err)
	}
	out := make([]string, 0, len(candidates))
	for dir := range candidates {
		out = append(out, dir)
	}
	slices.Sort(out)
	return out, nil
}

func pathDepth(rel string) int {
	if rel == "." || rel == "" {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}

// Load reads a config.yaml file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Version != 0 && cfg.Version != ConfigVersion {
		slog.Warn("config version mismatch", "have", cfg.Version, "want", ConfigVersion)
	}
	if err := validateModels(cfg.Models); err != nil {
		return nil, fmt.Errorf("invalid models config: %w", err)
	}
	if shouldNormalizeLegacyNodeCommands(&cfg) {
		projectDir := filepath.Dir(filepath.Dir(path))
		cfg.Commands = detectNodeCommands(projectDir, cfg.Commands)
	}
	return &cfg, nil
}

func shouldNormalizeLegacyNodeCommands(cfg *Config) bool {
	if cfg.Stack.BuildTool != "npm" {
		return false
	}
	return cfg.Commands == (Commands{
		Build:  "npm run build",
		Test:   "npm test",
		Lint:   "npm run lint",
		Format: "npm run format",
	})
}

// Save writes a Config to path as YAML.
func Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	return fsutil.AtomicWrite(path, data)
}

func defaultSkillsPath() string {
	return "" // embedded prompts are the default; set skills_path to override
}

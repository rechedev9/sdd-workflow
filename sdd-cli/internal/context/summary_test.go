package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/config"
)

func TestExtractDecisions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "kv pairs extracted",
			input: "## Overview\nSome prose here.\napproach: middleware\nfallback: noop",
			want:  "approach: middleware; fallback: noop",
		},
		{
			name:  "kv inside code fence skipped",
			input: "```\nkey: val\n```\nother: x",
			want:  "other: x",
		},
		{
			name:  "decisions header collected",
			input: "## Decisions\nUse adapter pattern\nNo ORM\nKeep it simple",
			want:  "Use adapter pattern No ORM Keep it simple",
		},
		{
			name:  "architecture header collected",
			input: "## Architecture\nLayer separation\nClean boundaries",
			want:  "Layer separation Clean boundaries",
		},
		{
			name:  "kv cap at 5",
			input: "a: 1\nb: 2\nc: 3\nd: 4\ne: 5\nf: 6",
			want:  "a: 1; b: 2; c: 3; d: 4; e: 5",
		},
		{
			name:  "header cap at 3 lines",
			input: "## Decisions\nLine 1\nLine 2\nLine 3\nLine 4",
			want:  "Line 1 Line 2 Line 3",
		},
		{
			name:  "fallback to extractFirst",
			input: "# Title\n## Section\nFirst line of content\nSecond line\nThird line",
			want:  extractFirst("# Title\n## Section\nFirst line of content\nSecond line\nThird line", "##", 3),
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "kv pairs take priority over header lines",
			input: "lang: Go\n## Decisions\nUse adapters",
			want:  "lang: Go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDecisions(tt.input)
			if got != tt.want {
				t.Errorf("extractDecisions() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsDecisionKey(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{strings.Repeat("x", 31), false},
		{"has space", false},
		{"has\ttab", false},
		{"http://example.com", false},
		{"https://example.com", false},
		{"-flag", false},
		{"approach", true},
		{"BuildTool", true},
		{"lang", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isDecisionKey(tt.input)
			if got != tt.want {
				t.Errorf("isDecisionKey(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractCompletedTasks(t *testing.T) {
	t.Run("with completed tasks", func(t *testing.T) {
		input := "## Phase 1\n- [x] Done task\n- [ ] Pending task\n## Phase 2\n- [x] Also done"
		got := extractCompletedTasks(input)
		if !strings.Contains(got, "Done task") || !strings.Contains(got, "Also done") {
			t.Errorf("expected completed tasks, got %q", got)
		}
	})

	t.Run("no completed tasks", func(t *testing.T) {
		input := "## Phase 1\n- [ ] Pending task"
		got := extractCompletedTasks(input)
		if got != "(no tasks completed yet)" {
			t.Errorf("expected sentinel, got %q", got)
		}
	})
}

func TestBuildSummary_NoArtifacts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := &Params{
		ChangeName:  "add-auth",
		Description: "Add authentication",
		Config: &config.Config{
			Stack: config.Stack{Language: "Go", BuildTool: "make"},
		},
	}
	got := buildSummary(dir, p)
	if !strings.Contains(got, "add-auth") {
		t.Errorf("expected change name in summary, got %q", got)
	}
	if !strings.Contains(got, "Go") {
		t.Errorf("expected stack language in summary, got %q", got)
	}
}

func TestBuildSummary_WithArtifacts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "exploration.md"), []byte("## Findings\nFound something useful"), 0o644)
	os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("approach: layered middleware\nfallback: noop"), 0o644)
	os.WriteFile(filepath.Join(dir, "design.md"), []byte("storage: postgres\npattern: repository"), 0o644)
	os.WriteFile(filepath.Join(dir, "review-report.md"), []byte("## Verdict\nLGTM - approved"), 0o644)

	p := &Params{
		ChangeName:  "feat-x",
		Description: "Feature X",
		Config: &config.Config{
			Stack: config.Stack{Language: "Go", BuildTool: "go"},
		},
	}
	got := buildSummary(dir, p)
	if !strings.Contains(got, "feat-x") {
		t.Errorf("expected change name, got %q", got)
	}
	if !strings.Contains(got, "Proposal:") {
		t.Errorf("expected Proposal section, got %q", got)
	}
	if !strings.Contains(got, "Review:") {
		t.Errorf("expected Review section, got %q", got)
	}
}

func TestLoadManifestContents(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/foo\n\ngo 1.21\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "go.sum"), []byte("checksums here\n"), 0o644)

	got := loadManifestContents(dir, []string{"go.mod", "go.sum"})
	if !strings.Contains(got, "go.mod") {
		t.Errorf("expected go.mod in output, got %q", got)
	}
	if !strings.Contains(got, "module example.com/foo") {
		t.Errorf("expected module path in output, got %q", got)
	}
}

func TestLoadManifestContents_Truncation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Write a file larger than 2KB.
	large := strings.Repeat("x", 3000)
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(large), 0o644)

	got := loadManifestContents(dir, []string{"Cargo.toml"})
	if !strings.Contains(got, "truncated") {
		t.Errorf("expected truncation marker, got %q", got[:100])
	}
}

func TestLoadManifestContents_MissingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	got := loadManifestContents(dir, []string{"nonexistent.toml"})
	if got != "" {
		t.Errorf("expected empty string for missing files, got %q", got)
	}
}

func TestLoadManifestContents_EmptyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Empty file — ReadFull returns n=0 → must be skipped (covers the n==0 continue branch).
	os.WriteFile(filepath.Join(dir, "empty.toml"), []byte{}, 0o644)

	got := loadManifestContents(dir, []string{"empty.toml"})
	if got != "" {
		t.Errorf("expected empty string for zero-byte file, got %q", got)
	}
}

func TestProjectContext_IncludesExecutionRootAndCommands(t *testing.T) {
	t.Parallel()
	p := &Params{
		ProjectDir: "/tmp/worktree/sdd-cli",
		Config: &config.Config{
			ProjectName: "sdd-cli",
			Stack: config.Stack{
				Language:  "go",
				BuildTool: "go",
				Manifests: []string{"go.mod"},
			},
			Commands: config.Commands{
				Build: "go build ./...",
				Test:  "go test ./...",
				Lint:  "golangci-lint run ./...",
			},
		},
	}

	got := projectContext(p)
	if !strings.Contains(got, "Project Root: /tmp/worktree/sdd-cli") {
		t.Fatalf("missing project root in project context: %q", got)
	}
	if !strings.Contains(got, "Execution Root: Run all build/test/lint commands from the Project Root above.") {
		t.Fatalf("missing execution root instruction: %q", got)
	}
	if !strings.Contains(got, "Build Command: go build ./...") {
		t.Fatalf("missing build command: %q", got)
	}
	if !strings.Contains(got, "Test Command: go test ./...") {
		t.Fatalf("missing test command: %q", got)
	}
	if !strings.Contains(got, "Lint Command: golangci-lint run ./...") {
		t.Fatalf("missing lint command: %q", got)
	}
}

func TestExtractFirst_SubHeaderBeforeContent(t *testing.T) {
	t.Parallel()
	// Hit the continue branch: collecting=true, sub-header encountered before any content.
	content := "## Target\n### SubHeader\nActual content here\nSecond line"
	got := extractFirst(content, "Target", 2)
	if !strings.Contains(got, "Actual content here") {
		t.Errorf("extractFirst = %q, want content after sub-header", got)
	}
}

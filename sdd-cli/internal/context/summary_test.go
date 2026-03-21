package context

import (
	"strings"
	"testing"
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

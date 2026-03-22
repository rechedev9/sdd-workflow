package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunCompletion_NoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runCompletion(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunCompletion_Bash(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runCompletion([]string{"bash"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "_sdd_completions") {
		t.Error("expected bash completion function in output")
	}
	if !strings.Contains(out, "sdd") {
		t.Error("expected 'sdd' in output")
	}
}

func TestRunCompletion_Zsh(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runCompletion([]string{"zsh"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "#compdef sdd") {
		t.Error("expected zsh #compdef in output")
	}
}

func TestRunCompletion_Fish(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runCompletion([]string{"fish"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "complete -c sdd") {
		t.Error("expected fish completion in output")
	}
}

func TestRunCompletion_UnknownShell(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runCompletion([]string{"powershell"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown shell")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

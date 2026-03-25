package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func TestRunShip_NoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runShip(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunShip_InvalidName(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runShip([]string{"../bad"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestRunShip_ChangeNotFound(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runShip([]string{"no-such-change-xyz"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for nonexistent change")
	}
}

func TestRunShip_UnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runShip([]string{"some-change", "--bad-flag"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunShip_PrerequisitesNotMet(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "ship-prereq")
	os.MkdirAll(changeDir, 0o755)

	// Create a change with only explore completed — ship requires clean.
	st := state.NewState("ship-prereq", "test ship prerequisites")
	st.Phases[state.PhaseExplore] = state.StatusCompleted
	st.BaseRef = "abc123"
	state.Save(st, filepath.Join(changeDir, "state.json"))

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runShip([]string{"ship-prereq"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for prerequisites not met")
	}
}

func TestRunShip_NoBaseRef(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "ship-noref")
	os.MkdirAll(changeDir, 0o755)

	// Complete up through clean but no BaseRef.
	st := state.NewState("ship-noref", "test ship no base ref")
	for _, p := range []state.Phase{
		state.PhaseExplore, state.PhasePropose, state.PhaseSpec,
		state.PhaseDesign, state.PhaseTasks, state.PhaseApply,
		state.PhaseReview, state.PhaseVerify, state.PhaseClean,
	} {
		st.Phases[p] = state.StatusCompleted
	}
	st.CurrentPhase = state.PhaseShip
	// BaseRef intentionally empty.
	state.Save(st, filepath.Join(changeDir, "state.json"))

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runShip([]string{"ship-noref"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing base_ref")
	}
}

func TestRunShip_TitleFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	// --title without value should error.
	err := runShip([]string{"some-change", "--title"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for --title without value")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestValidateChangeName_GitRefUnsafe(t *testing.T) {
	t.Parallel()
	bad := []string{
		"foo bar",    // space
		"foo:bar",    // colon
		"foo~bar",    // tilde
		"foo^bar",    // caret
		"foo?bar",    // question mark
		"foo*bar",    // asterisk
		"foo[bar",    // open bracket
		"foo..bar",   // double dot
		"foo@{bar",   // at-brace
		"foo.lock",   // .lock suffix
		".hidden",    // leading dot
		"-dashed",    // leading dash
		"foo.",       // trailing dot
		"foo\x00bar", // null byte (control char)
		"foo\x1fbar", // control char
		"foo\x7fbar", // DEL control char
	}
	for _, name := range bad {
		if err := validateChangeName(name); err == nil {
			t.Errorf("validateChangeName(%q) = nil, want error", name)
		}
	}

	// Valid names must still pass.
	good := []string{
		"add-auth",
		"feat-123",
		"my_feature",
		"v2.0",
		"UPPERCASE",
	}
	for _, name := range good {
		if err := validateChangeName(name); err != nil {
			t.Errorf("validateChangeName(%q) = %v, want nil", name, err)
		}
	}
}

func TestDetectGitBin(t *testing.T) {
	t.Parallel()
	bin := detectGitBin()
	if bin == "" {
		t.Fatal("detectGitBin returned empty string")
	}
	// Should find git in at least one well-known path or PATH.
	if bin == "git" {
		t.Log("detectGitBin fell back to PATH — no well-known path found")
	}
}

func TestExtractSection(t *testing.T) {
	t.Parallel()
	md := `# Proposal

## Intent

Add authentication to the API.

## Scope

- Login endpoint
- Session management

## Risks

None identified.
`
	intent := extractSection(md, "Intent")
	if intent != "Add authentication to the API." {
		t.Errorf("extractSection(Intent) = %q", intent)
	}

	scope := extractSection(md, "Scope")
	if scope != "- Login endpoint\n- Session management" {
		t.Errorf("extractSection(Scope) = %q", scope)
	}

	missing := extractSection(md, "Nonexistent")
	if missing != "" {
		t.Errorf("extractSection(Nonexistent) = %q, want empty", missing)
	}
}

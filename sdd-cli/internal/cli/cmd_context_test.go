package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestRunContext_NoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runContext(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunContext_UnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runContext([]string{"--bogus"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunContext_InvalidName(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runContext([]string{"../bad"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid change name")
	}
}

func TestRunContext_ChangeNotFound(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runContext([]string{"no-such-change-xyz"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for nonexistent change")
	}
}

func TestRunContext_AssemblyError(t *testing.T) {
	// Uses Chdir and setupChange — must not be parallel.
	// Pass a valid phase (propose) but the change has no exploration.md,
	// so assembly fails → covers the errs.WriteError path at line 86.
	root := setupChange(t, "ctx-asmerr", "assembly error test")
	writeConfig(t, root, "version: 0\nproject_name: test\n")
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runContext([]string{"ctx-asmerr", "propose"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when required artifact is missing")
	}
}

func TestRunContext_InvalidPhase(t *testing.T) {
	// Uses Chdir and setupChange — must not be parallel.
	root := setupChange(t, "ctx-badphase", "invalid phase test")
	writeConfig(t, root, "version: 0\nproject_name: test\n")
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runContext([]string{"ctx-badphase", "not-a-real-phase"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid phase name")
	}
}

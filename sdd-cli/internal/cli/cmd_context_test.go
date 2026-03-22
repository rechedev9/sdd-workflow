package cli

import (
	"bytes"
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

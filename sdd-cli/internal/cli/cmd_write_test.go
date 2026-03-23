package cli

import (
	"bytes"
	"testing"
)

func TestRunWrite_NoArgs(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runWrite(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunWrite_OneArg(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runWrite([]string{"only-name"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing phase arg")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunWrite_InvalidPhase(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runWrite([]string{"feat-x", "notaphase"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid phase")
	}
}

func TestRunWrite_InvalidName(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runWrite([]string{"../bad", "explore"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for invalid change name")
	}
}

func TestRunWrite_UnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runWrite([]string{"some-change", "explore", "--bad-flag"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunWrite_ChangeNotFound(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runWrite([]string{"no-such-change-xyz", "explore"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for nonexistent change")
	}
}

package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunErrors_UnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runErrors([]string{"--unknown"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunErrors_TextEmpty(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	// No errlog in cwd → empty log → text output.
	err := runErrors(nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "no recorded errors") {
		t.Errorf("expected 'no recorded errors', got %q", got)
	}
}

func TestRunErrors_JSONEmpty(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runErrors([]string{"--json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out struct {
		Command string        `json:"command"`
		Status  string        `json:"status"`
		Total   int           `json:"total"`
		Groups  []interface{} `json:"groups"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal JSON: %v\noutput: %s", err, stdout.String())
	}
	if out.Command != "errors" {
		t.Errorf("command = %q, want %q", out.Command, "errors")
	}
	if out.Status != "success" {
		t.Errorf("status = %q, want %q", out.Status, "success")
	}
}

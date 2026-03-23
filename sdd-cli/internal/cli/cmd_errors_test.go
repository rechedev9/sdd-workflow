package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/errlog"
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

func TestRunErrors_TextWithEntries(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	fp := errlog.Fingerprint("go build", []string{"error: undefined"})
	for i := 0; i < 3; i++ {
		errlog.Record(dir, errlog.ErrorEntry{
			Change: "feat-x", CommandName: "build",
			Command: "go build", ExitCode: 1,
			ErrorLines:  []string{"error: undefined"},
			Fingerprint: fp,
		})
	}

	var stdout, stderr bytes.Buffer
	if err := runErrors(nil, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "entries") {
		t.Errorf("expected 'entries' in output, got %q", got)
	}
	if !strings.Contains(got, "feat-x") {
		t.Errorf("expected change name in output, got %q", got)
	}
}

func TestRunErrors_JSONDedupeTimestamp(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	fp := errlog.Fingerprint("go test", []string{"FAIL"})
	// First entry — older timestamp.
	errlog.Record(dir, errlog.ErrorEntry{
		Change: "old", CommandName: "test",
		Command: "go test", ExitCode: 1,
		Timestamp:   "2026-01-01T00:00:00Z",
		ErrorLines:  []string{"FAIL old"},
		Fingerprint: fp,
	})
	// Second entry — same fingerprint, newer timestamp → should update LastSeen.
	errlog.Record(dir, errlog.ErrorEntry{
		Change: "new", CommandName: "test",
		Command: "go test", ExitCode: 1,
		Timestamp:   "2026-06-01T00:00:00Z",
		ErrorLines:  []string{"FAIL new"},
		Fingerprint: fp,
	})

	var stdout, stderr bytes.Buffer
	if err := runErrors([]string{"--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out struct {
		Total  int `json:"total"`
		Groups []struct {
			Count    int    `json:"count"`
			LastSeen string `json:"last_seen"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, stdout.String())
	}
	if out.Total != 2 {
		t.Errorf("total = %d, want 2", out.Total)
	}
	if len(out.Groups) != 1 {
		t.Fatalf("groups = %d, want 1 (deduped)", len(out.Groups))
	}
	if out.Groups[0].Count != 2 {
		t.Errorf("count = %d, want 2", out.Groups[0].Count)
	}
	if out.Groups[0].LastSeen != "2026-06-01T00:00:00Z" {
		t.Errorf("last_seen = %q, want newer timestamp", out.Groups[0].LastSeen)
	}
}

func TestRunErrors_TextMoreThanTen(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	// Record 12 entries to trigger the start = len-10 path.
	for i := 0; i < 12; i++ {
		fp := errlog.Fingerprint("go build", []string{"err"})
		errlog.Record(dir, errlog.ErrorEntry{
			Change: "feat-z", CommandName: "build",
			Command: "go build", ExitCode: 1,
			ErrorLines:  []string{"err"},
			Fingerprint: fp,
		})
	}

	var stdout, stderr bytes.Buffer
	if err := runErrors(nil, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "entries") {
		t.Errorf("expected 'entries' in output, got %q", got)
	}
}

func TestRunErrors_JSONNilErrorLines(t *testing.T) {
	// Uses Chdir — must not be parallel.
	// Covers the nil ErrorLines path in JSON grouping (lines = []string{}).
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	fp := errlog.Fingerprint("go build", nil)
	errlog.Record(dir, errlog.ErrorEntry{
		Change: "feat-nil", CommandName: "build",
		Command: "go build", ExitCode: 1,
		ErrorLines:  nil,
		Fingerprint: fp,
	})

	var stdout, stderr bytes.Buffer
	if err := runErrors([]string{"--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out struct {
		Groups []struct {
			ErrorLines []string `json:"error_lines"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, stdout.String())
	}
	if len(out.Groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(out.Groups))
	}
	if out.Groups[0].ErrorLines == nil {
		t.Error("expected non-nil ErrorLines slice for nil entry")
	}
}

func TestRunErrors_JSONMultipleGroups(t *testing.T) {
	// Uses Chdir — must not be parallel.
	// Two distinct fingerprints trigger the sort comparison closure.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	fp1 := errlog.Fingerprint("go build", []string{"err1"})
	fp2 := errlog.Fingerprint("go test", []string{"err2"})
	// fp1 gets 2 entries, fp2 gets 1 — sort should put fp1 first.
	for i := 0; i < 2; i++ {
		errlog.Record(dir, errlog.ErrorEntry{
			Change: "feat-a", CommandName: "build",
			Command: "go build", ExitCode: 1,
			ErrorLines: []string{"err1"}, Fingerprint: fp1,
		})
	}
	errlog.Record(dir, errlog.ErrorEntry{
		Change: "feat-b", CommandName: "test",
		Command: "go test", ExitCode: 1,
		ErrorLines: []string{"err2"}, Fingerprint: fp2,
	})

	var stdout, stderr bytes.Buffer
	if err := runErrors([]string{"--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out struct {
		Total  int `json:"total"`
		Groups []struct {
			Count int `json:"count"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, stdout.String())
	}
	if out.Total != 3 {
		t.Errorf("total = %d, want 3", out.Total)
	}
	if len(out.Groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(out.Groups))
	}
	// Sorted by count descending: fp1 (2) before fp2 (1).
	if out.Groups[0].Count != 2 {
		t.Errorf("groups[0].count = %d, want 2 (highest count first)", out.Groups[0].Count)
	}
}

func TestRunErrors_JSONWithEntries(t *testing.T) {
	// Uses Chdir — must not be parallel.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	fp := errlog.Fingerprint("go test", []string{"FAIL"})
	errlog.Record(dir, errlog.ErrorEntry{
		Change: "feat-y", CommandName: "test",
		Command: "go test", ExitCode: 1,
		ErrorLines:  []string{"FAIL"},
		Fingerprint: fp,
	})

	var stdout, stderr bytes.Buffer
	if err := runErrors([]string{"--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out struct {
		Command string `json:"command"`
		Status  string `json:"status"`
		Total   int    `json:"total"`
		Groups  []struct {
			Fingerprint string `json:"fingerprint"`
			Count       int    `json:"count"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal JSON: %v\noutput: %s", err, stdout.String())
	}
	if out.Total != 1 {
		t.Errorf("total = %d, want 1", out.Total)
	}
	if len(out.Groups) != 1 {
		t.Fatalf("groups len = %d, want 1", len(out.Groups))
	}
	if out.Groups[0].Fingerprint != fp {
		t.Errorf("fingerprint = %q, want %q", out.Groups[0].Fingerprint, fp)
	}
}

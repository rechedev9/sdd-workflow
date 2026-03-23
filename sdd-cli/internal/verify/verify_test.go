package verify

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRun_AllPass(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	commands := []CommandSpec{
		{Name: "build", Command: "echo build-ok"},
		{Name: "lint", Command: "echo lint-ok"},
		{Name: "test", Command: "echo test-ok"},
	}

	report, err := Run(dir, commands, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.Passed {
		t.Fatal("expected report.Passed to be true")
	}
	if len(report.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(report.Results))
	}
	for _, r := range report.Results {
		if !r.Passed {
			t.Errorf("expected %s to pass", r.Name)
		}
		if r.ExitCode != 0 {
			t.Errorf("expected exit code 0 for %s, got %d", r.Name, r.ExitCode)
		}
	}
}

func TestRun_OneFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	commands := []CommandSpec{
		{Name: "build", Command: "echo build-ok"},
		{Name: "lint", Command: "echo 'lint error: unused var' >&2; exit 1"},
		{Name: "test", Command: "echo test-ok"},
	}

	report, err := Run(dir, commands, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Passed {
		t.Fatal("expected report.Passed to be false")
	}
	// Should stop after lint failure — test never runs.
	if len(report.Results) != 2 {
		t.Fatalf("expected 2 results (stopped on failure), got %d", len(report.Results))
	}
	if !report.Results[0].Passed {
		t.Error("expected build to pass")
	}
	if report.Results[1].Passed {
		t.Error("expected lint to fail")
	}
	if report.Results[1].ExitCode != 1 {
		t.Errorf("expected exit code 1 for lint, got %d", report.Results[1].ExitCode)
	}
}

func TestRun_Timeout(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	commands := []CommandSpec{
		{Name: "hang", Command: "sleep 60"},
	}

	report, err := Run(dir, commands, 500*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Passed {
		t.Fatal("expected report.Passed to be false")
	}
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}
	r := report.Results[0]
	if !r.TimedOut {
		t.Error("expected TimedOut to be true")
	}
	if r.Passed {
		t.Error("expected command to fail on timeout")
	}
}

func TestRun_SkipsEmptyCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	commands := []CommandSpec{
		{Name: "build", Command: ""},
		{Name: "test", Command: "echo ok"},
	}

	report, err := Run(dir, commands, 30*time.Second, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.Passed {
		t.Fatal("expected report.Passed to be true")
	}
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result (empty skipped), got %d", len(report.Results))
	}
}

func TestErrorLines(t *testing.T) {
	t.Parallel()

	r := &CommandResult{
		Passed: false,
		Output: "line1\nline2\nline3\nline4\nline5\n",
	}

	lines := r.ErrorLines(3)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line1" || lines[2] != "line3" {
		t.Errorf("unexpected lines: %v", lines)
	}

	// Passed command returns nil.
	r2 := &CommandResult{Passed: true, Output: "something"}
	if r2.ErrorLines(5) != nil {
		t.Error("expected nil for passed command")
	}
}

func TestWriteReport_Pass(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	report := &Report{
		Timestamp: time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
		Passed:    true,
		Results: []*CommandResult{
			{Name: "build", Command: "go build ./...", Passed: true, Duration: 2 * time.Second, ExitCode: 0},
			{Name: "test", Command: "go test ./...", Passed: true, Duration: 5 * time.Second, ExitCode: 0},
		},
	}

	if err := WriteReport(report, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "verify-report.md"))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "PASSED") {
		t.Error("expected report to contain PASSED")
	}
	if !strings.Contains(content, "All commands passed") {
		t.Error("expected report to contain 'All commands passed'")
	}
	if !strings.Contains(content, "build — PASS") {
		t.Error("expected report to contain 'build — PASS'")
	}
}

func TestWriteReport_Fail(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	report := &Report{
		Timestamp: time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
		Passed:    false,
		Results: []*CommandResult{
			{Name: "build", Command: "go build ./...", Passed: true, Duration: 2 * time.Second, ExitCode: 0},
			{Name: "lint", Command: "golangci-lint run", Passed: false, Duration: 3 * time.Second, ExitCode: 1, Output: "main.go:10: unused variable\nmain.go:20: missing error check\n"},
		},
	}

	if err := WriteReport(report, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "verify-report.md"))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "FAILED") {
		t.Error("expected report to contain FAILED")
	}
	if !strings.Contains(content, "lint — FAIL") {
		t.Error("expected report to contain 'lint — FAIL'")
	}
	if !strings.Contains(content, "unused variable") {
		t.Error("expected report to contain error output")
	}
}

func TestRun_ProgressOutput(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	commands := []CommandSpec{
		{Name: "build", Command: "echo build-ok"},
		{Name: "lint", Command: "echo 'fail' >&2; exit 1"},
	}

	// Redirect slog to a buffer for this test.
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(prev) })

	_, err := Run(dir, commands, 30*time.Second, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "verify running") || !strings.Contains(out, "command=build") {
		t.Error("missing build start progress")
	}
	if !strings.Contains(out, "verify passed") || !strings.Contains(out, "command=build") {
		t.Error("missing build ok progress")
	}
	if !strings.Contains(out, "verify running") || !strings.Contains(out, "command=lint") {
		t.Error("missing lint start progress")
	}
	if !strings.Contains(out, "verify failed") || !strings.Contains(out, "command=lint") {
		t.Error("missing lint failed progress")
	}
}

func TestWriteReport_TimedOut(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	report := &Report{
		Timestamp: time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
		Passed:    false,
		Results: []*CommandResult{
			{Name: "build", Command: "go build ./...", Passed: false, TimedOut: true, Duration: 5 * time.Minute, ExitCode: -1},
		},
	}

	if err := WriteReport(report, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "verify-report.md"))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(data), "Timed out") {
		t.Error("expected report to contain 'Timed out'")
	}
}

func TestArchive_RenameError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	changeDir := filepath.Join(root, "changes", "my-change")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(changeDir, "state.json"), []byte(`{}`), 0o644)

	// Make the parent directory read-only so rename fails.
	parentDir := filepath.Join(root, "changes")
	os.Chmod(parentDir, 0o555)
	t.Cleanup(func() { os.Chmod(parentDir, 0o755) })

	_, err := Archive(changeDir)
	if err == nil {
		t.Fatal("expected error when rename fails due to read-only parent")
	}
}

func TestArchive_WithPendingDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	changeDir := filepath.Join(root, "changes", "feat")
	pendingDir := filepath.Join(changeDir, ".pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(changeDir, "exploration.md"), []byte("# Exploration"), 0o644)
	os.WriteFile(filepath.Join(pendingDir, "propose.md"), []byte("# Pending"), 0o644)

	result, err := Archive(changeDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	manifest, err := os.ReadFile(result.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	// .pending should not appear in manifest.
	if strings.Contains(string(manifest), ".pending") {
		t.Error("manifest should not list .pending directory")
	}
}

func TestArchive(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// Set up: openspec/changes/my-feature/ with some artifacts.
	changeDir := filepath.Join(root, "openspec", "changes", "my-feature")
	specsDir := filepath.Join(changeDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write some artifacts.
	artifacts := map[string]string{
		"state.json":       `{"name":"my-feature"}`,
		"exploration.md":   "# Exploration",
		"proposal.md":      "# Proposal",
		"design.md":        "# Design",
		"tasks.md":         "# Tasks",
		"verify-report.md": "# Verify Report\n\nPASSED",
	}
	for name, content := range artifacts {
		if err := os.WriteFile(filepath.Join(changeDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Spec file.
	if err := os.WriteFile(filepath.Join(specsDir, "api.md"), []byte("# API Spec"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run archive.
	result, err := Archive(changeDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Original directory should be gone.
	if _, err := os.Stat(changeDir); !os.IsNotExist(err) {
		t.Error("expected change directory to be moved")
	}

	// Archive directory should exist.
	if _, err := os.Stat(result.ArchivePath); err != nil {
		t.Fatalf("archive directory not found: %v", err)
	}

	// Check it's under archive/ with timestamp prefix.
	archiveBase := filepath.Base(result.ArchivePath)
	if !strings.HasSuffix(archiveBase, "-my-feature") {
		t.Errorf("expected archive name to end with -my-feature, got %s", archiveBase)
	}

	// Artifacts should be preserved.
	for name := range artifacts {
		path := filepath.Join(result.ArchivePath, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected artifact %s to exist in archive: %v", name, err)
		}
	}

	// Manifest should exist.
	manifestData, err := os.ReadFile(result.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest := string(manifestData)

	if !strings.Contains(manifest, "my-feature") {
		t.Error("expected manifest to contain change name")
	}
	if !strings.Contains(manifest, "specs/") {
		t.Error("expected manifest to list specs directory")
	}
	if !strings.Contains(manifest, "exploration.md") {
		t.Error("expected manifest to list exploration.md")
	}
}

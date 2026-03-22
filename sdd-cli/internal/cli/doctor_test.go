package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/config"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/errlog"
)

// --- helpers ---

// writeConfig creates a config.yaml at dir/openspec/config.yaml and returns the path.
func writeConfig(t *testing.T, dir, yaml string) string {
	t.Helper()
	configDir := filepath.Join(dir, "openspec")
	os.MkdirAll(configDir, 0o755)
	path := filepath.Join(configDir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0o644)
	return path
}

// writeErrLog populates an error log by recording entries via the errlog API.
func writeErrLog(t *testing.T, cwd string, entries []errlog.ErrorEntry) {
	t.Helper()
	for _, e := range entries {
		errlog.Record(cwd, e)
	}
}

// --- checkSkillsPath (existing) ---

func TestCheckSkillsPathEmpty(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{SkillsPath: ""}
	r := checkSkillsPath(cfg)
	if r.Status != "warn" {
		t.Errorf("expected warn, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "embedded") {
		t.Errorf("expected message about embedded prompts, got %q", r.Message)
	}
}

func TestCheckSkillsPathMissingDir(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{SkillsPath: "/nonexistent/skills/dir"}
	r := checkSkillsPath(cfg)
	if r.Status != "fail" {
		t.Errorf("expected fail, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "/nonexistent/skills/dir") {
		t.Errorf("expected message to contain path, got %q", r.Message)
	}
}

func TestCheckSkillsPathNilConfig(t *testing.T) {
	t.Parallel()
	r := checkSkillsPath(nil)
	if r.Status != "warn" {
		t.Errorf("expected warn, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "config unavailable") {
		t.Errorf("expected 'config unavailable', got %q", r.Message)
	}
}

// --- checkConfig ---

func TestCheckConfigMissingFile(t *testing.T) {
	t.Parallel()
	r, cfg := checkConfig("/nonexistent/path/config.yaml")
	if r.Status != "fail" {
		t.Errorf("expected fail, got %q", r.Status)
	}
	if cfg != nil {
		t.Error("expected nil config on fail")
	}
}

func TestCheckConfigInvalidYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeConfig(t, dir, "{{{{invalid yaml")
	r, cfg := checkConfig(path)
	if r.Status != "fail" {
		t.Errorf("expected fail, got %q", r.Status)
	}
	if cfg != nil {
		t.Error("expected nil config on invalid YAML")
	}
}

func TestCheckConfigVersionZero(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeConfig(t, dir, "version: 0\nproject_name: test\n")
	r, cfg := checkConfig(path)
	if r.Status != "pass" {
		t.Errorf("expected pass for version 0, got %q: %s", r.Status, r.Message)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestCheckConfigCurrentVersion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	yaml := fmt.Sprintf("version: %d\nproject_name: test\n", config.ConfigVersion)
	path := writeConfig(t, dir, yaml)
	r, cfg := checkConfig(path)
	if r.Status != "pass" {
		t.Errorf("expected pass, got %q: %s", r.Status, r.Message)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestCheckConfigVersionMismatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeConfig(t, dir, "version: 99\nproject_name: test\n")
	r, cfg := checkConfig(path)
	if r.Status != "warn" {
		t.Errorf("expected warn for version mismatch, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "99") {
		t.Errorf("expected message to mention version 99, got %q", r.Message)
	}
	// Warn path still returns a usable config.
	if cfg == nil {
		t.Error("expected non-nil config even on version mismatch")
	}
}

// --- checkCache ---

func TestCheckCacheMissingDir(t *testing.T) {
	t.Parallel()
	r := checkCache("/nonexistent/changes", nil)
	if r.Status != "warn" {
		t.Errorf("expected warn, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "cannot read") {
		t.Errorf("expected 'cannot read' message, got %q", r.Message)
	}
}

func TestCheckCacheEmptyDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r := checkCache(dir, nil)
	if r.Status != "pass" {
		t.Errorf("expected pass for empty dir, got %q", r.Status)
	}
}

func TestCheckCacheSkipsArchive(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create an "archive" subdirectory — should be skipped.
	os.MkdirAll(filepath.Join(dir, "archive"), 0o755)
	r := checkCache(dir, nil)
	if r.Status != "pass" {
		t.Errorf("expected pass (archive skipped), got %q", r.Status)
	}
}

func TestCheckCacheNoHashFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "my-change"), 0o755)
	r := checkCache(dir, nil)
	if r.Status != "pass" {
		t.Errorf("expected pass with no hash files, got %q", r.Status)
	}
}

func TestCheckCacheStaleEntry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "my-change")
	cacheDir := filepath.Join(changeDir, ".cache")
	os.MkdirAll(cacheDir, 0o755)
	// Write a hash file with a wrong hash — will be detected as stale.
	os.WriteFile(filepath.Join(cacheDir, "explore.hash"), []byte("wronghash|0"), 0o644)
	r := checkCache(dir, nil)
	if r.Status != "warn" {
		t.Errorf("expected warn for stale cache, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "stale") {
		t.Errorf("expected 'stale' in message, got %q", r.Message)
	}
}

func TestCheckCacheNilConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "my-change"), 0o755)
	// nil cfg means skillsPath defaults to "" — should not panic.
	r := checkCache(dir, nil)
	if r.Status != "pass" {
		t.Errorf("expected pass with nil config, got %q", r.Status)
	}
}

// --- checkOrphanedPending ---

func TestCheckOrphanedPendingMissingDir(t *testing.T) {
	t.Parallel()
	// Missing dir returns pass, not warn.
	r := checkOrphanedPending("/nonexistent/changes")
	if r.Status != "pass" {
		t.Errorf("expected pass for missing dir, got %q", r.Status)
	}
}

func TestCheckOrphanedPendingNoPending(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "my-change")
	os.MkdirAll(changeDir, 0o755)
	r := checkOrphanedPending(dir)
	if r.Status != "pass" {
		t.Errorf("expected pass, got %q", r.Status)
	}
}

func TestCheckOrphanedPendingUnmatched(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "my-change")
	pendingDir := filepath.Join(changeDir, ".pending")
	os.MkdirAll(pendingDir, 0o755)
	// Pending file exists but no promoted counterpart — not orphaned.
	os.WriteFile(filepath.Join(pendingDir, "explore.md"), []byte("pending"), 0o644)
	r := checkOrphanedPending(dir)
	if r.Status != "pass" {
		t.Errorf("expected pass (unmatched pending is not orphaned), got %q", r.Status)
	}
}

func TestCheckOrphanedPendingMatched(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "my-change")
	pendingDir := filepath.Join(changeDir, ".pending")
	os.MkdirAll(pendingDir, 0o755)
	// Both pending and promoted exist — this is an orphan.
	os.WriteFile(filepath.Join(pendingDir, "explore.md"), []byte("pending"), 0o644)
	os.WriteFile(filepath.Join(changeDir, "explore.md"), []byte("promoted"), 0o644)
	r := checkOrphanedPending(dir)
	if r.Status != "warn" {
		t.Errorf("expected warn for orphaned pending, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "1 orphaned") {
		t.Errorf("expected '1 orphaned' in message, got %q", r.Message)
	}
}

func TestCheckOrphanedPendingMultiple(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "my-change")
	pendingDir := filepath.Join(changeDir, ".pending")
	os.MkdirAll(pendingDir, 0o755)
	for _, phase := range []string{"explore", "propose"} {
		os.WriteFile(filepath.Join(pendingDir, phase+".md"), []byte("pending"), 0o644)
		os.WriteFile(filepath.Join(changeDir, phase+".md"), []byte("promoted"), 0o644)
	}
	r := checkOrphanedPending(dir)
	if r.Status != "warn" {
		t.Errorf("expected warn, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "2 orphaned") {
		t.Errorf("expected '2 orphaned', got %q", r.Message)
	}
}

func TestCheckOrphanedPendingSkipsArchive(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	pendingDir := filepath.Join(archiveDir, ".pending")
	os.MkdirAll(pendingDir, 0o755)
	os.WriteFile(filepath.Join(pendingDir, "explore.md"), []byte("pending"), 0o644)
	os.WriteFile(filepath.Join(archiveDir, "explore.md"), []byte("promoted"), 0o644)
	r := checkOrphanedPending(dir)
	if r.Status != "pass" {
		t.Errorf("expected pass (archive skipped), got %q", r.Status)
	}
}

func TestCheckOrphanedPendingSkipsNonMd(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "my-change")
	pendingDir := filepath.Join(changeDir, ".pending")
	os.MkdirAll(pendingDir, 0o755)
	// Non-.md file in pending — should be ignored.
	os.WriteFile(filepath.Join(pendingDir, "notes.txt"), []byte("notes"), 0o644)
	os.WriteFile(filepath.Join(changeDir, "notes.txt"), []byte("notes"), 0o644)
	r := checkOrphanedPending(dir)
	if r.Status != "pass" {
		t.Errorf("expected pass (non-.md skipped), got %q", r.Status)
	}
}

func TestCheckOrphanedPendingSkipsSubdirs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "my-change")
	pendingDir := filepath.Join(changeDir, ".pending")
	os.MkdirAll(filepath.Join(pendingDir, "subdir"), 0o755)
	r := checkOrphanedPending(dir)
	if r.Status != "pass" {
		t.Errorf("expected pass (subdirs skipped), got %q", r.Status)
	}
}

// --- checkBuildTools ---

func TestCheckBuildToolsNilConfig(t *testing.T) {
	t.Parallel()
	r := checkBuildTools(nil)
	if r.Status != "warn" {
		t.Errorf("expected warn, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "config unavailable") {
		t.Errorf("expected 'config unavailable', got %q", r.Message)
	}
}

func TestCheckBuildToolsAllEmpty(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	r := checkBuildTools(cfg)
	if r.Status != "pass" {
		t.Errorf("expected pass for empty commands, got %q", r.Status)
	}
}

func TestCheckBuildToolsAllFound(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Commands: config.Commands{Build: "sh -c true"},
	}
	r := checkBuildTools(cfg)
	if r.Status != "pass" {
		t.Errorf("expected pass, got %q: %s", r.Status, r.Message)
	}
}

func TestCheckBuildToolsOneMissing(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Commands: config.Commands{Build: "__sdd_no_such_binary_xyz build ./..."},
	}
	r := checkBuildTools(cfg)
	if r.Status != "fail" {
		t.Errorf("expected fail, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "__sdd_no_such_binary_xyz") {
		t.Errorf("expected missing binary name in message, got %q", r.Message)
	}
}

func TestCheckBuildToolsDedup(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Commands: config.Commands{
			Build: "sh -c true",
			Test:  "sh -c test",
		},
	}
	r := checkBuildTools(cfg)
	if r.Status != "pass" {
		t.Errorf("expected pass (dedup), got %q: %s", r.Status, r.Message)
	}
}

func TestCheckBuildToolsMultipleMissing(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Commands: config.Commands{
			Build: "__sdd_missing_a run",
			Test:  "__sdd_missing_b run",
		},
	}
	r := checkBuildTools(cfg)
	if r.Status != "fail" {
		t.Errorf("expected fail, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "__sdd_missing_a") || !strings.Contains(r.Message, "__sdd_missing_b") {
		t.Errorf("expected both missing binaries in message, got %q", r.Message)
	}
}

func TestCheckBuildToolsWhitespaceSkip(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Commands: config.Commands{Build: "  "},
	}
	r := checkBuildTools(cfg)
	if r.Status != "pass" {
		t.Errorf("expected pass (whitespace-only command skipped), got %q", r.Status)
	}
}

// --- checkErrors ---

func TestCheckErrorsNoFile(t *testing.T) {
	t.Parallel()
	r := checkErrors(t.TempDir())
	if r.Status != "pass" {
		t.Errorf("expected pass, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "no recorded") {
		t.Errorf("expected 'no recorded' message, got %q", r.Message)
	}
}

func TestCheckErrorsBelowThreshold(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	// Record 2 entries with same fingerprint — below threshold of 3.
	fp := errlog.Fingerprint("go build", []string{"error: something"})
	for i := 0; i < 2; i++ {
		writeErrLog(t, cwd, []errlog.ErrorEntry{{
			CommandName: "build",
			Command:     "go build",
			ExitCode:    1,
			ErrorLines:  []string{"error: something"},
			Fingerprint: fp,
		}})
	}
	r := checkErrors(cwd)
	if r.Status != "pass" {
		t.Errorf("expected pass (below threshold), got %q", r.Status)
	}
	if !strings.Contains(r.Message, "no recurring") {
		t.Errorf("expected 'no recurring' message, got %q", r.Message)
	}
}

func TestCheckErrorsAtThreshold(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	fp := errlog.Fingerprint("go build", []string{"error: repeated"})
	for i := 0; i < 3; i++ {
		writeErrLog(t, cwd, []errlog.ErrorEntry{{
			CommandName: "build",
			Command:     "go build",
			ExitCode:    1,
			ErrorLines:  []string{"error: repeated"},
			Fingerprint: fp,
		}})
	}
	r := checkErrors(cwd)
	if r.Status != "warn" {
		t.Errorf("expected warn at threshold, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "recurring") {
		t.Errorf("expected 'recurring' in message, got %q", r.Message)
	}
}

func TestCheckErrorsAboveThreshold(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	fp := errlog.Fingerprint("go test", []string{"FAIL"})
	for i := 0; i < 5; i++ {
		writeErrLog(t, cwd, []errlog.ErrorEntry{{
			CommandName: "test",
			Command:     "go test",
			ExitCode:    1,
			ErrorLines:  []string{"FAIL"},
			Fingerprint: fp,
		}})
	}
	r := checkErrors(cwd)
	if r.Status != "warn" {
		t.Errorf("expected warn above threshold, got %q", r.Status)
	}
}

func TestCheckErrorsAllDifferent(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf("unique error %d", i)
		fp := errlog.Fingerprint("cmd", []string{msg})
		writeErrLog(t, cwd, []errlog.ErrorEntry{{
			CommandName: "cmd",
			Command:     "cmd",
			ExitCode:    1,
			ErrorLines:  []string{msg},
			Fingerprint: fp,
		}})
	}
	r := checkErrors(cwd)
	if r.Status != "pass" {
		t.Errorf("expected pass (all different), got %q", r.Status)
	}
}

// --- checkPprof ---
// These MUST NOT call t.Parallel() because t.Setenv panics after t.Parallel().

func TestCheckPprofUnset(t *testing.T) {
	t.Setenv("SDD_PPROF", "")
	r := checkPprof()
	if r.Status != "pass" {
		t.Errorf("expected pass, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "not set") {
		t.Errorf("expected 'not set' message, got %q", r.Message)
	}
}

func TestCheckPprofSet(t *testing.T) {
	t.Setenv("SDD_PPROF", "cpu")
	r := checkPprof()
	if r.Status != "pass" {
		t.Errorf("expected pass, got %q", r.Status)
	}
	if !strings.Contains(r.Message, "SDD_PPROF=cpu") {
		t.Errorf("expected 'SDD_PPROF=cpu' in message, got %q", r.Message)
	}
}

// --- aggregateStatus ---

func TestAggregateStatusAllPass(t *testing.T) {
	t.Parallel()
	checks := []CheckResult{
		{Name: "a", Status: "pass"},
		{Name: "b", Status: "pass"},
	}
	if got := aggregateStatus(checks); got != "pass" {
		t.Errorf("aggregateStatus = %q, want pass", got)
	}
}

func TestAggregateStatusWithWarn(t *testing.T) {
	t.Parallel()
	checks := []CheckResult{
		{Name: "a", Status: "pass"},
		{Name: "b", Status: "warn"},
		{Name: "c", Status: "pass"},
	}
	if got := aggregateStatus(checks); got != "warn" {
		t.Errorf("aggregateStatus = %q, want warn", got)
	}
}

func TestAggregateStatusWithFail(t *testing.T) {
	t.Parallel()
	checks := []CheckResult{
		{Name: "a", Status: "warn"},
		{Name: "b", Status: "fail"},
		{Name: "c", Status: "pass"},
	}
	if got := aggregateStatus(checks); got != "fail" {
		t.Errorf("aggregateStatus = %q, want fail", got)
	}
}

func TestAggregateStatusEmpty(t *testing.T) {
	t.Parallel()
	if got := aggregateStatus(nil); got != "pass" {
		t.Errorf("aggregateStatus(nil) = %q, want pass", got)
	}
}

// --- printDoctorTable ---

func TestPrintDoctorTable(t *testing.T) {
	t.Parallel()
	checks := []CheckResult{
		{Name: "config", Status: "pass", Message: ""},
		{Name: "tools", Status: "warn", Message: "missing: foo"},
		{Name: "cache", Status: "fail", Message: "2 stale entries"},
	}
	var buf strings.Builder
	printDoctorTable(&buf, checks)
	out := buf.String()

	if !strings.Contains(out, "sdd doctor") {
		t.Error("output should contain 'sdd doctor' header")
	}
	if !strings.Contains(out, "config") {
		t.Error("output should contain check name 'config'")
	}
	if !strings.Contains(out, "missing: foo") {
		t.Error("output should contain message 'missing: foo'")
	}
	if !strings.Contains(out, "2 stale entries") {
		t.Error("output should contain '2 stale entries'")
	}
}

func TestPrintDoctorTableEmpty(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	printDoctorTable(&buf, nil)
	if !strings.Contains(buf.String(), "sdd doctor") {
		t.Error("output should still contain header even with no checks")
	}
}

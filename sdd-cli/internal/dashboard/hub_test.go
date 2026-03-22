package dashboard

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/store"
)

func newTestHub(t *testing.T) *Hub {
	t.Helper()
	dir := t.TempDir()
	return NewHub(&fakeMetrics{}, dir)
}

func TestParseSinceTS_Empty(t *testing.T) {
	t.Parallel()
	h := newTestHub(t)
	before := time.Now()
	got := h.parseSinceTS("")
	after := time.Now()
	// Empty string → defaultLookback ago.
	lo := before.Add(-defaultLookback)
	hi := after.Add(-defaultLookback)
	if got.Before(lo) || got.After(hi) {
		t.Errorf("parseSinceTS(\"\") = %v, want in [%v, %v]", got, lo, hi)
	}
}

func TestParseSinceTS_Valid(t *testing.T) {
	t.Parallel()
	h := newTestHub(t)
	ts := "2026-01-15T10:00:00Z"
	want, _ := time.Parse(time.RFC3339, ts)
	got := h.parseSinceTS(ts)
	if !got.Equal(want) {
		t.Errorf("parseSinceTS(%q) = %v, want %v", ts, got, want)
	}
}

func TestParseSinceTS_Invalid(t *testing.T) {
	t.Parallel()
	h := newTestHub(t)
	before := time.Now()
	got := h.parseSinceTS("not-a-timestamp")
	after := time.Now()
	// Invalid → defaultLookback ago.
	lo := before.Add(-defaultLookback)
	hi := after.Add(-defaultLookback)
	if got.Before(lo) || got.After(hi) {
		t.Errorf("parseSinceTS(invalid) = %v, want in [%v, %v]", got, lo, hi)
	}
}

func TestCachedVerifyStatus_NoFile(t *testing.T) {
	t.Parallel()
	h := newTestHub(t)
	dir := t.TempDir()
	// No verify-report.md → "ok".
	got := h.cachedVerifyStatus(dir)
	if got != "ok" {
		t.Errorf("cachedVerifyStatus (no file) = %q, want %q", got, "ok")
	}
}

func TestCachedVerifyStatus_Passed(t *testing.T) {
	t.Parallel()
	h := newTestHub(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "verify-report.md"), []byte("**Status:** PASSED\n"), 0o644)
	got := h.cachedVerifyStatus(dir)
	if got != "ok" {
		t.Errorf("cachedVerifyStatus (PASSED) = %q, want %q", got, "ok")
	}
}

func TestCachedVerifyStatus_Failed(t *testing.T) {
	t.Parallel()
	h := newTestHub(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "verify-report.md"), []byte("**Status:** FAILED\n"), 0o644)
	got := h.cachedVerifyStatus(dir)
	if got != "error" {
		t.Errorf("cachedVerifyStatus (FAILED) = %q, want %q", got, "error")
	}
}

func TestCachedVerifyStatus_CacheHit(t *testing.T) {
	t.Parallel()
	h := newTestHub(t)
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "verify-report.md")
	os.WriteFile(reportPath, []byte("**Status:** FAILED\n"), 0o644)
	// First call populates cache.
	h.cachedVerifyStatus(dir)
	// Second call should use cached value.
	got := h.cachedVerifyStatus(dir)
	if got != "error" {
		t.Errorf("cached second call = %q, want %q", got, "error")
	}
}

func TestPruneVerifyCache(t *testing.T) {
	t.Parallel()
	h := newTestHub(t)

	// Manually populate the verify cache.
	h.verifyCacheMu.Lock()
	h.verifyCache["active-dir"] = verifyCacheEntry{status: "ok"}
	h.verifyCache["stale-dir"] = verifyCacheEntry{status: "error"}
	h.verifyCacheMu.Unlock()

	active := map[string]struct{}{"active-dir": {}}
	h.pruneVerifyCache(active)

	h.verifyCacheMu.RLock()
	defer h.verifyCacheMu.RUnlock()

	if _, found := h.verifyCache["active-dir"]; !found {
		t.Error("expected active-dir to remain in cache")
	}
	if _, found := h.verifyCache["stale-dir"]; found {
		t.Error("expected stale-dir to be pruned from cache")
	}
}

func TestLoadChanges_Empty(t *testing.T) {
	t.Parallel()
	h := newTestHub(t)
	// changesDir is empty → no changes.
	changes := h.loadChanges()
	if len(changes) != 0 {
		t.Errorf("expected 0 changes for empty dir, got %d", len(changes))
	}
}

func TestLoadChanges_WithChange(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	changesDir := filepath.Join(dir, "changes")
	os.MkdirAll(changesDir, 0o755)
	h := NewHub(&fakeMetrics{}, changesDir)

	// Create a change directory with a valid state.json.
	changeDir := filepath.Join(changesDir, "feat-x")
	os.MkdirAll(changeDir, 0o755)
	st := state.NewState("feat-x", "test change")
	state.Save(st, filepath.Join(changeDir, "state.json"))

	changes := h.loadChanges()
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].state.Name != "feat-x" {
		t.Errorf("change name = %q, want %q", changes[0].state.Name, "feat-x")
	}
}

func TestLoadChanges_SkipsArchive(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	changesDir := filepath.Join(dir, "changes")
	os.MkdirAll(changesDir, 0o755)
	h := NewHub(&fakeMetrics{}, changesDir)

	// Create the special "archive" directory — should be skipped.
	os.MkdirAll(filepath.Join(changesDir, "archive"), 0o755)

	changes := h.loadChanges()
	if len(changes) != 0 {
		t.Errorf("expected 0 changes (archive skipped), got %d", len(changes))
	}
}

func TestBuildErrors_Empty(t *testing.T) {
	t.Parallel()
	h := newTestHub(t)
	data := h.buildErrors(context.Background())
	if data == nil {
		t.Error("expected non-nil slice for empty errors")
	}
	if len(data) != 0 {
		t.Errorf("expected 0 errors, got %d", len(data))
	}
}

func TestBuildErrors_WithRows(t *testing.T) {
	t.Parallel()
	fm := &fakeMetrics{
		errors: []store.ErrorRow{
			{
				Timestamp:   "2026-01-01T00:00:00Z",
				CommandName: "build",
				ExitCode:    1,
				Change:      "feat-a",
				Fingerprint: "abcdef0123456789",
				FirstLine:   "error: undefined",
			},
		},
	}
	h := NewHub(fm, t.TempDir())
	data := h.buildErrors(context.Background())
	if len(data) != 1 {
		t.Fatalf("expected 1 error row, got %d", len(data))
	}
	if data[0].Fingerprint != "abcdef01" {
		t.Errorf("fingerprint = %q, want %q", data[0].Fingerprint, "abcdef01")
	}
}

func TestBuildHeatmapFromChanges_Empty(t *testing.T) {
	t.Parallel()
	grid := buildHeatmapFromChanges(nil)
	if len(grid) != 0 {
		t.Errorf("expected 0 rows for empty changes, got %d", len(grid))
	}
}

func TestBuildHeatmapFromChanges_WithChange(t *testing.T) {
	t.Parallel()
	st := state.NewState("feat-y", "test")
	changes := []changeSnapshot{{dir: "/tmp/feat-y", state: st}}
	grid := buildHeatmapFromChanges(changes)
	allPhases := state.AllPhases()
	if len(grid) != len(allPhases) {
		t.Errorf("grid rows = %d, want %d", len(grid), len(allPhases))
	}
	for _, row := range grid {
		if row.Change != "feat-y" {
			t.Errorf("change = %q, want %q", row.Change, "feat-y")
		}
		if row.Status == "" {
			t.Errorf("status empty for phase %s", row.Phase)
		}
	}
}

func TestBuildPipelinesFromChanges_Empty(t *testing.T) {
	t.Parallel()
	h := newTestHub(t)
	pipelines := h.buildPipelinesFromChanges(context.Background(), nil)
	if len(pipelines) != 0 {
		t.Errorf("expected 0 pipelines for nil changes, got %d", len(pipelines))
	}
}

func TestBuildPipelinesFromChanges_WithChange(t *testing.T) {
	t.Parallel()
	fm := &fakeMetrics{
		tokens: []store.ChangeTokens{
			{Change: "feat-z", Tokens: 1000},
		},
	}
	h := NewHub(fm, t.TempDir())
	st := state.NewState("feat-z", "test")
	changes := []changeSnapshot{{dir: t.TempDir(), state: st}}

	pipelines := h.buildPipelinesFromChanges(context.Background(), changes)
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}
	p := pipelines[0]
	if p.Name != "feat-z" {
		t.Errorf("name = %q, want %q", p.Name, "feat-z")
	}
	if p.Tokens != 1000 {
		t.Errorf("tokens = %d, want 1000", p.Tokens)
	}
	if p.Total != len(state.AllPhases()) {
		t.Errorf("total = %d, want %d", p.Total, len(state.AllPhases()))
	}
}

func TestBuildPipelinesFromChanges_StaleStatus(t *testing.T) {
	t.Parallel()
	h := newTestHub(t)

	// Create a state that is old enough to be stale (> 24h).
	st := state.NewState("feat-stale", "test")
	// Override UpdatedAt to a time far in the past.
	st.UpdatedAt = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	changes := []changeSnapshot{{dir: t.TempDir(), state: st}}
	pipelines := h.buildPipelinesFromChanges(context.Background(), changes)
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}
	// Stale change with no FAILED verify report → "warn".
	if pipelines[0].Status != "warn" {
		t.Errorf("status = %q, want %q", pipelines[0].Status, "warn")
	}
}

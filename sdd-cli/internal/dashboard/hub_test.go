package dashboard

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

package dashboard

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

const defaultLookback = 24 * time.Hour

// wsMessage is the JSON envelope sent over WebSocket.
type wsMessage struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// PhaseStatusRow holds one cell of the pipeline heatmap.
type PhaseStatusRow struct {
	Change string `json:"change"`
	Phase  string `json:"phase"`
	Status string `json:"status"`
}

// changeSnapshot holds the loaded state for a single active change directory.
type changeSnapshot struct {
	dir   string
	state *state.State
}

// Hub manages WebSocket clients and pushes data deltas.
type Hub struct {
	metrics    MetricsReader
	changesDir string

	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}

	// Last-known state for diffing via content hashes.
	lastKPI           KPIData
	lastPipelinesHash [sha256.Size]byte
	lastErrorsHash    [sha256.Size]byte
	lastHeatmapHash   [sha256.Size]byte
	lastDurationsHash [sha256.Size]byte
	lastTokenTS       string // max timestamp seen in phase_events
	lastVerifyTS      string // max timestamp seen in verify_results
	lastCacheTS       string // tracks cache history watermark independently

	// Cached verify-report status per change directory.
	verifyCache   map[string]verifyCacheEntry
	verifyCacheMu sync.RWMutex
}

// verifyCacheEntry caches the status derived from a verify-report.md file.
type verifyCacheEntry struct {
	modTime time.Time
	status  string // "error" or "ok"
}

// NewHub creates a hub. Call Run() to start the poll loop.
func NewHub(m MetricsReader, changesDir string) *Hub {
	return &Hub{
		metrics:     m,
		changesDir:  changesDir,
		clients:     make(map[*websocket.Conn]struct{}),
		verifyCache: make(map[string]verifyCacheEntry),
	}
}

// Run starts the 1-second poll loop. Blocks until ctx is cancelled.
func (h *Hub) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.poll(ctx)
		}
	}
}

// HandleWS upgrades an HTTP connection to WebSocket and sends the initial snapshot.
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // local dev tool, accept any origin
	})
	if err != nil {
		slog.Error("ws accept", "error", err)
		return
	}

	ctx := r.Context()

	h.sendSnapshot(ctx, conn)

	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()

	// Keep connection alive — read loop discards incoming messages.
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}

	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	_ = conn.Close(websocket.StatusNormalClosure, "") // best-effort close
}

// jsonHash returns the SHA-256 hash of the JSON encoding of v.
// broadcastIfChanged broadcasts data only when its JSON hash differs from lastHash.
func (h *Hub) broadcastIfChanged(ctx context.Context, msgType string, data any, lastHash *[sha256.Size]byte) {
	if hash := jsonHash(data); hash != *lastHash {
		h.broadcast(ctx, wsMessage{Type: msgType, Data: data})
		*lastHash = hash
	}
}

func jsonHash(v any) [sha256.Size]byte {
	data, err := json.Marshal(v)
	if err != nil {
		return [sha256.Size]byte{}
	}
	return sha256.Sum256(data)
}

// poll queries DB + filesystem, diffs against last state, broadcasts deltas.
func (h *Hub) poll(ctx context.Context) {
	h.mu.RLock()
	empty := len(h.clients) == 0
	h.mu.RUnlock()
	if empty {
		return
	}

	// Single filesystem walk — shared by KPI, pipelines, and heatmap.
	changes := h.loadChanges()

	// Prune verify cache to prevent unbounded growth.
	activeDirs := make(map[string]struct{}, len(changes))
	for _, ch := range changes {
		activeDirs[ch.dir] = struct{}{}
	}
	h.pruneVerifyCache(activeDirs)

	// KPIs.
	kpi := h.buildKPIFromChanges(ctx, changes)
	if kpi != h.lastKPI {
		h.broadcast(ctx, wsMessage{Type: "kpi", Data: kpi})
		h.lastKPI = kpi
	}

	// Pipelines — hash-based diffing instead of reflect.DeepEqual.
	h.broadcastIfChanged(ctx, "pipelines", h.buildPipelinesFromChanges(ctx, changes), &h.lastPipelinesHash)

	// Errors — hash-based diffing.
	h.broadcastIfChanged(ctx, "errors", h.buildErrors(ctx), &h.lastErrorsHash)

	// Heatmap — hash-based diffing.
	h.broadcastIfChanged(ctx, "chart:heatmap", buildHeatmapFromChanges(changes), &h.lastHeatmapHash)

	// Chart data — incremental by timestamp.
	tokenSince := h.parseSinceTS(h.lastTokenTS)

	if rows, err := h.metrics.TokenHistory(ctx, tokenSince); err == nil && len(rows) > 0 {
		h.broadcast(ctx, wsMessage{Type: "chart:tokens", Data: rows})
		h.lastTokenTS = rows[len(rows)-1].Timestamp
	}

	if rows, err := h.metrics.PhaseDurations(ctx); err == nil && len(rows) > 0 {
		h.broadcastIfChanged(ctx, "chart:durations", rows, &h.lastDurationsHash)
	}

	cacheSince := h.parseSinceTS(h.lastCacheTS)
	if rows, err := h.metrics.CacheHistory(ctx, cacheSince); err == nil && len(rows) > 0 {
		h.broadcast(ctx, wsMessage{Type: "chart:cache", Data: rows})
		h.lastCacheTS = rows[len(rows)-1].Timestamp
	}

	verifySince := h.parseSinceTS(h.lastVerifyTS)
	if rows, err := h.metrics.VerifyHistory(ctx, verifySince); err == nil && len(rows) > 0 {
		h.broadcast(ctx, wsMessage{Type: "chart:verify", Data: rows})
		h.lastVerifyTS = rows[len(rows)-1].Timestamp
	}
}

// parseSinceTS converts a stored timestamp to time.Time, falling back to defaultLookback.
func (h *Hub) parseSinceTS(ts string) time.Time {
	if ts == "" {
		return time.Now().Add(-defaultLookback)
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Now().Add(-defaultLookback)
	}
	return t
}

// sendSnapshot sends the full current state to a single client.
func (h *Hub) sendSnapshot(ctx context.Context, conn *websocket.Conn) {
	since := time.Now().Add(-defaultLookback)
	changes := h.loadChanges()

	msgs := []wsMessage{
		{Type: "kpi", Data: h.buildKPIFromChanges(ctx, changes)},
		{Type: "pipelines", Data: h.buildPipelinesFromChanges(ctx, changes)},
		{Type: "errors", Data: h.buildErrors(ctx)},
		{Type: "chart:heatmap", Data: buildHeatmapFromChanges(changes)},
	}

	if rows, err := h.metrics.TokenHistory(ctx, since); err == nil {
		msgs = append(msgs, wsMessage{Type: "chart:tokens", Data: rows})
	}
	if rows, err := h.metrics.PhaseDurations(ctx); err == nil {
		msgs = append(msgs, wsMessage{Type: "chart:durations", Data: rows})
	}
	if rows, err := h.metrics.CacheHistory(ctx, since); err == nil {
		msgs = append(msgs, wsMessage{Type: "chart:cache", Data: rows})
	}
	if rows, err := h.metrics.VerifyHistory(ctx, since); err == nil {
		msgs = append(msgs, wsMessage{Type: "chart:verify", Data: rows})
	}

	for _, msg := range msgs {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
			return
		}
	}
}

// broadcast sends a message to all connected clients, removing dead ones.
func (h *Hub) broadcast(ctx context.Context, msg wsMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
			_ = conn.Close(websocket.StatusGoingAway, "") // best-effort close dead client
			delete(h.clients, conn)
		}
	}
}

// loadChanges walks changesDir once and loads state.json for each active change.
func (h *Hub) loadChanges() []changeSnapshot {
	entries, err := os.ReadDir(h.changesDir)
	if err != nil {
		return nil
	}

	changes := make([]changeSnapshot, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		changeDir := filepath.Join(h.changesDir, e.Name())
		statePath := filepath.Join(changeDir, "state.json")
		st, err := state.Load(statePath)
		if err != nil {
			continue
		}
		changes = append(changes, changeSnapshot{dir: changeDir, state: st})
	}
	return changes
}

// buildKPIFromChanges computes KPI data using pre-loaded changes.
func (h *Hub) buildKPIFromChanges(ctx context.Context, changes []changeSnapshot) KPIData {
	data := KPIData{ActiveChanges: len(changes)}

	if stats, err := h.metrics.TokenSummary(ctx); err == nil {
		data.TotalTokens = stats.TotalTokens
		data.CacheHitPct = stats.CacheHitPct
		data.ErrorCount = stats.ErrorCount
	}

	return data
}

// buildPipelinesFromChanges computes pipeline data using pre-loaded changes.
func (h *Hub) buildPipelinesFromChanges(ctx context.Context, changes []changeSnapshot) []PipelineData {
	tokenMap := make(map[string]int, len(changes))
	if ct, err := h.metrics.PhaseTokensByChange(ctx); err == nil {
		for _, c := range ct {
			tokenMap[c.Change] = c.Tokens
		}
	}

	allPhases := state.AllPhases()
	total := len(allPhases)
	pipelines := make([]PipelineData, 0, len(changes))

	for _, ch := range changes {
		completed := 0
		for _, p := range allPhases {
			if ch.state.Phases[p] == state.StatusCompleted {
				completed++
			}
		}

		pct := 0
		if total > 0 {
			pct = completed * 100 / total
		}

		status := h.cachedVerifyStatus(ch.dir)
		if status == "ok" && ch.state.IsStale(defaultLookback) {
			status = "warn"
		}

		pipelines = append(pipelines, PipelineData{
			Name:         ch.state.Name,
			CurrentPhase: string(ch.state.CurrentPhase),
			Completed:    completed,
			Total:        total,
			Tokens:       tokenMap[ch.state.Name],
			ProgressPct:  pct,
			Status:       status,
		})
	}

	return pipelines
}

// verifyReportReadLimit caps how much of verify-report.md we read.
// The "**Status:** FAILED" marker appears in the first few hundred bytes.
const verifyReportReadLimit = 4096

// verifyBufPool reuses 4 KiB read buffers across cachedVerifyStatus calls.
var verifyBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, verifyReportReadLimit)
		return &buf
	},
}

// cachedVerifyStatus returns "error" if verify-report.md contains FAILED,
// "ok" otherwise. Results are cached by file modification time to avoid
// re-reading the file on every poll tick. Uses open+fstat to avoid TOCTOU
// races between stat and read.
func (h *Hub) cachedVerifyStatus(changeDir string) string {
	reportPath := filepath.Join(changeDir, "verify-report.md")

	// Open the file once — get handle for both stat and read atomically.
	f, err := os.Open(reportPath)
	if err != nil {
		return "ok"
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "ok"
	}

	h.verifyCacheMu.RLock()
	entry, found := h.verifyCache[changeDir]
	h.verifyCacheMu.RUnlock()

	if found && entry.modTime.Equal(info.ModTime()) {
		return entry.status
	}

	// File changed or not cached — read capped prefix from open handle.
	status := "ok"
	bufp := verifyBufPool.Get().(*[]byte)
	n, _ := f.Read(*bufp)
	if n > 0 && bytes.Contains((*bufp)[:n], []byte("**Status:** FAILED")) {
		status = "error"
	}
	verifyBufPool.Put(bufp)

	h.verifyCacheMu.Lock()
	h.verifyCache[changeDir] = verifyCacheEntry{modTime: info.ModTime(), status: status}
	h.verifyCacheMu.Unlock()

	return status
}

// pruneVerifyCache removes entries for change directories no longer active.
// Called once per poll to bound cache size to active changes only.
func (h *Hub) pruneVerifyCache(activeChangeDirs map[string]struct{}) {
	h.verifyCacheMu.Lock()
	defer h.verifyCacheMu.Unlock()
	for dir := range h.verifyCache {
		if _, active := activeChangeDirs[dir]; !active {
			delete(h.verifyCache, dir)
		}
	}
}

// buildErrors fetches recent errors from the store.
func (h *Hub) buildErrors(ctx context.Context) []ErrorData {
	rows, err := h.metrics.RecentErrors(ctx, 20)
	if err != nil {
		return []ErrorData{}
	}

	data := make([]ErrorData, 0, len(rows))
	for _, r := range rows {
		data = append(data, ErrorData{
			Timestamp:   r.Timestamp[:min(len(r.Timestamp), 19)],
			CommandName: r.CommandName,
			ExitCode:    r.ExitCode,
			Change:      r.Change,
			Fingerprint: r.Fingerprint[:min(len(r.Fingerprint), 8)],
			FirstLine:   r.FirstLine,
		})
	}

	return data
}

// buildHeatmapFromChanges builds the phase status grid from pre-loaded changes.
func buildHeatmapFromChanges(changes []changeSnapshot) []PhaseStatusRow {
	allPhases := state.AllPhases()
	grid := make([]PhaseStatusRow, 0, len(changes)*len(allPhases))

	for _, ch := range changes {
		for _, p := range allPhases {
			status := string(ch.state.Phases[p])
			if status == "" {
				status = string(state.StatusPending)
			}
			grid = append(grid, PhaseStatusRow{
				Change: ch.state.Name,
				Phase:  string(p),
				Status: status,
			})
		}
	}

	return grid
}

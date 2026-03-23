package context

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/fsutil"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/phase"
)

// cacheVersion is bumped when assembler output format changes.
// Any cache entry written with a different version is treated as stale.
// Bump this when: adding new sections to assemblers, changing section
// labels, modifying summary format, or changing what artifacts are loaded.
// Bumped to 7: embedded skills fallback; hash now always includes skill bytes.
const cacheVersion = 7

// cacheDir returns the cache directory for a change.
func cacheDir(changeDir string) string {
	return filepath.Join(changeDir, ".cache")
}

// contextCachePath returns the path to the cached context for a phase.
func contextCachePath(changeDir, phase string) string {
	return filepath.Join(cacheDir(changeDir), phase+".ctx")
}

// hashCachePath returns the path to the hash file for a phase.
func hashCachePath(changeDir, phase string) string {
	return filepath.Join(cacheDir(changeDir), phase+".hash")
}

// phaseCacheInputs returns CacheInputs for a phase from the registry.
func phaseCacheInputs(name string) []string {
	desc, ok := phase.DefaultRegistry.Get(name)
	if !ok {
		return nil
	}
	return desc.CacheInputs
}

// phaseCacheTTL returns CacheTTL for a phase from the registry.
func phaseCacheTTL(name string) time.Duration {
	desc, ok := phase.DefaultRegistry.Get(name)
	if !ok {
		return 0
	}
	return desc.CacheTTL
}

// readSkillBytes reads skill content for hashing.
// phaseName is the bare name, e.g. "explore" (no sdd- prefix).
func readSkillBytes(skillsPath, phaseName string) []byte {
	data, _ := loadSkill(skillsPath, "sdd-"+phaseName)
	return data
}

// inputHash computes a SHA256 hash of all input artifacts + SKILL.md for a phase.
// Includes cacheVersion so format changes auto-invalidate.
// Includes SKILL.md so skill edits invalidate the cache (essentially an ETag pattern).
func inputHash(changeDir string, inputs []string, skillsPath, phaseName string) string {
	h := sha256.New()
	var intBuf [32]byte // scratch buffer for strconv.AppendInt — avoids fmt allocations

	// Version prefix — constant string, no allocation.
	io.WriteString(h, "v7:") //nolint:errcheck // hash.Hash.Write never errors; matches cacheVersion

	// Hash the SKILL.md for this phase — fixes correctness bug where
	// editing a skill wouldn't invalidate cached context.
	if phaseName != "" {
		if data := readSkillBytes(skillsPath, phaseName); data != nil {
			io.WriteString(h, "skill:")                                       //nolint:errcheck
			h.Write(strconv.AppendInt(intBuf[:0], int64(len(data)), 10))     //nolint:errcheck
			io.WriteString(h, ":")                                            //nolint:errcheck
			h.Write(data)                                                     //nolint:errcheck
		}
	}

	// CacheInputs are pre-sorted at phase registration time; iterate directly.
	for _, name := range inputs {
		if name == "specs/" {
			hashSpecsDir(h, changeDir)
			continue
		}
		data, err := os.ReadFile(filepath.Join(changeDir, name))
		if err != nil {
			continue
		}
		io.WriteString(h, name)                                          //nolint:errcheck
		io.WriteString(h, ":")                                           //nolint:errcheck
		h.Write(strconv.AppendInt(intBuf[:0], int64(len(data)), 10))    //nolint:errcheck
		io.WriteString(h, ":")                                           //nolint:errcheck
		h.Write(data)                                                    //nolint:errcheck
	}

	return hex.EncodeToString(h.Sum(nil))
}

// hashSpecsDir hashes all .md files in specs/ into the provided hasher.
func hashSpecsDir(h io.Writer, changeDir string) {
	specsDir := filepath.Join(changeDir, "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return
	}
	var intBuf [32]byte
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(specsDir, e.Name()))
		if err != nil {
			continue
		}
		io.WriteString(h, "specs/")                                      //nolint:errcheck
		io.WriteString(h, e.Name())                                      //nolint:errcheck
		io.WriteString(h, ":")                                           //nolint:errcheck
		h.Write(strconv.AppendInt(intBuf[:0], int64(len(data)), 10))    //nolint:errcheck
		io.WriteString(h, ":")                                           //nolint:errcheck
		h.Write(data)                                                    //nolint:errcheck
	}
}

// tryCachedContext checks if a cached context exists, its input hash
// matches the current artifacts, and the TTL hasn't expired.
// Hash file format: "{hex_hash}|{unix_seconds}"
// Legacy files without "|" produce a cache miss (silent upgrade).
func tryCachedContext(changeDir, phaseName, skillsPath string) ([]byte, bool) {
	inputs := phaseCacheInputs(phaseName)

	raw, err := os.ReadFile(hashCachePath(changeDir, phaseName))
	if err != nil {
		return nil, false
	}

	// Parse "hash|timestamp" format.
	hashB, tsB, ok := bytes.Cut(bytes.TrimSpace(raw), []byte("|"))
	if !ok {
		return nil, false // legacy format without timestamp → miss
	}
	storedHash := string(hashB)
	tsStr := string(tsB)

	// Check content hash (includes SKILL.md).
	currentHash := inputHash(changeDir, inputs, skillsPath, phaseName)
	if storedHash != currentHash {
		return nil, false
	}

	// Check TTL.
	if ttl := phaseCacheTTL(phaseName); ttl > 0 {
		ts := mustParseInt64(tsStr)
		age := time.Since(time.Unix(ts, 0))
		if age > ttl {
			return nil, false // expired
		}
	}

	cached, err := os.ReadFile(contextCachePath(changeDir, phaseName))
	if err != nil {
		return nil, false
	}

	return cached, true
}

func mustParseInt64(s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0 // epoch → forces TTL miss (safe fallback)
	}
	return v
}

// saveContextCache stores the assembled context and its input hash with timestamp.
// Format: "{hash}|{unix_seconds}"
func saveContextCache(changeDir, phaseName, skillsPath string, content []byte) error {
	dir := cacheDir(changeDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	inputs := phaseCacheInputs(phaseName)
	hash := inputHash(changeDir, inputs, skillsPath, phaseName)
	hashWithTS := fmt.Sprintf("%s|%d", hash, time.Now().Unix())

	if err := fsutil.AtomicWrite(hashCachePath(changeDir, phaseName), []byte(hashWithTS)); err != nil {
		return err
	}
	return fsutil.AtomicWrite(contextCachePath(changeDir, phaseName), content)
}

// estimateTokens provides a rough token estimate from byte count.
// ~4 bytes per token for English/code mixed content.
func estimateTokens(size int) int {
	return size / 4
}

// maxContextBytes is the default size limit for assembled context.
// ~100KB ≈ 25K tokens — keeps sub-agents within context window.
const maxContextBytes = 100 * 1024

// contextMetrics holds measurements from a context assembly operation.
type contextMetrics struct {
	Phase      string
	Bytes      int
	Tokens     int
	Cached     bool
	DurationMs int64
}

// metricsFromPayload converts a PhaseAssembledPayload to contextMetrics.
func metricsFromPayload(p events.PhaseAssembledPayload) *contextMetrics {
	return &contextMetrics{
		Phase:      p.Phase,
		Bytes:      p.Bytes,
		Tokens:     p.Tokens,
		Cached:     p.Cached,
		DurationMs: p.DurationMs,
	}
}

// writeMetrics logs context metrics via slog.
func writeMetrics(m *contextMetrics, verbosity int) {
	if verbosity < 0 {
		return
	}
	source := "assembled"
	if m.Cached {
		source = "cached"
	}
	slog.Info("phase assembled",
		"phase", m.Phase,
		"bytes", m.Bytes,
		"tokens_k", m.Tokens/1000,
		"duration_ms", m.DurationMs,
		"source", source,
	)
}

// PipelineMetrics tracks cumulative token usage across all phases of a change.
// Exported for use by sdd health command.
type PipelineMetrics struct {
	Version     int                     `json:"version"`
	Phases      map[string]PhaseMetrics `json:"phases"`
	TotalBytes  int                     `json:"total_bytes"`
	TotalTokens int                     `json:"total_tokens"`
	CacheHits   int                     `json:"cache_hits"`
	CacheMisses int                     `json:"cache_misses"`
}

// PhaseMetrics holds per-phase metrics. Exported for sdd health.
type PhaseMetrics struct {
	Bytes      int   `json:"bytes"`
	Tokens     int   `json:"tokens"`
	Cached     bool  `json:"cached"`
	DurationMs int64 `json:"duration_ms"`
}

// metricsPath returns the path to the cumulative metrics file.
func metricsPath(changeDir string) string {
	return filepath.Join(cacheDir(changeDir), "metrics.json")
}

// recordMetrics appends a phase's metrics to the cumulative tracker.
// Best-effort — failures are silently ignored.
func recordMetrics(changeDir string, m *contextMetrics) {
	pm := LoadPipelineMetrics(changeDir)

	pm.Phases[m.Phase] = PhaseMetrics{
		Bytes:      m.Bytes,
		Tokens:     m.Tokens,
		Cached:     m.Cached,
		DurationMs: m.DurationMs,
	}

	// Recompute totals.
	pm.TotalBytes = 0
	pm.TotalTokens = 0
	pm.CacheHits = 0
	pm.CacheMisses = 0
	for _, p := range pm.Phases {
		pm.TotalBytes += p.Bytes
		pm.TotalTokens += p.Tokens
		if p.Cached {
			pm.CacheHits++
		} else {
			pm.CacheMisses++
		}
	}

	data, err := json.MarshalIndent(pm, "", "  ")
	if err != nil {
		return
	}

	_ = os.MkdirAll(cacheDir(changeDir), 0o755)          // best-effort dir creation
	_ = fsutil.AtomicWrite(metricsPath(changeDir), data) // best-effort metrics persistence
}

// LoadPipelineMetrics reads the cumulative metrics file for a change.
// Exported for use by sdd health command.
func LoadPipelineMetrics(changeDir string) *PipelineMetrics {
	data, err := os.ReadFile(metricsPath(changeDir))
	if err != nil {
		return &PipelineMetrics{
			Version: cacheVersion,
			Phases:  make(map[string]PhaseMetrics),
		}
	}

	var pm PipelineMetrics
	if err := json.Unmarshal(data, &pm); err != nil || pm.Version != cacheVersion {
		return &PipelineMetrics{
			Version: cacheVersion,
			Phases:  make(map[string]PhaseMetrics),
		}
	}

	if pm.Phases == nil {
		pm.Phases = make(map[string]PhaseMetrics)
	}
	return &pm
}

func formatBytes(b int) string {
	if b < 1024 {
		return fmt.Sprintf("%dB", b)
	}
	return fmt.Sprintf("%dKB", b/1024)
}

// CheckCacheIntegrity counts stale cache entries in a change directory.
// Returns the number of .hash files whose stored hash no longer matches
// the current input hash (content drift).
func CheckCacheIntegrity(changeDir, skillsPath string) (int, error) {
	stale := 0
	hashFiles, err := filepath.Glob(filepath.Join(cacheDir(changeDir), "*.hash"))
	if err != nil || len(hashFiles) == 0 {
		return 0, nil
	}
	for _, hf := range hashFiles {
		phase := strings.TrimSuffix(filepath.Base(hf), ".hash")
		raw, err := os.ReadFile(hf)
		if err != nil {
			continue
		}
		storedHashB, _, ok := bytes.Cut(bytes.TrimSpace(raw), []byte("|"))
		storedHash := string(storedHashB)
		if !ok {
			stale++
			continue
		}
		inputs := phaseCacheInputs(phase)
		current := inputHash(changeDir, inputs, skillsPath, phase)
		if storedHash != current {
			stale++
		}
	}
	return stale, nil
}

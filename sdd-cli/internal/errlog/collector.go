// Package errlog persists verify failures to a global error log
// so sdd doctor and sdd errors can surface recurring patterns.
package errlog

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/fsutil"
)

const (
	maxEntries = 100
	logVersion = 1
)

// ErrorEntry is a single recorded verify failure.
type ErrorEntry struct {
	Timestamp   string   `json:"timestamp"`
	Change      string   `json:"change"`
	CommandName string   `json:"command_name"`
	Command     string   `json:"command"`
	ExitCode    int      `json:"exit_code"`
	ErrorLines  []string `json:"error_lines"`
	Fingerprint string   `json:"fingerprint"`
}

// ErrorLog is the persistent error store.
type ErrorLog struct {
	Version int          `json:"version"`
	Entries []ErrorEntry `json:"entries"`
}

// Fingerprint computes a stable 16-hex-char hash from command + first error line.
func Fingerprint(command string, errorLines []string) string {
	h := sha256.New()
	io.WriteString(h, command) //nolint:errcheck // hash.Hash.Write never errors
	h.Write([]byte{0})         //nolint:errcheck
	if len(errorLines) > 0 {
		io.WriteString(h, errorLines[0]) //nolint:errcheck
	}
	return fmt.Sprintf("%x", h.Sum(nil)[:8])
}

// LogPath returns the path to the global error log.
func LogPath(cwd string) string {
	return filepath.Join(cwd, "openspec", ".cache", "errors.json")
}

// Load reads the error log from disk. Returns empty log on any error.
func Load(cwd string) *ErrorLog {
	data, err := os.ReadFile(LogPath(cwd))
	if err != nil {
		return &ErrorLog{Version: logVersion}
	}
	var log ErrorLog
	if json.Unmarshal(data, &log) != nil || log.Version != logVersion {
		return &ErrorLog{Version: logVersion}
	}
	return &log
}

// Record appends an entry, evicts oldest beyond maxEntries, and writes atomically.
// Best-effort: failures are silently ignored.
func Record(cwd string, entry ErrorEntry) {
	log := Load(cwd)
	log.Entries = append(log.Entries, entry)
	if len(log.Entries) > maxEntries {
		log.Entries = log.Entries[len(log.Entries)-maxEntries:]
	}

	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return
	}

	path := LogPath(cwd)
	_ = os.MkdirAll(filepath.Dir(path), 0o755) // best-effort dir creation
	_ = fsutil.AtomicWrite(path, data)          // best-effort error log persistence
}

// RecurringFingerprints returns fingerprints seen >= threshold times with their counts.
func (l *ErrorLog) RecurringFingerprints(threshold int) map[string]int {
	counts := make(map[string]int)
	for _, e := range l.Entries {
		counts[e.Fingerprint]++
	}
	result := make(map[string]int, len(counts))
	for fp, n := range counts {
		if n >= threshold {
			result[fp] = n
		}
	}
	return result
}

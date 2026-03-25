package artifacts

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

// WritePending writes content to the pending artifact location for a phase.
// Directory-backed phases write a default file within their pending directory.
func WritePending(changeDir string, phase state.Phase, data []byte) error {
	path := PendingPath(changeDir, phase)
	if isDirectoryArtifact(phase) {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create pending directory: %w", err)
		}
		path = filepath.Join(path, PendingFileName(phase))
	} else {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create .pending directory: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write pending %s: %w", filepath.Base(path), err)
	}
	return nil
}

// PendingPath returns the path to the pending artifact for a phase.
func PendingPath(changeDir string, phase state.Phase) string {
	return filepath.Join(changeDir, ".pending", pendingRelativePath(phase))
}

// PendingExists reports whether a pending artifact exists for the given phase.
func PendingExists(changeDir string, phase state.Phase) bool {
	path := PendingPath(changeDir, phase)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return true
	}
	files, err := collectRegularFiles(path)
	return err == nil && len(files) > 0
}

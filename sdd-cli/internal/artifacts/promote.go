package artifacts

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

var ErrNoPending = errors.New("no pending artifact")

// Promote moves .pending/{phase}.md to its final location in the change directory.
// For spec phase, the pending file is moved into the specs/ directory.
// If force is false, content is validated against phase-specific rules before promotion.
func Promote(changeDir string, phase state.Phase, force bool) (string, error) {
	src := PendingPath(changeDir, phase)

	if !PendingExists(changeDir, phase) {
		return "", fmt.Errorf("%w: %s (expected at %s)", ErrNoPending, phase, src)
	}

	finalName, ok := ArtifactFileName(phase)
	if !ok {
		return "", fmt.Errorf("no artifact mapping for phase: %s", phase)
	}

	if !force {
		if err := ValidatePending(phase, src); err != nil {
			return "", err
		}
	}

	dst := filepath.Join(changeDir, finalName)
	if isDirectoryArtifact(phase) {
		if err := promoteDir(src, dst); err != nil {
			return "", err
		}
		return dst, nil
	}

	// Read, validate, write to destination, remove source (cross-device safe).
	data, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("read pending: %w", err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return "", fmt.Errorf("write promoted: %w", err)
	}
	if err := os.Remove(src); err != nil {
		slog.Warn("promote: failed to remove pending artifact after promotion", "path", src, "err", err)
		return dst, nil //nolint:nilerr // non-fatal: artifact is promoted; source cleanup failure is not an error
	}

	return dst, nil
}

func promoteDir(src, dst string) error {
	// Atomic directory promotion: copy to a temp sibling, then rename.
	// This avoids a window where dst is deleted but copy hasn't finished.
	tmpDir := dst + ".tmp"
	if err := os.RemoveAll(tmpDir); err != nil {
		return fmt.Errorf("clean temp directory: %w", err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}

	if err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(tmpDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	}); err != nil {
		os.RemoveAll(tmpDir) //nolint:errcheck // best-effort cleanup on copy failure
		return fmt.Errorf("copy promoted directory: %w", err)
	}

	// Swap: remove old dst, rename tmp → dst.
	if err := os.RemoveAll(dst); err != nil {
		os.RemoveAll(tmpDir) //nolint:errcheck // best-effort cleanup
		return fmt.Errorf("remove old promoted directory: %w", err)
	}
	if err := os.Rename(tmpDir, dst); err != nil {
		return fmt.Errorf("rename temp to promoted: %w", err)
	}

	if err := os.RemoveAll(src); err != nil {
		slog.Warn("promote: failed to remove pending artifact after promotion", "path", src, "err", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close destination file: %w", err)
	}
	return nil
}

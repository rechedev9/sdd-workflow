package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWrite writes data to path via a temp file + rename.
// Safe on POSIX: rename is atomic within the same filesystem.
func AtomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // best-effort cleanup
		return fmt.Errorf("rename %s: %w", filepath.Base(path), err)
	}
	return nil
}

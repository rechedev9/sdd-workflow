package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWrite writes data to path via a temp file + rename.
// Safe on POSIX: rename is atomic within the same filesystem.
// Uses os.CreateTemp for unique temp files — safe under concurrent calls.
func AtomicWrite(path string, data []byte) error {
	dir, base := filepath.Dir(path), filepath.Base(path)
	f, err := os.CreateTemp(dir, base+".tmp.*")
	if err != nil {
		return fmt.Errorf("create temp for %s: %w", base, err)
	}
	tmp := f.Name()

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp) // best-effort cleanup
		return fmt.Errorf("write %s: %w", base, err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close temp for %s: %w", base, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // best-effort cleanup
		return fmt.Errorf("rename %s: %w", base, err)
	}
	return nil
}

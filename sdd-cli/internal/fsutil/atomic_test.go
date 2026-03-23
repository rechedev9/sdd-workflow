package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWrite(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "test.txt")
	if err := AtomicWrite(path, []byte("hello")); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestAtomicWrite_NoTmpLeftOnSuccess(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "test.txt")
	AtomicWrite(path, []byte("data"))
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("tmp file still exists after successful write")
	}
}

func TestAtomicWrite_Overwrite(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "test.txt")
	if err := AtomicWrite(path, []byte("first")); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := AtomicWrite(path, []byte("second")); err != nil {
		t.Fatalf("second write: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "second" {
		t.Errorf("got %q, want %q", got, "second")
	}
}

func TestAtomicWrite_MissingDir(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nonexistent", "test.txt")
	if err := AtomicWrite(path, []byte("data")); err == nil {
		t.Error("expected error writing to missing dir, got nil")
	}
}

func TestAtomicWrite_RenameFailsDirAtDest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create a directory at the destination path so os.Rename fails.
	dest := filepath.Join(dir, "output.txt")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := AtomicWrite(dest, []byte("data")); err == nil {
		t.Error("expected error when destination is a directory, got nil")
	}
}

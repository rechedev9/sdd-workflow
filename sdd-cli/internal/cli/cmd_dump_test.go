package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunDump_HappyPath(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "dump-feat", "test dump")
	writeConfig(t, root, "version: 0\nproject_name: test\n")

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runDump([]string{"dump-feat"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runDump: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nstdout: %s", err, stdout.String())
	}
	if out["command"] != "dump" {
		t.Errorf("command = %v, want dump", out["command"])
	}
	if out["change"] != "dump-feat" {
		t.Errorf("change = %v, want dump-feat", out["change"])
	}
	// Artifacts and pending should be JSON arrays (not null).
	if arts, ok := out["artifacts"].([]interface{}); !ok || arts == nil {
		t.Errorf("artifacts should be a JSON array, got %T: %v", out["artifacts"], out["artifacts"])
	}
	// Cache keys should be a JSON object.
	if _, ok := out["cache_keys"].(map[string]interface{}); !ok {
		t.Errorf("cache_keys should be a JSON object, got %T", out["cache_keys"])
	}
}

func TestRunDump_WithCacheKeys(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "dump-cache", "test cache keys")
	writeConfig(t, root, "version: 0\nproject_name: test\n")

	// Plant a fake .hash file in .cache/.
	changeDir := filepath.Join(root, "openspec", "changes", "dump-cache")
	cacheDir := filepath.Join(changeDir, ".cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "explore.hash"), []byte("  abc123  \n"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runDump([]string{"dump-cache"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runDump: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	keys, ok := out["cache_keys"].(map[string]interface{})
	if !ok {
		t.Fatalf("cache_keys should be object, got %T", out["cache_keys"])
	}
	if keys["explore"] != "abc123" {
		t.Errorf("cache_keys[explore] = %v, want abc123", keys["explore"])
	}
}

func TestRunDump_StatusAndPendingArray(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "dump-status", "status check")
	writeConfig(t, root, "version: 0\nproject_name: test\n")

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	if err := runDump([]string{"dump-status"}, &stdout, &stderr); err != nil {
		t.Fatalf("runDump: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out["status"] != "success" {
		t.Errorf("status = %v, want success", out["status"])
	}
	// pending must be a JSON array (not null) even when .pending/ is absent.
	if pending, ok := out["pending"].([]interface{}); !ok || pending == nil {
		t.Errorf("pending should be a JSON array, got %T: %v", out["pending"], out["pending"])
	}
}

func TestRunDump_CacheHashWithPipe(t *testing.T) {
	// Uses Chdir — must not be parallel.
	// Covers the bytes.Cut pipe-format branch: "{hash}|{unix_seconds}".
	root := setupChange(t, "dump-pipe", "test pipe hash")
	writeConfig(t, root, "version: 0\nproject_name: test\n")

	changeDir := filepath.Join(root, "openspec", "changes", "dump-pipe")
	cacheDir := filepath.Join(changeDir, ".cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write hash in "{hash}|{unix_seconds}" format.
	if err := os.WriteFile(filepath.Join(cacheDir, "design.hash"), []byte("deadbeef|1700000000"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	if err := runDump([]string{"dump-pipe"}, &stdout, &stderr); err != nil {
		t.Fatalf("runDump: %v\nstderr: %s", err, stderr.String())
	}

	var out map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	keys, ok := out["cache_keys"].(map[string]interface{})
	if !ok {
		t.Fatalf("cache_keys should be object, got %T", out["cache_keys"])
	}
	if keys["design"] != "deadbeef" {
		t.Errorf("cache_keys[design] = %v, want deadbeef", keys["design"])
	}
}

func TestRunDump_UnknownFlag(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	err := runDump([]string{"some-change", "--bad-flag"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunDump_NoConfig(t *testing.T) {
	// Uses Chdir — must not be parallel.
	root := setupChange(t, "dump-nocfg", "no config")
	// Deliberately no config.yaml written.

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	var stdout, stderr bytes.Buffer
	err := runDump([]string{"dump-nocfg"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when config.yaml missing")
	}
}

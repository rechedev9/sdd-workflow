package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func TestRunWatch_NoArgs(t *testing.T) {
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	err := runWatch(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
}

func TestRunWatch_UnknownFlag(t *testing.T) {
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	err := runWatch([]string{"foo", "--invalid-flag"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("error = %q, want to contain 'unknown flag'", err.Error())
	}
}

func TestRunWatch_DebounceZero(t *testing.T) {
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	err := runWatch([]string{"foo", "--debounce", "0"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for debounce 0")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
	if !strings.Contains(err.Error(), "debounce") {
		t.Errorf("error = %q, want to contain 'debounce'", err.Error())
	}
}

func TestRunWatch_DebounceNonNumeric(t *testing.T) {
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	err := runWatch([]string{"foo", "--debounce", "abc"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for non-numeric debounce")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code = %d, want 2", ExitCode(err))
	}
	if !strings.Contains(err.Error(), "debounce") {
		t.Errorf("error = %q, want to contain 'debounce'", err.Error())
	}
}

func TestRunWatch_NonexistentChange(t *testing.T) {
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	err := runWatch([]string{"nonexistent"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for nonexistent change")
	}
	if ExitCode(err) != 1 {
		t.Errorf("exit code = %d, want 1", ExitCode(err))
	}
	if !strings.Contains(stderr.String(), `"command":"watch"`) {
		t.Errorf("stderr = %q, want JSON with command:watch", stderr.String())
	}
}

func TestRunWatch_StartupJSON(t *testing.T) {
	// This test sends SIGINT to stop the watcher; must not be parallel.
	root := setupChange(t, "foo", "test feature")
	writeConfig(t, root, "version: 1\n")

	changeDir := filepath.Join(root, "openspec", "changes", "foo")
	if _, err := state.Load(filepath.Join(changeDir, "state.json")); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(root)

	// Use io.Pipe so reading the startup JSON is race-free.
	pr, pw := io.Pipe()
	var stderr bytes.Buffer

	done := make(chan error, 1)
	go func() {
		done <- runWatch([]string{"foo"}, pw, &stderr)
		pw.Close()
	}()

	// Read the startup JSON from the pipe.
	dec := json.NewDecoder(pr)
	var startupMsg struct {
		Command string `json:"command"`
		Status  string `json:"status"`
		Change  string `json:"change"`
		Phase   string `json:"phase"`
		Dir     string `json:"dir"`
	}
	if err := dec.Decode(&startupMsg); err != nil {
		t.Fatalf("failed to decode startup JSON: %v", err)
	}

	// Now that we've read the startup JSON, send SIGINT to stop the watcher.
	// signal.NotifyContext in runWatch intercepts SIGINT and cancels ctx.
	time.Sleep(20 * time.Millisecond) // ensure signal.NotifyContext is registered
	p, _ := os.FindProcess(os.Getpid())
	p.Signal(syscall.SIGINT) //nolint:errcheck

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("runWatch did not return after SIGINT")
	}
	pr.Close()

	if startupMsg.Command != "watch" {
		t.Errorf("command = %q, want watch", startupMsg.Command)
	}
	if startupMsg.Status != "watching" {
		t.Errorf("status = %q, want watching", startupMsg.Status)
	}
	if startupMsg.Change != "foo" {
		t.Errorf("change = %q, want foo", startupMsg.Change)
	}
	if startupMsg.Phase != "explore" {
		t.Errorf("phase = %q, want explore", startupMsg.Phase)
	}
	if !filepath.IsAbs(startupMsg.Dir) {
		t.Errorf("dir = %q, want absolute path", startupMsg.Dir)
	}
}

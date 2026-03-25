package watch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
)

// runWatch starts Run in a goroutine and returns a done channel.
// Cancel ctx to stop; read from done to wait for Run to finish.
func runWatch(ctx context.Context, opts Options) <-chan error {
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, opts)
	}()
	// Allow watcher goroutine to start.
	time.Sleep(50 * time.Millisecond)
	return done
}

// --- Unit tests for helpers ---

func TestShouldFilter_CachePath(t *testing.T) {
	t.Parallel()
	if !shouldFilter("/foo/.cache/bar.ctx", fsnotify.Write) {
		t.Error("expected .cache path to be filtered")
	}
}

func TestShouldFilter_PendingPath(t *testing.T) {
	t.Parallel()
	if !shouldFilter("/foo/.pending/spec.md", fsnotify.Write) {
		t.Error("expected .pending path to be filtered")
	}
}

func TestShouldFilter_ArtifactPath(t *testing.T) {
	t.Parallel()
	if shouldFilter("/foo/proposal.md", fsnotify.Write) {
		t.Error("artifact path should not be filtered")
	}
}

func TestShouldFilter_ChmodOnly(t *testing.T) {
	t.Parallel()
	if !shouldFilter("/foo/proposal.md", fsnotify.Chmod) {
		t.Error("Chmod-only event should be filtered")
	}
}

func TestShouldFilter_WriteAndChmod(t *testing.T) {
	t.Parallel()
	if shouldFilter("/foo/proposal.md", fsnotify.Write|fsnotify.Chmod) {
		t.Error("Write|Chmod should NOT be filtered (not Chmod-only)")
	}
}

func TestAddRecursive_EnumeratesSubdirs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "specs", "watch-cli"), 0o755)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0o755)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	count, err := addRecursive(w, dir)
	if err != nil {
		t.Fatalf("addRecursive error: %v", err)
	}
	// dir + specs + specs/watch-cli + subdir = 4
	if count != 4 {
		t.Errorf("count = %d, want 4", count)
	}
}

func TestAddRecursive_SkipsCacheAndPending(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".cache", "deep"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".pending"), 0o755)
	os.MkdirAll(filepath.Join(dir, "specs"), 0o755)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	count, err := addRecursive(w, dir)
	if err != nil {
		t.Fatalf("addRecursive error: %v", err)
	}
	// dir + specs = 2 (skips .cache, .cache/deep, .pending)
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

// --- Integration tests for Run ---

func TestRun_BlocksUntilContextCancelled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	done := runWatch(ctx, Options{
		ChangeDir:  dir,
		Debounce:   20 * time.Millisecond,
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Reassemble: func(_ context.Context, _ io.Writer) error { return nil },
	})

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func TestRun_SingleFileWriteTriggersOneReassembly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	var count atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	done := runWatch(ctx, Options{
		ChangeDir: dir,
		Debounce:  20 * time.Millisecond,
		Stdout:    &bytes.Buffer{},
		Stderr:    &bytes.Buffer{},
		Reassemble: func(_ context.Context, _ io.Writer) error {
			count.Add(1)
			return nil
		},
	})

	os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("content"), 0o644)
	time.Sleep(200 * time.Millisecond)

	cancel()
	<-done

	got := count.Load()
	if got != 1 {
		t.Errorf("reassembly count = %d, want 1", got)
	}
}

func TestRun_RapidWritesCoalesceIntoOne(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	var count atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	done := runWatch(ctx, Options{
		ChangeDir: dir,
		Debounce:  50 * time.Millisecond,
		Stdout:    &bytes.Buffer{},
		Stderr:    &bytes.Buffer{},
		Reassemble: func(_ context.Context, _ io.Writer) error {
			count.Add(1)
			return nil
		},
	})

	// Write 5 files rapidly within 20ms of each other.
	for i := range 5 {
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("file%d.md", i)), []byte("data"), 0o644)
		time.Sleep(4 * time.Millisecond)
	}

	time.Sleep(300 * time.Millisecond)
	cancel()
	<-done

	got := count.Load()
	if got != 1 {
		t.Errorf("reassembly count = %d, want 1 (coalesced)", got)
	}
}

func TestRun_WidelySpacedWritesTriggerSeparate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	var count atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	done := runWatch(ctx, Options{
		ChangeDir: dir,
		Debounce:  30 * time.Millisecond,
		Stdout:    &bytes.Buffer{},
		Stderr:    &bytes.Buffer{},
		Reassemble: func(_ context.Context, _ io.Writer) error {
			count.Add(1)
			return nil
		},
	})

	// First write.
	os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("v1"), 0o644)
	time.Sleep(200 * time.Millisecond)

	// Second write.
	os.WriteFile(filepath.Join(dir, "design.md"), []byte("v1"), 0o644)
	time.Sleep(200 * time.Millisecond)

	cancel()
	<-done

	got := count.Load()
	if got != 2 {
		t.Errorf("reassembly count = %d, want 2", got)
	}
}

func TestRun_DynamicSubdirWatched(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	var count atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	done := runWatch(ctx, Options{
		ChangeDir: dir,
		Debounce:  20 * time.Millisecond,
		Stdout:    &bytes.Buffer{},
		Stderr:    &bytes.Buffer{},
		Reassemble: func(_ context.Context, _ io.Writer) error {
			count.Add(1)
			return nil
		},
	})

	// Create a new subdirectory.
	newDir := filepath.Join(dir, "newdir")
	os.MkdirAll(newDir, 0o755)
	// Wait for the directory creation event to be processed and watch to be added.
	time.Sleep(100 * time.Millisecond)

	// Reset count (directory creation may have triggered one reassembly).
	count.Store(0)

	// Write a file inside the new subdirectory.
	os.WriteFile(filepath.Join(newDir, "test.md"), []byte("data"), 0o644)
	time.Sleep(200 * time.Millisecond)

	cancel()
	<-done

	got := count.Load()
	if got < 1 {
		t.Errorf("reassembly count = %d, want >= 1 (dynamic subdir)", got)
	}
}

func TestRun_ReassembleErrorLoggedWatchContinues(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	var count atomic.Int32
	var callNum atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	done := runWatch(ctx, Options{
		ChangeDir: dir,
		Debounce:  20 * time.Millisecond,
		Stdout:    &bytes.Buffer{},
		Stderr:    &bytes.Buffer{},
		Reassemble: func(_ context.Context, _ io.Writer) error {
			n := callNum.Add(1)
			count.Add(1)
			if n == 1 {
				return fmt.Errorf("simulated error")
			}
			return nil
		},
	})

	// First write -- triggers error in callback.
	os.WriteFile(filepath.Join(dir, "file1.md"), []byte("v1"), 0o644)
	time.Sleep(200 * time.Millisecond)

	// Second write -- should still trigger (watch continues after error).
	os.WriteFile(filepath.Join(dir, "file2.md"), []byte("v2"), 0o644)
	time.Sleep(200 * time.Millisecond)

	cancel()
	<-done

	got := count.Load()
	if got < 2 {
		t.Errorf("reassembly count = %d, want >= 2 (watch should continue after error)", got)
	}
}

func TestRun_NoGoroutineLeak(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Give background goroutines time to settle.
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	before := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	done := runWatch(ctx, Options{
		ChangeDir: dir,
		Debounce:  20 * time.Millisecond,
		Stdout:    &bytes.Buffer{},
		Stderr:    &bytes.Buffer{},
		Reassemble: func(_ context.Context, _ io.Writer) error {
			return nil
		},
	})

	// Trigger at least one reassembly.
	os.WriteFile(filepath.Join(dir, "test.md"), []byte("data"), 0o644)
	time.Sleep(200 * time.Millisecond)

	cancel()
	<-done

	// Let goroutines wind down.
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	after := runtime.NumGoroutine()
	if after > before+2 {
		t.Errorf("goroutine leak: before=%d, after=%d (tolerance +2)", before, after)
	}
}

func TestRun_SeparatorOnStderrBetweenReassemblies(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	var stderrBuf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())

	done := runWatch(ctx, Options{
		ChangeDir: dir,
		Debounce:  20 * time.Millisecond,
		Stdout:    &bytes.Buffer{},
		Stderr:    &stderrBuf,
		Reassemble: func(_ context.Context, _ io.Writer) error {
			return nil
		},
	})

	// First write.
	os.WriteFile(filepath.Join(dir, "file1.md"), []byte("v1"), 0o644)
	time.Sleep(200 * time.Millisecond)

	// Second write.
	os.WriteFile(filepath.Join(dir, "file2.md"), []byte("v2"), 0o644)
	time.Sleep(200 * time.Millisecond)

	// Stop watcher and wait for goroutine to exit before reading buffer.
	cancel()
	<-done

	stderr := stderrBuf.String()
	if !strings.Contains(stderr, "--- reassembled at") {
		t.Errorf("stderr = %q, want separator line", stderr)
	}
}

func TestRun_WatchReassembledEventEmitted(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	broker := events.NewBroker()
	var received atomic.Int32
	broker.Subscribe(events.WatchReassembled, func(_ events.Event) {
		received.Add(1)
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := runWatch(ctx, Options{
		ChangeDir:  dir,
		Debounce:   20 * time.Millisecond,
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Broker:     broker,
		ChangeName: "foo",
		Reassemble: func(_ context.Context, _ io.Writer) error {
			// In real usage the CLI callback emits the event.
			// Simulate that here.
			broker.Emit(events.Event{
				Type: events.WatchReassembled,
				Payload: events.WatchReassembledPayload{
					Change:     "foo",
					Phase:      "propose",
					DurationMs: 1,
				},
			})
			return nil
		},
	})

	os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("content"), 0o644)
	time.Sleep(200 * time.Millisecond)

	cancel()
	<-done

	got := received.Load()
	if got < 1 {
		t.Errorf("event received count = %d, want >= 1", got)
	}
}

func TestRun_WatchReassembledEventNotEmittedOnError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	broker := events.NewBroker()
	var received atomic.Int32
	broker.Subscribe(events.WatchReassembled, func(_ events.Event) {
		received.Add(1)
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := runWatch(ctx, Options{
		ChangeDir:  dir,
		Debounce:   20 * time.Millisecond,
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Broker:     broker,
		ChangeName: "foo",
		Reassemble: func(_ context.Context, _ io.Writer) error {
			// Error -- should not emit event.
			return fmt.Errorf("simulated error")
		},
	})

	os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("content"), 0o644)
	time.Sleep(200 * time.Millisecond)

	cancel()
	<-done

	got := received.Load()
	if got != 0 {
		t.Errorf("event received count = %d, want 0 (error should suppress event)", got)
	}
}

func TestWatchPackageDoesNotImportCLI(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("go", "list", "-deps", "./internal/watch/")
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(), "GOCACHE=/tmp/sdd-gocache")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		if strings.Contains(line, "internal/cli") {
			t.Fatalf("watch package imports internal/cli: %s", line)
		}
	}
}

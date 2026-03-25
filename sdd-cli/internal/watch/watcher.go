// Package watch implements a debounced filesystem watcher that monitors
// a change directory and invokes a callback when artifacts change.
// Import constraint: this package MUST NOT import internal/cli.
package watch

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
)

// ReassembleFunc is the callback invoked on each debounce fire.
// The watcher passes the current stdout writer; the callback is
// responsible for re-reading state, resolving phases, and calling
// context.Assemble or context.AssembleConcurrent.
//
// If the callback returns an error, the watcher logs it and continues.
// The callback receives the same context as Run -- it should respect
// cancellation for long-running assemblies.
type ReassembleFunc func(ctx context.Context, stdout io.Writer) error

// Options configures a watch.Run invocation.
// All fields are required unless noted.
type Options struct {
	// ChangeDir is the absolute path to openspec/changes/{name}/.
	ChangeDir string

	// Debounce is the duration to wait after the last filesystem event
	// before invoking the reassembly callback. Must be positive.
	Debounce time.Duration

	// Stdout receives assembled context output on each reassembly.
	Stdout io.Writer

	// Stderr receives log messages, errors, and reassembly separators.
	Stderr io.Writer

	// Reassemble is called on each debounce fire. Must not be nil.
	Reassemble ReassembleFunc

	// Broker emits WatchReassembled events. May be nil (no events emitted).
	Broker *events.Broker

	// ChangeName is used in event payloads. Required if Broker is non-nil.
	ChangeName string
}

// shouldFilter reports whether an fsnotify event should be discarded
// (not trigger debounce). Filters:
//   - paths containing "/.cache/"
//   - paths containing "/.pending/"
//   - Chmod-only events (Op == fsnotify.Chmod with no other bits)
func shouldFilter(eventPath string, op fsnotify.Op) bool {
	if op == fsnotify.Chmod {
		return true
	}
	if strings.Contains(eventPath, string(filepath.Separator)+".cache"+string(filepath.Separator)) ||
		strings.HasSuffix(eventPath, string(filepath.Separator)+".cache") {
		return true
	}
	if strings.Contains(eventPath, string(filepath.Separator)+".pending"+string(filepath.Separator)) ||
		strings.HasSuffix(eventPath, string(filepath.Separator)+".pending") {
		return true
	}
	return false
}

// addRecursive walks dir and adds all subdirectories to w,
// skipping .cache/ and .pending/ directories.
// Returns the count of directories added.
func addRecursive(w *fsnotify.Watcher, dir string) (int, error) {
	count := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if name == ".cache" || name == ".pending" {
			return filepath.SkipDir
		}
		if err := w.Add(path); err != nil {
			return fmt.Errorf("watch add %s: %w", path, err)
		}
		count++
		return nil
	})
	return count, err
}

// Run starts the filesystem watcher and blocks until ctx is cancelled.
// Returns nil on clean shutdown (context cancellation).
// Returns an error only for fatal failures (e.g., fsnotify init failure).
//
// Lifecycle:
//  1. Create fsnotify watcher
//  2. Enumerate all subdirs of opts.ChangeDir (skip .cache/, .pending/)
//  3. Add each to watcher
//  4. Enter event loop goroutine
//  5. Block until ctx.Done()
//  6. Close fsnotify watcher, return nil
func Run(ctx context.Context, opts Options) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}

	if _, err := addRecursive(watcher, opts.ChangeDir); err != nil {
		watcher.Close() //nolint:errcheck // best-effort cleanup on init failure
		return fmt.Errorf("initial watch setup: %w", err)
	}

	var (
		mu              sync.Mutex
		timer           *time.Timer
		reassembleCount int
		done            = make(chan struct{})
	)

	// Initialize a stopped timer.
	timer = time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}

	// Fire function called when debounce timer expires.
	fire := func() {
		mu.Lock()
		reassembleCount++
		count := reassembleCount
		mu.Unlock()

		if count > 1 {
			fmt.Fprintf(opts.Stderr, "--- reassembled at %s ---\n", time.Now().Format("15:04:05"))
		}

		if err := opts.Reassemble(ctx, opts.Stdout); err != nil {
			slog.Error("reassembly failed", "err", err)
			return
		}
	}

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				mu.Lock()
				timer.Stop()
				mu.Unlock()
				watcher.Close() //nolint:errcheck // best-effort cleanup on shutdown
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if shouldFilter(event.Name, event.Op) {
					continue
				}
				// Dynamic recursive watch: add newly created directories.
				if event.Op&fsnotify.Create != 0 {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						watcher.Add(event.Name) //nolint:errcheck
					}
				}
				mu.Lock()
				timer.Reset(opts.Debounce)
				mu.Unlock()

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Error("fsnotify error", "err", err)

			case <-timer.C:
				fire()
			}
		}
	}()

	<-done
	return nil
}

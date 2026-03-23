package context

import (
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/errlog"
)

func TestRegisterSubscribers_NilBroker(t *testing.T) {
	t.Parallel()
	// Should not panic.
	RegisterSubscribers(nil, 0)
}

func TestRegisterSubscribers_PhaseAssembled_RecordsMetrics(t *testing.T) {
	t.Parallel()
	b := events.NewBroker()
	dir := t.TempDir()
	RegisterSubscribers(b, 0)

	b.Emit(events.Event{
		Type: events.PhaseAssembled,
		Payload: events.PhaseAssembledPayload{
			Phase:      "explore",
			Bytes:      1024,
			Tokens:     256,
			Cached:     false,
			DurationMs: 80,
			ChangeDir:  dir,
		},
	})

	// Verify metrics were persisted.
	pm := LoadPipelineMetrics(dir)
	if _, ok := pm.Phases["explore"]; !ok {
		t.Error("expected explore phase metrics to be recorded")
	}
}

func TestRegisterSubscribers_PhaseAssembled_NilStderr(t *testing.T) {
	t.Parallel()
	b := events.NewBroker()
	dir := t.TempDir()
	// nil stderr — writeMetrics subscriber should skip (no panic).
	RegisterSubscribers(b, 0)

	b.Emit(events.Event{
		Type: events.PhaseAssembled,
		Payload: events.PhaseAssembledPayload{
			Phase:     "propose",
			Bytes:     512,
			Tokens:    128,
			Cached:    false,
			ChangeDir: dir,
		},
	})
}

func TestRegisterSubscribers_PhaseAssembled_CacheContent(t *testing.T) {
	t.Parallel()
	b := events.NewBroker()
	dir := t.TempDir()
	RegisterSubscribers(b, -1)

	// Emit with non-nil Content — should trigger cache persistence.
	b.Emit(events.Event{
		Type: events.PhaseAssembled,
		Payload: events.PhaseAssembledPayload{
			Phase:     "propose",
			Bytes:     100,
			Tokens:    25,
			Cached:    false,
			ChangeDir: dir,
			Content:   []byte("cached context"),
		},
	})

	// The cache file should now exist.
	if _, _, ok := tryCachedContext(dir, "propose", ""); !ok {
		t.Error("expected cached context to be stored after PhaseAssembled with Content")
	}
}

func TestRegisterSubscribers_WrongPayloadType_NoOp(t *testing.T) {
	t.Parallel()
	b := events.NewBroker()
	RegisterSubscribers(b, 0)

	// Both PhaseAssembled subscribers have a !ok guard — wrong payload must not panic.
	b.Emit(events.Event{Type: events.PhaseAssembled, Payload: "not a struct"})
	// VerifyFailed subscriber has a !ok guard.
	b.Emit(events.Event{Type: events.VerifyFailed, Payload: 42})
}

func TestRegisterSubscribers_PhaseAssembled_Cached_SkipsCache(t *testing.T) {
	t.Parallel()
	b := events.NewBroker()
	dir := t.TempDir()
	RegisterSubscribers(b, -1)

	// Emit cached=true with Content — cache subscriber must skip.
	b.Emit(events.Event{
		Type: events.PhaseAssembled,
		Payload: events.PhaseAssembledPayload{
			Phase:     "explore",
			Bytes:     50,
			Tokens:    10,
			Cached:    true,
			ChangeDir: dir,
			Content:   []byte("should not be cached"),
		},
	})

	// No cache should have been written.
	if _, _, ok := tryCachedContext(dir, "explore", ""); ok {
		t.Error("cached=true should skip cache persistence")
	}
}

func TestRegisterSubscribers_VerifyFailed_RecordsErrors(t *testing.T) {
	t.Parallel()
	b := events.NewBroker()
	cwd := t.TempDir()

	RegisterSubscribers(b, 0)

	b.Emit(events.Event{
		Type: events.VerifyFailed,
		Payload: events.VerifyFailedPayload{
			Change:     "my-change",
			ProjectDir: cwd, // required: subscriber skips when empty
			Results: []events.VerifyFailedCommand{
				{
					Name:       "lint",
					Command:    "golangci-lint run",
					ExitCode:   1,
					ErrorLines: []string{"unused var"},
				},
			},
		},
	})

	// Verify the entry was actually recorded in the errlog.
	log := errlog.Load(cwd)
	if len(log.Entries) != 1 {
		t.Fatalf("expected 1 errlog entry, got %d", len(log.Entries))
	}
	if log.Entries[0].Change != "my-change" {
		t.Errorf("entry.Change = %q, want my-change", log.Entries[0].Change)
	}
}

func TestRegisterSubscribers_VerifyFailed_EmptyProjectDir(t *testing.T) {
	t.Parallel()
	b := events.NewBroker()
	RegisterSubscribers(b, 0)

	// Emit with empty ProjectDir — subscriber must skip without panic.
	b.Emit(events.Event{
		Type: events.VerifyFailed,
		Payload: events.VerifyFailedPayload{
			Change:     "my-change",
			ProjectDir: "", // triggers early return guard
			Results: []events.VerifyFailedCommand{
				{Name: "build", Command: "go build ./...", ExitCode: 1},
			},
		},
	})
	// No assertion needed — test passes if no panic occurs.
}

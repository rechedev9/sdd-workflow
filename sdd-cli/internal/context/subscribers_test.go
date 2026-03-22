package context

import (
	"bytes"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
)

func TestRegisterSubscribers_NilBroker(t *testing.T) {
	t.Parallel()
	// Should not panic.
	RegisterSubscribers(nil, nil, 0)
}

func TestRegisterSubscribers_PhaseAssembled_RecordsMetrics(t *testing.T) {
	t.Parallel()
	b := events.NewBroker()
	dir := t.TempDir()
	var stderr bytes.Buffer
	RegisterSubscribers(b, &stderr, 0)

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
	RegisterSubscribers(b, nil, 0)

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
	RegisterSubscribers(b, nil, -1)

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
	if _, ok := tryCachedContext(dir, "propose", ""); !ok {
		t.Error("expected cached context to be stored after PhaseAssembled with Content")
	}
}

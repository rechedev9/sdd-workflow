package context

import (
	"io"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
)

// RegisterSubscribers wires the default event subscribers for metrics
// recording, stderr output, and cache persistence.
// Safe to call with a nil broker (no-op).
func RegisterSubscribers(broker *events.Broker, stderr io.Writer, verbosity int) {
	if broker == nil {
		return
	}

	// Metrics recording — writes to metrics.json (serialized via broker mutex).
	broker.Subscribe(events.PhaseAssembled, func(e events.Event) {
		p, ok := e.Payload.(events.PhaseAssembledPayload)
		if !ok {
			return
		}
		m := &contextMetrics{
			Phase:      p.Phase,
			Bytes:      p.Bytes,
			Tokens:     p.Tokens,
			Cached:     p.Cached,
			DurationMs: p.DurationMs,
		}
		recordMetrics(p.ChangeDir, m)
	})

	// Stderr output — prints metrics line.
	broker.Subscribe(events.PhaseAssembled, func(e events.Event) {
		p, ok := e.Payload.(events.PhaseAssembledPayload)
		if !ok || stderr == nil {
			return
		}
		m := &contextMetrics{
			Phase:      p.Phase,
			Bytes:      p.Bytes,
			Tokens:     p.Tokens,
			Cached:     p.Cached,
			DurationMs: p.DurationMs,
		}
		writeMetrics(stderr, m, verbosity)
	})

	// Cache persistence — saves assembled context for next run.
	broker.Subscribe(events.PhaseAssembled, func(e events.Event) {
		p, ok := e.Payload.(events.PhaseAssembledPayload)
		if !ok || p.Cached || p.Content == nil {
			return
		}
		_ = saveContextCache(p.ChangeDir, p.Phase, p.SkillsPath, p.Content)
	})
}

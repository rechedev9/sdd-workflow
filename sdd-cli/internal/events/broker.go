package events

import (
	"fmt"
	"io"
	"sync"
)

// EventType identifies the kind of event.
type EventType string

// Event type constants.
const (
	PhaseAssembled   EventType = "PhaseAssembled"
	CacheHit         EventType = "CacheHit"
	CacheMiss        EventType = "CacheMiss"
	ArtifactPromoted EventType = "ArtifactPromoted"
	StateAdvanced    EventType = "StateAdvanced"
)

// Event carries a typed payload through the broker.
type Event struct {
	Type    EventType
	Payload any
}

// PhaseAssembledPayload is emitted after a phase's context is assembled.
type PhaseAssembledPayload struct {
	Phase      string
	Bytes      int
	Tokens     int
	Cached     bool
	DurationMs int64
	ChangeDir  string
	SkillsPath string
	Content    []byte // non-nil only on cache miss (for cache subscriber)
}

// CacheHitPayload is emitted when cached context is reused.
type CacheHitPayload struct {
	Phase string
	Bytes int
}

// CacheMissPayload is emitted when context must be freshly assembled.
type CacheMissPayload struct {
	Phase string
}

// ArtifactPromotedPayload is emitted when a pending artifact is promoted.
type ArtifactPromotedPayload struct {
	Change     string
	Phase      string
	PromotedTo string
}

// StateAdvancedPayload is emitted when the state machine advances.
type StateAdvancedPayload struct {
	Change    string
	FromPhase string
	ToPhase   string
}

// Handler processes an event.
type Handler func(Event)

// Broker dispatches events to registered subscribers.
// Safe for concurrent Emit() calls from multiple goroutines.
// A nil *Broker is safe to call Emit() and Subscribe() on (no-op).
type Broker struct {
	mu     sync.Mutex
	subs   map[EventType][]Handler
	stderr io.Writer
}

// NewBroker creates a Broker. stderr is used for panic diagnostics.
func NewBroker(stderr io.Writer) *Broker {
	return &Broker{
		subs:   make(map[EventType][]Handler),
		stderr: stderr,
	}
}

// Subscribe registers a handler for the given event type.
// Nil-safe: calling on a nil *Broker is a no-op.
func (b *Broker) Subscribe(t EventType, h Handler) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[t] = append(b.subs[t], h)
}

// Emit dispatches an event to all subscribers for its type.
// Nil-safe: calling on a nil *Broker is a no-op.
// Serialized via mutex — concurrent Emit() calls are safe.
// Each subscriber is called with panic recovery.
func (b *Broker) Emit(e Event) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	handlers := b.subs[e.Type]
	for _, h := range handlers {
		func(handler Handler) {
			defer func() {
				if r := recover(); r != nil {
					if b.stderr != nil {
						fmt.Fprintf(b.stderr, "sdd: event subscriber panic [%s]: %v\n", e.Type, r)
					}
				}
			}()
			handler(e)
		}(h)
	}
}

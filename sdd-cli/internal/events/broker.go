package events

import (
	"fmt"
	"log/slog"
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
	VerifyFailed     EventType = "VerifyFailed"
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

// VerifyFailedPayload is emitted when sdd verify has one or more failed commands.
type VerifyFailedPayload struct {
	Change  string
	Results []VerifyFailedCommand
}

// VerifyFailedCommand captures one failed command from a verify run.
type VerifyFailedCommand struct {
	Name       string
	Command    string
	ExitCode   int
	ErrorLines []string // first 5 lines
}

// Handler processes an event.
type Handler func(Event)

// Broker dispatches events to registered subscribers.
// Safe for concurrent Emit() calls from multiple goroutines.
// A nil *Broker is safe to call Emit() and Subscribe() on (no-op).
type Broker struct {
	mu   sync.Mutex
	subs map[EventType][]Handler
}

// NewBroker creates a Broker. Panic diagnostics are logged via slog.
func NewBroker() *Broker {
	return &Broker{
		subs: make(map[EventType][]Handler),
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
// The handler slice is copied under the lock, then handlers are called
// without holding the lock — allowing handlers to call Subscribe or Emit
// without deadlocking.
// Each subscriber is called with panic recovery.
func (b *Broker) Emit(e Event) {
	if b == nil {
		return
	}
	b.mu.Lock()
	handlers := make([]Handler, len(b.subs[e.Type]))
	copy(handlers, b.subs[e.Type])
	b.mu.Unlock()

	for _, h := range handlers {
		func(handler Handler) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("event subscriber panic", "event", string(e.Type), "panic", fmt.Sprint(r))
				}
			}()
			handler(e)
		}(h)
	}
}

package live

import (
	"sync"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// subscriberBufferSize is the per-client event buffer. Events that overflow
// are dropped for that client — the snapshot mechanism on reconnect will
// recover the full state.
const subscriberBufferSize = 128

// subscriber represents a single SSE client connection.
type subscriber struct {
	id        uint64
	ch        chan auditlog.Event
	done      chan struct{}
	closeOnce sync.Once
}

// ID returns the subscriber's unique identifier.
func (s *subscriber) ID() uint64 { return s.id }

// Events returns the channel that receives broadcast events.
func (s *subscriber) Events() <-chan auditlog.Event { return s.ch }

// Done returns a channel that is closed when SignalComplete is called
// or the subscriber is removed.
func (s *subscriber) Done() <-chan struct{} { return s.done }

// closeDone closes the done channel exactly once.
func (s *subscriber) closeDone() {
	s.closeOnce.Do(func() { close(s.done) })
}

// Hub fans out auditlog events to all connected SSE clients. It is the
// real-time backbone of the live dashboard.
//
// The hub is safe for concurrent use. OnEvent is called from recorder
// goroutines (the recorder fires callbacks outside its lock), and
// Subscribe/Unsubscribe are called from HTTP handler goroutines.
type Hub struct {
	mu       sync.RWMutex
	plugin   *auditlog.Plugin
	clients  map[uint64]*subscriber
	nextID   uint64
	complete bool
}

// NewHub creates a Hub that sources report data from plugin.
// The hub does not take ownership of the plugin — it only reads from it.
// Pass nil when using live.New() (the plugin is set internally).
func NewHub(plugin *auditlog.Plugin) *Hub {
	return &Hub{ //nolint:exhaustruct
		plugin:  plugin,
		clients: make(map[uint64]*subscriber),
	}
}

// SetPlugin sets the plugin after construction. This is used by
// the live.New() convenience constructor and for setups where the
// plugin is created after the hub.
func (h *Hub) SetPlugin(plugin *auditlog.Plugin) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.plugin = plugin
}

// OnEvent broadcasts an event to all connected subscribers. It is designed
// to be passed directly to auditlog.Config.OnEvent.
//
// If a subscriber's buffer is full, the event is dropped for that subscriber
// (non-blocking send). This prevents one slow client from blocking the
// recorder's callback chain. The client will recover on reconnect via the
// snapshot mechanism.
func (h *Hub) OnEvent(evt auditlog.Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, sub := range h.clients {
		select {
		case sub.ch <- evt:
		default:
		}
	}
}

// Subscribe registers a new SSE client and returns a subscriber.
// The caller must call Unsubscribe with the returned subscriber's ID when
// the connection closes.
func (h *Hub) Subscribe() *subscriber {
	h.mu.Lock()
	defer h.mu.Unlock()

	subID := h.nextID
	h.nextID++

	sub := &subscriber{ //nolint:exhaustruct
		id:   subID,
		ch:   make(chan auditlog.Event, subscriberBufferSize),
		done: make(chan struct{}),
	}
	h.clients[subID] = sub

	return sub
}

// Unsubscribe removes a subscriber by ID and signals its done channel.
// Safe to call multiple times — subsequent calls are no-ops.
func (h *Hub) Unsubscribe(subscriberID uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sub, ok := h.clients[subscriberID]
	if !ok {
		return
	}

	sub.closeDone()
	delete(h.clients, subscriberID)
}

// SignalComplete marks the container lifecycle as finished. All subscribers
// receive a done signal so the SSE handler can send the final report.
//
// After SignalComplete, new subscribers still receive the snapshot
// (which will include the final report) but no live events are expected.
func (h *Hub) SignalComplete() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.complete = true

	for _, sub := range h.clients {
		sub.closeDone()
	}
}

// IsComplete returns whether the lifecycle has been marked as complete.
func (h *Hub) IsComplete() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.complete
}

// ClientCount returns the number of currently connected subscribers.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients)
}

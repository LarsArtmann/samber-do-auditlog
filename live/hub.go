package live

import (
	"encoding/json"
	"strconv"
	"sync/atomic"

	"github.com/larsartmann/go-sse"
	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// subscriberBufferSize is the per-client event buffer. Events that overflow
// are dropped for that client — the snapshot mechanism on reconnect will
// recover the full state.
const subscriberBufferSize = 128

// Hub fans out container lifecycle events to all connected SSE clients via
// sse.Broadcaster[sse.Event]. It wraps the generic broadcaster with
// domain-specific lifecycle semantics (SignalComplete/IsComplete/Done) that
// go-sse has no concept of.
//
// The hub is safe for concurrent use. OnEvent is called from plugin
// callbacks, and Subscribe/Unsubscribe are called from HTTP handler goroutines.
type Hub struct {
	bc       *sse.Broadcaster[sse.Event]
	complete atomic.Bool
	doneCh   chan struct{}
}

// NewHub creates a Hub ready for use.
func NewHub() *Hub {
	return &Hub{
		bc:       sse.NewBroadcaster[sse.Event](sse.WithBufferSize[sse.Event](subscriberBufferSize)),
		complete: atomic.Bool{},
		doneCh:   make(chan struct{}),
	}
}

// OnEvent marshals an auditlog.Event to JSON and broadcasts it to all
// connected SSE clients as a ready-to-send sse.Event. The auditlog
// event's Sequence is used as the SSE event ID so that reconnection
// replay can filter by sequence number.
func (h *Hub) OnEvent(evt auditlog.Event) {
	payload, err := json.Marshal(evt)
	if err != nil {
		return
	}

	h.bc.Broadcast(sse.Event{
		Event: "event",
		Data:  string(payload),
		ID:    sse.NewEventID(strconv.Itoa(evt.Sequence)),
	})
}

// Subscribe returns a channel that receives broadcast SSE events.
// The channel has a buffer of subscriberBufferSize; events that overflow
// are dropped for that subscriber — the snapshot mechanism on reconnect
// recovers the full state.
//
// Call Unsubscribe when the client disconnects to prevent memory leaks.
func (h *Hub) Subscribe() <-chan sse.Event {
	return h.bc.Subscribe()
}

// Unsubscribe removes a subscriber channel and closes it.
// Call this when a client disconnects to prevent memory leaks.
func (h *Hub) Unsubscribe(ch <-chan sse.Event) {
	h.bc.Unsubscribe(ch)
}

// Done returns a channel that is closed when the container lifecycle is
// marked as complete via SignalComplete. Handlers select on this to know
// when to send the final report.
func (h *Hub) Done() <-chan struct{} {
	return h.doneCh
}

// SignalComplete marks the lifecycle as finished. All handlers waiting
// on Done() are unblocked so they can send the final report.
func (h *Hub) SignalComplete() {
	if h.complete.CompareAndSwap(false, true) {
		close(h.doneCh)
	}
}

// IsComplete returns whether the lifecycle has been marked as complete.
func (h *Hub) IsComplete() bool {
	return h.complete.Load()
}

// ClientCount returns the number of currently connected subscribers.
func (h *Hub) ClientCount() int {
	return h.bc.SubscriberCount()
}

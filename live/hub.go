package live

import (
	"context"
	"encoding/json"
	"strconv"
	"sync/atomic"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// sseEventType is the SSE event name for auditlog lifecycle events broadcast
// by the hub.
const sseEventType = "event"

// Hub fans out container lifecycle events to all connected SSE clients. It
// wraps the broadcaster with domain-specific lifecycle semantics
// (SignalComplete/IsComplete/Done) and keeps a replay ring buffer for SSE
// reconnection.
//
// The hub is safe for concurrent use. OnEvent is called from plugin
// callbacks, and Subscribe/Unsubscribe are called from HTTP handler goroutines.
type Hub struct {
	bc         *broadcaster
	complete   atomic.Bool
	doneCh     chan struct{}
	ringBuffer *eventRingBuffer
}

// NewHub creates a Hub ready for use with the default replay buffer size.
func NewHub() *Hub {
	return NewHubWithReplay(0)
}

// NewHubWithReplay creates a Hub with a replay ring buffer of the given
// capacity. Non-positive capacity uses the default (1000).
func NewHubWithReplay(replayBufferSize int) *Hub {
	return &Hub{
		bc:         newBroadcaster(subscriberBufferSize),
		complete:   atomic.Bool{},
		doneCh:     make(chan struct{}),
		ringBuffer: newEventRingBuffer(replayBufferSize),
	}
}

// OnEvent marshals an auditlog.Event to JSON and broadcasts it to all
// connected SSE clients. The auditlog event's Sequence is used as the SSE
// event ID so that reconnection replay can filter by sequence number.
func (h *Hub) OnEvent(evt auditlog.Event) {
	payload, err := json.Marshal(evt)
	if err != nil {
		return
	}

	sseEvt := sseEvent{
		Name: sseEventType,
		Data: string(payload),
		ID:   strconv.Itoa(evt.Sequence),
	}

	h.ringBuffer.add(sseEvt)

	h.bc.broadcast(sseEvt)
}

// Subscribe returns a channel that receives broadcast SSE events.
// The channel has a buffer of subscriberBufferSize; events that overflow
// are dropped for that subscriber — the snapshot mechanism on reconnect
// recovers the full state.
//
// Call Unsubscribe when the client disconnects to prevent memory leaks.
func (h *Hub) Subscribe() <-chan sseEvent {
	return h.bc.subscribe()
}

// Unsubscribe removes a subscriber channel and closes it.
// Call this when a client disconnects to prevent memory leaks.
func (h *Hub) Unsubscribe(ch <-chan sseEvent) {
	h.bc.unsubscribe(ch)
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
	return h.bc.subscriberCount()
}

// Shutdown gracefully drains the broadcaster: stops accepting new subscribers,
// waits for active subscriber buffers to empty, then closes all channels.
// Returns an error wrapping ctx.Err() if the context fires before the drain
// completes.
func (h *Hub) Shutdown(ctx context.Context) error {
	return h.bc.shutdown(ctx)
}

// Health returns a structured snapshot of the broadcaster's lifecycle state
// for health checks and observability dashboards.
func (h *Hub) Health() broadcasterHealth {
	return h.bc.health()
}

// ReplayStore returns the replay ring buffer holding recently broadcast
// events for SSE reconnection replay.
func (h *Hub) ReplayStore() *eventRingBuffer {
	return h.ringBuffer
}

// BufferedEventCount returns the number of events currently stored in the
// replay ring buffer.
func (h *Hub) BufferedEventCount() int {
	return h.ringBuffer.len()
}

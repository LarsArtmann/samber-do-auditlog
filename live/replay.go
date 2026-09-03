package live

import (
	"strconv"
	"sync"
)

const (
	// defaultReplayBufferSize is the maximum number of events retained for
	// SSE reconnection replay. Events older than this are evicted (FIFO).
	defaultReplayBufferSize = 1000
)

// eventRingBuffer is a bounded, thread-safe ring buffer of SSE events used
// for reconnection replay.
//
// Events are stored with monotonically increasing numeric IDs (assigned by
// the Hub). eventsAfter returns all events with IDs strictly greater than
// the given lastID, ordered ascending.
type eventRingBuffer struct {
	mu     sync.RWMutex
	events []sseEvent
	cap    int
}

// newEventRingBuffer creates a ring buffer with the given capacity.
// Non-positive capacity falls back to defaultReplayBufferSize.
func newEventRingBuffer(capacity int) *eventRingBuffer {
	if capacity <= 0 {
		capacity = defaultReplayBufferSize
	}

	return &eventRingBuffer{
		mu:     sync.RWMutex{},
		events: make([]sseEvent, 0, capacity),
		cap:    capacity,
	}
}

// add appends an event to the buffer. If the buffer is full, the oldest
// event is evicted (FIFO).
func (rb *eventRingBuffer) add(evt sseEvent) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if len(rb.events) >= rb.cap {
		rb.events = rb.events[1:]
	}

	rb.events = append(rb.events, evt)
}

// eventsAfter returns all events with IDs strictly greater than lastID,
// ordered ascending. Returns nil if lastID is not a number or no events
// match (an unparseable lastID means "nothing to replay from").
func (rb *eventRingBuffer) eventsAfter(lastID string) []sseEvent {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	lastSeq, err := strconv.ParseUint(lastID, base10, 64)
	if err != nil {
		return nil
	}

	var result []sseEvent

	for _, evt := range rb.events {
		seq, err := strconv.ParseUint(evt.ID, base10, 64)
		if err != nil {
			continue
		}

		if seq > lastSeq {
			result = append(result, evt)
		}
	}

	return result
}

// len returns the number of events currently stored.
func (rb *eventRingBuffer) len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	return len(rb.events)
}

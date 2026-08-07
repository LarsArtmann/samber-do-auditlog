package live

import (
	"strconv"
	"sync"

	"github.com/larsartmann/go-sse"
)

const (
	// defaultReplayBufferSize is the maximum number of events retained for
	// SSE reconnection replay. Events older than this are evicted (FIFO).
	defaultReplayBufferSize = 1000
)

// eventRingBuffer is a bounded, thread-safe ring buffer of SSE events used
// for reconnection replay. It implements sse.EventStore.
//
// Events are stored with monotonically increasing numeric IDs (assigned by
// the Hub). EventsAfter returns all events with IDs strictly greater than
// the given lastID, ordered ascending.
type eventRingBuffer struct {
	mu     sync.RWMutex
	events []sse.Event
	cap    int
}

// newEventRingBuffer creates a ring buffer with the given capacity.
// Non-positive capacity falls back to defaultReplayBufferSize.
func newEventRingBuffer(capacity int) *eventRingBuffer {
	if capacity <= 0 {
		capacity = defaultReplayBufferSize
	}

	return &eventRingBuffer{ //nolint:exhaustruct // mu zero-value is correct
		events: make([]sse.Event, 0, capacity),
		cap:    capacity,
	}
}

// add appends an event to the buffer. If the buffer is full, the oldest
// event is evicted (FIFO).
func (rb *eventRingBuffer) add(evt sse.Event) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if len(rb.events) >= rb.cap {
		rb.events = rb.events[1:]
	}

	rb.events = append(rb.events, evt)
}

// EventsAfter implements sse.EventStore. It returns all events with IDs
// strictly greater than lastID, ordered ascending. Returns an empty slice
// if lastID is unknown or no events match.
func (rb *eventRingBuffer) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	lastSeq, err := strconv.ParseUint(lastID.Get(), 10, 64)
	if err != nil {
		return nil, nil
	}

	var result []sse.Event

	for _, evt := range rb.events {
		seq, err := strconv.ParseUint(evt.ID.Get(), 10, 64)
		if err != nil {
			continue
		}

		if seq > lastSeq {
			result = append(result, evt)
		}
	}

	return result, nil
}

// len returns the number of events currently stored.
func (rb *eventRingBuffer) len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	return len(rb.events)
}

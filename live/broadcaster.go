package live

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"sync"
	"time"
)

// subscriberBufferSize is the per-client event buffer. Events that overflow
// are dropped for that client — the snapshot mechanism on reconnect will
// recover the full state.
const subscriberBufferSize = 128

// drainPollInterval is how often broadcaster.shutdown re-checks whether all
// subscriber buffers have been drained while waiting for consumers to catch
// up. Short enough that an idle consumer registers a drain promptly, long
// enough to avoid burning CPU.
const drainPollInterval = time.Millisecond

// broadcasterHealth is a snapshot of a broadcaster's lifecycle state,
// returned by broadcaster.health for health checks and observability.
type broadcasterHealth struct {
	// Draining is true while shutdown is waiting for subscriber buffers
	// to drain. During draining, new subscribe calls return a closed
	// channel so no new work piles up.
	Draining bool
	// SubscriberCount is the number of currently registered subscribers.
	SubscriberCount int
	// BufferSize is the per-subscriber channel capacity, in events.
	BufferSize int
}

// broadcaster fans out SSE events to all connected clients with
// non-blocking sends: a subscriber whose buffer is full has the message
// silently dropped (the snapshot mechanism on reconnect recovers state).
//
// The zero value is not usable; construct via newBroadcaster. All methods
// are safe for concurrent use.
type broadcaster struct {
	mu          sync.RWMutex
	subscribers map[uintptr]chan sseEvent
	bufferSize  int
	draining    bool
}

// newBroadcaster creates a broadcaster with the given per-subscriber channel
// capacity. Non-positive values fall back to subscriberBufferSize.
func newBroadcaster(bufferSize int) *broadcaster {
	if bufferSize <= 0 {
		bufferSize = subscriberBufferSize
	}

	return &broadcaster{
		mu:          sync.RWMutex{},
		subscribers: make(map[uintptr]chan sseEvent),
		bufferSize:  bufferSize,
		draining:    false,
	}
}

// subscribe registers a new subscriber and returns its event channel. After
// shutdown has started, subscribe returns a closed channel (no-op).
func (b *broadcaster) subscribe() <-chan sseEvent {
	subCh := make(chan sseEvent, b.bufferSize)

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.subscribers == nil || b.draining {
		close(subCh)

		return subCh
	}

	b.subscribers[sseChannelPtr(subCh)] = subCh

	return subCh
}

// unsubscribe removes a subscriber channel and closes it. Unknown channels
// are ignored.
func (b *broadcaster) unsubscribe(ch <-chan sseEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := sseChannelPtr(ch)

	if sub, ok := b.subscribers[key]; ok {
		delete(b.subscribers, key)
		close(sub)
	}
}

// broadcast sends msg to all active subscribers. The iteration runs under
// the read lock so a concurrent unsubscribe cannot close a channel mid-send;
// sends are non-blocking, so slow subscribers drop the message.
func (b *broadcaster) broadcast(msg sseEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers {
		select {
		case ch <- msg:
		default:
		}
	}
}

// subscriberCount returns the number of active subscribers.
func (b *broadcaster) subscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.subscribers)
}

// shutdown gracefully drains the broadcaster: it stops accepting new
// subscribers, waits for every active subscriber's buffer to empty, then
// closes all subscriber channels. Returns ctx.Err()-wrapped if the context
// fires before the drain completes (nothing is closed in that case; retry
// with a fresh context).
func (b *broadcaster) shutdown(ctx context.Context) error {
	b.mu.Lock()

	if b.subscribers == nil {
		b.mu.Unlock()

		return nil
	}

	subs := slices.Collect(maps.Values(b.subscribers))
	b.draining = true

	b.mu.Unlock()

	if err := b.waitForDrain(ctx, subs); err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range subs {
		key := sseChannelPtr(ch)
		if _, stillTracked := b.subscribers[key]; stillTracked {
			delete(b.subscribers, key)
			close(ch)
		}
	}

	b.subscribers = nil
	b.draining = false

	return nil
}

// waitForDrain blocks until every subscriber buffer is empty or ctx is
// cancelled. Returns nil once all buffers drain, or a wrapped context error
// if the deadline fires first.
func (b *broadcaster) waitForDrain(ctx context.Context, subs []chan sseEvent) error {
	for {
		notDrained := 0

		for _, ch := range subs {
			if len(ch) > 0 {
				notDrained++
			}
		}

		if notDrained == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("drain %d subscribers: %w", len(subs), ctx.Err())
		case <-time.After(drainPollInterval):
		}
	}
}

// health returns a snapshot of the broadcaster's lifecycle state.
func (b *broadcaster) health() broadcasterHealth {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return broadcasterHealth{
		Draining:        b.draining,
		SubscriberCount: len(b.subscribers),
		BufferSize:      b.bufferSize,
	}
}

// sseChannelPtr returns a stable identity for a receive-only channel so it
// can be tracked in the subscriber map and later unsubscribed by value.
func sseChannelPtr(ch <-chan sseEvent) uintptr {
	return reflect.ValueOf(ch).Pointer()
}

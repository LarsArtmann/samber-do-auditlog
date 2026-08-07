package live_test

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/larsartmann/go-sse"
	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
)

func makeReplayEvent(seq int) auditlog.Event {
	return auditlog.Event{
		Sequence:  seq,
		EventType: auditlog.EventTypeRegistration,
		Phase:     auditlog.PhaseBefore,
		ServiceRef: auditlog.ServiceRef{
			ServiceName: auditlog.ServiceName(fmt.Sprintf("svc-%d", seq)),
		},
	}
}

func TestHub_EventStore_EventsAfterUnknownID(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	hub.OnEvent(makeReplayEvent(1))
	hub.OnEvent(makeReplayEvent(2))

	store := hub.EventStore()

	events, err := store.EventsAfter(sse.NewEventID("99"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("expected 0 events after unknown ID, got %d", len(events))
	}
}

func TestHub_EventStore_ReplayMatchingEvents(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	for i := 1; i <= 5; i++ {
		hub.OnEvent(makeReplayEvent(i))
	}

	store := hub.EventStore()

	events, err := store.EventsAfter(sse.NewEventID("2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events after ID 2, got %d", len(events))
	}

	for i, evt := range events {
		expectedID := strconv.Itoa(i + 3)

		if evt.ID.Get() != expectedID {
			t.Errorf("event %d: expected ID %s, got %s", i, expectedID, evt.ID.Get())
		}
	}
}

func TestHub_RingBufferOverflow(t *testing.T) {
	t.Parallel()

	hub := live.NewHubWithReplay(3)

	for i := 1; i <= 5; i++ {
		hub.OnEvent(makeReplayEvent(i))
	}

	if hub.BufferedEventCount() != 3 {
		t.Fatalf("expected 3 buffered events, got %d", hub.BufferedEventCount())
	}

	store := hub.EventStore()

	events, err := store.EventsAfter(sse.NewEventID("0"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events in buffer, got %d", len(events))
	}

	expectedIDs := []string{"3", "4", "5"}

	for i, evt := range events {
		if evt.ID.Get() != expectedIDs[i] {
			t.Errorf("event %d: expected ID %s, got %s", i, expectedIDs[i], evt.ID.Get())
		}
	}
}

func TestHub_EventStore_ConcurrentReplaySafety(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	for i := 1; i <= 50; i++ {
		hub.OnEvent(makeReplayEvent(i))
	}

	store := hub.EventStore()

	var wg sync.WaitGroup

	for i := range 4 {
		wg.Add(1)

		go func(offset int) {
			defer wg.Done()

			for j := range 50 {
				hub.OnEvent(makeReplayEvent(100 + offset*50 + j))
			}
		}(i)
	}

	for i := range 4 {
		wg.Add(1)

		go func(n int) {
			defer wg.Done()

			for j := range 50 {
				lastID := sse.NewEventID(strconv.Itoa(n*10 + j))
				_, _ = store.EventsAfter(lastID)
			}
		}(i)
	}

	wg.Wait()
}

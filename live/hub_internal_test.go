package live

import (
	"strconv"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestHubBufferedEventCountAndReplayStore(t *testing.T) {
	t.Parallel()

	hub := NewHubWithReplay(3)

	for i := range 5 {
		hub.OnEvent(auditlogEvent(i + 1))
	}

	if got := hub.BufferedEventCount(); got != 3 {
		t.Errorf("BufferedEventCount = %d, want 3 (FIFO eviction)", got)
	}

	store := hub.ReplayStore()

	if got := len(store.eventsAfter("2")); got != 3 {
		t.Errorf("eventsAfter(2) = %d events, want 3", got)
	}

	if got := store.eventsAfter("garbage"); got != nil {
		t.Errorf("eventsAfter(garbage) should be nil, got %d", len(got))
	}
}

func TestHubConcurrentOnEventAndReplay(t *testing.T) {
	t.Parallel()

	hub := NewHubWithReplay(64)

	done := make(chan struct{})

	go func() {
		defer close(done)

		for i := range 200 {
			hub.OnEvent(auditlogEvent(i + 1))
		}
	}()

	stop := time.After(200 * time.Millisecond)

	for {
		select {
		case <-stop:
			<-done

			return
		default:
			_ = hub.ReplayStore().eventsAfter("0")
		}
	}
}

func TestHubSubscribeDeliversSequenceIDs(t *testing.T) {
	t.Parallel()

	hub := NewHub()

	ch := hub.Subscribe()
	defer func() { hub.Unsubscribe(ch) }()

	hub.OnEvent(auditlogEvent(7))

	select {
	case evt := <-ch:
		if evt.ID != strconv.Itoa(7) {
			t.Errorf("event ID = %q, want %q", evt.ID, "7")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for hub event")
	}
}

func auditlogEvent(seq int) auditlog.Event {
	return auditlog.Event{Sequence: seq, EventType: auditlog.EventTypeRegistration, Phase: auditlog.PhaseAfter}
}

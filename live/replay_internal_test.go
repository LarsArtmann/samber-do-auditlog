package live

import (
	"testing"

	"github.com/larsartmann/go-sse"
	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestEventStore_EventsAfter(t *testing.T) {
	t.Parallel()

	store := &eventStore{
		events: []auditlog.Event{
			{Sequence: 1},
			{Sequence: 2},
			{Sequence: 3},
			{Sequence: 4},
			{Sequence: 5},
		},
	}

	tests := []struct {
		name    string
		lastID  string
		zero    bool
		wantIDs []string
	}{
		{"after_1_returns_4_events", "1", false, []string{"2", "3", "4", "5"}},
		{"after_3_returns_2_events", "3", false, []string{"4", "5"}},
		{"after_5_returns_none", "5", false, nil},
		{"after_0_returns_all", "0", false, []string{"1", "2", "3", "4", "5"}},
		{"zero_id_returns_none", "", true, nil},
		{"invalid_id_returns_none", "abc", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var id sse.EventID

			if tt.zero {
				id = sse.EventID{}
			} else {
				id = sse.NewEventID(tt.lastID)
			}

			events, err := store.EventsAfter(id)
			if err != nil {
				t.Fatalf("EventsAfter error: %v", err)
			}

			if len(events) != len(tt.wantIDs) {
				t.Fatalf("got %d events, want %d", len(events), len(tt.wantIDs))
			}

			for i, evt := range events {
				if evt.ID.Get() != tt.wantIDs[i] {
					t.Errorf("event %d: got ID %q, want %q", i, evt.ID.Get(), tt.wantIDs[i])
				}

				if evt.Event != "event" {
					t.Errorf("event %d: got type %q, want %q", i, evt.Event, "event")
				}

				if evt.Data == "" {
					t.Errorf("event %d: empty data payload", i)
				}
			}
		})
	}
}

func TestEventStore_Empty(t *testing.T) {
	t.Parallel()

	store := &eventStore{events: nil}

	events, err := store.EventsAfter(sse.NewEventID("1"))
	if err != nil {
		t.Fatalf("EventsAfter error: %v", err)
	}

	if events != nil {
		t.Errorf("expected nil for empty store, got %d events", len(events))
	}
}

package live

import (
	"encoding/json"
	"strconv"

	"github.com/larsartmann/go-sse"
	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// eventStore adapts a snapshot of auditlog events into sse.EventStore
// for reconnection replay. The snapshot is taken at handler entry — events
// before the handler call are available for replay; events after arrive
// via the live broadcaster channel.
type eventStore struct {
	events []auditlog.Event
}

// EventsAfter returns SSE events with sequence IDs strictly after lastID.
// The auditlog event's Sequence field is the SSE event ID, so this filters
// by the same numbering scheme used for live broadcasts.
// Events are ordered ascending by sequence (matching the input order from
// plugin.Events()).
func (s *eventStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	if lastID.IsZero() {
		return nil, nil
	}

	after, err := strconv.ParseInt(lastID.Get(), 10, 64)
	if err != nil {
		return nil, nil
	}

	var result []sse.Event

	for _, evt := range s.events {
		if int64(evt.Sequence) <= after {
			continue
		}

		payload, err := json.Marshal(evt)
		if err != nil {
			continue
		}

		result = append(result, sse.Event{
			Event: "event",
			Data:  string(payload),
			ID:    sse.NewEventID(strconv.Itoa(evt.Sequence)),
		})
	}

	return result, nil
}

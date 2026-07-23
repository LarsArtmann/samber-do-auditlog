package auditlog

import (
	"errors"
	"fmt"
	"io"

	"github.com/larsartmann/go-ndjson"
)

// Sentinel errors for NDJSON reading. Re-exported from auditlog-core/ndjson
// so existing callers can continue to match with errors.Is.
var (
	ErrEmpty         = ndjson.ErrEmpty
	ErrNoEvents      = ndjson.ErrNoEvents
	ErrOversizedLine = ndjson.ErrOversizedLine
)

// Domain-specific validation errors.
var (
	errUnknownEventType = errors.New("unknown event_type")
	errUnknownPhase     = errors.New("unknown phase")
)

// ReadEvents reads line-delimited JSON events from reader.
// Each line must be a single JSON-encoded Event object.
// Blank lines are skipped. Returns the parsed events in order.
//
// Returns ErrEmpty if the input contains no bytes, ErrNoEvents if all lines
// were blank, or ErrOversizedLine if any line exceeds 1 MB.
func ReadEvents(reader io.Reader) ([]Event, error) {
	events, err := ndjson.Read(reader, validateEvent)
	if err != nil {
		return nil, fmt.Errorf("read ndjson events: %w", err)
	}

	return events, nil
}

// validateEvent checks that event_type and phase are recognized values.
func validateEvent(lineNum int, evt Event) error {
	if evt.EventType != "" && !evt.EventType.IsKnown() {
		return fmt.Errorf("line %d: %w: %q", lineNum, errUnknownEventType, evt.EventType)
	}

	if evt.Phase != "" && !evt.Phase.IsKnown() {
		return fmt.Errorf("line %d: %w: %q", lineNum, errUnknownPhase, evt.Phase)
	}

	return nil
}

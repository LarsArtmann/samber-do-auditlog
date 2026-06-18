package auditlog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Maximum line size for NDJSON reading (1 MB).
const ndjsonMaxLineBytes = 1 << 20

// Errors returned by ReadEvents.
var (
	ErrEmpty            = errors.New("ndjson input is empty")
	ErrNoEvents         = errors.New("ndjson input contains no events")
	ErrOversizedLine    = errors.New("ndjson line exceeds maximum size")
	errUnknownEventType = errors.New("unknown event_type")
	errUnknownPhase     = errors.New("unknown phase")
)

// ReadEvents reads line-delimited JSON events from reader.
// Each line must be a single JSON-encoded Event object.
// Blank lines are skipped. Returns the parsed events in order.
func ReadEvents(reader io.Reader) ([]Event, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, ndjsonMaxLineBytes), ndjsonMaxLineBytes)

	var events []Event

	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		// Skip blank/whitespace-only lines.
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var evt Event

		err := json.Unmarshal(line, &evt)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		if evt.EventType != "" && !evt.EventType.IsKnown() {
			return nil, fmt.Errorf("line %d: %w: %q", lineNum, errUnknownEventType, evt.EventType)
		}

		if evt.Phase != "" && !evt.Phase.IsKnown() {
			return nil, fmt.Errorf("line %d: %w: %q", lineNum, errUnknownPhase, evt.Phase)
		}

		events = append(events, evt)
	}

	err := scanner.Err()
	if err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return nil, fmt.Errorf("%w (max %d bytes)", ErrOversizedLine, ndjsonMaxLineBytes)
		}

		return nil, fmt.Errorf("reading ndjson: %w", err)
	}

	if lineNum == 0 {
		return nil, ErrEmpty
	}

	if len(events) == 0 {
		return nil, ErrNoEvents
	}

	return events, nil
}

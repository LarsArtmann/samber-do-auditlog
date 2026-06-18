package auditlog

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Maximum line size for NDJSON reading (1 MB).
const ndjsonMaxLineBytes = 1 << 20

// Errors returned by ReadEvents.
var (
	ErrEmpty         = errors.New("ndjson input is empty")
	ErrNoEvents      = errors.New("ndjson input contains no events")
	ErrOversizedLine = errors.New("ndjson line exceeds maximum size")
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
		if len(trimWhitespace(line)) == 0 {
			continue
		}

		var evt Event

		err := json.Unmarshal(line, &evt)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
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

// trimWhitespace removes leading/trailing ASCII whitespace from a byte slice.
func trimWhitespace(data []byte) []byte {
	start := 0
	end := len(data)

	for start < end && (data[start] == ' ' || data[start] == '\t' || data[start] == '\r' || data[start] == '\n') {
		start++
	}

	for end > start && (data[end-1] == ' ' || data[end-1] == '\t' || data[end-1] == '\r' || data[end-1] == '\n') {
		end--
	}

	return data[start:end]
}

package auditlog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/larsartmann/go-ndjson"
)

// Sentinel errors for NDJSON reading. Re-exported from go-ndjson
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

// errNilStreamCallback is returned by [StreamEvents] when fn is nil.
var errNilStreamCallback = errors.New("auditlog: StreamEvents callback is nil")

// StreamEventsCallback is the per-event callback signature for [StreamEvents].
// lineNum is the 1-based line number from which the event was parsed (useful
// for error reporting in the callback itself).
type StreamEventsCallback func(lineNum int, evt Event) error

// StreamEvents reads line-delimited JSON events from reader and invokes fn
// for each parsed event in order, without materializing the entire event
// stream in memory. This is the streaming counterpart to [ReadEvents] —
// designed for containers that produce more events than fit comfortably in
// RAM (10k+ events, long-running processes).
//
// The validate function is called for each parsed event with its 1-based line
// number (same semantics as [ReadEvents]). Pass nil to skip validation.
//
// If fn returns a non-nil error, StreamEvents stops reading and returns that
// error wrapped with the line number, so callers can halt cleanly on a
// downstream failure (disk full, network drop, etc.).
//
// Returns ErrEmpty if the input contains no bytes, ErrNoEvents if all lines
// were blank, or ErrOversizedLine if any line exceeds 1 MB — identical to
// [ReadEvents].
func StreamEvents(reader io.Reader, validate func(lineNum int, evt Event) error, fn StreamEventsCallback) error {
	if fn == nil {
		return errNilStreamCallback
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, ndjson.MaxLineBytes), ndjson.MaxLineBytes)

	lineNum := 0

	delivered := 0

	for scanner.Scan() {
		lineNum++

		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var evt Event

		err := json.Unmarshal(line, &evt)
		if err != nil {
			return fmt.Errorf("ndjson line %d: %w", lineNum, err)
		}

		if validate != nil {
			err = validate(lineNum, evt)
			if err != nil {
				return err
			}
		}

		err = fn(lineNum, evt)
		if err != nil {
			return fmt.Errorf("ndjson line %d: callback: %w", lineNum, err)
		}

		delivered++
	}

	err := scanner.Err()
	if err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return fmt.Errorf("%w (max %d bytes)", ErrOversizedLine, ndjson.MaxLineBytes)
		}

		return fmt.Errorf("scan ndjson: %w", err)
	}

	if delivered == 0 {
		if lineNum == 0 {
			return ErrEmpty
		}

		return ErrNoEvents
	}

	return nil
}

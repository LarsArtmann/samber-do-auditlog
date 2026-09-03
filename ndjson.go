package auditlog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// MaxLineBytes is the maximum allowed size for a single NDJSON line (1 MB).
const MaxLineBytes = 1 << 20

// Sentinel errors for NDJSON reading.
var (
	ErrEmpty         = errors.New("ndjson input is empty")
	ErrNoEvents      = errors.New("ndjson input contains no events")
	ErrOversizedLine = errors.New("ndjson line exceeds maximum size")
)

// Domain-specific validation errors.
var (
	errUnknownEventType    = errors.New("unknown event_type")
	errUnknownPhase        = errors.New("unknown phase")
	errUnknownProviderType = errors.New("unknown provider_type")
)

// scanNDJSONLines scans reader line by line (1 MB line cap), invoking fn for
// each non-blank line with its 1-based line number. Returns ErrEmpty when the
// input holds no bytes at all and ErrNoEvents when every line was blank.
// Oversized lines surface as ErrOversizedLine; other scanner errors are
// wrapped with a "scan ndjson" prefix.
func scanNDJSONLines(reader io.Reader, fn func(lineNum int, line []byte) error) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, MaxLineBytes), MaxLineBytes)

	lineNum := 0

	nonBlank := 0

	for scanner.Scan() {
		lineNum++

		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		nonBlank++

		if err := fn(lineNum, line); err != nil {
			return err
		}
	}

	err := scanner.Err()
	if err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return fmt.Errorf("%w (max %d bytes)", ErrOversizedLine, MaxLineBytes)
		}

		return fmt.Errorf("scan ndjson: %w", err)
	}

	if lineNum == 0 {
		return ErrEmpty
	}

	if nonBlank == 0 {
		return ErrNoEvents
	}

	return nil
}

// ReadEvents reads line-delimited JSON events from reader.
// Each line must be a single JSON-encoded Event object.
// Blank lines are skipped. Returns the parsed events in order.
//
// Returns ErrEmpty if the input contains no bytes, ErrNoEvents if all lines
// were blank, or ErrOversizedLine if any line exceeds MaxLineBytes.
func ReadEvents(reader io.Reader) ([]Event, error) {
	var events []Event

	err := scanNDJSONLines(reader, func(lineNum int, line []byte) error {
		var evt Event

		if err := json.Unmarshal(line, &evt); err != nil {
			return fmt.Errorf("ndjson line %d: %w", lineNum, err)
		}

		if err := validateEvent(lineNum, evt); err != nil {
			return err
		}

		events = append(events, evt)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read ndjson events: %w", err)
	}

	return events, nil
}

// validateEvent checks that event_type, phase, and provider_type are
// recognized values. An empty provider_type is allowed: it legitimately means
// "could not be determined" for some registration paths.
func validateEvent(lineNum int, evt Event) error {
	if err := validateEventEnums(evt); err != nil {
		return fmt.Errorf("line %d: %w", lineNum, err)
	}

	return nil
}

// validateEventEnums checks an Event's enum fields without position context.
// Shared by the NDJSON load path (which adds the line number) and the replay
// path (which adds the event index) so corrupt data fails loudly at both
// boundaries instead of being silently dropped.
func validateEventEnums(evt Event) error {
	if evt.EventType != "" && !evt.EventType.IsKnown() {
		return fmt.Errorf("%w: %q", errUnknownEventType, evt.EventType)
	}

	if evt.Phase != "" && !evt.Phase.IsKnown() {
		return fmt.Errorf("%w: %q", errUnknownPhase, evt.Phase)
	}

	if evt.ServiceType != "" && !evt.ServiceType.IsKnown() {
		return fmt.Errorf("%w: %q", errUnknownProviderType, evt.ServiceType)
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
// were blank, or ErrOversizedLine if any line exceeds MaxLineBytes — identical
// to [ReadEvents].
func StreamEvents(reader io.Reader, validate func(lineNum int, evt Event) error, fn StreamEventsCallback) error {
	if fn == nil {
		return errNilStreamCallback
	}

	err := scanNDJSONLines(reader, func(lineNum int, line []byte) error {
		var evt Event

		if err := json.Unmarshal(line, &evt); err != nil {
			return fmt.Errorf("ndjson line %d: %w", lineNum, err)
		}

		if validate != nil {
			if err := validate(lineNum, evt); err != nil {
				return err
			}
		}

		if err := fn(lineNum, evt); err != nil {
			return fmt.Errorf("ndjson line %d: callback: %w", lineNum, err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

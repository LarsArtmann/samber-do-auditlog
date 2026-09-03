package auditlog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrUnsupportedFormat is returned when the format value is not recognized.
var ErrUnsupportedFormat = errors.New("unsupported format")

// errNoContent is returned by format detection when no non-blank content exists.
var errNoContent = errors.New("no content found for format detection")

// Format identifies the serialization format of a report file.
type Format int

const (
	// FormatAuto auto-detects JSON vs NDJSON by inspecting the first line.
	FormatAuto Format = iota
	// FormatJSON is a single JSON Report object (contains "version" key).
	FormatJSON
	// FormatNDJSON is newline-delimited Event objects (contains "event_type" key).
	FormatNDJSON
)

// String returns the human-readable format name.
func (f Format) String() string {
	switch f {
	case FormatAuto:
		return "auto"
	case FormatJSON:
		return "json"
	case FormatNDJSON:
		return "ndjson"
	default:
		return "unknown"
	}
}

// detectFormat inspects raw bytes to determine whether they contain a JSON
// report or NDJSON events by checking the first non-blank line.
//
// A JSON report is identified by a top-level "version" key; an NDJSON event
// by an "event_type" key. Multi-line JSON (pretty-printed) that cannot be
// parsed as a single-line object defaults to FormatJSON.
func detectFormat(data []byte) (Format, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, MaxLineBytes), MaxLineBytes)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) > 0 {
			return detectLineFormat(line), nil
		}
	}

	err := scanner.Err()
	if err != nil {
		return FormatAuto, fmt.Errorf("scan for format detection: %w", err)
	}

	return FormatAuto, errNoContent
}

// detectLineFormat inspects a single JSON line for Report vs Event keys.
func detectLineFormat(line []byte) Format {
	var probe struct {
		Version   string `json:"version"`
		EventType string `json:"event_type"`
	}

	err := json.Unmarshal(line, &probe)
	if err != nil {
		// Not valid single-line JSON — probably a multi-line JSON Report.
		return FormatJSON
	}

	if probe.Version != "" {
		return FormatJSON
	}

	if probe.EventType != "" {
		return FormatNDJSON
	}

	// Default: single-line object without version or event_type.
	return FormatNDJSON
}

// LoadOption configures LoadReport behavior.
type LoadOption func(*loadConfig)

type loadConfig struct {
	format Format
}

// WithFormat forces a specific format instead of auto-detection.
func WithFormat(format Format) LoadOption {
	return func(cfg *loadConfig) { cfg.format = format }
}

// LoadReport reads a report from a file path, auto-detecting the format.
//
// JSON files (single Report object) are loaded via MigrateReport, which
// handles both v0.1.0 and v0.2.0 schemas. NDJSON files (line-delimited
// events) are loaded via ReadEvents + ReplayEvents.
//
// Returns the loaded Report, the detected Format, and any error.
func LoadReport(path string, opts ...LoadOption) (Report, Format, error) {
	cfg := loadConfig{format: FormatAuto}
	for _, opt := range opts {
		opt(&cfg)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Report{}, FormatAuto, fmt.Errorf("read %q: %w", path, err)
	}

	return LoadReportFromBytes(data, cfg.format)
}

// LoadReportFromReader reads a report from an io.Reader.
// If format is FormatAuto, the entire content is read and the format is
// detected by inspecting the first non-blank line for a "version" key
// (JSON Report) or "event_type" key (NDJSON event).
func LoadReportFromReader(reader io.Reader, format Format) (Report, Format, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return Report{}, FormatAuto, fmt.Errorf("read report data: %w", err)
	}

	return LoadReportFromBytes(data, format)
}

// LoadReportFromBytes loads a report from raw bytes with format detection.
func LoadReportFromBytes(data []byte, format Format) (Report, Format, error) {
	if format == FormatAuto {
		detected, err := detectFormat(data)
		if err != nil {
			if errors.Is(err, errNoContent) {
				return Report{}, FormatAuto, ErrEmpty
			}

			return Report{}, FormatAuto, fmt.Errorf("detect format: %w", err)
		}

		format = detected
	}

	switch format {
	case FormatJSON:
		return loadJSONFromBytes(data)
	case FormatNDJSON:
		return loadNDJSONFromBytes(data)
	default:
		return Report{}, FormatAuto, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
}

func loadJSONFromBytes(data []byte) (Report, Format, error) {
	report, err := MigrateReport(data)
	if err != nil {
		return Report{}, FormatJSON, err
	}

	return report, FormatJSON, nil
}

func loadNDJSONFromBytes(data []byte) (Report, Format, error) {
	events, err := ReadEvents(strings.NewReader(string(data)))
	if err != nil {
		return Report{}, FormatNDJSON, err
	}

	report, err := ReplayEvents(events)
	if err != nil {
		return Report{}, FormatNDJSON, err
	}

	return report, FormatNDJSON, nil
}

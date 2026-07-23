package auditlog

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/larsartmann/auditlog-core/loader"
)

// errUnsupportedFormat is returned when the format value is not recognized.
var errUnsupportedFormat = errors.New("unsupported format")

// Format identifies the serialization format of a report file.
// Re-exported from auditlog-core/loader so existing callers are unaffected.
type Format = loader.Format

// Format constants (re-exported from auditlog-core/loader).
const (
	// FormatAuto auto-detects JSON vs NDJSON by inspecting the first line.
	FormatAuto = loader.FormatAuto
	// FormatJSON is a single JSON Report object (contains "version" key).
	FormatJSON = loader.FormatJSON
	// FormatNDJSON is newline-delimited Event objects (contains "event_type" key).
	FormatNDJSON = loader.FormatNDJSON
)

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
//
//nolint:gosec // G304: path is user-provided CLI input, not a security risk
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
		detected, err := loader.Detect(data)
		if err != nil {
			if errors.Is(err, loader.ErrNoContent) {
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
		return Report{}, FormatAuto, fmt.Errorf("%w: %s", errUnsupportedFormat, format)
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

package main

import (
	"fmt"
	"io"
	"os"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// loadFile loads a report from path (auto-detecting JSON vs NDJSON).
// A path of "-" reads from stdin. Returns an error so callers can produce
// consistent error messages with the subcommand prefix.
func loadFile(path string) (auditlog.Report, error) {
	if path == "-" {
		return loadFromReader(os.Stdin, "stdin")
	}

	report, _, err := auditlog.LoadReport(path)
	if err != nil {
		return auditlog.Report{}, fmt.Errorf("load %s: %w", path, err)
	}

	return report, nil
}

// loadFromReader loads from any io.Reader using auto-detection.
func loadFromReader(r io.Reader, label string) (auditlog.Report, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return auditlog.Report{}, fmt.Errorf("read %s: %w", label, err)
	}

	report, _, err := auditlog.LoadReportFromBytes(data, auditlog.FormatAuto)
	if err != nil {
		return auditlog.Report{}, fmt.Errorf("load %s: %w", label, err)
	}

	return report, nil
}

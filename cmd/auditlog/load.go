package main

import (
	"fmt"
	"os"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// loadFile loads a report from path (auto-detecting JSON vs NDJSON).
// A path of "-" reads from stdin. Returns an error so callers can produce
// consistent error messages with the subcommand prefix.
func loadFile(path string) (auditlog.Report, error) {
	if path == "-" {
		report, _, err := auditlog.LoadReportFromReader(os.Stdin, auditlog.FormatAuto)
		if err != nil {
			return auditlog.Report{}, fmt.Errorf("load stdin: %w", err)
		}

		return report, nil
	}

	report, _, err := auditlog.LoadReport(path)
	if err != nil {
		return auditlog.Report{}, fmt.Errorf("load %s: %w", path, err)
	}

	return report, nil
}

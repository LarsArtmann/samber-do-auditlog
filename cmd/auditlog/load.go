package main

import (
	"errors"
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

// parseAndLoadSingleReport parses args for a single-report subcommand and
// loads the report at the given positional index. Returns the report, its
// source path, and any error. The expectedNArg guard ensures exactly N
// positional arguments (use 1 for `cmd <file>`, 2 for `cmd <a> <b>`).
func parseAndLoadSingleReport(name string, args []string, expectedNArg int, usage string) (auditlog.Report, string, error) {
	fs := newFlagSet(name)

	if err := fs.Parse(args); err != nil {
		return auditlog.Report{}, "", err
	}

	if fs.NArg() != expectedNArg {
		return auditlog.Report{}, "", errors.New(usage)
	}

	path := fs.Arg(0)
	report, err := loadFile(path)
	if err != nil {
		return auditlog.Report{}, "", err
	}

	return report, path, nil
}

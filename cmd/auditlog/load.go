package main

import (
	"errors"
	"flag"
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

// parseFlagSet parses flags for a subcommand and verifies the positional
// argument count. Returns the flag set so callers can read the positional
// arguments themselves (used for both single-report subcommands like `info`
// and two-report subcommands like `diff`).
func parseFlagSet(name string, args []string, expectedNArg int, usage string) (*flag.FlagSet, error) {
	fs := newFlagSet(name)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if fs.NArg() != expectedNArg {
		return nil, errors.New(usage)
	}

	return fs, nil
}

// loadSingleReportSubcommand is the common preamble for subcommands that take
// exactly one positional report file: parse flags, enforce the arg count,
// load the report, and return it together with the source path. The usage
// string is used in the "usage: ..." error returned when the arg count is wrong.
func loadSingleReportSubcommand(name string, args []string, usage string) (auditlog.Report, string, error) {
	fs, err := parseFlagSet(name, args, 1, usage)
	if err != nil {
		return auditlog.Report{}, "", err
	}

	report, err := loadFile(fs.Arg(0))
	if err != nil {
		return auditlog.Report{}, "", err
	}

	return report, fs.Arg(0), nil
}

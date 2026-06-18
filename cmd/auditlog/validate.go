package main

import (
	"errors"
	"flag"
	"fmt"
)

// runValidate loads a report and verifies internal consistency via Validate().
func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("usage: auditlog validate <file>")
	}

	path := fs.Arg(0)

	report, err := loadFile(path)
	if err != nil {
		return err
	}

	if err := report.Validate(); err != nil {
		return fmt.Errorf("%s: invalid: %w", path, err)
	}

	fmt.Printf("OK: %s is valid (%d services, %d events, %d scopes)\n",
		path, report.ServiceCount, report.EventCount, report.ScopeCount)

	return nil
}

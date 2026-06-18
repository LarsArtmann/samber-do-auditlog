package main

import (
	"errors"
	"flag"
	"fmt"
)

// runValidate loads a report and verifies internal consistency via Validate().
func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return errors.New("usage: auditlog validate <file>")
	}

	report := loadFile(fs.Arg(0))

	if err := report.Validate(); err != nil {
		return fmt.Errorf("invalid: %w", err)
	}

	fmt.Printf("OK: %s is valid (%d services, %d events, %d scopes)\n",
		fs.Arg(0), report.ServiceCount, report.EventCount, report.ScopeCount)

	return nil
}

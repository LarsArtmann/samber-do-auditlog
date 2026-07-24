package main

import (
	"fmt"
)

// runValidate loads a report and verifies internal consistency via Validate().
func runValidate(args []string) error {
	report, path, err := loadSingleReportSubcommand("validate", args, "usage: auditlog validate <file>")
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

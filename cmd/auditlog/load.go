package main

import (
	"fmt"
	"os"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// loadFile loads a report from path (auto-detecting JSON vs NDJSON) and exits
// the process with a user-friendly message on failure.
func loadFile(path string) auditlog.Report {
	report, _, err := auditlog.LoadReport(path)
	if err != nil {
		failf("load %s: %v", path, err)
	}

	return report
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "auditlog: "+format+"\n", args...)
	os.Exit(1)
}

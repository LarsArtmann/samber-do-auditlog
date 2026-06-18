package main

import (
	"flag"
	"fmt"
	"os"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// runSchema prints the canonical JSON Schema for the report format.
func runSchema(args []string) error {
	fs := flag.NewFlagSet("schema", flag.ExitOnError)

	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, auditlog.JSONSchema())

	return nil
}

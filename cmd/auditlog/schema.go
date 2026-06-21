package main

import (
	"errors"
	"fmt"
	"os"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// runSchema prints the canonical JSON Schema for the report format.
func runSchema(args []string) error {
	fs := newFlagSet("schema")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() > 0 {
		return errors.New("usage: auditlog schema (takes no arguments)")
	}

	_, err := fmt.Fprintln(os.Stdout, auditlog.JSONSchema())

	return err
}

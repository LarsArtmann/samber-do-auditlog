// Package main is the JSON Schema generator for the auditlog report format.
//
// It reflects over the library's public types (Report, Event, ServiceInfo, etc.)
// and emits schema/report.schema.json so report consumers can validate exported
// JSON without the schema drifting from the Go types.
//
// Usage: go run ./cmd/genschema
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"
	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func main() {
	r := new(jsonschema.Reflector)
	// Keep the schema self-contained (no external $ref to a definitions map
	// is required for the top-level document) while still inlining complex
	// sub-types for human readability.
	r.DoNotReference = false
	r.ExpandedStruct = true

	schema := r.Reflect(auditlog.Report{})
	schema.ID = "https://github.com/larsartmann/samber-do-auditlog/schema/report.schema.json"
	schema.Title = "do-auditlog Report"
	schema.Description = "Report exported by the samber-do-auditlog plugin."

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		die("marshal schema: %v\n", err)
	}

	data = append(data, '\n')

	outPath := filepath.Join("schema", "report.schema.json")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		die("mkdir: %v\n", err)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		die("write schema: %v\n", err)
	}

	fmt.Printf("wrote %s (%d bytes)\n", outPath, len(data))
}

// die prints a formatted message to stderr and exits with code 1.
// Centralizes the "print + exit" pattern shared by every error path in main.
func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

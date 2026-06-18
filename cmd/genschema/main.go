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
		fmt.Fprintf(os.Stderr, "marshal schema: %v\n", err)
		os.Exit(1)
	}

	data = append(data, '\n')

	outPath := filepath.Join("schema", "report.schema.json")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("wrote %s (%d bytes)\n", outPath, len(data))
}

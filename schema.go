package auditlog

//go:generate go run ./cmd/genschema

import _ "embed"

// reportSchemaJSON is the canonical JSON Schema (Draft 2020-12) for the Report
// format, generated from the Go types by cmd/genschema. Embedding it keeps the
// schema available to library consumers (and the CLI validate command) without
// a separate file lookup at runtime.
//
//go:embed schema/report.schema.json
var reportSchemaJSON string

// JSONSchema returns the canonical JSON Schema (Draft 2020-12) describing the
// Report export format, derived from this package's public types. Consumers can
// use it to validate exported report JSON or to generate client code.
func JSONSchema() string { return reportSchemaJSON }

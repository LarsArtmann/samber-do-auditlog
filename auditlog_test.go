// Package auditlog_test contains the test suite for the auditlog package.
//
// Tests are organized by feature area into separate _test.go files:
//
//   helpers_test.go            — shared test types, providers, assertions
//   plugin_basic_test.go       — enabled/disabled/env var/ContainerID/version
//   plugin_lifecycle_test.go   — registration, invocation, ordering, dependencies
//   plugin_errors_test.go      — service status, provider/shutdown errors
//   plugin_export_test.go      — JSON/NDJSON/HTML export and writer errors
//   plugin_scope_test.go       — scope tree, scope ID, resolve service scope
//   plugin_provider_test.go    — service type capture, capability tracking
//   plugin_html_test.go        — HTML output structure and tab content
//   healthcheck_basic_test.go  — core health check tests
//   healthcheck_export_test.go — health check reporting, callbacks, edge cases
//   type_method_test.go        — Event/ServiceInfo/ServiceRef/ServiceStatus methods
//   report_query_test.go       — ServiceBy*, EventsBy*, Failed, Unhealthy, Index
//   report_filter_test.go      — all Filtered* tests
//   diagram_test.go            — Mermaid and PlantUML output
//   migration_test.go          — MigrateReport tests
//   extra_test.go              — EventHandler, RealWorldScenario, EventsCount
//   benchmarks_test.go         — benchmarks
package auditlog_test

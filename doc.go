// Package auditlog provides an audit-log plugin for samber/do v2 that tracks
// every service registration, invocation, shutdown, and health check with timestamps,
// dependency graph inference, build duration measurement, and provider type tracking.
//
// Each Event carries a ServiceType (ProviderType) identifying the provider kind
// (lazy, eager, transient, alias). ServiceInfo includes IsHealthchecker and
// IsShutdowner capabilities detected via do.ExplainInjector.
//
// Config.Validate() checks configuration constraints. Export formats include JSON
// reports, NDJSON event streams, CSV/TSV, self-contained HTML, Mermaid,
// PlantUML, Graphviz DOT, and D2 diagrams.
//
// # Build requirement
//
// This module builds on Go 1.23+ with no GOEXPERIMENT and no third-party
// runtime dependencies (samber/do/v2 is the only runtime require). Do NOT set
// GOEXPERIMENT=jsonv2 here — a stale ambient value breaks the go1.23
// toolchain. (The upstream samber-do-auditlog master line requires it on Go
// 1.26.x; that constraint does not apply to this module.)
package auditlog

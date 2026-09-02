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
// Building (not importing) this library requires GOEXPERIMENT=jsonv2 on Go
// 1.26.x, because a transitive dependency (go-output) uses encoding/json/v2
// features. Set the env var or the build fails with "package encoding/json/v2
// is not in std". The flag will not be needed once Go 1.27 stabilizes json/v2.
package auditlog

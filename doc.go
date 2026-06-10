// Package auditlog provides an audit-log plugin for samber/do v2 that tracks
// every service registration, invocation, shutdown, and health check with timestamps,
// dependency graph inference, build duration measurement, and provider type tracking.
//
// Each Event carries a ServiceType (ProviderType) identifying the provider kind
// (lazy, eager, transient, alias). ServiceInfo includes IsHealthchecker and
// IsShutdowner capabilities detected via do.ExplainInjector.
//
// Config.Validate() checks configuration constraints. Export formats include JSON
// reports, NDJSON event streams, and a self-contained HTML visualization with a
// force-directed dependency graph.
package auditlog

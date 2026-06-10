// Package auditlog provides an audit-log plugin for samber/do v2 that tracks
// every service registration, invocation, shutdown, and health check with timestamps,
// dependency graph inference, and build duration measurement.
//
// Export formats include JSON reports, NDJSON event streams, and a
// self-contained HTML visualization with a force-directed dependency graph.
package auditlog

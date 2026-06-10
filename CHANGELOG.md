# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Security

- **Fix XSS in HTML dependency rendering**: `d.service_name` and `s.status` are now escaped via `esc()` before interpolation
- **Add CSP meta tag**: `Content-Security-Policy` restricts the HTML page to inline styles/scripts and Google Fonts
- **Expand fuzz coverage**: 3 fuzz targets covering malicious service names, error strings, and dependency chains with 6+ injection vector checks

### Fixed

- **Fix broken Events tab**: `allEvents` was referenced but undefined — the Events tab rendered nothing. Now shows full event table with sequence, timestamp, type badge, provider badge, phase icon, scope, service, duration, and error
- **MigrateReport validation**: rejects empty input and missing version field; preserves original `ExportedAt`; passes through unchanged if already at current schema
- **writeToFile error handling**: `Close()` errors are now returned properly instead of silently discarded
- **Non-deterministic scope tree**: `buildScopeTreeLocked` now iterates scopes in sorted order
- **HTML tooltip positioning**: error tooltip used `position:absolute` with viewport-relative coords — now `position:fixed`
- **HTML `esc()` function**: now escapes `"` and `'` to prevent broken `data-error` attributes on quoted messages
- **Duplicate `data-error` attributes**: when both invocation and shutdown errors exist, they are now concatenated into a single attribute

### Changed

- **Config.Validate()** now validates `ContainerID` for path separators (`/` and `\`)
- **RootScopeName constant**: replaces the `"[root]"` magic string
- **OnBeforeShutdown** now calls `recordScope` for consistency with other `OnBefore*` hooks
- **HTML template** uses server-computed `s.status` instead of client-side derivation
- **Complete HTML visualization rewrite**: dark-themed dashboard with services table, scope tree, Sugiyama DAG graph, dual timeline, events table, stat cards, responsive layout, and keyboard navigation
- **Key format consolidation**: `serviceKey(scopeID, serviceName)` is now the single canonical function
- **Deduplicated duration helpers**: `sumBuildDurationMs` and `sumShutdownDurationMs` merged into generic `sumDurationField()`
- **Removed `countScopesLocked()` wrapper**: inlined `len(r.scopes)` in `BuildReport()`
- **DependencyRef renamed to ServiceRef**: embedded in `Event` and `ServiceInfo` as the single source of truth for service identity
- **Stack pop optimization**: LIFO fast path checks last element first (O(1) common case)
- **Expanded godoc**: 7 exported methods now have proper documentation comments

### Added

- **ServiceStatus type**: computed lifecycle state (`registered`, `active`, `invocation_error`, `shutdown`, `shutdown_error`) on `ServiceInfo`
- **Event convenience methods**: `IsRegistration()`, `IsInvocation()`, `IsShutdown()`, `IsBefore()`, `IsAfter()`
- **ServiceRef.String()**: human-readable `"scope/name"` format
- **ServiceStatus.IsError()**: `true` for invocation or shutdown errors
- **Report query methods**: `ServiceByName`, `ServiceByRef`, `ServicesByScope`, `EventsByService`, `EventsByRef`, `EventsByType`, `FailedServices`, `UnhealthyServices`
- **ProviderType named type**: `lazy`, `eager`, `transient`, `alias` with `String()` and `Icon()` methods
- **Event.ServiceType field**: provider type carried on every event
- **Capability detection**: `IsHealthchecker` / `IsShutdowner` populated via `do.ExplainInjector` in `BuildReport()`
- **Config.OnEvent callback**: real-time event streaming, called outside the mutex
- **Config.Validate()**: configuration validation entry point
- **Health check auditing**: `RecordHealthCheck()` / `RecordHealthCheckWithContext()` wrap `injector.HealthCheckWithContext()`
- **Health check events**: `EventTypeHealthCheck` with `IsHealthCheck()` (PhaseAfter only, no DurationMs)
- **Health check service fields**: `LastHealthCheckAt`, `HealthCheckError`, `HealthCheckCount`
- **Health check report fields**: `HealthCheckSucceeded`, `HealthCheckedCount` on `Report`
- **Health check HTML visualization**: health column in services table, amber event badge, filter chip, conditional stat card
- **Report filtering**: `Report.Filtered(opts...)` with `WithServicesByName`, `WithServicesByType`, `WithEventsByType`, `WithTimeRange`, `WithScope`
- **Plugin.ReportFiltered**: convenience method for filtered reports
- **ExportFilteredToFile**: write filtered JSON report to file
- **Mermaid export**: `Report.WriteMermaid(writer)` for flowchart TD diagrams
- **Type helpers**: `ProviderType.IsKnown()`, `ServiceRef.IsRoot()`, `Event.HasError()`, `ServiceInfo.HasHealthError()`
- **Schema migration**: `MigrateReport([]byte)` upgrades v0.1.0 JSON to current schema
- **Single-lock Recorder**: 4 mutexes → 1 `RWMutex` + 2 atomics; 23% faster, 50% fewer allocations
- **Locking protocol docs**: comprehensive godoc on Recorder write/read paths and deadlock risk
- **Capability emojis** in HTML: services table, scope tree, graph nodes, timeline
- **Provider column** in HTML Events tab
- **Godoc examples**: 7 runnable `Example*` functions for pkg.go.dev
- **HTML fuzz test**: `FuzzPluginHTML` verifies templ XSS escaping
- **Comprehensive example** (`example/main.go`): 19 samber/do v2 features with self-checking checklist
- **Documentation**: `FEATURES.md`, `TODO_LIST.md`, `docs/DOMAIN_LANGUAGE.md`, architecture D2 diagrams, deepening opportunities report

## [0.1.0] - 2026-06-09

### Added

- Initial release
- Audit-log plugin for samber/do v2 with lifecycle hooks
- Event capture for registration, invocation, and shutdown
- Stack-based dependency graph inference
- Reverse dependency computation
- Scope tree building with per-scope service lists
- JSON report export
- NDJSON event stream export
- Self-contained HTML visualization with force-directed graph
- Environment variable toggle (`DO_AUDITLOG_ENABLED`)
- Zero-cost disabled mode
- Strict golangci-lint configuration
- Benchmarks for hook overhead measurement

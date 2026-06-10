# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- `ServiceStatus` type with computed `status` field on `ServiceInfo` — derives lifecycle state (registered, active, invocation_error, shutdown, shutdown_error) from existing fields
- `//go:generate templ generate` directive in `html.go` for self-documenting templ regeneration
- Tests for `ServiceStatus` computation across lifecycle states
- Comprehensive codebase analysis: code quality scan, naming review, full code review, architecture review, architecture visualization, features audit, TODO list builder
- `FEATURES.md` — honest feature inventory with verified status
- `TODO_LIST.md` — prioritized improvement list
- `docs/DOMAIN_LANGUAGE.md` — filled with 17 project domain terms
- Architecture diagrams (D2) for current and improved state
- Architecture deepening opportunities HTML report
- `ServiceRef` type (renamed from `DependencyRef`) — embedded in `Event` and `ServiceInfo` for single source of truth on service identity. JSON output unchanged (embedded struct fields are flattened)
- Event convenience methods: `IsRegistration()`, `IsInvocation()`, `IsShutdown()`, `IsBefore()`, `IsAfter()`
- `Config.OnEvent` callback for real-time event streaming — called after each event is captured, outside the mutex, enabling live observability without polling
- `ServiceRef.String()` — human-readable `"scope/name"` format for compact display
- `ServiceStatus.IsError()` — `true` for invocation_error or shutdown_error
- `Report.ServiceByName(name)` — find service by exact name
- `Report.EventsByType(t)` — filter events by EventType
- `Report.FailedServices()` — all services with invocation or shutdown errors
- `ProviderType` named type with `String()` and `Icon()` methods — represents the DI provider kind (lazy, eager, transient, alias)
- `Event.ServiceType` field — provider type carried on every event
- `IsHealthchecker`/`IsShutdowner` capability tracking via `do.ExplainInjector` in `BuildReport()`
- `Config.Validate()` method for configuration validation
- `enrichCapabilities()` and `buildCapabilityMap()` for capability detection outside the recorder mutex
- Capability emojis in HTML: services table, scope tree, graph nodes, and timeline
- Provider column in HTML Events tab
- 62 total tests covering ProviderType, capabilities, Event.ServiceType, Config.Validate, and error paths
- Comprehensive example (`example/main.go`) demonstrating 18 samber/do v2 features with self-checking feature checklist

### Changed

- `OnBeforeShutdown` now calls `recordScope` for consistency with other `OnBefore*` hooks
- `buildScopeTreeLocked` now uses sorted scope iteration for deterministic output
- HTML template now uses server-computed `s.status` instead of client-side derivation
- Complete HTML visualization rewrite: services table with status badges, shutdown duration,
  reverse dependencies, search filter, timestamps on hover; stats cards with schema version,
  total build duration, error count; events table with full type names and filter chips;
  dependency graph with status-colored nodes and SVG tooltips; scopes tab with collapsible
  tree; timeline with dual build+shutdown bars; responsive layout, keyboard nav, footer
- `stackEntry` and `serviceRecord` now have `key()` methods centralizing the scope/service key format
- `OnBeforeInvocation` computes `depKey` before acquiring the stack lock
- Consolidated 3 key format implementations into single `serviceKey()` function, removing `stackEntry.key()` and `serviceRecord.key()` methods
- Deduplicated `sumBuildDurationMs`/`sumShutdownDurationMs` into generic `sumDurationField()` with thin wrappers
- Removed `countScopesLocked()` wrapper — inlined `len(r.scopes)` in `BuildReport()`
- `DependencyRef` renamed to `ServiceRef` and embedded in `Event` and `ServiceInfo` (breaking change, pre-1.0)

### Fixed

- Non-deterministic scope tree construction due to random map iteration order
- Dead `classList &&` check in HTML template JavaScript
- Test file used custom `contains`/`searchString` instead of `strings.Contains`
- Error tooltip used `position:absolute` with viewport-relative coords — wrong position when scrolled (now `position:fixed`)
- HTML `esc()` function did not escape `"` or `'` — broken `data-error` attributes on error messages containing quotes (now regex-based)
- Status badge emitted duplicate `data-error` attributes when both invocation and shutdown errors exist — now concatenated into single attribute

## [0.1.0] - 2026-01-01

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

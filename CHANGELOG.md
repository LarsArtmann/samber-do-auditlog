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

### Fixed

- Non-deterministic scope tree construction due to random map iteration order
- Dead `classList &&` check in HTML template JavaScript
- Test file used custom `contains`/`searchString` instead of `strings.Contains`

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

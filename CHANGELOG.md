# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> **Release vs. schema versions.** Release tags follow `v0.0.x`. The report
> `schema_version` (currently `0.2.0`, see `types.go`) is a **separate**, independent
> version for the JSON report format and is upgraded via `MigrateReport`. The two
> version numbers are unrelated.

## [Unreleased]

### Added

- **CI pipeline** (`.github/workflows/ci.yml`): four parallel jobs — test (with
  `-race`), lint (golangci-lint v2.12.2), vulnerability scan (govulncheck), and
  stale-generation check (catches `html_templ.go` drift).
- **`Report.WriteNDJSON(writer)`**: streams events as NDJSON directly from the
  report without a defensive copy.
- **`Report.WriteJSON(writer)`**: writes the full report as indented JSON.
- **`Report.Diff(other Report) DiffResult`**: structural comparison of two
  reports (added/removed/changed services, event count delta).
- **`flake.nix` devShell**: pins Go 1.26.3, golangci-lint, govulncheck, templ,
  golines for contributor reproducibility.
- **`BENCHMARKS.md`**: post-v0.0.3 baseline of all 13 benchmarks for regression
  detection.
- **`STABILITY.md`**: 0.x API stability promise (stable vs evolving vs internal).
- **OTel reference example** (`docs/examples/otel-bridge.md`): shows how to
  bridge `Config.OnEvent` to OpenTelemetry spans without adding a dependency.
- **"Releasing" section** in `CONTRIBUTING.md` documenting the release procedure
  and the release-vs-schema-version distinction.

### Tests

- `deriveServiceStatus` exhaustive 16-case property test (all 2^4 input
  combinations + priority ordering).
- `MaxEvents` concurrency stress test (50 goroutines, 20x repeat, `-race`).
- Atomic-write crash path tests (rename failure + write error cleanup).
- Migration full round-trip test (v0.2.0 → downgrade → migrate → assert
  equality).
- Diff tests (identical, added/removed, changed, new-error).

## [0.0.3] - 2026-06-17

First release cut from a clean lint baseline. A complete HTML redesign, new
observability and memory controls, export performance work, and a codebase-wide
split of the monolithic source files into focused modules.

### Breaking

- **`New(Config)` now returns `(*Plugin, error)`**: `Config.Validate()` is enforced
  at construction time. All callers must handle the returned error. Tests use the
  new `mustNew()` helper.

### Added

- **Memory-bounded event capture**: `Config.MaxEvents` caps the number of stored
  events; `Config.InitialEventCapacity` pre-sizes the events slice to avoid
  reallocation; `Plugin.DroppedEventCount()` reports how many events were discarded
  once the cap is reached. Prevents OOM in long-running processes.
- **`Report.Validate()`**: verifies internal consistency of the denormalized count
  fields (`EventCount`, `ServiceCount`, `ScopeCount`, `HealthCheckedCount`) against
  the actual data.
- **`BuildTypeMetadata()` / `TypeMetadata`**: single source of truth for provider
  icons, status icons, event labels, and event colors, injected into the HTML via
  `@templ.JSONScript` so the client reads metadata instead of hardcoding constants.
- **HTML pagination**: the services table shows the first 50 rows and the events
  table the first 100; a "show all" button reveals the remainder. Search and filter
  bypass the limits.
- **Touch support**: one-finger pan and two-finger pinch-zoom on the dependency graph.
- **Accessibility**: ARIA roles, labels, and `sr-only` text across the HTML report.
- **Diagram themes**: Mermaid `%%{init}%%` and PlantUML `skinparam` directives using
  the warm amber palette.
- **Shared diagram formatter**: a `diagramFormatter` interface removes duplication
  between the Mermaid and PlantUML exporters.
- **Hardened CSP**: `base-uri 'none'; frame-ancestors 'none'` to block base injection
  and clickjacking.
- **Robust fuzz XSS**: `stripJSONScripts` replaces the old character-by-character
  parser; three fuzz targets now cover service names, error strings, and dependency
  chains with six-plus injection-vector checks.
- **`mustNew()` test helper** and shared providers/assertions in `helpers_test.go`.
- Security and archive documentation, an integration test, and planning/status reports.

### Performance

- **Atomic file writes**: exports write to a temp file in the target directory then
  `os.Rename` it into place, so a crash leaves the previous file intact instead of a
  partial write.
- **Zero-allocation struct map key** for `serviceKey`, removing the per-key string
  concatenation allocation.
- **Buffered export I/O**: a 64 KB `bufio` block batches writes, cutting syscall
  count 10–100x compared to writing straight to `os.File`.

### Changed

- **Complete HTML redesign**: warm amber "Container Telemetry" aesthetic with a
  lifecycle waveform signature element, a five-tab layout (Services / Scopes /
  Graph / Timeline / Events), stat cards, and keyboard navigation.
- **Pinned Go 1.26.3** in `go.mod` and the lint config.

### Fixed

- Bugs, lint failures, and XSS gaps surfaced by a full code review.

### Refactor

- **Monolith split**: `recorder.go` decomposed into focused modules — `hooks.go`,
  `report.go`, `report_builder.go`, `report_helpers.go`, `service.go`, `event.go`,
  `export.go`, `healthcheck.go`, `filter.go`, `metadata.go`.
- **Test split**: the monolithic `auditlog_test.go` split into 14 feature-focused
  test files.
- **Example split**: `example/main.go` split into `register.go`, `services.go`, and
  `summary.go`.
- Removed dead code and stale comments.

## [0.0.2] - 2026-06-11

### Fixed

- Commit the generated `html_templ.go` so the project builds without running the
  `templ` generator (Nix build compatibility).

## [0.0.1] - 2026-06-10

First alpha release. An audit-log plugin for samber/do v2 that records every DI
container lifecycle event with timestamps, dependency-graph inference, build-duration
tracking, health-check auditing, and export to JSON / NDJSON / self-contained HTML.

### Added

- Drop-in plugin: `New(Config)` + `Opts()` → one-line integration.
- Service registration, invocation, shutdown, and health-check tracking.
- Stack-based dependency-graph inference with reverse dependencies.
- Scope tree with hierarchical visualization.
- Provider-type detection (`lazy` / `eager` / `transient` / `alias`).
- Configurable `OnEvent` callback for real-time observability.
- JSON, NDJSON, Mermaid, PlantUML, and self-contained HTML export.
- Report filtering (`Report.Filtered`) with five filter options.
- Schema migration (`MigrateReport`) upgrading v0.1.0 JSON reports to the current
  schema.
- Zero-cost disabled mode via `DO_AUDITLOG_ENABLED`.
- Concurrent-safe single-lock `Recorder` design.
- ~95% test coverage, 140 tests, 11 benchmarks.
- XSS-hardened HTML with a Content-Security-Policy.
- Strict `golangci-lint` configuration.

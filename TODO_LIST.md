# TODO List

Comprehensive list of improvement tasks, verified against actual code state.
Last updated: 2026-06-10

---

## Priority 1 — Features (Future)

- [ ] **Add `ReportOption` functional options** for filtering reports by service name, time range, event type. Enables efficient consumption for large containers. Currently `Report()` returns everything.

## Priority 2 — Polish

- [x] **`Config.Validate() error`** — Added as forward-compatible API placeholder. Currently always returns nil.

## Priority 3 — Consider

- [ ] **Versioned report schema with migration** — `SchemaVersion` exists but has no migration function. Postpone until v1.0 planning.
- [ ] **Additional export formats** — Mermaid diagram, PlantUML. Only if users request them.

## Not Planned (Explicitly Rejected)

- **Multi-module split** — Project is too small (1 package). Revisit at 5+ packages.
- **External storage backends** — File and io.Writer exports are sufficient.
- **Prometheus/OpenTelemetry integration** — Out of scope. Use OnEvent callback when available.
- **`samber/lo` dependency** — Current stdlib `slices`/`cmp` usage is sufficient for this project size.
- **`encoding/json/v2` migration** — Current `encoding/json` works fine. Risk of breaking JSON output format for consumers.

---

## Completed (2026-06-10)

- [x] Fill DOMAIN_LANGUAGE.md with actual domain terms
- [x] Fix `OnBeforeShutdown` missing `recordScope` call
- [x] Fix non-deterministic scope tree construction in `buildScopeTreeLocked`
- [x] Add `ServiceStatus` type with computed field on `ServiceInfo`
- [x] Update HTML template to use server-computed status
- [x] Add `//go:generate templ generate` directive in `html.go`
- [x] Remove dead `classList &&` check in HTML template JS
- [x] Replace custom `contains`/`searchString` with `strings.Contains` in tests
- [x] Add shutdown error test
- [x] Add ServiceStatus computation tests
- [x] Update CHANGELOG.md with actual development history
- [x] Move `depKey` computation before lock in `OnBeforeInvocation`
- [x] Comprehensive codebase analysis (code quality, naming, architecture, features)
- [x] Complete HTML visualization rewrite (T2-T8): services table with status badges, shutdown duration, reverse deps, search; stats cards; events with filter chips; graph improvements; scopes tab; timeline dual bars; responsive UX
- [x] Fix error tooltip positioning: `position:fixed` for scroll support
- [x] Fix HTML esc() XSS: escape quotes for attribute safety, improve performance with regex
- [x] Fix error tooltip: concatenate invocation+shutdown errors into single data-error attribute
- [x] Remove `countScopesLocked` wrapper — inline `len(r.scopes)`
- [x] Consolidate key format: `serviceKey()` replaces `scopeKey`/`stackEntry.key`/`serviceRecord.key`
- [x] Deduplicate `sumBuildDurationMs`/`sumShutdownDurationMs` into `sumDurationField`
- [x] Rename `DependencyRef` to `ServiceRef` and embed in `Event`/`ServiceInfo`
- [x] Add Event convenience methods: `IsRegistration`, `IsInvocation`, `IsShutdown`, `IsBefore`, `IsAfter`
- [x] Add `Config.OnEvent` callback for real-time event streaming
- [x] Add health check auditing: EventTypeHealthCheck, RecordHealthCheck/RecordHealthCheckWithContext
- [x] Add ProviderType named type with Icon()/String() methods
- [x] Add IsHealthchecker/IsShutdowner fields to ServiceInfo
- [x] Add health check HTML visualization (health column, event badge, stat card)
- [x] Fix health check duration bug: remove misleading HealthCheckDurationMs (per-service timing unavailable)
- [x] Fix HealthCheckSucceeded semantics: false when no health checks ran
- [x] Refactor: extract newEventFromRef and newServiceRecordFromMeta helpers
- [x] Add test coverage for convenience methods (ServiceByName, FailedServices, IsError, String)
- [x] Add health check test coverage (OnEvent, phase, JSON/NDJSON export)
- [x] Add Event.ServiceType field with provider type per event
- [x] Add capability tracking via enrichCapabilities() in BuildReport()
- [x] Remove dead inferCapabilities code
- [x] Fix all golangci-lint issues: 0 issues across entire project (was 28)
- [x] Use maps.Copy in BuildReport for scope map copying
- [x] Simplify Config.Validate() to honest placeholder

## Completed (Historical)

- [x] Initial plugin structure with Config, New, Opts
- [x] Event capture for registration, invocation, shutdown
- [x] Stack-based dependency inference
- [x] Reverse dependency computation
- [x] Scope tree building
- [x] JSON report export
- [x] NDJSON event stream export
- [x] Self-contained HTML visualization with force-directed graph
- [x] Environment variable toggle (DO_AUDITLOG_ENABLED)
- [x] Zero-cost disabled mode
- [x] Strict golangci-lint configuration
- [x] External test package

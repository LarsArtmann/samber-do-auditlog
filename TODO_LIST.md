# TODO List

Comprehensive list of improvement tasks, verified against actual code state.
Last updated: 2026-06-10

---

## Priority 1 — Features (Future)

- [ ] **Add `ReportOption` functional options** for filtering reports by service name, time range, event type. Enables efficient consumption for large containers. Currently `Report()` returns everything.
- [ ] **Add `EventHandler` callback** in `Config` for real-time event streaming. `type EventHandler func(Event)` — called after each event is captured. Zero cost when nil. Enables live dashboards and metrics integration.

## Priority 2 — Polish

- [ ] **Add `Config.Validate() error`** method — Currently validation is ad-hoc in `New()`. A proper `Validate()` method would centralize it and make the API more extensible.

## Priority 3 — Consider

- [ ] **Versioned report schema with migration** — `SchemaVersion` exists but has no migration function. Postpone until v1.0 planning.
- [ ] **Additional export formats** — Mermaid diagram, PlantUML. Only if users request them.

## Not Planned (Explicitly Rejected)

- **Multi-module split** — Project is too small (1 package). Revisit at 5+ packages.
- **External storage backends** — File and io.Writer exports are sufficient.
- **Prometheus/OpenTelemetry integration** — Out of scope. Use EventHandler when available.
- **`samber/lo` dependency** — Current stdlib `slices`/`cmp` usage is sufficient for this project size.

---

## Completed (2026-06-10)

- [x] Fill DOMAIN_LANGUAGE.md with actual domain terms
- [x] Fix `OnBeforeShutdown` missing `recordScope` call
- [x] Fix non-deterministic scope tree construction in `buildScopeTreeLocked`
- [x] Add `ServiceStatus` type with computed field on `ServiceInfo`
- [x] Update HTML template to use server-computed status
- [x] Add `stackEntry.key()` and `serviceRecord.key()` methods
- [x] Add `//go:generate templ generate` directive in `html.go`
- [x] Remove dead `classList &&` check in HTML template JS
- [x] Replace custom `contains`/`searchString` with `strings.Contains` in tests
- [x] Add shutdown error test
- [x] Add ServiceStatus computation tests
- [x] Update CHANGELOG.md with actual development history
- [x] Move `depKey` computation before lock in `OnBeforeInvocation`
- [x] Comprehensive codebase analysis (code quality, naming, architecture, features)
- [x] Complete HTML visualization rewrite (T2-T8): services table with status badges, shutdown duration, reverse deps, search; stats cards; events with filter chips; graph improvements; scopes tab; timeline dual bars; responsive UX

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

# TODO List

Comprehensive list of improvement tasks, verified against actual code state.
Last updated: 2026-06-10

---

## Priority 0 — Documentation (Quick Wins)

- [ ] **Update CHANGELOG.md** — Current content is placeholder ("Initial release"). Should reflect actual development history: plugin creation, event capture, export formats, HTML visualization, scope tree, transient/value provider support, concurrent tests, etc.
- [x] ~~Fill DOMAIN_LANGUAGE.md with actual domain terms~~ — Done 2026-06-10

## Priority 1 — Code Quality

- [x] ~~Fix `parentKey` construction in `OnBeforeInvocation`~~ — Moved `depKey` computation before lock for consistency. Done 2026-06-10.
- [x] ~~Remove dead `classList &&` check in `html.templ` JS~~ — Done 2026-06-10.
- [x] ~~Replace custom `contains()`/`searchString()` with `strings.Contains` in tests~~ — Done 2026-06-10.
- [x] ~~Add concurrent access test for Recorder~~ — `TestPlugin_ConcurrentInvocations` already exists.
- [x] ~~Add empty container edge case test~~ — `TestPlugin_EmptyReport` already exists.
- [x] ~~Add `WriteReportJSON` test~~ — `TestPlugin_WriteReportJSON` already exists.
- [x] ~~Add `WriteEventsNDJSON` to writer test~~ — `TestPlugin_WriteEventsNDJSON` already exists.

## Priority 2 — Features (Future)

- [ ] **Add `ReportOption` functional options** for filtering reports by service name, time range, event type. Enables efficient consumption for large containers. Currently `Report()` returns everything.
- [ ] **Add `EventHandler` callback** in `Config` for real-time event streaming. `type EventHandler func(Event)` — called after each event is captured. Zero cost when nil. Enables live dashboards and metrics integration.
- [ ] **Add convenience methods on `Event`** — `IsRegistration()`, `IsInvocation()`, `IsShutdown()`, `IsBefore()`, `IsAfter()`. Simple boolean methods that improve readability at call sites.

## Priority 3 — Polish

- [ ] **Add `Config.Validate() error`** method — Currently validation is ad-hoc in `New()`. A proper `Validate()` method would centralize it and make the API more extensible.
- [ ] **Consider `stackEntry.key()` method** — The `scopeID + "/" + serviceName` format is used in multiple places. A method would centralize the format.
- [ ] **Upgrade templ dependency** — go.mod has v0.3.1020 but generator is v0.3.1036. Run `go get -u github.com/a-h/templ`.
- [ ] **Verify LICENSE file** — Ensure MIT license text is correct and complete.

## Not Planned (Explicitly Rejected)

- **Multi-module split** — Project is too small (1 package). Revisit at 5+ packages or when external consumers have conflicting dependency needs.
- **External storage backends** — File and io.Writer exports are sufficient. EventHandler callback covers streaming use cases.
- **Prometheus/OpenTelemetry integration** — Out of scope. Use EventHandler when available.

---

## Completed (Historical)

- [x] Initial plugin structure with Config, New, Opts
- [x] Event capture for registration, invocation, shutdown
- [x] Stack-based dependency inference
- [x] Reverse dependency computation
- [x] Scope tree building
- [x] JSON report export
- [x] NDJSON event stream export
- [x] Self-contained HTML visualization with D3-like force graph
- [x] Environment variable toggle (DO_AUDITLOG_ENABLED)
- [x] Zero-cost disabled mode
- [x] Strict golangci-lint configuration
- [x] External test package
- [x] Transient and value provider tests
- [x] Concurrent invocation test
- [x] Empty report test
- [x] Writer-based export tests
- [x] Comprehensive code review and architecture documentation (2026-06-10)

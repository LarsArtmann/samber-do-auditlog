# Execution Plan — 2026-06-10

Comprehensive, exhaustive TODO list. Every task ≤ 12 min. Sorted by importance/impact/effort/customer-value.

**Total: 42 tasks across 5 tiers · Est. ~6h**

---

## Tier 1 — Bugs & Data Correctness (DO FIRST)

Customer-facing breakage. Ship-blocking.

| #   | Task                                                                                                                                                                                                       | File(s)                                | Est | Status |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- | --- | ------ |
| 1   | Fix missing `<th>Health</th>` in services table — 11 `<td>` vs 10 `<th>` causes misaligned columns                                                                                                         | `html.templ`                           | 2m  | ❌     |
| 2   | Fix health check duration: remove per-service `HealthCheckDurationMs` from `ServiceInfo`, add `TotalHealthCheckDurationMs` to `Report` instead — current values are batch-total, misleading as per-service | `plugin.go`, `recorder.go`, `types.go` | 10m | ❌     |
| 3   | Fix health check duration: update `RecordHealthCheck` recorder to store batch total on Report, not per-service                                                                                             | `recorder.go`                          | 5m  | ❌     |
| 4   | Fix health check duration: update HTML stat card + remove per-service health duration column from services table                                                                                           | `html.templ`                           | 5m  | ❌     |
| 5   | Fix health check duration: update tests for new field layout                                                                                                                                               | `auditlog_test.go`                     | 8m  | ❌     |
| 6   | Fix `RecordHealthCheck` creating records with empty `serviceType` — call `inferServiceType()` in that codepath too                                                                                         | `recorder.go`                          | 3m  | ❌     |
| 7   | Run `go generate ./...` + `go test -count=1 ./...` to verify all fixes                                                                                                                                     | CLI                                    | 2m  | ❌     |

---

## Tier 2 — Type Architecture (HIGH VALUE)

Improves type safety, code quality, and visual richness. Medium effort, high payoff.

| #   | Task                                                                                                                       | File(s)            | Est | Status |
| --- | -------------------------------------------------------------------------------------------------------------------------- | ------------------ | --- | ------ |
| 8   | Define `ProviderType` named type in `types.go` with constants (`ProviderTypeLazy`, etc.) and `String()` + `Icon()` methods | `types.go`         | 8m  | ❌     |
| 9   | Change `ServiceInfo.ServiceType` from `string` to `ProviderType`                                                           | `types.go`         | 3m  | ❌     |
| 10  | Change `serviceRecord.serviceType` from `string` to `ProviderType`, update `inferServiceType` to return `ProviderType`     | `recorder.go`      | 5m  | ❌     |
| 11  | Update `buildServicesLocked` to use `ProviderType`, update `newServiceRecord`                                              | `recorder.go`      | 3m  | ❌     |
| 12  | Add `IsHealthchecker` and `IsShutdowner` bool fields to `ServiceInfo`                                                      | `types.go`         | 3m  | ❌     |
| 13  | Update `inferServiceType` → `inferServiceInfo` that also fetches Healthchecker/Shutdowner via `do.ExplainNamedService`     | `recorder.go`      | 5m  | ❌     |
| 14  | Update `serviceRecord` + `newServiceRecord` + `buildServicesLocked` for Healthchecker/Shutdowner fields                    | `recorder.go`      | 5m  | ❌     |
| 15  | Update HTML: add 🫀 Healthchecker / 🙅 Shutdowner capability icons in services table, scope tree, graph tooltips           | `html.templ`       | 8m  | ❌     |
| 16  | Add tests for `ProviderType` constants, `Icon()`, `String()` methods                                                       | `auditlog_test.go` | 5m  | ❌     |
| 17  | Add test for Healthchecker/Shutdowner capability tracking                                                                  | `auditlog_test.go` | 5m  | ❌     |
| 18  | Run `go generate` + `go test -count=1 ./...` to verify Tier 2                                                              | CLI                | 2m  | ❌     |

---

## Tier 3 — Missing Tests & Lint (QUALITY GATE)

Unblocks confidence in future changes. Low effort, high reliability payoff.

| #   | Task                                                                                | File(s)            | Est | Status |
| --- | ----------------------------------------------------------------------------------- | ------------------ | --- | ------ |
| 19  | Add test for `Report.ServiceByName()` — found and not-found cases                   | `auditlog_test.go` | 5m  | ❌     |
| 20  | Add test for `Report.FailedServices()` — with failing and healthy services          | `auditlog_test.go` | 5m  | ❌     |
| 21  | Add test for `Report.EventsByType()` — registration, invocation, shutdown filtering | `auditlog_test.go` | 5m  | ❌     |
| 22  | Add test for `ServiceStatus.IsError()` — all 5 statuses                             | `auditlog_test.go` | 3m  | ❌     |
| 23  | Add test for `ServiceRef.String()` — root scope, named scope, empty scope           | `auditlog_test.go` | 3m  | ❌     |
| 24  | Add test for `WriteReportJSON` error path (failing writer)                          | `auditlog_test.go` | 3m  | ❌     |
| 25  | Add test for `WriteEventsNDJSON` error path (failing writer)                        | `auditlog_test.go` | 3m  | ❌     |
| 26  | Add test for `WriteHTML` error path (failing writer)                                | `auditlog_test.go` | 3m  | ❌     |
| 27  | Add test for `writeToFile` error path (invalid directory)                           | `auditlog_test.go` | 3m  | ❌     |
| 28  | Run `golangci-lint run` on `auditlog` package only, fix formatting issues (gci)     | CLI                | 5m  | ❌     |
| 29  | Run `go generate` + `go test -count=1 ./...` to verify Tier 3                       | CLI                | 2m  | ❌     |

---

## Tier 4 — UX Enhancements (CUSTOMER VALUE)

Visible improvements users will notice. Medium effort.

| #   | Task                                                                                              | File(s)                   | Est | Status |
| --- | ------------------------------------------------------------------------------------------------- | ------------------------- | --- | ------ |
| 30  | Add service type filter chips to Events tab (filter by lazy/eager/transient/alias)                | `html.templ`              | 8m  | ❌     |
| 31  | Add "Type" column to Events table showing provider type per event                                 | `html.templ`              | 5m  | ❌     |
| 32  | Add service type to Event struct (`ServiceType ProviderType`) — populate in `newEvent` via lookup | `types.go`, `recorder.go` | 8m  | ❌     |
| 33  | Update Events tab JS to use `service_type` from event data                                        | `html.templ`              | 5m  | ❌     |
| 34  | Add health check event type filter chip + health_check badge color in Events tab                  | `html.templ`              | 3m  | ❌     |
| 35  | Run `go generate` + `go test -count=1 ./...` to verify Tier 4                                     | CLI                       | 2m  | ❌     |

---

## Tier 5 — Documentation & Polish (LONG-TERM VALUE)

Keeps project healthy. Can be done incrementally.

| #   | Task                                                                                                 | File(s)                         | Est | Status |
| --- | ---------------------------------------------------------------------------------------------------- | ------------------------------- | --- | ------ |
| 36  | Update `FEATURES.md`: add health check auditing, service type tracking, ProviderType, capabilities   | `FEATURES.md`                   | 8m  | ❌     |
| 37  | Update `TODO_LIST.md`: mark done items, add new ones from this plan                                  | `TODO_LIST.md`                  | 5m  | ❌     |
| 38  | Update `AGENTS.md`: ProviderType, capabilities, health check duration semantics, Tier 2 changes      | `AGENTS.md`                     | 5m  | ❌     |
| 39  | Update `doc.go`: add health checks, service type, capabilities to package description                | `doc.go`                        | 3m  | ❌     |
| 40  | Update `CHANGELOG.md` with all new features (health check, service type, ProviderType, capabilities) | `CHANGELOG.md`                  | 5m  | ❌     |
| 41  | Add `Config.Validate() error` method + test                                                          | `plugin.go`, `auditlog_test.go` | 8m  | ❌     |
| 42  | Write status report after all tiers complete                                                         | `docs/status/`                  | 5m  | ❌     |

---

## Already Done (from UI/UX plan — all completed in HTML rewrite)

~~T1-1 Report fields~~ ✓ · ~~T1-2 BuildReport~~ ✓ · ~~T1-3 Tests~~ ✓ · ~~T2-1 Shutdown column~~ ✓ · ~~T2-2 Error hover~~ ✓ · ~~T2-3 Dependents~~ ✓ · ~~T2-4 Search filter~~ ✓ · ~~T2-5 Timestamps~~ ✓ · ~~T3-1 Schema version~~ ✓ · ~~T3-2 Stat cards~~ ✓ · ~~T3-3 Error stat~~ ✓ · ~~T4-1 Event names~~ ✓ · ~~T4-2 Color badges~~ ✓ · ~~T4-3 Filter chips~~ ✓ · ~~T5-2 Node colors~~ ✓ · ~~T5-3 Tooltips~~ ✓ · ~~T6-1 Scopes tab~~ ✓ · ~~T6-2 Scope tree~~ ✓ · ~~T6-3 Expand/collapse~~ ✓ · ~~T7-1 Dual bars~~ ✓ · ~~T7-2 Order numbers~~ ✓ · ~~T8-1 Responsive~~ ✓ · ~~T8-2 Keyboard nav~~ ✓ · ~~T8-3 Footer~~ ✓

## Already Done (from final-polish plan)

~~F3 SchemaVersion 0.2.0~~ ✓ (bumped in health check commit)

## Explicitly NOT Included (deferred / rejected)

- `ReportOption` functional options — P2, ~35min, no user demand yet
- Mermaid/PlantUML export — Future, no user demand
- Versioned schema migration — Postpone to v1.0
- `sortScopeNodes` by ID instead of Name — cosmetic, Name is better for display
- Stale FEATURES.md PLANNED entries — folded into Task #36
- Arrow endpoint overlap fix (T5-1) — cosmetic, graph already functional

---

## Commit Strategy

One commit per completed tier (5 commits total). Each tier is self-contained and leaves tests green.

1. **Tier 1 commit**: "Fix HTML table alignment and health check duration semantics"
2. **Tier 2 commit**: "Add ProviderType named type and Healthchecker/Shutdowner capability tracking"
3. **Tier 3 commit**: "Add missing tests for convenience methods, error paths, and type methods"
4. **Tier 4 commit**: "Add service type to events and type filter chips in Events tab"
5. **Tier 5 commit**: "Update docs: FEATURES, TODO_LIST, CHANGELOG, AGENTS, doc.go, Config.Validate"

# Execution Plan — 2026-06-10

Comprehensive, sorted by importance/impact/effort/customer-value.
Every task ≤12 minutes. Total: 42 tasks.

---

## Tier 1: Test Coverage Gaps (Safety Net)

Production code with <100% coverage = latent bugs. Fix first.

| #   | Task                                                                         | Files              | Est. | Impact | Rationale                                                                                                                                                        |
| --- | ---------------------------------------------------------------------------- | ------------------ | ---- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Cover `newServiceRecordFromMeta` — test health check on unregistered service | `auditlog_test.go` | 8min | HIGH   | 0% coverage. Only call site is `RecordHealthCheck` line 802. A service discovered ONLY via health check (never registered through hooks) exercises this path.    |
| 2   | Cover `RecordHealthCheckWithContext` with real `context.Context`             | `auditlog_test.go` | 8min | HIGH   | 88.9% coverage. Only tested via `RecordHealthCheck` wrapper (always `context.Background()`). Add test with cancellable context.                                  |
| 3   | Cover `ResolveServiceScope` ancestor-walking branch                          | `auditlog_test.go` | 8min | MED    | 90% coverage. The `scope.Ancestors()` walk is untested. Need a child scope with a service registered in root, resolved from child.                               |
| 4   | Cover `recordInvocationResult` late-registration branch                      | `auditlog_test.go` | 6min | MED    | 88.9%. The `!ok` branch (service not in map) only fires if `OnAfterRegistration` was skipped but `OnAfterInvocation` fires. Need direct `Recorder` manipulation. |
| 5   | Cover `enrichCapabilities` skip-when-ref-nil branch                          | `auditlog_test.go` | 5min | LOW    | 91.7%. The `meta.ref == nil` guard. Only happens if `recordScope` is never called for a scope. Edge case.                                                        |

## Tier 2: Type Model Improvements (Architecture)

Stronger types = fewer bugs, better API. Do before adding features.

| #   | Task                                                                         | Files                          | Est. | Impact | Rationale                                                                                                             |
| --- | ---------------------------------------------------------------------------- | ------------------------------ | ---- | ------ | --------------------------------------------------------------------------------------------------------------------- |
| 6   | Add `ProviderType.IsKnown() bool` method                                     | `types.go`, `auditlog_test.go` | 4min | MED    | Empty string = unknown is implicit. `IsKnown()` makes intent explicit. Replaces `!= ""` checks throughout.            |
| 7   | Consolidate `newServiceRecord` + `newServiceRecordFromMeta` into shared core | `recorder.go`                  | 6min | MED    | 14 of 16 fields identical. Extract `newServiceRecordCore(scopeID, scopeName, name, serviceType, now)` called by both. |
| 8   | Add `ServiceRef.IsRoot() bool` helper                                        | `types.go`, `auditlog_test.go` | 3min | LOW    | `ScopeName == "" \|\| ScopeName == "[root]"` pattern repeated in `String()` and HTML. Centralize.                     |
| 9   | Add `Event.HasError() bool` helper                                           | `types.go`, `auditlog_test.go` | 3min | LOW    | `e.Error != nil` scattered in HTML template and tests. Makes intent clearer.                                          |
| 10  | Add `ServiceInfo.HasHealthError() bool` helper                               | `types.go`, `auditlog_test.go` | 3min | LOW    | `s.HealthCheckError != nil` used in `UnhealthyServices()` and HTML.                                                   |

## Tier 3: ReportOption Functional Options (P1 Feature)

Top customer-requested feature. Split into smallest possible steps.

| #   | Task                                                                         | Files              | Est. | Impact    | Rationale                                                                                          |
| --- | ---------------------------------------------------------------------------- | ------------------ | ---- | --------- | -------------------------------------------------------------------------------------------------- |
| 11  | Define `ReportOption` type + `WithServicesByName(names ...string)`           | `types.go`         | 6min | VERY HIGH | Core type + first filter option. `func(*reportFilter) error` pattern.                              |
| 12  | Define `WithServicesByType(pt ProviderType)` option                          | `types.go`         | 4min | HIGH      | Filter by provider type. Natural companion to `WithServicesByName`.                                |
| 13  | Define `WithEventsByType(et EventType)` option                               | `types.go`         | 4min | HIGH      | Filter events by type. Most common query pattern.                                                  |
| 14  | Define `WithTimeRange(from, to time.Time)` option                            | `types.go`         | 4min | HIGH      | Filter events by timestamp range. Essential for time-bounded queries.                              |
| 15  | Define `WithScope(scopeID string)` option                                    | `types.go`         | 4min | MED       | Filter to single scope. Natural with `ServicesByScope`.                                            |
| 16  | Add `reportFilter` internal struct + `newReportFilter(opts ...ReportOption)` | `types.go`         | 5min | VERY HIGH | Internal plumbing. Holds parsed options. Validates them.                                           |
| 17  | Add `Report.Filtered(opts ...ReportOption) Report` method                    | `types.go`         | 8min | VERY HIGH | The main public API. Returns new filtered Report value. Apply filters to services + events slices. |
| 18  | Add `Plugin.ReportFiltered(opts ...ReportOption) Report` convenience         | `plugin.go`        | 3min | HIGH      | User-friendly: `p.ReportFiltered(auditlog.WithServicesByType("lazy"))`.                            |
| 19  | Test: `WithServicesByName` filter                                            | `auditlog_test.go` | 6min | VERY HIGH | Verify services and events are correctly filtered by name.                                         |
| 20  | Test: `WithServicesByType` filter                                            | `auditlog_test.go` | 5min | HIGH      | Verify provider type filtering works.                                                              |
| 21  | Test: `WithEventsByType` filter                                              | `auditlog_test.go` | 5min | HIGH      | Verify event type filtering works.                                                                 |
| 22  | Test: `WithTimeRange` filter                                                 | `auditlog_test.go` | 5min | HIGH      | Verify time-bounded event filtering.                                                               |
| 23  | Test: `WithScope` filter                                                     | `auditlog_test.go` | 5min | MED       | Verify scope-scoped filtering.                                                                     |
| 24  | Test: combined filters (name + type)                                         | `auditlog_test.go` | 5min | MED       | Multiple options applied simultaneously.                                                           |
| 25  | Test: `ReportFiltered` on Plugin                                             | `auditlog_test.go` | 4min | HIGH      | End-to-end through Plugin API.                                                                     |

## Tier 4: Documentation & Project Hygiene

Keep docs honest. Low effort, high trust value.

| #   | Task                                                                      | Files          | Est. | Impact | Rationale                                                             |
| --- | ------------------------------------------------------------------------- | -------------- | ---- | ------ | --------------------------------------------------------------------- |
| 26  | Update `TODO_LIST.md` — reflect completed coverage + ReportOption         | `TODO_LIST.md` | 4min | MED    | Currently stale. Missing coverage improvements and ReportOption work. |
| 27  | Update `FEATURES.md` — add ReportOption, helpers, coverage                | `FEATURES.md`  | 4min | MED    | Missing new features from this session.                               |
| 28  | Update `AGENTS.md` — add ReportOption, consolidated constructors, helpers | `AGENTS.md`    | 4min | MED    | Keeps future AI sessions informed.                                    |
| 29  | Add `ReportOption` section to README.md                                   | `README.md`    | 6min | HIGH   | Users need to know about filtering API.                               |

## Tier 5: Export Enhancements

| #   | Task                                                                       | Files                                             | Est.  | Impact | Rationale                                                                                                |
| --- | -------------------------------------------------------------------------- | ------------------------------------------------- | ----- | ------ | -------------------------------------------------------------------------------------------------------- |
| 30  | Add `Plugin.ExportFilteredToFile(path string, opts ...ReportOption) error` | `plugin.go`, `auditlog_test.go`                   | 6min  | MED    | Combines `ReportFiltered` + `ExportToFile`. Most common export use case.                                 |
| 31  | Add Mermaid export: `Plugin.ExportMermaid(path string) error`              | `plugin.go`, new `mermaid.go`, `auditlog_test.go` | 10min | MED    | Text-based dependency graph. Uses existing `buildDepsLocked` data. Mermaid syntax is trivial: `A --> B`. |
| 32  | Add Mermaid export test                                                    | `auditlog_test.go`                                | 5min  | MED    | Verify Mermaid output is valid syntax.                                                                   |

## Tier 6: Deeper Architecture Improvements

Nice-to-have. Can defer.

| #   | Task                                                                    | Files                          | Est. | Impact | Rationale                                                                              |
| --- | ----------------------------------------------------------------------- | ------------------------------ | ---- | ------ | -------------------------------------------------------------------------------------- |
| 33  | Refactor `buildCapabilityMap` to iterative (remove nolint)              | `recorder.go`                  | 8min | LOW    | `//nolint:modernize` suppresses a real smell. Replace recursion with stack-based walk. |
| 34  | Add `Report.EventsByRef(scopeID, serviceName)` method                   | `types.go`, `auditlog_test.go` | 5min | MED    | Scoped event lookup. Natural companion to `EventsByService`.                           |
| 35  | Document Recorder locking protocol in godoc                             | `recorder.go`                  | 4min | MED    | 4 mutexes, complex ordering. A doc block prevents future deadlocks.                    |
| 36  | Add `ProviderType.Valid() bool` (alias for IsKnown, but more idiomatic) | `types.go`, `auditlog_test.go` | 3min | LOW    | `Valid()` is more idiomatic Go for type validation.                                    |
| 37  | Add HTML export test verifying health check data renders                | `auditlog_test.go`             | 6min | MED    | Current HTML test only checks structure. Verify health fields appear in output.        |
| 38  | Add runnable godoc examples in `example_test.go`                        | new `example_test.go`          | 8min | MED    | Godoc examples are executable documentation. Users learn from code.                    |
| 39  | Refactor `buildScopeTreeLocked` — extract scope service grouping        | `recorder.go`                  | 6min | LOW    | 55-line function. Extract `buildScopeServicesLocked` helper.                           |

## Tier 7: Future Consideration (Not This Round)

| #   | Task                        | Files              | Est.  | Impact | Rationale                                                          |
| --- | --------------------------- | ------------------ | ----- | ------ | ------------------------------------------------------------------ |
| 40  | Schema migration function   | `types.go`         | 30min | MED    | Only needed at v1.0. SchemaVersion exists, no consumers yet.       |
| 41  | PlantUML export             | new `plantuml.go`  | 20min | LOW    | No known users. Mermaid is more useful.                            |
| 42  | Fuzz test for HTML template | `auditlog_test.go` | 15min | LOW    | Good security practice, but templ provides escaping. Low XSS risk. |

---

## Execution Order (What to do in sequence)

**Phase A** (Safety): #1→#2→#3→#4→#5 — Coverage gaps. ~35 min.
**Phase B** (Types): #6→#7→#8→#9→#10 — Type model. ~19 min.
**Phase C** (Feature): #11→#12→#13→#14→#15→#16→#17→#18→#19→#20→#21→#22→#23→#24→#25 — ReportOption. ~74 min.
**Phase D** (Polish): #26→#27→#28→#29 — Docs. ~18 min.
**Phase E** (Exports): #30→#31→#32 — Export enhancements. ~21 min.
**Phase F** (Architecture): #33→#34→#35→#36→#37→#38→#39 — Deeper improvements. ~35 min.
**Phase G** (Future): #40→#41→#42 — Defer. ~65 min.

**Total executable this round: ~202 minutes (Phases A–F)**
**Deferred: ~65 minutes (Phase G)**

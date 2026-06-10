# Status Report — 2026-06-10 15:54

**Project**: `samber-do-auditlog` — Go plugin for samber/do v2 DI container observability  
**Module**: `github.com/larsartmann/samber-do-auditlog`  
**Branch**: `master` (1 commit ahead of origin)  
**Status**: ALPHA  
**Coverage**: 94.9% (library package), 61.4% total (example pulls it down)

---

## A. FULLY DONE

### Core Library (Production Code)

- [x] **Plugin API** — `New(Config)`, `Opts()`, `Report()`, `Events()`, `EventsCount()`
- [x] **Event capture** — Registration, Invocation, Shutdown, Health Check with timestamps + sequence numbers
- [x] **Dependency inference** — Stack-based: push on before-hook, pop on after-hook, record edges
- [x] **Service lifecycle tracking** — `ServiceStatus` (registered → active → shutdown, with error states)
- [x] **ProviderType detection** — lazy/eager/transient/alias via `do.ExplainNamedService`
- [x] **Capability detection** — `IsHealthchecker`, `IsShutdowner` via `do.ExplainInjector` (with recursion fix for child scopes)
- [x] **Scope tree** — Parent/child hierarchy with sorted services per scope
- [x] **Health check wrapper** — `RecordHealthCheck[WithContext](injector)` wrapping `injector.HealthCheckWithContext`
- [x] **Config.OnEvent callback** — Real-time event streaming without polling
- [x] **Env var toggle** — `DO_AUDITLOG_ENABLED=true|1|yes` with zero-cost disabled mode
- [x] **Build duration tracking** — Microsecond-precision per-service first build + shutdown
- [x] **Concurrency model** — `RWMutex` for events/services/scopes, separate `Mutex` for stack and invocation order, atomic sequence counters

### Export Formats

- [x] **JSON** — Full `Report` as indented JSON via `WriteReportJSON()` / `ExportToFile()`
- [x] **NDJSON** — Line-delimited event stream via `WriteEventsNDJSON()` / `ExportEventsToNDJSON()`
- [x] **HTML** — Self-contained single-file dashboard via `WriteHTML()` / `ExportToHTML()` with:
  - 5-tab layout: Services / Scopes / Graph / Timeline / Events
  - Sugiyama layered DAG graph with pan/zoom/click-to-highlight
  - Dual build+shutdown timeline bars
  - Service type badges (lazy/eager/transient/alias with canonical emojis)
  - Search & filter, keyboard nav (1-5), animated transitions
  - Dark observability dashboard aesthetic

### Query Methods (Added This Session)

- [x] `Report.ServiceByName(name)` — First match by service name
- [x] `Report.ServiceByRef(scopeID, name)` — Exact scoped lookup
- [x] `Report.ServicesByScope(scopeID)` — All services in scope
- [x] `Report.EventsByService(name)` — All events for a service
- [x] `Report.EventsByType(t)` — All events of given type
- [x] `Report.FailedServices()` — Services with invocation or shutdown errors
- [x] `Report.UnhealthyServices()` — Services with health check errors
- [x] `Event.Duration()` — Nil-safe duration in ms
- [x] `ServiceInfo.Uptime()` — Time since registration

### Testing

- [x] **71 tests passing** (69 unit + 2 benchmarks)
- [x] **94.9% statement coverage** on library package
- [x] **0 golangci-lint issues** (108 linters enabled, extremely strict config)
- [x] **0 build errors or warnings**
- [x] Tests for: disabled/enabled toggle, env var, registration/invocation, dependency tracking, shutdown (clean + error), scope tree, scope_id, export formats, error paths, container_id, version, sequence numbers, empty report, concurrent invocations, ServiceStatus, transient/value providers, health checks (13 tests), capability tracking with child scopes, all query methods, benchmarks
- [x] Table-driven test patterns where applicable

### Code Quality

- [x] `.golangci.yml` with 108 linters — exhaustruct, depguard (strict allowlist), err113, wsl_v5, nlreturn, modernize, ireturn, gosec, gocritic, etc.
- [x] `serviceKey()` as single canonical key function
- [x] `ServiceRef` embedded in `Event` and `ServiceInfo` for single source of truth
- [x] `writeToFile()` helper eliminates create-defer-close duplication
- [x] `newEvent()`/`newEventFromRef()`/`newServiceRecord()`/`newServiceRecordFromMeta()` constructors centralize field init for exhaustruct compliance
- [x] Deadlock-aware: `do.ExplainInjector()` called outside recorder mutex

---

## B. PARTIALLY DONE

### Schema Versioning

- **What exists**: `SchemaVersion = "0.2.0"` constant in `types.go`
- **What's missing**: No migration function, no backward compatibility logic, no schema evolution strategy
- **Impact**: Low — still ALPHA, no consumers yet. Can defer to v1.0.

### HTML Visualization

- **What exists**: Full 5-tab interactive dashboard
- **What could improve**:
  - No test coverage for the JS/visualization logic (only tests that HTML exports valid markup)
  - ~500 lines of inline JS in `html.templ` — could benefit from better structuring
  - The Sugiyama graph layout algorithm is custom — potential for edge cases with very deep/wide graphs

---

## C. NOT STARTED

### Priority 1: ReportOption Functional Options

- Filter reports by service name, time range, event type, scope
- This is the top TODO item — enables consumers to get targeted views without processing full reports

### Priority 2: Additional Export Formats

- Mermaid diagram export (text-based dependency graph)
- PlantUML export
- Both are user-driven — no known consumers yet

### Other Not Started

- **`newServiceRecordFromMeta` has 0% coverage** — only called when health checks discover services not yet registered through hooks. Path exists but no test exercises it.
- **`ResolveServiceScope` has 90% coverage** — the ancestor-walking branch is untested
- **`RecordHealthCheckWithContext` is untested with actual context** — all tests use `RecordHealthCheck` which delegates to `RecordHealthCheckWithContext(context.Background(), ...)`. No test passes a cancellable/timed-out context.

---

## D. TOTALLY FUCKED UP

### Nothing Is Broken

- Build: clean
- Tests: 71/71 pass
- Lint: 0 issues
- No known bugs or data corruption risks

### Near Misses (Fixed This Session)

1. **buildCapabilityMap recursion was deleted as "dead code"** — This was a REAL BUG. Services in child scopes silently got `IsHealthchecker=false`. Fixed and tested.
2. **3 lint issues found this session** — gocritic unlambda, ireturn on test helper, wsl violation in new test. All fixed.

### Architectural Concerns (Not Bugs, But Smells)

1. **`newServiceRecord` and `newServiceRecordFromMeta` are near-duplicates** — Only differ in how they get scopeID/scopeName and serviceType. Should be consolidated.
2. **`serviceRecord` is a god struct** — 16 fields mixing lifecycle, health, timing, error, and dependency concerns. Could benefit from grouping into embedded structs.
3. **`Recorder` has 4 different mutexes** — `mu`, `stackMu`, `invocationMu`, `shutdownMu`. Correct but complex. The locking protocol is undocumented beyond code comments.
4. **`buildCapabilityMap` uses recursive map copy** — `result[k] = v` inside a recursive loop triggers `modernize` lint. Suppressed with `//nolint:modernize` because `maps.Copy` doesn't work recursively.
5. **Report query methods are O(n) linear scans** — `ServiceByName`, `ServiceByRef`, `ServicesByScope`, `EventsByService`, `EventsByType` all iterate. Fine for current scale (<1000 services), but worth noting for future.

---

## E. WHAT WE SHOULD IMPROVE

### Type Model Improvements

1. **Consolidate `serviceRecord` constructors** — `newServiceRecord(scope, name, now)` and `newServiceRecordFromMeta(scopeID, scopeName, name, now)` share 14 of 16 field assignments. Extract a shared `newServiceRecordCore(scopeID, scopeName, name, serviceType, now)` that both call.

2. **Consider `ServiceCapabilities` struct** — `IsHealthchecker` and `IsShutdowner` are always set together in `enrichCapabilities`. A small struct would group them semantically.

3. **Consider `ServiceHealth` struct** — `LastHealthCheckAt`, `HealthCheckError`, `HealthCheckCount` are always mutated together in `RecordHealthCheck`. Grouping them makes the domain clearer.

4. **`ProviderType` should have `IsKnown()` method** — Currently empty string means "unknown". An `IsKnown()` or `Valid()` method would make intent explicit instead of checking `!= ""`.

### Library Usage Improvements

5. **`slices.Collect` + `slices.All`** — Go 1.26 has better iterator support. Some filter/map patterns in `types.go` could use these. However, the current patterns are clear and performant — this is cosmetic.

6. **`maps.Keys` / `maps.Values`** — Already using `maps.Copy` in `BuildReport`. Could use `maps.Keys` in `buildDepsLocked` and `buildDependentsMapLocked` instead of manual range.

### Testing Improvements

7. **Cover `newServiceRecordFromMeta`** — 0% coverage. Add a test that does health check on a service not yet registered through hooks (this is the only code path that calls it).

8. **Test `RecordHealthCheckWithContext` with real context** — Currently only tested via `RecordHealthCheck` wrapper. Should test context cancellation/timeout behavior.

9. **Cover `ResolveServiceScope` ancestor-walking** — 90% coverage, missing the ancestor walk branch.

10. **Add fuzz test for `html.templ`** — The template renders arbitrary report data. A fuzz test would catch XSS or malformed output edge cases.

---

## F. Top #25 Things We Should Get Done Next

### High Impact, Low Effort (Do First)

| #   | Task                                                                          | Impact   | Effort |
| --- | ----------------------------------------------------------------------------- | -------- | ------ |
| 1   | **Push to origin** — 1 commit ahead                                           | Critical | 0      |
| 2   | **Cover `newServiceRecordFromMeta`** — 0% → ~100%                             | High     | Low    |
| 3   | **Consolidate `newServiceRecord` / `newServiceRecordFromMeta`** — deduplicate | Medium   | Low    |
| 4   | **Add `ProviderType.IsKnown()` method**                                       | Medium   | Low    |
| 5   | **Cover `ResolveServiceScope` ancestor-walking branch**                       | Medium   | Low    |
| 6   | **Test `RecordHealthCheckWithContext` with cancellable context**              | Medium   | Low    |
| 7   | **Update AGENTS.md with new convenience methods**                             | Medium   | Low    |

### High Impact, Medium Effort (Do Next)

| #   | Task                                                                                     | Impact    | Effort |
| --- | ---------------------------------------------------------------------------------------- | --------- | ------ |
| 8   | **ReportOption functional options** — filter reports by service/time/event type/scope    | Very High | Medium |
| 9   | **Group health fields into `ServiceHealth` struct** in `serviceRecord` and `ServiceInfo` | Medium    | Medium |
| 10  | **Add `Event.RelativeTime() string`** — human-readable relative timestamps for events    | Medium    | Medium |
| 11  | **Update TODO_LIST.md / FEATURES.md** — reflect current state                            | Medium    | Low    |
| 12  | **Add `Report.EventsByRef(scopeID, serviceName)`** — scoped event lookup                 | Medium    | Low    |

### Medium Impact, Medium Effort

| #   | Task                                                                                     | Impact | Effort |
| --- | ---------------------------------------------------------------------------------------- | ------ | ------ |
| 13  | **Add `Report.ServiceCount()` / `EventCount()` cached accessors**                        | Low    | Low    |
| 14  | **Document locking protocol** on `Recorder` (4 mutexes)                                  | Medium | Medium |
| 15  | **Refactor `buildCapabilityMap`** — use iterative approach instead of recursion + nolint | Low    | Medium |
| 16  | **Add HTML export test that verifies health check data renders**                         | Medium | Low    |
| 17  | **Add example_test.go** with runnable examples for godoc                                 | Medium | Medium |

### Lower Priority, Higher Effort

| #   | Task                                                                      | Impact | Effort |
| --- | ------------------------------------------------------------------------- | ------ | ------ |
| 18  | **Mermaid export** — text-based dependency graph                          | Medium | High   |
| 19  | **Schema migration function** — version compatibility                     | Medium | High   |
| 20  | **Fuzz test for HTML template rendering**                                 | Low    | Medium |
| 21  | **Optimize Report query methods with pre-built indices**                  | Low    | Medium |
| 22  | **PlantUML export**                                                       | Low    | High   |
| 23  | **Consider `iter.Seq` for event streaming** — Go 1.26 iterators           | Low    | Medium |
| 24  | **Structured logging integration** — `slog.Handler` for events            | Low    | High   |
| 25  | **Benchmark with realistic workloads** — 1000+ services, deep scope trees | Low    | Medium |

---

## G. Top #1 Question I Cannot Figure Out Myself

**Should `ReportOption` filter in-place (modify the `Report`) or return a new filtered `Report`?**

Arguments:

- **In-place**: Avoids allocation, but mutates shared state. Dangerous if caller holds a reference.
- **New value**: Safe, idiomatic Go, but copies slices. `Report` can be large with many events.

The `Report` struct is currently a value type (returned by value from `BuildReport()`). All query methods (`ServiceByName`, `EventsByType`, etc.) are on `Report` (value receiver). This suggests **returning a new filtered `Report`** is the right call — consistent with existing patterns. But I'd want your confirmation before implementing, because this API decision affects all future consumers.

Also: should `ReportOption` be a functional option pattern (`func(*Report)` with filtered slices) or should we add dedicated methods like `Report.FilteredBy(...)` that return new reports? The functional option pattern is more extensible but heavier. Dedicated methods are simpler but less flexible.

---

## Session Summary

| Metric                  | Value                                                             |
| ----------------------- | ----------------------------------------------------------------- |
| Tests                   | 71 passing, 0 failing                                             |
| Coverage (library)      | 94.9%                                                             |
| Coverage (total)        | 61.4%                                                             |
| Lint issues             | 0                                                                 |
| Build                   | Clean                                                             |
| LOC (production)        | ~1,720 (recorder 850 + types 264 + plugin 194 + html 26 + doc 12) |
| LOC (test)              | 2,288                                                             |
| LOC (example)           | 787                                                               |
| Commits this session    | 8                                                                 |
| Commits ahead of origin | 1                                                                 |

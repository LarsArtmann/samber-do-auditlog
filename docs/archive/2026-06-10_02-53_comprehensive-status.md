# Status Report — 2026-06-10 02:53

**Project**: samber-do-auditlog · **Module**: `github.com/larsartmann/samber-do-auditlog` · **Go**: 1.26.3

---

## A. FULLY DONE

### Core Library (production-quality)

| Item                 | Detail                                                                                                        |
| -------------------- | ------------------------------------------------------------------------------------------------------------- |
| Plugin lifecycle     | `New(Config)` → `Opts()` → `do.NewWithOpts` — one-line integration                                            |
| Event capture        | 6 hooks: before/after registration, invocation, shutdown                                                      |
| Dependency graph     | Stack-based inference: parent→child from call chain                                                           |
| Reverse deps         | `Dependents` computed at report time                                                                          |
| Scope tracking       | Scope tree with parent metadata, per-scope service lists                                                      |
| Service status       | `ServiceStatus` computed: registered → active → shutdown, with error states                                   |
| Duration tracking    | First build duration (ms) + shutdown duration (ms) per service                                                |
| Sequence numbers     | Per-recorder atomic counter, monotonic, no global state                                                       |
| JSON export          | Indented JSON to `io.Writer` or file                                                                          |
| NDJSON export        | Line-delimited event stream to `io.Writer` or file                                                            |
| HTML visualization   | Self-contained HTML: 5 tabs (Services/Scopes/Graph/Timeline/Events), force-directed graph, search, responsive |
| Env var toggle       | `DO_AUDITLOG_ENABLED` = true/1/yes                                                                            |
| Zero-cost disabled   | Empty `InjectorOpts`, no hooks, no allocation                                                                 |
| OnEvent callback     | Real-time event streaming via `Config.OnEvent func(Event)`                                                    |
| Thread safety        | 4-lock design: RWMutex (state), Mutex (stack, ordering, shutdown)                                             |
| Deterministic output | Services sorted by (scope_name, service_name), scopes sorted by ID                                            |

### Type Model (clean, well-factored)

| Type                                                              | Purpose                                                                 |
| ----------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `ServiceRef`                                                      | Embedded in `Event`/`ServiceInfo` — single source of truth for identity |
| `ServiceRef.String()`                                             | Human-readable `"scope/name"` format                                    |
| `ServiceStatus.IsError()`                                         | `true` for invocation_error or shutdown_error                           |
| `Event.IsRegistration/IsInvocation/IsShutdown/IsBefore/IsAfter()` | Convenience predicates                                                  |
| `Report.ServiceByName(name)`                                      | Exact name lookup                                                       |
| `Report.EventsByType(t)`                                          | Filter events by EventType                                              |
| `Report.FailedServices()`                                         | All services with errors                                                |

### Test Suite

- **42 tests, all passing**
- **87.7% statement coverage** (library package)
- External test package (`auditlog_test`)
- Table-driven, no ginkgo/testify
- CrashingService test type for reliable shutdown error testing
- No shared state between tests

### Example

- **18/18 features** verified by self-checking feature checklist
- Ride-sharing domain model demonstrating every samber/do v2 feature
- Output to temp dir, not CWD

### Code Quality

- **0 lint issues** on library code
- Strict `.golangci.yml` with nearly all linters enabled
- `go vet` clean
- Deduplicated: single `serviceKey()`, single `sumDurationField()`

---

## B. PARTIALLY DONE

| Item                        | What's Done                  | What's Missing                                                                                             |
| --------------------------- | ---------------------------- | ---------------------------------------------------------------------------------------------------------- |
| Documentation sync          | CHANGELOG, AGENTS.md updated | FEATURES.md has stale PLANNED items (Event convenience methods, EventHandler callback already implemented) |
| `ServiceByName` doc comment | Function works correctly     | Comment says "suffix" but does exact match — misleading                                                    |

---

## C. NOT STARTED

| Item                               | Notes                                                                              |
| ---------------------------------- | ---------------------------------------------------------------------------------- |
| Report filtering                   | `ReportOption` functional options for filtering by service, time range, event type |
| Config validation                  | `Config.Validate() error` method — currently ad-hoc in `New()`                     |
| Schema migration                   | `SchemaVersion` constant exists but no migration function                          |
| Additional export formats          | Mermaid diagram, PlantUML                                                          |
| Benchmarks for convenience methods | No perf baseline for `ServiceByName`, `EventsByType`, `FailedServices`             |

---

## D. TOTALLY FUCKED UP / ISSUES

| Issue                                     | Severity     | Detail                                                                                                                                                                                 |
| ----------------------------------------- | ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ServiceByName` doc comment says "suffix" | **Low**      | Line 121 of types.go: "returns the first ServiceInfo matching the given service name suffix" — but it does exact match, not suffix match. Misleading.                                  |
| FEATURES.md stale PLANNED section         | **Low**      | "Event streaming callback" and "Convenience methods on Event" listed as PLANNED but both are DONE                                                                                      |
| CHANGELOG missing recent entries          | **Low**      | Missing: ServiceRef.String(), ServiceStatus.IsError(), Report convenience methods, example rewrite                                                                                     |
| gopls stale diagnostics                   | **Cosmetic** | 17 false-positive errors from gopls not understanding embedded struct fields in recorder.go and Config.OnEvent. `go build` and `go test` pass fine. No actual issue.                   |
| Example lint warnings                     | **Expected** | Example has `err113` (dynamic errors), `gocognit` (complex main), `gochecknoglobals` (rideCounter). All expected for demo code, all excluded in `.golangci.yml` for example directory. |

---

## E. WHAT WE SHOULD IMPROVE

### Type Model

1. **`*float64` for durations is primitive obsession** — Every consumer has to nil-check before dereferencing. A `Millis` type with `Value() float64` and `String() string` would be more ergonomic. But: might be over-engineering for the current API surface.

2. **`ServiceInfo` has too many `*string` error fields** — `ShutdownError` and `InvocationError` are `*string` not `error`. This is correct for JSON serialization but consumers lose error typing. Could add `HasErrors() bool` as a convenience.

3. **`ScopeNode` lacks service count** — The HTML visualization counts services per scope but the type doesn't have a `ServiceCount int` field. Minor.

### Architecture

4. **`Recorder` is a god struct** — 648 lines, 4 mutexes, handles event capture AND report building AND dependency inference AND scope tracking. Could split into `EventCollector`, `DependencyTracker`, `ReportBuilder`. But: would add complexity for a 1-package library at ~1800 LOC.

5. **No structured logging** — `Config` accepts no logger. The library is silent by design, which is good, but OnEvent is the only way to observe internals. Could accept `slog.Logger` via `Config.Logger` for debug-mode diagnostics.

### Testing

6. **No tests for convenience methods** — `ServiceByName`, `EventsByType`, `FailedServices`, `ServiceRef.String()`, `ServiceStatus.IsError()` are untested. They're trivial but should have coverage.

7. **No benchmark for OnEvent callback** — The callback fires on every event. Should verify it doesn't significantly impact throughput.

### Ecosystem

8. **No `go doc` examples** — The package has no `Example*` test functions that show up in `go doc`. The `example/` directory exists but pkg.go.dev won't render it as an example.

---

## F. TOP 25 THINGS TO DO NEXT

Sorted by **impact / effort** (highest impact per unit of work first):

| #   | Task                                                                                                           | Impact                   | Effort  | Category |
| --- | -------------------------------------------------------------------------------------------------------------- | ------------------------ | ------- | -------- |
| 1   | Fix `ServiceByName` doc comment: "suffix" → "exact match"                                                      | High (correctness)       | 1 min   | Bug      |
| 2   | Update FEATURES.md: move Event convenience methods and EventHandler from PLANNED → DONE                        | Medium (accuracy)        | 2 min   | Docs     |
| 3   | Update CHANGELOG.md: add ServiceRef.String, ServiceStatus.IsError, Report convenience methods, example rewrite | Medium (accuracy)        | 5 min   | Docs     |
| 4   | Add tests for `ServiceRef.String()`                                                                            | Medium (coverage)        | 5 min   | Tests    |
| 5   | Add tests for `ServiceStatus.IsError()`                                                                        | Medium (coverage)        | 3 min   | Tests    |
| 6   | Add tests for `Report.ServiceByName`, `EventsByType`, `FailedServices`                                         | Medium (coverage)        | 10 min  | Tests    |
| 7   | Add `Example*` test functions in `auditlog_test.go` for pkg.go.dev                                             | High (discoverability)   | 15 min  | API      |
| 8   | Add `ServiceInfo.HasErrors() bool` convenience                                                                 | Low (ergonomics)         | 2 min   | API      |
| 9   | Add `ScopeNode.ServiceCount int` computed field                                                                | Low (completeness)       | 5 min   | API      |
| 10  | Add benchmark for OnEvent callback overhead                                                                    | Medium (perf)            | 10 min  | Tests    |
| 11  | Fix example `log.Fatal` calls that skip `defer cancel()` (gocritic)                                            | Low (correctness)        | 3 min   | Example  |
| 12  | Add `Config.Validate() error` method                                                                           | Medium (robustness)      | 10 min  | API      |
| 13  | Consider `Millis` named type for duration fields                                                               | Medium (ergonomics)      | 30 min  | API      |
| 14  | Add `Report.EventsByPhase(phase Phase)` method                                                                 | Low (completeness)       | 3 min   | API      |
| 15  | Add `Report.ServicesByScope(scopeName string)` method                                                          | Low (completeness)       | 5 min   | API      |
| 16  | Add `Report.ServiceByRef(ref ServiceRef)` method                                                               | Low (completeness)       | 3 min   | API      |
| 17  | Document thread-safety guarantees in doc comments                                                              | Medium (docs)            | 10 min  | Docs     |
| 18  | Add `//go:build` constraints or build tags                                                                     | None                     | —       | Skip     |
| 19  | Consider `encoding/json/v2` migration                                                                          | Low (perf)               | High    | Defer    |
| 20  | Add Mermaid export format                                                                                      | Medium (visualization)   | 30 min  | Feature  |
| 21  | Add Report filtering with functional options                                                                   | High (usability)         | 1-2 hrs | Feature  |
| 22  | Schema migration for report versioning                                                                         | Low                      | 1 hr    | Feature  |
| 23  | PlantUML export format                                                                                         | Low                      | 30 min  | Feature  |
| 24  | Split Recorder into smaller structs                                                                            | Medium (maintainability) | 2 hrs   | Refactor |
| 25  | Accept `slog.Logger` in Config for debug diagnostics                                                           | Medium (observability)   | 30 min  | API      |

---

## G. TOP #1 QUESTION

**Is `*float64` for duration fields (`FirstBuildDurationMs`, `ShutdownDurationMs`, `DurationMs`) worth replacing with a named type?**

The nil-check pattern is repeated everywhere (`if s.FirstBuildDurationMs != nil { ... *s.FirstBuildDurationMs }`). A `Millis` type with `Value() float64`, `String() string`, and `IsZero() bool` would eliminate this. But it changes the JSON serialization and breaks the public API. Given we're pre-1.0, now is the time — but is the ergonomics gain worth the complexity of a wrapper type?

---

## Metrics

| Metric                 | Value                                                                |
| ---------------------- | -------------------------------------------------------------------- |
| Library LOC            | ~1,000 (excluding generated html_templ.go)                           |
| Total LOC (with tests) | ~2,278                                                               |
| Source files           | 6 (plugin.go, recorder.go, types.go, html.go, doc.go, html_templ.go) |
| Test files             | 1 (auditlog_test.go: 1,208 lines)                                    |
| Tests                  | 42, all passing                                                      |
| Coverage               | 87.7%                                                                |
| Lint issues (library)  | 0                                                                    |
| go vet                 | clean                                                                |
| Schema version         | 0.1.0                                                                |
| Dependencies           | samber/do v2.0.0, a-h/templ v0.3.1020                                |
| Go version             | 1.26.3                                                               |

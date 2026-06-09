# Status Report — 2026-06-09 23:18

**Project**: samber-do-auditlog
**Module**: `github.com/larsartmann/samber-do-auditlog`
**Go**: 1.26.3 | **Status**: ALPHA
**Tests**: 16/16 PASS | **Build**: CLEAN | **Lint**: 0 ISSUES

---

## a) FULLY DONE

### Core Library

- **Plugin lifecycle hooks**: Registration, invocation, shutdown — all with before/after phases
- **Dependency graph inference**: Stack-based — if A is on-stack when B's before-invocation fires, A→B is recorded
- **Build duration tracking**: First build duration per service (ms precision from µs)
- **Shutdown duration tracking**: Before/after shutdown timestamps, duration computed per service
- **Scope hierarchy**: Tree structure with parent-child relationships, services mapped to scopes
- **ContainerID on every Event**: Supports multi-container NDJSON consumers
- **Report schema versioning**: `Version` field ("0.1.0") for schema evolution
- **Deterministic output**: Services sorted by (scope_name, service_name) in reports

### Export Formats

- **JSON**: Full report, indented, via `WriteReportJSON` / `ExportToFile`
- **NDJSON**: Line-delimited events via `WriteEventsNDJSON` / `ExportEventsToNDJSON`
- **HTML**: Self-contained single-file visualization via templ (`html.templ`)
  - Dark theme dashboard with stats cards
  - Tabbed UI: Services table, Dependency Graph (SVG force simulation), Timeline, Events table
  - JSON data injected via `templ.JSONScript` (XSS-safe)
  - No external dependencies in output

### Configuration

- **`DO_AUDITLOG_ENABLED` env var**: "true"/"1"/"yes" enables without code changes
- **`Config.Enabled`**: Explicit override (takes precedence over env var)
- **`Config.ContainerID`**: Optional human-readable identifier (defaults to "default")
- **Disabled mode**: Zero-overhead no-op (all hooks are nil)

### Code Quality

- **0 lint issues** on non-test, non-example code (golangci-lint with ~70 linters)
- **`writeToFile()` helper**: Eliminates 3× duplicated create-defer-close pattern
- **No redundant state**: `Plugin.containerID` removed — stored only in `Recorder`
- **`BuildReport()` parameterless**: Uses `Recorder.containerID` instead
- **exhaustruct**: All struct fields explicitly initialized via constructor helpers
- **depguard**: Only `$gostd`, `github.com/a-h/templ`, `github.com/samber` allowed
- **Dead code removed**: ServiceType legend, unused CSS variables, stale references

### Tests (16 tests, all passing)

| Test                            | Coverage                                         |
| ------------------------------- | ------------------------------------------------ |
| DisabledIsNoOp                  | Plugin disabled = zero events                    |
| EnvVarEnables                   | DO_AUDITLOG_ENABLED=true activates               |
| EnvVarValues (7 subtests)       | "true"/"1"/"yes" on; "false"/"0"/""/"random" off |
| ExplicitEnabledOverridesEnv     | Config.Enabled=true wins over unset env          |
| RegistrationAndInvocation       | 4 events, service info populated                 |
| InvocationOrder                 | Monotonically increasing order                   |
| DependencyTracking              | Stack-based A→B inference                        |
| ShutdownTracking                | Before/after shutdown events                     |
| ExportToFile                    | JSON round-trip                                  |
| ExportEventsToNDJSON            | 4 NDJSON lines                                   |
| ScopeTree                       | Parent-child scope hierarchy                     |
| CachedInvocation                | Cached service re-invocation tracking            |
| EventSequenceNumbers            | Monotonically increasing sequence IDs            |
| ProviderError                   | Error captured in InvocationError                |
| ExportToHTML                    | Full HTML page with service data                 |
| ProvideTransient / ProvideValue | Transient and value provider tracking            |

---

## b) PARTIALLY DONE

- **HTML timeline**: Shows build duration bars, but no interactivity (click to drill down, zoom)
- **HTML dependency graph**: SVG force simulation works, but no click-to-highlight, no filtering
- **Documentation**: `AGENTS.md` is solid but no README.md, no godoc-quality API docs

---

## c) NOT STARTED

- **Mermaid/DOT graph export**: Would enable integration with GitHub, VS Code, etc.
- **Streaming/chunked HTML**: For reports with 10k+ services, the current approach embeds all JSON in-page
- **Profiling integration**: CPU/memory profiling of the hooks themselves
- **OpenTelemetry bridge**: Export events as OTel spans
- **Configurable filtering**: Only record certain services, scopes, or event types
- **Custom event enrichers**: User-defined metadata attached to events
- **HTML dark/light theme toggle**: Only dark theme exists
- **Benchmarks in CI**: Benchmarks exist but aren't tracked over time
- **Module rename**: Repo dir is `samber-do-metrics` but module is `samber-do-auditlog`

---

## d) TOTALLY FUCKED UP

Nothing. Clean state. All tests pass, 0 lint issues.

Previous close call: `.golangci.yml` had a duplicate `rules` key (tagliatelle + revive merged) that silently disabled ALL linting. Fixed in commit `a92ebf5`.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **`Recorder` is doing too much**: It's both the event capturer AND the report builder. These could be separated — a `Recorder` that only captures, and a `ReportBuilder` that assembles from recorded data.
2. **`serviceRecord` internal type is a parallel copy of `ServiceInfo`**: Every field exists in both. Consider whether `ServiceInfo` could be the internal type too, or whether a cleaner separation exists.
3. **No interfaces**: `Recorder` and `Plugin` are concrete types. For testability and composability, consider `Recorder` as an interface.
4. **Constructor helpers (`newRegistrationEvent`, etc.) take too many params**: 7-8 parameters. A `eventBuilder` or options pattern would be cleaner.

### Type Models

5. **`DurationMs *float64`**: Pointer-to-float64 is a code smell. A custom `Duration` type with `Valid`/`Present` semantics would be more honest.
6. **`Error *string`**: Same pattern. Consider `ErrorMessage string` with `HasError bool`, or a custom type.
7. **`ServiceType` was removed but nothing replaced it**: The concept of "lazy vs eager vs transient" is real in DI, samber/do just doesn't expose it. The HTML still has a "Type" column header with nothing to show.

### Testing

8. **External test package only**: No internal tests for private functions like `buildServicesLocked`, `buildScopeTreeLocked`, `buildDependentsMapLocked`.
9. **No integration test with real DI container lifecycle**: Tests create services but don't test the full `do.Injector` lifecycle including health checks, scopes, etc.
10. **No fuzz tests**: Event recorder is a good fuzz target.

### Developer Experience

11. **`templ generate` required before `go build`**: Not obvious to new contributors. Should be documented in README or enforced via build tag/script.
12. **No README.md**: Project has no user-facing documentation.

---

## f) Top 25 Things to Do Next (Sorted by Impact vs Effort)

| #   | Task                                                                  | Impact | Effort |
| --- | --------------------------------------------------------------------- | ------ | ------ |
| 1   | Write README.md with usage examples                                   | HIGH   | LOW    |
| 2   | Add `Opts()` disabled case exhaustruct exemption or zero-value return | MED    | LOW    |
| 3   | Extract `Duration` custom type to replace `*float64`                  | MED    | MED    |
| 4   | Add DOT/Mermaid graph export                                          | HIGH   | MED    |
| 5   | Add internal tests for `buildServicesLocked`, `buildScopeTreeLocked`  | MED    | LOW    |
| 6   | Remove "Type" column from HTML services table (nothing to show)       | LOW    | LOW    |
| 7   | Make `templ generate` a `go:generate` directive                       | MED    | LOW    |
| 8   | Add `go:generate` to `html.templ` header                              | MED    | LOW    |
| 9   | Add example with real DI container lifecycle (health checks, scopes)  | MED    | MED    |
| 10  | Rename repo dir to match module name (`samber-do-auditlog`)           | MED    | LOW    |
| 11  | Add `Report` round-trip test (JSON → unmarshal → re-marshal)          | MED    | LOW    |
| 12  | Add `ContainerID` validation (no slashes, not empty)                  | LOW    | LOW    |
| 13  | Consider `Recorder` interface for testability                         | MED    | MED    |
| 14  | Add OpenTelemetry bridge                                              | HIGH   | HIGH   |
| 15  | Add configurable event filtering                                      | MED    | MED    |
| 16  | Add `WithContainerID` functional option pattern                       | LOW    | MED    |
| 17  | HTML: add click-to-highlight on dependency graph nodes                | MED    | MED    |
| 18  | HTML: responsive design for mobile                                    | MED    | MED    |
| 19  | Track benchmark results over time in CI                               | MED    | MED    |
| 20  | Add `sampler` interface for sampling high-volume events               | LOW    | HIGH   |
| 21  | Add context.Context support to `BuildReport`/export methods           | MED    | MED    |
| 22  | Consider event streaming channel API (`Events() <-chan Event`)        | MED    | HIGH   |
| 23  | Add Prometheus metrics endpoint export                                | MED    | MED    |
| 24  | Structured logging integration (slog)                                 | LOW    | MED    |
| 25  | Add PProf endpoints to HTML visualization                             | LOW    | LOW    |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Does samber/do v2 expose any way to determine if a service was registered as lazy (Provide), eager (ProvideValue), or transient (ProvideTransient)?**

The `HookBeforeRegistration`/`HookAfterRegistration` callbacks receive `(scope, serviceName)` only. There's no registration option or metadata exposed. `ServiceType` was previously a field that always returned `unknown` — it was removed because it was dead code. If samber/do ever adds this capability, we could restore meaningful service classification.

---

## File Inventory

```
plugin.go        (142 lines) — Public API: New(), Opts(), Report(), Export*(), Events(), writeToFile()
recorder.go      (546 lines) — Core state machine: event capture, invocation/shutdown stacks, BuildReport()
types.go         (82 lines)  — Domain types: Event, ServiceInfo, Report, ScopeNode, DependencyRef
html.go          (24 lines)  — HTML export entry points (calls templ-generated component)
html.templ       (334 lines) — Templ template for self-contained HTML visualization
doc.go           (7 lines)   — Package doc comment
auditlog_test.go (688 lines) — External test package (16 tests)
example/main.go  (175 lines) — Runnable demo
html_templ.go    (61 lines)  — Generated (gitignored, requires `templ generate`)
```

**Dependencies**: `github.com/a-h/templ v0.3.1020`, `github.com/samber/do/v2 v2.0.0`
**Total hand-written Go**: ~1,664 lines

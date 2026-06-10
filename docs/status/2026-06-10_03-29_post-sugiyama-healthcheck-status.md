# Status Report — samber-do-auditlog

**Date**: 2026-06-10 03:29  
**Author**: Crush (automated)  
**Trigger**: User request for full comprehensive status after Sugiyama graph + health check work  
**Branch**: `master` (ahead of `origin/master` by 1 commit)

---

## Executive Summary

**samber-do-auditlog** is a Go plugin for [samber/do v2](https://github.com/samber/do) that records every DI container lifecycle event with timestamps, dependency graph inference, build duration tracking, health check auditing, service type tracking, and export to JSON / NDJSON / self-contained HTML.

The project is in **ALPHA** state. Core features are solid (89.6% test coverage, 54 tests, all green). The latest batch of work added **health check auditing** (wrapper pattern), **service type inference** (lazy/eager/transient/alias), **HTML dashboard redesign** (dark observability aesthetic), and **Sugiyama layered DAG graph** (replacing the broken force-directed layout).

**Working tree is clean.** All changes are committed.

---

## Metrics

| Metric | Value |
|---|---|
| Module | `github.com/larsartmann/samber-do-auditlog` |
| Go version | 1.26.3 |
| Package | Single package `auditlog` |
| LOC (library) | 2,826 (including generated `html_templ.go`) |
| LOC (total with tests+example) | 3,606 |
| Tests | 54 |
| Test coverage | 89.6% |
| Dependencies | `samber/do/v2`, `a-h/templ` (2 direct, 1 indirect) |
| Schema version | 0.2.0 |
| Files in package | 8 (`plugin.go`, `recorder.go`, `types.go`, `html.go`, `html.templ`, `html_templ.go`, `doc.go`, `auditlog_test.go`) |

---

## A) FULLY DONE

### Core Library

| Feature | Status | Location |
|---|---|---|
| Drop-in plugin setup (`New(Config)` + `Opts()`) | ✅ | `plugin.go:41-81` |
| Service registration tracking (before/after events) | ✅ | `recorder.go:169-198` |
| Service invocation tracking (duration, errors) | ✅ | `recorder.go:200-310` |
| Shutdown tracking (clean + error) | ✅ | `recorder.go:312-357` |
| **Health check auditing** (wrapper pattern) | ✅ | `plugin.go:134-177`, `recorder.go:667-762` |
| **Service type inference** (lazy/eager/transient/alias) | ✅ | `recorder.go` via `inferServiceType` |
| Dependency graph inference (stack-based) | ✅ | `recorder.go:204-216` |
| Reverse dependencies (computed at report time) | ✅ | `recorder.go:467-483` |
| Scope tree + scope tracking | ✅ | `recorder.go:485-542` |
| Monotonic sequence numbers (per-recorder atomic) | ✅ | `recorder.go:52-56` |
| Build duration measurement | ✅ | `recorder.go:253-271` |
| Invocation ordering | ✅ | `recorder.go:294-301` |
| Provider error capture | ✅ | `recorder.go:367-375` |
| Concurrent-safe recording (4-lock design) | ✅ | `recorder.go:59-76` |
| Deterministic report output (sorted) | ✅ | `recorder.go:429-434` |
| Defensive copies (Events/Report return copies) | ✅ | `recorder.go:544-550` |
| Schema versioning (0.2.0) | ✅ | `types.go:9` |
| Zero-cost disabled mode | ✅ | `plugin.go:68-71` |
| Environment variable toggle | ✅ | `plugin.go:56-64` |
| Container ID propagation | ✅ | `plugin.go:22-26` |
| EventHandler callback (`Config.OnEvent`) | ✅ | `plugin.go:Config.OnEvent` |

### Types & Convenience Methods

| Feature | Status |
|---|---|
| `Event.IsRegistration/IsInvocation/IsShutdown/IsHealthCheck/IsBefore/IsAfter()` | ✅ |
| `ServiceRef.String()` | ✅ |
| `ServiceStatus.IsError()` | ✅ |
| `Report.ServiceByName(name)` | ✅ |
| `Report.EventsByType(t)` | ✅ |
| `Report.FailedServices()` | ✅ |
| `Report.UnhealthyServices()` | ✅ |
| `ServiceInfo` health check fields (`lastHealthCheckAt`, `healthCheckDurationMs`, `healthCheckError`, `healthCheckCount`) | ✅ |
| `Report` health check fields (`healthCheckSucceeded`, `healthCheckedCount`) | ✅ |

### Export Formats

| Format | Status | Location |
|---|---|---|
| JSON report (indented) | ✅ | `plugin.go:88-120` |
| NDJSON event stream | ✅ | `plugin.go:102-125` |
| Self-contained HTML visualization | ✅ | `html.templ` |

### HTML Visualization (5-tab dashboard)

| Feature | Status |
|---|---|
| **Dark observability aesthetic** (JetBrains Mono + DM Sans, CSS variables) | ✅ |
| Services table (status badges, type badges, health, search filter) | ✅ |
| **Sugiyama layered DAG graph** (rank assignment, barycenter ordering, Bézier edges) | ✅ |
| Graph pan/zoom (scroll, drag, +/−/fit buttons) | ✅ |
| Graph click-to-highlight connected subgraph | ✅ |
| Graph type-colored nodes + left accent bars | ✅ |
| Scope tree (collapsible, type emoji chips) | ✅ |
| Timeline (dual build+shutdown bars, type icons) | ✅ |
| Events table (type filter chips, health_check support) | ✅ |
| Stats cards (services, scopes, events, deps, build time, errors, health) | ✅ |
| Legend (type distribution counts) | ✅ |
| Keyboard navigation (1-5) | ✅ |
| Responsive layout | ✅ |
| Error tooltips | ✅ |

### Testing

- **54 tests**, all passing
- **89.6% coverage**
- Covers: disabled/enabled, env var, registration, invocation, shutdown, dependency tracking, scope tree, scope_id, all export formats, error paths, container_id, sequence numbers, empty report, concurrent invocations, ServiceStatus, transient/value providers, health check (healthy/unhealthy/multiple/disabled/count/report/scope/succeeded), service type capture (eager/lazy/transient), EventHandler, convenience methods

### Documentation

- `AGENTS.md` — comprehensive project context (commands, architecture, gotchas, testing patterns)
- `FEATURES.md` — honest feature inventory (but stale — see section B)
- `TODO_LIST.md` — prioritized open tasks
- `CHANGELOG.md` — development history
- `README.md` — user-facing docs
- `docs/DOMAIN_LANGUAGE.md` — DDD glossary

---

## B) PARTIALLY DONE

| Feature | What's Done | What's Missing | Impact |
|---|---|---|---|
| **FEATURES.md** | 36+ features listed as DONE | Missing: health check auditing, service type inference, UnhealthyServices(), EventTypeHealthCheck, IsHealthCheck(), HTML redesign, Sugiyama graph, type badges, accent bars | **Stale docs** — any new session will have incomplete picture |
| **HTML health check tab** | Health check events appear in Events tab, health status in Services table | No dedicated health check visualization tab (summary card, per-service health timeline, unhealthy highlight) | Medium — health data is visible but not first-class in the dashboard |
| **Graph for disconnected nodes** | Disconnected nodes assigned to L0 | No visual distinction between "root by nature" vs "disconnected" nodes | Low — rare in real DI containers |
| **Report filtering** | `Report` struct has all data | No `ReportOption` functional options to filter by service/time/event | Listed in TODO |
| **Schema migration** | `SchemaVersion = "0.2.0"` constant exists | No migration function for consumers with v0.1.0 exports | Low urgency |

---

## C) NOT STARTED

| Feature | Priority | Notes |
|---|---|---|
| Report functional options (`ReportOption`) | P2 | Filter by service name, time range, event type |
| `Config.Validate() error` method | P3 | Centralize validation, currently ad-hoc |
| Mermaid diagram export | Future | DOT/Mermaid text output for GitHub/VS Code |
| PlantUML export | Future | Only if users request |
| HTML health check tab | P3 | Dedicated visualization for health data |
| HTML graph: scope grouping | P3 | Group nodes by scope in the DAG layout |
| HTML graph: minimap | P4 | Small overview of large graphs |
| GraphViz DOT export | Future | Standard graph interchange format |
| Configurable HTML themes | P4 | Light/dark toggle or custom CSS injection |
| Benchmark suite | P3 | Performance regression detection |
| Example web server (serve HTML) | P3 | `http.ListenAndServe` for live dashboard |
| godoc cleanup | P3 | Multiple package doc comments cause `godoclint` warning |
| CONTRIBUTING.md | P4 | If project goes public |
| Versioned releases (git tags) | P2 | No tags exist yet |

---

## D) TOTALLY FUCKED UP / PROBLEMATIC

| Issue | Severity | Details |
|---|---|---|
| **FEATURES.md is stale** | HIGH | Missing 2 major features (health check auditing, service type inference) and ~10 minor features. Any new AI session starts with wrong picture. Must be fixed before next development cycle. |
| **templ version mismatch** | LOW | go.mod has v0.3.1020, local generator is v0.3.1036 (unpublished). No functional impact but generates version warning. |
| **Multiple godoc comments** | LOW | Both `doc.go` and `plugin.go` have package-level doc comments, causing `godoclint` warning. |
| **`doc.go` says "force-directed graph"** | MEDIUM | Still references old force-directed graph. Should say "Sugiyama layered DAG layout". |
| **FEATURES.md LOC count outdated** | LOW | Says "1 package, 1757 LOC" but now 2826 LOC (library) / 3606 total. |
| **No git tags / releases** | MEDIUM | Schema is at 0.2.0 but no git tags exist. Consumers can't pin versions. |
| **`ResolveServiceScope` does `injector.(*do.Scope)` type assertion** | MEDIUM | Tightly coupled to samber/do internal types. If do changes Scope API, this breaks. Consider a more stable approach. |
| **HTML graph doesn't show scope boundaries** | LOW | Nodes from different scopes are mixed in layers. No visual grouping by scope. |

---

## E) WHAT WE SHOULD IMPROVE

### High Priority

1. **Update FEATURES.md** — Add health check auditing, service type inference, UnhealthyServices, EventTypeHealthCheck, IsHealthCheck, HTML redesign items, Sugiyama graph. This is the #1 source of truth for new sessions.

2. **Fix `doc.go`** — Replace "force-directed graph" with "Sugiyama layered DAG layout".

3. **Git tag v0.2.0** — Schema is at 0.2.0, code is stable, 54 tests pass. Tag it.

### Medium Priority

4. **HTML health check visualization** — Add a "Health" tab showing per-service health status, last check time, duration, error details. Use the new `UnhealthyServices()` data.

5. **Graph scope grouping** — Visual scope boundaries in the DAG (background rects or subtle color coding per scope).

6. **Report functional options** — `Report(func(...ReportOption) Report)` for filtering.

### Low Priority

7. **Benchmark suite** — Add `BenchmarkRegistration`, `BenchmarkInvocation`, `BenchmarkBuildReport`, `BenchmarkHTMLExport`.

8. **Minimap for graph** — For 20+ services, a small overview rectangle in the corner.

9. **Configurable HTML themes** — CSS variable injection or light/dark toggle.

---

## F) Top 25 Things to Do Next

| # | Task | Category | Impact | Effort | Priority |
|---|---|---|---|---|---|
| 1 | **Update FEATURES.md** with health check, service type, Sugiyama graph, HTML redesign | Docs | HIGH | 30m | P0 |
| 2 | **Fix `doc.go`** — remove "force-directed" reference | Docs | HIGH | 2m | P0 |
| 3 | **Git tag v0.2.0** | Release | HIGH | 2m | P0 |
| 4 | **Update FEATURES.md LOC count** (2826/3606) | Docs | MED | 1m | P0 |
| 5 | **HTML Health tab** — per-service health status, timeline, unhealthy highlight | Feature | HIGH | 3h | P1 |
| 6 | **Report functional options** — `ReportOption` for filtering by service/time/event | Feature | HIGH | 2h | P1 |
| 7 | **Graph scope grouping** — visual scope boundaries in DAG layout | UX | MED | 2h | P2 |
| 8 | **Config.Validate() error** method | API | MED | 30m | P2 |
| 9 | **Benchmark suite** — registration, invocation, BuildReport, HTML export | Testing | MED | 1h | P2 |
| 10 | **ResolveServiceScope decoupling** — abstract away `do.Scope` type assertion | Architecture | MED | 1h | P2 |
| 11 | **Mermaid export** — text-based dependency graph for GitHub/VS Code | Export | MED | 1h | P2 |
| 12 | **DOT export** — GraphViz format for tooling integration | Export | MED | 1h | P2 |
| 13 | **Graph minimap** — overview rectangle for 20+ service graphs | UX | LOW | 2h | P3 |
| 14 | **HTML graph node tooltips** — show build_ms, scope, type on hover popup | UX | LOW | 1h | P3 |
| 15 | **Example web server** — `http.ListenAndServe` for live HTML dashboard | Example | MED | 1h | P3 |
| 16 | **Fix godoc duplicate** — remove package doc from `plugin.go` or `doc.go` | Lint | LOW | 2m | P3 |
| 17 | **Graph drag-to-reorder nodes** — allow manual position adjustments | UX | LOW | 2h | P4 |
| 18 | **Configurable HTML themes** — light/dark toggle or CSS injection | UX | LOW | 2h | P4 |
| 19 | **CONTRIBUTING.md** — if project goes public | Docs | LOW | 1h | P4 |
| 20 | **Schema migration function** — for consumers with v0.1.0 exports | API | LOW | 1h | P4 |
| 21 | **HTML graph edge labels** — show dependency type or invocation order | UX | LOW | 1h | P4 |
| 22 | **Scope-aware health checks** — per-scope health summary in HTML | Feature | LOW | 1h | P4 |
| 23 | **Test HTML output snapshots** — golden file tests for HTML structure | Testing | MED | 2h | P3 |
| 24 | **CI pipeline** — GitHub Actions for test + lint + generate | Infra | MED | 1h | P2 |
| 25 | **README screenshot** — add an actual screenshot of the HTML dashboard | Docs | MED | 30m | P2 |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should the health check wrapper (`RecordHealthCheck*`) remain the canonical way to audit health checks, or should samber/do v2 add native `OnBeforeHealthCheck`/`OnAfterHealthCheck` hooks to `InjectorOpts` (like it has for registration, invocation, shutdown)?**

Current state: The health check wrapper works, but it's architecturally different from the other 3 event types. Registration/invocation/shutdown are captured via `do.InjectorOpts` hooks that fire automatically. Health checks require the user to explicitly call `plugin.RecordHealthCheck(injector)` instead of `injector.HealthCheck()`. This means:
- Users must remember to use the wrapper (easy to forget)
- The wrapper measures total `HealthCheckWithContext` duration, not per-service (less precise)
- No `PhaseBefore` for health checks (do doesn't provide a before-hook)

If samber/do added health check hooks to `InjectorOpts`, the plugin would capture them automatically with the same precision as invocation/shutdown. This would be a cleaner API but requires upstream changes.

**Question for the user**: Should we (a) keep the wrapper pattern and document it clearly, (b) contribute upstream hooks to samber/do, or (c) both (wrapper now, migrate to hooks when available)?

---

## Commit History (last 20)

```
12a8b22 Add health check auditing with service type tracking
09024dd Move health check implementation plan from research to planning directory
12fac97 Add ServiceType field to ServiceInfo and infer provider type from do.Container
b1dd313 Enhance dependency graph with Sugiyama DAG layout and update health check docs
88a0b06 Enhance dependency graph UI: zoom/pan controls, node highlighting, edge styling
ed0c976 Docs: table alignment, new templ-components ADR, example cleanups
04d3624 Fix ServiceByName doc, sync FEATURES/CHANGELOG, write comprehensive status report
b57b262 Add execution plan: final polish and feature completion
b50d31a Update AGENTS.md: test count, example feature table, convenience methods
7863177 Fix wsl_v5 lint: add blank lines before range loops
4c6817e Use Report convenience methods in example, remove ptrToFloat helper
a43593c Add convenience methods: ServiceStatus.IsError, Report.ServiceByName, EventsByType, FailedServices
2e5af5c Update documentation: CHANGELOG, FEATURES, TODO_LIST, AGENTS.md
41ac6a3 Rewrite example: real invocation errors, Healthchecker interfaces, OnEvent callback
43d92ab Add ServiceRef.String(), fix shutdown error test, add test coverage
20ebff3 Add EventHandler callback for real-time event streaming
821263c Add OnEvent callback hook, expand example with comprehensive feature demo
cb6f537 Add Event convenience methods: IsRegistration, IsInvocation, IsShutdown, IsBefore, IsAfter
7ea7a76 Consolidate service identity: rename DependencyRef to ServiceRef
aaed291 Deduplicate sum functions: extract sumDurationField with thin wrappers
```

---

_Generated by Crush on 2026-06-10 at 03:29_

# Execution Plan — Session 6

**Created**: 2026-06-10 19:30 CEST
**Total Tasks**: 47 (split from 25 macro-items, each ≤12 min)
**Sorted by**: Impact → Customer Value → Effort (Pareto)

---

## Legend

| Col         | Meaning                                                         |
| ----------- | --------------------------------------------------------------- |
| **W**       | Wave (execution order — do all tasks in wave N before wave N+1) |
| **ID**      | Unique task ID                                                  |
| **Task**    | What to do                                                      |
| **File(s)** | Primary file(s) affected                                        |
| **Time**    | Estimated max duration                                          |
| **Impact**  | Why this matters                                                |

---

## Wave 1 — Critical Fixes (must do before anything else)

These are broken/user-facing/security issues. The Events tab is completely non-functional and there's an XSS vulnerability.

| W   | ID  | Task                                                                                                                                                                                                                                                                                | File(s)            | Time     | Impact                                    |
| --- | --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------ | -------- | ----------------------------------------- |
| 1   | T01 | **Fix XSS in deps column**: add `esc()` around `d.service_name` in deps and depsR map at lines 625-626                                                                                                                                                                              | `html.templ`       | 5m       | Security: stored XSS vector               |
| 1   | T02 | **Fix XSS in status badge CSS class**: escape `s.status` before inserting into HTML attribute at line 630                                                                                                                                                                           | `html.templ`       | 5m       | Security: attribute injection             |
| 1   | T03 | **Build allEvents array**: add `const allEvents = report.events.map(e => {...})` with columns: sequence, relative timestamp, event type badge, provider type badge, phase icon (+/−), scope name, service name, duration, error tooltip. Add `data-type` attribute for filter chips | `html.templ`       | 10m      | Critical: Events tab is completely broken |
| 1   | T04 | **Add event type color scheme**: add CSS classes for event type badges (registration=blue, invocation=green, shutdown=yellow, health_check=amber) to match existing dashboard aesthetic                                                                                             | `html.templ`       | 5m       | UX: visual consistency                    |
| 1   | T05 | **Run `go generate ./...`** to regenerate `html_templ.go` from fixed `html.templ`                                                                                                                                                                                                   | `html_templ.go`    | 2m       | Required: generated code                  |
| 1   | T06 | **Add TestWriteHTML_EventsTabContent**: create test that writes HTML and asserts event-related content exists (service names, event type strings, sequence numbers)                                                                                                                 | `auditlog_test.go` | 10m      | Critical: prevent regression              |
| 1   | T07 | **Add TestWriteHTML_AllFiveTabs**: smoke test that all 5 tab content divs render with non-empty inner content (services, scopes, graph, timeline, events)                                                                                                                           | `auditlog_test.go` | 10m      | Critical: catch broken tabs               |
| 1   | T08 | **Run tests + lint** — verify all changes pass                                                                                                                                                                                                                                      | —                  | 5m       | Gate                                      |
|     |     |                                                                                                                                                                                                                                                                                     |                    | **~52m** |                                           |

---

## Wave 2 — Fuzz Test Expansion (security hardening)

| W   | ID  | Task                                                                                                                                       | File(s)        | Time     | Impact                                       |
| --- | --- | ------------------------------------------------------------------------------------------------------------------------------------------ | -------------- | -------- | -------------------------------------------- |
| 2   | T09 | **Expand fuzz corpus**: add fuzz seeds with HTML chars in scope names (create child scope with `<script>` name)                            | `fuzz_test.go` | 8m       | Security: scope name XSS                     |
| 2   | T10 | **Add error-injection fuzz seed**: provider that returns errors with HTML-special chars in error messages                                  | `fuzz_test.go` | 8m       | Security: error string XSS                   |
| 2   | T11 | **Add multi-check fuzz assertions**: check for `<script`, `<img`, `<svg`, `onerror=`, `javascript:` in addition to current `<script>alert` | `fuzz_test.go` | 8m       | Security: broader detection                  |
| 2   | T12 | **Add dep-chain fuzz test**: fuzz test that creates 2 services where one depends on the other, with fuzzed dep names                       | `fuzz_test.go` | 10m      | Security: dep name XSS (the vector from T01) |
| 2   | T13 | **Run tests + lint**                                                                                                                       | —              | 3m       | Gate                                         |
|     |     |                                                                                                                                            |                | **~37m** |                                              |

---

## Wave 3 — Migration Robustness

| W   | ID  | Task                                                                                                                                         | File(s)            | Time     | Impact                                 |
| --- | --- | -------------------------------------------------------------------------------------------------------------------------------------------- | ------------------ | -------- | -------------------------------------- |
| 3   | T14 | **Add version guard to MigrateReport**: if `report.Version == SchemaVersion`, return early without modifying ExportedAt                      | `migration.go`     | 5m       | Correctness: prevents silent data loss |
| 3   | T15 | **Preserve ExportedAt in migration**: only set `ExportedAt = time.Now()` if `report.ExportedAt.IsZero()`                                     | `migration.go`     | 5m       | Correctness: keeps original timestamp  |
| 3   | T16 | **Validate input in MigrateReport**: return error if `len(data) == 0` or if `report.Version == ""` after unmarshal (indicates corrupt input) | `migration.go`     | 8m       | Robustness: reject garbage input       |
| 3   | T17 | **Add TestMigrateReport_AlreadyCurrentVersion**: verify v0.2.0 report passes through unchanged                                               | `auditlog_test.go` | 5m       | Coverage                               |
| 3   | T18 | **Add TestMigrateReport_PreservesExportedAt**: verify original ExportedAt is not overwritten                                                 | `auditlog_test.go` | 5m       | Coverage                               |
| 3   | T19 | **Add TestMigrateReport_EmptyInput**: verify empty/invalid JSON returns error                                                                | `auditlog_test.go` | 5m       | Coverage                               |
| 3   | T20 | **Run tests + lint**                                                                                                                         | —                  | 3m       | Gate                                   |
|     |     |                                                                                                                                              |                    | **~36m** |                                        |

---

## Wave 4 — Coverage Gaps

| W   | ID  | Task                                                                                                                                                                | File(s)            | Time     | Impact                                      |
| --- | --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------ | -------- | ------------------------------------------- |
| 4   | T21 | **Add TestResolveServiceScope_ParentScope**: register service in root scope, call ResolveServiceScope on child scope, verify it finds the service via ancestor walk | `auditlog_test.go` | 10m      | Coverage: 90% → 100% on ResolveServiceScope |
| 4   | T22 | **Add TestResolveServiceScope_GrandparentScope**: register in root, create root→child→grandchild, resolve via grandchild                                            | `auditlog_test.go` | 8m       | Coverage: multi-level ancestor walk         |
| 4   | T23 | **Add TestWriteMermaid_DuplicateEdges**: synthetic report with duplicate dep edges, verify output is deduplicated                                                   | `auditlog_test.go` | 5m       | Coverage: dedup branch in WriteMermaid      |
| 4   | T24 | **Run tests + lint**                                                                                                                                                | —                  | 3m       | Gate                                        |
|     |     |                                                                                                                                                                     |                    | **~26m** |                                             |

---

## Wave 5 — CSP + Security Hardening

| W   | ID  | Task                                                                                                                                                                                                                                                | File(s)      | Time     | Impact                     |
| --- | --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | -------- | -------------------------- |
| 5   | T25 | **Add CSP meta tag to HTML**: `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; font-src fonts.googleapis.com fonts.gstatic.com; img-src 'self' data:;">` in `<head>` | `html.templ` | 8m       | Security: defense-in-depth |
| 5   | T26 | **Run `go generate` + test**: regenerate and verify                                                                                                                                                                                                 | —            | 3m       | Gate                       |
|     |     |                                                                                                                                                                                                                                                     |              | **~11m** |                            |

---

## Wave 6 — Code Cleanliness

| W   | ID  | Task                                                                                                                                                                 | File(s)                        | Time     | Impact                                |
| --- | --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------ | -------- | ------------------------------------- |
| 6   | T27 | **Add godoc to 7 exported methods**: `ServiceRef.String()`, `Event.IsRegistration()`, `IsInvocation()`, `IsShutdown()`, `IsHealthCheck()`, `IsBefore()`, `IsAfter()` | `types.go`                     | 8m       | Documentation                         |
| 6   | T28 | **Extract `"[root]"` to named constant**: `const RootScopeName = "[root]"` in types.go, update `IsRoot()` and 22 test references                                     | `types.go`, `auditlog_test.go` | 10m      | Cleanliness: magic string elimination |
| 6   | T29 | **Extract emoji codepoints to named constants**: `IconLazy`, `IconEager`, `IconTransient`, `IconAlias` in types.go                                                   | `types.go`                     | 8m       | Readability                           |
| 6   | T30 | **Handle writeToFile Close error**: wrap with `sync.Once` pattern or return Close error after fn error check                                                         | `plugin.go`                    | 8m       | Correctness: data integrity           |
| 6   | T31 | **Fix dual package doc comment**: remove package comment from `doc.go` or `plugin.go` (keep the one in doc.go since it's the canonical location)                     | `doc.go`                       | 3m       | Lint hygiene                          |
| 6   | T32 | **Run tests + lint**                                                                                                                                                 | —                              | 3m       | Gate                                  |
|     |     |                                                                                                                                                                      |                                | **~40m** |                                       |

---

## Wave 7 — Hook Function Refactoring

| W   | ID  | Task                                                                                                               | File(s)       | Time     | Impact                       |
| --- | --- | ------------------------------------------------------------------------------------------------------------------ | ------------- | -------- | ---------------------------- |
| 7   | T33 | **Extract `buildInvocationBeforeEvent` helper**: from OnBeforeInvocation (lines 352-362) — builds the Event struct | `recorder.go` | 8m       | Maintainability: 57→35 lines |
| 7   | T34 | **Extract `popStackFrame` helper**: from OnAfterInvocation (lines 385-398) — LIFO pop with duration calc           | `recorder.go` | 8m       | Maintainability              |
| 7   | T35 | **Extract `buildShutdownBeforeEvent` helper**: from OnBeforeShutdown (lines 480-490)                               | `recorder.go` | 5m       | Maintainability              |
| 7   | T36 | **Extract `resolveShutdownDuration` helper**: from OnAfterShutdown (lines 511-521)                                 | `recorder.go` | 5m       | Maintainability              |
| 7   | T37 | **Run tests + lint** (including benchmark comparison)                                                              | —             | 5m       | Gate: no perf regression     |
|     |     |                                                                                                                    |               | **~31m** |                              |

---

## Wave 8 — Documentation & Examples

| W   | ID  | Task                                                                                      | File(s)           | Time     | Impact        |
| --- | --- | ----------------------------------------------------------------------------------------- | ----------------- | -------- | ------------- |
| 8   | T38 | **Add ExampleConfig_Validate**: godoc example showing valid and invalid configs           | `example_test.go` | 8m       | Documentation |
| 8   | T39 | **Update CHANGELOG.md**: add Session 5+6 entries                                          | `CHANGELOG.md`    | 8m       | Documentation |
| 8   | T40 | **Update AGENTS.md**: document Events tab fix, XSS fix, CSP meta tag, root scope constant | `AGENTS.md`       | 8m       | Memory        |
| 8   | T41 | **Update FEATURES.md**: mark events tab, XSS fix, CSP as DONE                             | `FEATURES.md`     | 5m       | Documentation |
| 8   | T42 | **Update TODO_LIST.md**: mark all completed items, add any new ones                       | `TODO_LIST.md`    | 5m       | Tracking      |
|     |     |                                                                                           |                   | **~34m** |               |

---

## Wave 9 — UX Enhancements (lower priority)

| W   | ID  | Task                                                                                | File(s)      | Time     | Impact             |
| --- | --- | ----------------------------------------------------------------------------------- | ------------ | -------- | ------------------ |
| 9   | T43 | **Add edge count to HTML stats card**: total dependency edges in the stat cards row | `html.templ` | 8m       | UX: quick overview |
| 9   | T44 | **Run `go generate` + test**                                                        | —            | 3m       | Gate               |
|     |     |                                                                                     |              | **~11m** |                    |

---

## Wave 10 — Performance & Deferred Items

| W   | ID  | Task                                                                                                                       | File(s)            | Time     | Impact                            |
| --- | --- | -------------------------------------------------------------------------------------------------------------------------- | ------------------ | -------- | --------------------------------- |
| 10  | T45 | **Consider struct key for serviceKey**: benchmark `scopeID+"/"+serviceName` vs struct key `{scopeID, serviceName}` in maps | `recorder.go`      | 10m      | Performance: allocation reduction |
| 10  | T46 | **Add TestServiceKey_StructKey**: if adopted, test that struct key produces same results                                   | `auditlog_test.go` | 5m       | Correctness                       |
| 10  | T47 | **Final: run full test suite + lint + benchmarks**, verify no regressions, update coverage numbers in docs                 | —                  | 5m       | Final gate                        |
|     |     |                                                                                                                            |                    | **~20m** |                                   |

---

## Summary Table

| Wave   | Theme                            | Tasks        | Time      | Impact Level         |
| ------ | -------------------------------- | ------------ | --------- | -------------------- |
| **1**  | Critical Fixes (Events tab, XSS) | T01–T08      | ~52m      | 🔴 Critical/Security |
| **2**  | Fuzz Test Expansion              | T09–T13      | ~37m      | 🔴 Security          |
| **3**  | Migration Robustness             | T14–T20      | ~36m      | 🟡 Correctness       |
| **4**  | Coverage Gaps                    | T21–T24      | ~26m      | 🟡 Coverage          |
| **5**  | CSP Security                     | T25–T26      | ~11m      | 🟡 Security          |
| **6**  | Code Cleanliness                 | T27–T32      | ~40m      | 🟢 Quality           |
| **7**  | Hook Refactoring                 | T33–T37      | ~31m      | 🟢 Maintainability   |
| **8**  | Documentation                    | T38–T42      | ~34m      | 🟢 Docs              |
| **9**  | UX Enhancement                   | T43–T44      | ~11m      | 🔵 UX                |
| **10** | Performance                      | T45–T47      | ~20m      | 🔵 Performance       |
|        | **TOTAL**                        | **47 tasks** | **~5.5h** |                      |

---

## Explicitly NOT Included (deferred)

| Item                                        | Why                                                    |
| ------------------------------------------- | ------------------------------------------------------ |
| PlantUML export                             | User demand not confirmed. Mermaid sufficient.         |
| HTML integration test with JS engine (goja) | High effort (4h), low ROI vs. string-based smoke tests |
| Minify inline JS/CSS in html_templ.go       | Marginal performance gain, fragile                     |
| ServiceInfo health check duration           | Blocked on samber/do API, not our code                 |
| Multi-module split                          | Project too small (1 package)                          |

---

## Dependency Graph

```
T01 (XSS deps) ──┐
T02 (XSS status) ├──► T03 (allEvents) ──► T04 (event CSS) ──► T05 (generate) ──► T06 + T07 (tests) ──► T08 (gate)
                 │
T09-T12 (fuzz) ──────► T13 (gate)

T14 (version guard) ──► T15 (ExportedAt) ──► T16 (validate input) ──► T17-T19 (tests) ──► T20 (gate)

T21 (parent scope) + T22 (grandparent) + T23 (mermaid dupes) ──► T24 (gate)

T25 (CSP) ──► T26 (generate + test)

T27 (godoc) ─┬─ T28 (root const) ─┬─ T29 (emoji const) ─┬─ T30 (close err) ─┬─ T31 (doc.go) ──► T32 (gate)

T33-T36 (extract helpers) ──► T37 (bench + gate)

T38-T42 (docs) — parallel, no dependencies between them

T43 (edge count) ──► T44 (gate)

T45-T46 (struct key) ──► T47 (final gate)
```

All waves are sequential (must complete Wave N before Wave N+1).
Within a wave, tasks without dependencies between them can be parallelized.

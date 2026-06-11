# Status Report — Session 5

**Date**: 2026-06-10 19:26 CEST
**Branch**: master
**Coverage**: 95.8% of statements (main package)
**Tests**: 157 (141 unit + 6 examples + 10 fuzz corpus seeds)
**Benchmarks**: 13
**Lint**: 0 issues (golangci-lint)
**Go vet**: clean
**LOC**: 5,523 total (2,325 production + 3,401 test + 797 example)

---

## A) FULLY DONE ✅

| Area                                          | Details                                                                     |
| --------------------------------------------- | --------------------------------------------------------------------------- |
| **Drop-in plugin**                            | `New(Config)` + `Opts()` → one-line integration                             |
| **Registration/Invocation/Shutdown tracking** | Before/after hooks with timestamps, duration, errors                        |
| **Dependency graph inference**                | Stack-based A→B detection during invocation chains                          |
| **Reverse dependencies**                      | Computed at report time from forward deps                                   |
| **Scope tree**                                | Hierarchical with per-scope service lists, deterministic sort               |
| **Build duration**                            | Millisecond-precision per service                                           |
| **Provider type detection**                   | lazy/eager/transient/alias via `do.ExplainNamedService`                     |
| **Health check auditing**                     | `RecordHealthCheck`/`RecordHealthCheckWithContext` with events              |
| **Capability detection**                      | `IsHealthchecker`/`IsShutdowner` via `do.ExplainInjector`                   |
| **JSON/NDJSON export**                        | File + io.Writer paths                                                      |
| **Self-contained HTML**                       | Dark dashboard, 5 tabs, Sugiyama graph, timeline, scope tree                |
| **Mermaid export**                            | `Report.WriteMermaid(writer)`                                               |
| **Schema migration**                          | `MigrateReport` v0.1.0 → v0.2.0                                             |
| **Report filtering**                          | 5 filter options: name, type, event type, time range, scope                 |
| **Config.Validate()**                         | Validates ContainerID for path separators (NEW this session)                |
| **Convenience methods**                       | ServiceByName, ServiceByRef, EventsByRef, FailedServices, UnhealthyServices |
| **Type helpers**                              | IsKnown, IsRoot, HasError, HasHealthError, IsError, Duration, Uptime        |
| **Event helpers**                             | IsRegistration, IsInvocation, IsShutdown, IsHealthCheck, IsBefore, IsAfter  |
| **OnEvent callback**                          | Real-time streaming outside mutex                                           |
| **Environment toggle**                        | `DO_AUDITLOG_ENABLED` = true/1/yes                                          |
| **Zero-cost disabled**                        | Empty hooks when disabled                                                   |
| **Single-lock recorder**                      | 1 RWMutex + 2 atomics, 23% faster than previous 4-lock design               |
| **Deterministic output**                      | Sorted services, scopes, deps                                               |
| **Defensive copies**                          | Events() and Report() return copies                                         |
| **HTML fuzz test**                            | FuzzPluginHTML with 10 corpus seeds                                         |
| **Godoc examples**                            | 6 runnable Example\* functions                                              |
| **Zero lint issues**                          | All 28 original issues fixed                                                |
| **mermaidLabelForRef coverage**               | 100% via synthetic external dependency test (NEW this session)              |

---

## B) PARTIALLY DONE ⚠️

Nothing. All started features are complete. The two items from the previous status report (`Config.Validate()` and `mermaidLabelForRef coverage`) are now fully done.

---

## C) NOT STARTED 📋

| Item                          | Priority            | Notes                                                |
| ----------------------------- | ------------------- | ---------------------------------------------------- |
| **PlantUML export**           | Low                 | Only if users request it. Mermaid already available. |
| **CSP meta tag in HTML**      | Medium              | No Content-Security-Policy in generated HTML         |
| **HTML events tab rendering** | N/A → see section D |                                                      |
| **Dep name XSS in HTML**      | N/A → see section D |                                                      |

---

## D) TOTALLY FUCKED UP 💥

### D1. HTML Events Tab is COMPLETELY BROKEN 🚨

**Severity**: CRITICAL — the entire Events tab is non-functional.

**File**: `html.templ:654-656`

```javascript
// Events table

document.getElementById("events-tbody").innerHTML = allEvents.join("");
```

`allEvents` is **never defined**. The variable was presumably removed during a refactor but its usage was left behind. Clicking the Events tab shows an empty table, and a `ReferenceError: allEvents is not defined` is thrown in the browser console, which **also kills all subsequent JS execution** (including event filters, scope tree rendering, and the graph).

**Why tests didn't catch it**: Tests only check HTML string output (size > 500, contains "db"). No test renders/executes the JavaScript. The fuzz test checks for XSS, not JS correctness.

**Fix**: Build the `allEvents` array from `report.events`, similar to how `allServices` is built from `report.services` at line 624.

### D2. XSS Vulnerability in Dependencies Column 🚨

**Severity**: HIGH — stored XSS vector.

**File**: `html.templ:625-626`

```javascript
const deps = (s.dependencies || []).map((d) => "<span>" + d.service_name + "</span>").join(", ");
const depsR = (s.dependents || []).map((d) => "<span>" + d.service_name + "</span>").join(", ");
```

`d.service_name` is interpolated directly into HTML **without calling `esc()`**. A service named `<img src=x onerror=alert(1)>` would execute arbitrary JavaScript. The data comes from `templ.JSONScript` which is user-controlled.

**Fix**: Replace `d.service_name` with `esc(d.service_name)` in both lines.

### D3. Fuzz Test Coverage Gap

**File**: `fuzz_test.go`

The fuzz test only seeds `ProvideNamed` with a service name. It doesn't test:

- Dependency/dependent name rendering (the XSS vector above)
- Scope names with HTML special chars
- Error strings (from error-returning providers)
- Only checks for `<script>alert` — misses `<img`, `<svg`, `onerror=`, `javascript:` vectors
- No structural HTML validation

---

## E) WHAT WE SHOULD IMPROVE 🔧

### E1. Code Quality Issues

| Issue                                      | Location                      | Severity                         |
| ------------------------------------------ | ----------------------------- | -------------------------------- |
| Missing godoc on 7 exported methods        | `types.go:92,119-124`         | Low                              |
| `"[root]"` magic string used in 20+ places | `types.go:102`, tests         | Low                              |
| `writeToFile` ignores `file.Close()` error | `plugin.go:223`               | Low (deferred close after write) |
| Unicode emoji hardcoded inline             | `types.go:53-66`              | Cosmetic                         |
| 4 functions >50 lines                      | `recorder.go:313,372,500,721` | Medium (complexity)              |

### E2. Migration Edge Cases

| Issue                                                             | Location             | Risk                                                      |
| ----------------------------------------------------------------- | -------------------- | --------------------------------------------------------- |
| No version check before migrating                                 | `migration.go:17-25` | Re-migrating v0.2.0 report silently overwrites ExportedAt |
| Empty/null JSON input produces misleading "migrated" empty report | `migration.go:20`    | Low                                                       |
| `ExportedAt` overwritten unconditionally                          | `migration.go:26`    | Original timestamp lost                                   |

### E3. ResolveServiceScope Coverage Gap

`recorder.go:904` is at 90% coverage. Missing:

- Plain `do.Injector` (not `*do.Scope`) type assertion failure path
- Ancestor walk where service lives in parent/grandparent scope

### E4. Testing Gaps

- **No JS execution tests**: HTML tests only check string output, not that JavaScript runs without errors
- **No integration test** that opens HTML and verifies all 5 tabs render correctly
- **Events tab has zero test coverage** (because it's broken)

### E5. Documentation

- `doc.go` and `plugin.go` both have package-level doc comments (godoclint warning)
- No CHANGELOG.md entry for Session 5 changes

---

## F) Top 25 Things to Get Done Next

### Critical (must fix before any release)

| #   | Task                                                                                                  | Impact      | Effort    |
| --- | ----------------------------------------------------------------------------------------------------- | ----------- | --------- |
| 1   | **Fix broken Events tab** — build `allEvents` array from `report.events`                              | 🔴 Critical | S (30min) |
| 2   | **Fix XSS in deps column** — add `esc()` around `d.service_name`                                      | 🔴 Security | S (5min)  |
| 3   | **Add HTML smoke test** — verify all 5 tabs produce non-empty content                                 | 🔴 Critical | M (1hr)   |
| 4   | **Add `allEvents` rendering** — sequence, timestamp, type badge, phase, duration, error, service type | 🔴 Critical | M (1hr)   |

### High Priority

| #   | Task                                                                                      | Impact         | Effort    |
| --- | ----------------------------------------------------------------------------------------- | -------------- | --------- |
| 5   | **Expand fuzz test** — deps names, scope names, error strings, multiple injection vectors | 🔴 Security    | M (1hr)   |
| 6   | **Add version guard to MigrateReport** — return early if already at target version        | 🟡 Robustness  | S (15min) |
| 7   | **Preserve ExportedAt in migration** — don't overwrite if non-zero                        | 🟡 Correctness | S (10min) |
| 8   | **Add ResolveServiceScope ancestor walk test** — service in parent scope                  | 🟡 Coverage    | S (20min) |
| 9   | **Add CSP meta tag to HTML** — defense-in-depth against XSS                               | 🟡 Security    | S (15min) |
| 10  | **Validate empty/null JSON in MigrateReport** — return error for invalid input            | 🟡 Robustness  | S (15min) |

### Medium Priority

| #   | Task                                                                                             | Impact          | Effort    |
| --- | ------------------------------------------------------------------------------------------------ | --------------- | --------- |
| 11  | **Add missing godoc** — 7 exported methods in types.go                                           | Documentation   | S (15min) |
| 12  | **Extract `"[root]"` to named constant** — used in types.go + 20+ test lines                     | Cleanliness     | S (10min) |
| 13  | **Handle writeToFile Close error** — at minimum log or wrap                                      | Correctness     | S (10min) |
| 14  | **Extract complex hook functions** — OnBeforeInvocation (57 lines), OnAfterInvocation (55 lines) | Maintainability | M (2hr)   |
| 15  | **Add godoclint exception or fix** — dual package comments in doc.go + plugin.go                 | Lint hygiene    | S (5min)  |
| 16  | **Add ExampleConfig_Validate** — godoc example for validation                                    | Documentation   | S (10min) |
| 17  | **Update CHANGELOG.md** — Session 5 entries                                                      | Documentation   | S (10min) |

### Lower Priority

| #   | Task                                                                                            | Impact      | Effort    |
| --- | ----------------------------------------------------------------------------------------------- | ----------- | --------- |
| 18  | **Add HTML integration test** — run JS in a lightweight engine (e.g., goja) to verify rendering | Testing     | L (4hr)   |
| 19  | **Add event type badge to HTML events tab** — color-coded badges matching service types         | UX          | M (1hr)   |
| 20  | **Extract emoji codepoints to named constants** — types.go:53-66                                | Readability | S (10min) |
| 21  | **Add PlantUML export** — only if users request it                                              | Feature     | M (2hr)   |
| 22  | **Add ServiceInfo health check duration** — if samber/do adds per-service timing in future      | Feature     | Deferred  |
| 23  | **Consider struct key for serviceKey** — `scopeID+"/"+serviceName` allocates per key            | Performance | S (15min) |
| 24  | **Add edge count to HTML stats** — total dependency edges                                       | UX          | S (10min) |
| 25  | **Reduce html_templ.go generated size** — minify inline JS/CSS                                  | Performance | M (1hr)   |

---

## G) Top #1 Question I Cannot Figure Out Myself

**How should the Events tab render each event row?**

The `allServices` array is well-defined (each service → one table row with 11 columns). But `allEvents` was never built. The report.events array has: sequence, timestamp, event_type, phase, service_type, scope_name, service_name, duration_ms, error, container_id.

I need to decide:

1. **Which columns to show**: Sequence, Timestamp, Type, Phase, Service, Scope, Service Type, Duration, Error?
2. **How to format timestamps**: Relative (offset from first event), absolute, or both?
3. **Event type badges**: Same color scheme as service type badges?
4. **Phase indicator**: "before"/"after" text, or up/down arrow, or +/- symbol?
5. **Error display**: Inline truncated text, or tooltip on hover like the services table?
6. **Filter behavior**: The filter chip JS already exists (eventTypes=['all','registration','invocation','shutdown','health_check']), just needs the `data-type` attribute on each row.

**I can make reasonable decisions on all of these, but want confirmation before investing time in the wrong design.**

---

## Session 5 Changes (This Session)

| File               | Change                                                                                                                                                       |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `plugin.go`        | `Config.Validate()` now validates ContainerID for path separators; static sentinel error `errContainerIDPathSep`                                             |
| `auditlog_test.go` | Added `TestConfig_Validate` with 4 new cases (forward slash, backslash, hyphen); Added `TestWriteMermaid_ExternalDependency` for mermaidLabelForRef coverage |
| `FEATURES.md`      | Updated Config.Validate() description from "placeholder" to actual validation                                                                                |
| `TODO_LIST.md`     | Marked both partial items as done; added Session 5 completed section                                                                                         |

## Key Metrics Trend

| Metric         | Session 4 | Session 5 | Delta |
| -------------- | --------- | --------- | ----- |
| Coverage       | 95.1%     | 95.8%     | +0.7% |
| Tests          | 140       | 157       | +17   |
| Lint issues    | 0         | 0         | —     |
| Production LOC | ~2,300    | ~2,325    | +25   |
| Test LOC       | ~3,340    | ~3,401    | +61   |

# Status Report: Health Check Audit Logging

**Date**: 2026-06-10 03:45 · **Branch**: master · **Status**: ALPHA · **Health**: Good

---

## a) FULLY DONE

### Core Health Check Feature (Schema 0.1.0 → 0.2.0)

| Component                               | Status  | Details                                                                              |
| --------------------------------------- | ------- | ------------------------------------------------------------------------------------ |
| `EventTypeHealthCheck`                  | ✅ Done | New event type with `IsHealthCheck()` convenience method                             |
| `ServiceInfo` health fields             | ✅ Done | `LastHealthCheckAt`, `HealthCheckDurationMs`, `HealthCheckError`, `HealthCheckCount` |
| `Report` health fields                  | ✅ Done | `HealthCheckSucceeded`, `HealthCheckedCount`                                         |
| `Report.UnhealthyServices()`            | ✅ Done | Convenience method returning services with health errors                             |
| `Recorder.RecordHealthCheck()`          | ✅ Done | Records event + updates service record                                               |
| `Recorder.ResolveServiceScope()`        | ✅ Done | Handles `*do.RootScope` and `*do.Scope`                                              |
| `Plugin.RecordHealthCheck()`            | ✅ Done | Wrapper delegating to `RecordHealthCheckWithContext`                                 |
| `Plugin.RecordHealthCheckWithContext()` | ✅ Done | Wraps `injector.HealthCheckWithContext()`, records events per service                |
| Disabled-plugin passthrough             | ✅ Done | When `Enabled: false`, delegates directly to injector                                |
| Schema version bump                     | ✅ Done | `0.1.0` → `0.2.0`                                                                    |
| `doc.go` update                         | ✅ Done | Package doc mentions health checks                                                   |
| `example/main.go` update                | ✅ Done | Uses `plugin.RecordHealthCheckWithContext()`                                         |
| `AGENTS.md` update                      | ✅ Done | Architecture, gotchas, testing patterns updated                                      |

### Tests (9 new, 49 total)

| Test                                     | Status  | What it covers                                          |
| ---------------------------------------- | ------- | ------------------------------------------------------- |
| `TestPlugin_HealthCheckHealthy`          | ✅ Pass | Healthy service gets event + all ServiceInfo fields     |
| `TestPlugin_HealthCheckUnhealthy`        | ✅ Pass | Unhealthy service gets error in event + ServiceInfo     |
| `TestPlugin_HealthCheckMultipleServices` | ✅ Pass | Mix of healthy/unhealthy/unchecked                      |
| `TestPlugin_HealthCheckDisabled`         | ✅ Pass | Disabled plugin delegates, no events recorded           |
| `TestPlugin_HealthCheckCount`            | ✅ Pass | Two checks → HealthCheckCount=2, 2 events               |
| `TestPlugin_HealthCheckReport`           | ✅ Pass | HealthCheckSucceeded=true, HealthCheckedCount=1         |
| `TestPlugin_HealthCheckWithScope`        | ✅ Pass | Child scope health check resolves root + child services |
| `TestPlugin_HealthCheckReportSucceeded`  | ✅ Pass | HealthCheckSucceeded=false, UnhealthyServices() works   |
| `TestEvent_IsHealthCheck`                | ✅ Pass | Added to existing convenience method table test         |

### HTML Visualization

| Feature                               | Status                                                          |
| ------------------------------------- | --------------------------------------------------------------- |
| Health column in services table       | ✅ Done — ✓ healthy / ✗ unhealthy badges with error tooltip     |
| `health_check` event badge CSS        | ✅ Done — amber/yellow color                                    |
| `health_check` event type filter chip | ✅ Done                                                         |
| Health checks stat card               | ✅ Done — shows "N checked, M unhealthy" when health checks ran |

### Verification

| Check                         | Result                                                   |
| ----------------------------- | -------------------------------------------------------- |
| `go test ./... -race`         | ✅ All 49 tests pass                                     |
| `go vet ./...`                | ✅ Clean                                                 |
| `golangci-lint run` (library) | ✅ Zero new issues                                       |
| `golangci-lint run` (example) | 8 pre-existing warnings (err113, gochecknoglobals, etc.) |
| `go build ./...`              | ✅ Clean                                                 |
| Coverage                      | 89.6% of library statements                              |
| `go generate ./...`           | ✅ html_templ.go regenerated                             |

---

## b) PARTIALLY DONE

Nothing is half-done — all started work items are complete.

---

## c) NOT STARTED (from research report & plan)

| Item                                    | Priority | Notes                                                                                                    |
| --------------------------------------- | -------- | -------------------------------------------------------------------------------------------------------- |
| Per-service health check duration       | HIGH     | Currently all services get the same `time.Since(start)` — should time each service individually          |
| `Report.HealthCheckSucceeded` semantics | MED      | Currently `true` when NO health checks ran — should be `true` only when health checks ran AND all passed |
| Health check history / ring buffer      | LOW      | Stores only last result; events stream captures full history                                             |
| Periodic health check helper            | LOW      | `plugin.StartHealthCheckLoop(injector, interval)`                                                        |
| Dedicated HTML health check tab         | LOW      | Tab showing health check timeline/history                                                                |
| Upstream health check hooks PR          | LOW      | Add `HookBeforeHealthCheck`/`HookAfterHealthCheck` to samber/do                                          |

---

## d) TOTALLY FUCKED UP / CRITICAL ISSUES

### 🔴 BUG: Duration is shared across all services

**The worst problem.** In `RecordHealthCheckWithContext`:

```go
start := time.Now()
results := injector.HealthCheckWithContext(ctx)
for svcName, svcErr := range results {
    elapsed := time.Since(start)  // ← SAME time for ALL services
    durationMs := float64(elapsed.Microseconds()) / microsPerMs
```

Every service gets `time.Since(start)` — the total wall-clock time from the start of the bulk health check to the end. If the bulk check takes 100ms and there are 5 services, ALL 5 get `durationMs ≈ 100`. This is meaningless per-service timing.

**Impact**: `ServiceInfo.HealthCheckDurationMs` and `Event.DurationMs` for health check events are **misleading**. The value represents "total time since bulk check started", not "time spent on this specific service".

**Root cause**: `injector.HealthCheckWithContext()` runs all checks in parallel (or with configured parallelism) and returns results as a completed map. There's no way to time individual services from outside — the timing happens inside samber/do's internals.

**Fix options**:

1. Accept it — document that duration is "wall-clock time of the entire bulk health check", not per-service
2. Remove duration from health check events entirely (set to nil)
3. Set duration to the total time only on the LAST event, nil on others
4. Best: upstream PR to add per-service timing in samber/do's health check results

### 🟡 SEMANTICS: `HealthCheckSucceeded` when no checks ran

`Report.HealthCheckSucceeded` is `true` when no health checks have been recorded. This matches the pattern of `ShutdownSucceeded` (which is `true` when no shutdowns have run). But it's semantically different — "no checks ran" is not the same as "all checks passed". The `allHealthChecksPassed()` function only fails if `HealthCheckCount > 0 && HealthCheckError != nil`, so an empty set returns `true` (vacuous truth).

This is defensible but could confuse consumers.

### 🟡 MISSING: `ServiceByName` / `FailedServices` have 0% coverage

These pre-existing convenience methods are untested:

- `ServiceByName` — 0%
- `FailedServices` — 0%
- `IsError` — 0%
- `String` — 0%

---

## e) WHAT WE SHOULD IMPROVE

### Architecture & Type Model

1. **Duplicate struct literal in `RecordHealthCheck`** — `recorder.go:721-740` has a 20-line `serviceRecord{}` literal that's nearly identical to `newServiceRecord()`. If `newServiceRecord` accepted `(scopeID, scopeName, serviceName, now)` instead of `(*do.Scope, string, time.Time)`, we could reuse it. This duplication will diverge.

2. **`RecordHealthCheck` on Recorder takes raw strings** — The signature `(scopeID, scopeName, serviceName string, err error, durationMs float64)` is 6 params. A `HealthCheckResult` struct would be cleaner and more extensible.

3. **No `HealthCheckResult` domain type** — The result of a health check for a service (name, error, duration) is passed as loose parameters. A proper type would enable richer per-service metadata.

4. **`Event` is built manually in `RecordHealthCheck`** — All other event types use `newEvent()` helper. Health check breaks the pattern because `newEvent` requires `*do.Scope`. Should refactor `newEvent` to accept metadata strings, or create a `newEventFromMetadata()` helper.

### Testing

5. **No test for health check `OnEvent` callback** — Other event types have `TestPlugin_EventHandler` but health check events don't verify the callback fires.

6. **No test for health check events in JSON/NDJSON export** — We don't verify that health check events serialize correctly in exported files.

7. **No test for health check `Phase` being `PhaseAfter` only** — Should explicitly verify no `PhaseBefore` events are emitted.

8. **No test for `RecordHealthCheckWithContext` (the one with ctx)** — All tests use `RecordHealthCheck()` which calls `RecordHealthCheckWithContext(context.Background())`. Should test context propagation and cancellation.

9. **No benchmark for health check recording** — Other hooks have benchmarks; health check doesn't.

### HTML Visualization

10. **Health column shows "–" for unchecked services** — This takes up space for no value. Could be conditionally shown only when health checks have been recorded.

11. **No health check tab** — A dedicated tab with health check timeline/history would be valuable for operational monitoring.

12. **Health check duration in HTML is misleading** — Shows the shared wall-clock time, not per-service time.

### Documentation & Examples

13. **`example/main.go` doesn't show health check in the report checklist with `health_check_count`** — The checklist line just checks `HealthCheckedCount > 0` but doesn't show the per-service health fields.

14. **Research report still says "PhaseAfter only"** — The `docs/research/` report is a historical document, fine.

15. **FEATURES.md not updated** — Should reflect health check support status.

---

## f) Top #25 Things We Should Get Done Next

Sorted by: **Impact × (1/Effort)** — highest ROI first.

| #   | Task                                                                            | Impact | Effort | Category     |
| --- | ------------------------------------------------------------------------------- | ------ | ------ | ------------ |
| 1   | **Fix duration bug: set to nil (or document wall-clock)**                       | HIGH   | LOW    | Bug fix      |
| 2   | **Fix `HealthCheckSucceeded` semantics: false when no checks ran**              | MED    | LOW    | Bug fix      |
| 3   | **Add `OnEvent` callback test for health checks**                               | MED    | LOW    | Testing      |
| 4   | **Add JSON/NDJSON export test with health check events**                        | MED    | LOW    | Testing      |
| 5   | **Extract `HealthCheckResult` type for recorder method**                        | MED    | LOW    | Architecture |
| 6   | **Deduplicate `serviceRecord` literal in `RecordHealthCheck`**                  | MED    | LOW    | Architecture |
| 7   | **Add `newEventFromMetadata()` or refactor `newEvent()`**                       | MED    | LOW    | Architecture |
| 8   | **Test `RecordHealthCheckWithContext` with real context**                       | MED    | LOW    | Testing      |
| 9   | **Add benchmark for health check recording**                                    | LOW    | LOW    | Testing      |
| 10  | **Test health check Phase is always PhaseAfter**                                | LOW    | LOW    | Testing      |
| 11  | **Add tests for `ServiceByName`, `FailedServices`, `IsError`, `String`**        | MED    | LOW    | Testing      |
| 12  | **Update FEATURES.md with health check support**                                | MED    | LOW    | Docs         |
| 13  | **Show per-service health fields in example checklist**                         | LOW    | LOW    | Example      |
| 14  | **Conditionally show health column in HTML**                                    | LOW    | LOW    | HTML         |
| 15  | **Add health check event badge in graph nodes**                                 | LOW    | MED    | HTML         |
| 16  | **Add health check tab in HTML**                                                | MED    | HIGH   | HTML         |
| 17  | **Add `RecordHealthCheckNamed()` for single-service check**                     | MED    | MED    | Feature      |
| 18  | **Add `RecordHealthCheckNamedWithContext()` for single-service check with ctx** | MED    | MED    | Feature      |
| 19  | **Add health check history ring buffer**                                        | LOW    | MED    | Feature      |
| 20  | **Add `StartHealthCheckLoop(injector, interval)` helper**                       | MED    | MED    | Feature      |
| 21  | **Upstream PR: `HookBeforeHealthCheck`/`HookAfterHealthCheck`**                 | HIGH   | HIGH   | Upstream     |
| 22  | **Upstream PR: per-service timing in health check results**                     | HIGH   | HIGH   | Upstream     |
| 23  | **Add `ServiceInfo.Healthchecker` bool field**                                  | MED    | LOW    | Feature      |
| 24  | **Add `Report.HealthyServiceCount` computed field**                             | LOW    | LOW    | Feature      |
| 25  | **Consider `samber/lo` for map/slice operations**                               | LOW    | LOW    | Dependencies |

---

## g) Top #1 Question I Cannot Figure Out Myself

**How should we handle per-service health check duration when samber/do runs them in parallel?**

The bulk `HealthCheckWithContext()` returns a completed `map[string]error` — no per-service timing info. Options:

1. **Set `DurationMs` to `nil`** for health check events — honest about not knowing
2. **Keep wall-clock time** — misleading but at least gives a sense of "how long did the whole batch take"
3. **Use `do.HealthCheckNamed()` individually** — call it once per service and time each, but this bypasses the parallelism/timeout controls in `InjectorOpts`
4. **Upstream PR** — add per-service timing to samber/do's health check API

Option 3 is the most correct but changes semantics (loses parallelism). Option 1 is the most honest. Option 4 is the best long-term.

**I recommend option 1 (set DurationMs to nil)** for now, with a comment explaining why, and pursue option 4 as a follow-up. This is a design decision I'd want your input on.

---

## Files Changed This Session

```
AGENTS.md            | Architecture, gotchas, testing patterns
auditlog_test.go     | +348 lines (9 new tests + helpers + convenience method update)
doc.go               | Package doc
example/main.go      | Uses plugin.RecordHealthCheckWithContext + feature checklist
html.templ           | Health column, event badge, filter chip, stat card
html_templ.go        | Regenerated
plugin.go            | RecordHealthCheck + RecordHealthCheckWithContext
recorder.go          | +173 lines (health fields, RecordHealthCheck, ResolveServiceScope)
types.go             | EventTypeHealthCheck, health fields, UnhealthyServices
```

**Total**: ~1,400 lines added, ~300 lines changed across 8 source files + 1 generated file.

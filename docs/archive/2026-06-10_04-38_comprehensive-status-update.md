# Status Report: Full Comprehensive Update

**Date**: 2026-06-10 04:38 · **Branch**: master · **Status**: ALPHA · **Health**: Good (with known limitations)

---

## a) FULLY DONE

### Core DI Observability (Registration + Invocation + Shutdown)

| Feature                                                                                                            | Status  | Verified        |
| ------------------------------------------------------------------------------------------------------------------ | ------- | --------------- |
| Drop-in plugin setup (`New` + `Opts`)                                                                              | ✅ Done | ✓ `plugin.go`   |
| Service registration tracking (before/after)                                                                       | ✅ Done | ✓ `recorder.go` |
| Service invocation tracking (before/after, duration, errors)                                                       | ✅ Done | ✓ `recorder.go` |
| Shutdown tracking (before/after, duration, errors)                                                                 | ✅ Done | ✓ `recorder.go` |
| Stack-based dependency graph inference                                                                             | ✅ Done | ✓ `recorder.go` |
| Reverse dependencies (Dependents)                                                                                  | ✅ Done | ✓ `recorder.go` |
| Scope tree + scope tracking                                                                                        | ✅ Done | ✓ `recorder.go` |
| Monotonic sequence numbers (per-recorder atomic)                                                                   | ✅ Done | ✓ `recorder.go` |
| Build duration measurement (first build per service)                                                               | ✅ Done | ✓ `recorder.go` |
| Invocation ordering                                                                                                | ✅ Done | ✓ `recorder.go` |
| Provider error capture                                                                                             | ✅ Done | ✓ `recorder.go` |
| Service lifecycle status (`ServiceStatus`)                                                                         | ✅ Done | ✓ `types.go`    |
| `ProviderType` (lazy/eager/transient/alias) with `Icon()`/`String()`                                               | ✅ Done | ✓ `types.go`    |
| `ServiceRef` embedded in Event/ServiceInfo                                                                         | ✅ Done | ✓ `types.go`    |
| Event convenience methods (`IsRegistration`, `IsInvocation`, `IsShutdown`, `IsHealthCheck`, `IsBefore`, `IsAfter`) | ✅ Done | ✓ `types.go`    |
| `ServiceRef.String()` (human-readable scope/name format)                                                           | ✅ Done | ✓ `types.go`    |
| `ServiceStatus.IsError()`                                                                                          | ✅ Done | ✓ `types.go`    |
| Report convenience methods (`ServiceByName`, `EventsByType`, `FailedServices`, `UnhealthyServices`)                | ✅ Done | ✓ `types.go`    |
| `Config.OnEvent` callback for real-time streaming                                                                  | ✅ Done | ✓ `plugin.go`   |

### Health Check Auditing

| Feature                                                                                | Status  | Verified        |
| -------------------------------------------------------------------------------------- | ------- | --------------- |
| `EventTypeHealthCheck` with `IsHealthCheck()`                                          | ✅ Done | ✓ `types.go`    |
| `Plugin.RecordHealthCheck()` / `RecordHealthCheckWithContext()`                        | ✅ Done | ✓ `plugin.go`   |
| `Recorder.RecordHealthCheck()` emits PhaseAfter events                                 | ✅ Done | ✓ `recorder.go` |
| `Recorder.ResolveServiceScope()` for RootScope + child Scope                           | ✅ Done | ✓ `recorder.go` |
| ServiceInfo health fields: `LastHealthCheckAt`, `HealthCheckError`, `HealthCheckCount` | ✅ Done | ✓ `types.go`    |
| Report health fields: `HealthCheckSucceeded`, `HealthCheckedCount`                     | ✅ Done | ✓ `types.go`    |
| Disabled-plugin passthrough (delegates directly to injector)                           | ✅ Done | ✓ `plugin.go`   |
| Schema version bump (0.1.0 → 0.2.0)                                                    | ✅ Done | ✓ `types.go`    |

### Bug Fixes (This + Previous Session)

| Fix                                                         | Status   | Details                                                                      |
| ----------------------------------------------------------- | -------- | ---------------------------------------------------------------------------- |
| Removed misleading `HealthCheckDurationMs`                  | ✅ Fixed | Per-service timing unavailable from bulk API. Events have `DurationMs: nil`. |
| Restored accidentally deleted `lastHealthCheckAt` field     | ✅ Fixed | Previous session removed it along with `healthCheckDurationMs`               |
| `HealthCheckSucceeded` = `false` when no checks ran         | ✅ Fixed | `allHealthChecksPassed()` now requires `checked > 0`                         |
| Deduplicated `serviceRecord` literal in `RecordHealthCheck` | ✅ Fixed | Extracted `newServiceRecordFromMeta()`                                       |

### Refactoring

| Refactor                                                     | Status  |
| ------------------------------------------------------------ | ------- |
| `newEventFromRef()` — shared event builder from `ServiceRef` | ✅ Done |
| `newServiceRecordFromMeta()` — factory from metadata strings | ✅ Done |

### Export Formats

| Format                                                             | Status  |
| ------------------------------------------------------------------ | ------- |
| JSON report (`WriteReportJSON` / `ExportToFile`)                   | ✅ Done |
| NDJSON event stream (`WriteEventsNDJSON` / `ExportEventsToNDJSON`) | ✅ Done |
| Self-contained HTML visualization (dark theme, 5 tabs)             | ✅ Done |

### HTML Visualization

| Feature                                                                                                | Status  |
| ------------------------------------------------------------------------------------------------------ | ------- |
| Services table with type badges, status badges, shutdown duration, reverse deps, health column, search | ✅ Done |
| Collapsible scope tree with type emoji chips                                                           | ✅ Done |
| Sugiyama layered DAG dependency graph with pan/zoom/click                                              | ✅ Done |
| Dual build+shutdown timeline bars                                                                      | ✅ Done |
| Event type filter chips (registration/invocation/shutdown/health_check)                                | ✅ Done |
| Keyboard nav (1-5), responsive layout, stat cards                                                      | ✅ Done |
| Health column (✓/✗ badges with error tooltip)                                                          | ✅ Done |
| Health check stat card (when checks ran)                                                               | ✅ Done |

### Testing

| Metric        | Value                               |
| ------------- | ----------------------------------- |
| Total tests   | 53                                  |
| Coverage      | 91.1%                               |
| Race detector | Clean                               |
| Framework     | Standard `testing.T` + table-driven |

### Verification

| Check                         | Result                       |
| ----------------------------- | ---------------------------- |
| `go test ./... -race`         | ✅ 53 tests pass, no races   |
| `go vet ./...`                | ✅ Clean                     |
| `go build ./...`              | ✅ Clean                     |
| `go generate ./...`           | ✅ html_templ.go regenerated |
| `golangci-lint run` (library) | ✅ Zero issues               |

---

## b) PARTIALLY DONE

| Item                                                | What's Done                                                 | What's Missing                                                                                                                                                                                                                                        |
| --------------------------------------------------- | ----------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `IsHealthchecker` / `IsShutdowner` on `ServiceInfo` | Fields exist in types, populated to `false` in constructors | **Always `false`** — `do.ExplainServiceOutput` doesn't expose capability flags. Previous session started WIP using `do.ExplainInjector()` API (which DOES have these fields) but it was never committed due to race condition. See stash `stash@{0}`. |
| `Event.ServiceType`                                 | Field added in WIP (uncommitted)                            | Not in committed code. Would require `inferServiceType` at hook time which races with concurrent builds. WIP tried this but introduced a race.                                                                                                        |

---

## c) NOT STARTED

| Item                                                        | Priority | Notes                                                                                           |
| ----------------------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------- |
| Per-service health check duration                           | MED      | Requires upstream PR to samber/do to add per-service timing to `HealthCheckWithContext` results |
| Single-service `RecordHealthCheckNamed()`                   | MED      | Call `do.HealthCheckNamed()` per service and time individually. Loses parallelism.              |
| Health check history ring buffer                            | LOW      | Events stream captures full history; ServiceInfo only stores last result                        |
| Periodic health check helper                                | LOW      | `plugin.StartHealthCheckLoop(injector, interval)`                                               |
| Dedicated HTML health check tab                             | LOW      | Separate tab with health check timeline/history                                                 |
| Upstream PR: `HookBeforeHealthCheck`/`HookAfterHealthCheck` | LOW      | Would eliminate need for wrapper method                                                         |
| `ReportOption` functional options for filtering             | MED      | Filter report by service, time range, event type                                                |
| `Config.Validate() error`                                   | LOW      | Centralize config validation                                                                    |
| Versioned report schema with migration                      | LOW      | `SchemaVersion` exists, no migration function                                                   |
| Additional export formats (Mermaid, PlantUML)               | LOW      | Only if users request                                                                           |
| Benchmarks for health check recording                       | LOW      | Other hooks have benchmarks                                                                     |

---

## d) TOTALLY FUCKED UP / CRITICAL ISSUES

### 🔴 WIP STASH: Race condition in `inferServiceType` during concurrent invocations

**Stash**: `stash@{0}` — "WIP: enrichCapabilities, Event.ServiceType, OffsetNs refactor"

The previous session attempted a major refactor to:

1. Add `ServiceType` field to `Event` (populated from `inferServiceType` called inside hooks)
2. Populate `IsHealthchecker`/`IsShutdowner` via `do.ExplainInjector()` (called at report time, outside mutex)
3. Replace `Event.Timestamp` with `Event.OffsetNs` (relative offset from registration)

**Critical bug**: `inferServiceType` calls `do.ExplainNamedService()` which accesses `serviceLazy.getBuildTime()` — this races with concurrent `serviceLazy.build()` calls in samber/do. The race is detected by `TestPlugin_ConcurrentInvocations`.

**Root cause**: `newEvent()` (called from every hook) called `inferServiceType(scope, serviceName)` which is NOT safe during concurrent invocations. The fix requires either:

- Looking up the already-recorded type from `serviceRecord` (requires `lookupServiceType` under RLock)
- Or only populating `ServiceType` on events at report-build time

### 🟡 `IsHealthchecker`/`IsShutdowner` always false

Fields exist but are hardcoded to `false`. The stash has a working `enrichCapabilities()` using `do.ExplainInjector()` which correctly populates them — but it's tangled with the broken `Event.ServiceType` refactor.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Extract `enrichCapabilities` from the stash and land it separately** — The `do.ExplainInjector()` approach works and is safe (called outside mutex at report time). The stash has working code for this. The only issue was the separate `Event.ServiceType` change that caused the race.

2. **`Event.ServiceType` should be populated at report time, not hook time** — Instead of calling `inferServiceType` in every hook (racy), enrich events with `ServiceType` during `BuildReport()` by joining against the `serviceRecord` map. This is safe because it happens under a single read lock.

3. **`Event.OffsetNs` refactor in stash is unnecessary** — Relative offsets from registration time add complexity without clear benefit. Absolute timestamps (`time.Time`) are simpler and more useful.

### Testing

4. **No benchmark for health check recording** — Other hooks have benchmarks; health check doesn't.

5. **No test for `RecordHealthCheckWithContext` with context cancellation** — All tests use `RecordHealthCheck()`.

6. **No HTML test verifying health column rendering** — HTML test doesn't check health-specific elements.

### Documentation

7. **AGENTS.md mentions `IsHealthchecker`/`IsShutdowner` always-false** — Should be updated once we land `enrichCapabilities`.

---

## f) Top #25 Things We Should Get Done Next

Sorted by: Impact × (1/Effort).

| #   | Task                                                                                              | Impact | Effort | Category     |
| --- | ------------------------------------------------------------------------------------------------- | ------ | ------ | ------------ |
| 1   | **Land `enrichCapabilities` from stash** (extract from broken WIP, keep ExplainInjector approach) | HIGH   | LOW    | Bug fix      |
| 2   | **Remove always-false `IsHealthchecker`/`IsShutdowner` or populate via enrichCapabilities**       | HIGH   | LOW    | Bug fix      |
| 3   | **Add `Event.ServiceType` at report-build time** (safe, no race)                                  | MED    | LOW    | Feature      |
| 4   | **Update AGENTS.md with enrichment approach**                                                     | MED    | LOW    | Docs         |
| 5   | **Add benchmark for health check recording**                                                      | LOW    | LOW    | Testing      |
| 6   | **Test `RecordHealthCheckWithContext` with cancelled context**                                    | MED    | LOW    | Testing      |
| 7   | **`ReportOption` functional options for filtering**                                               | MED    | MED    | Feature      |
| 8   | **`Config.Validate() error` method**                                                              | LOW    | LOW    | Polish       |
| 9   | **Single-service `RecordHealthCheckNamed()`**                                                     | MED    | MED    | Feature      |
| 10  | **Dedicated HTML health check tab**                                                               | MED    | HIGH   | HTML         |
| 11  | **Conditionally show health column in HTML**                                                      | LOW    | LOW    | HTML         |
| 12  | **Add health check event badge in graph nodes**                                                   | LOW    | MED    | HTML         |
| 13  | **HTML test verifying health column rendering**                                                   | LOW    | LOW    | Testing      |
| 14  | **Versioned report schema with migration**                                                        | MED    | MED    | Feature      |
| 15  | **Upstream PR: per-service timing in health check results**                                       | HIGH   | HIGH   | Upstream     |
| 16  | **Upstream PR: `HookBeforeHealthCheck`/`HookAfterHealthCheck`**                                   | HIGH   | HIGH   | Upstream     |
| 17  | **Health check history ring buffer**                                                              | LOW    | MED    | Feature      |
| 18  | **Periodic health check helper**                                                                  | MED    | MED    | Feature      |
| 19  | **Add `Report.HealthyServiceCount` computed field**                                               | LOW    | LOW    | Feature      |
| 20  | **Additional export formats (Mermaid, PlantUML)**                                                 | LOW    | HIGH   | Feature      |
| 21  | **Show per-service health fields in example checklist**                                           | LOW    | LOW    | Example      |
| 22  | **Add `RecordHealthCheckNamedWithContext()`**                                                     | MED    | MED    | Feature      |
| 23  | **Add `ServiceInfo.Healthy()` bool method**                                                       | LOW    | LOW    | Feature      |
| 24  | **Consider `samber/lo` for map/slice operations**                                                 | LOW    | LOW    | Dependencies |
| 25  | **Drop the stash after extracting the good parts**                                                | LOW    | LOW    | Cleanup      |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `Event.ServiceType` be populated at event creation time or at report-build time?**

Arguments for **event creation time** (in hooks):

- Events are immutable once created — the type is known and fixed
- More correct: the event captures what happened at that moment

Arguments for **report-build time** (in `BuildReport`):

- Safe from race conditions — no concurrent access to `do.ExplainNamedService`
- Simpler: just join `Event.ServiceRef` → `serviceRecord.serviceType`
- The type rarely changes (it's determined at registration)

The stash tried event-creation-time and introduced a race. Report-build-time is safe and simple. But it means raw events streamed via `OnEvent` callback won't have `ServiceType` — only the assembled `Report` will.

I lean toward **report-build-time** because the `OnEvent` callback is a streaming use case where extra per-event type info is less critical than correctness. But this is a design decision worth your input.

---

## Stash Contents

```
stash@{0}: WIP: enrichCapabilities, Event.ServiceType, OffsetNs refactor - HAS RACE CONDITION
```

This stash contains:

- `scopeMeta.ref *do.Scope` — stores scope reference for later capability lookup
- `enrichCapabilities()` — uses `do.ExplainInjector()` to populate IsHealthchecker/IsShutdowner (WORKS, safe outside mutex)
- `buildCapabilityMap()` — helper for recursive scope DAG traversal
- `Event.ServiceType` field — populates at hook time (RACY — do not use as-is)
- `Event.OffsetNs` replacing `Timestamp` — unnecessary refactor
- `TestPlugin_CapabilityTracking`, `TestProviderType_String`, `TestProviderType_Icon` — new tests
- `BuildReport()` releases RLock before `enrichCapabilities` — good pattern

**Recommendation**: Extract only `enrichCapabilities`, `buildCapabilityMap`, `scopeMeta.ref`, and the test. Drop the `Event.ServiceType`/`OffsetNs` changes.

---

## Stats

| Metric                          | Value                  |
| ------------------------------- | ---------------------- |
| Total tests                     | 53                     |
| Coverage                        | 91.1%                  |
| Source files                    | 8 (_.go + _.templ)     |
| Total LOC                       | 4,208                  |
| Race detector                   | Clean (committed code) |
| Stash items                     | 1 (WIP with race)      |
| Commits on health check feature | 9                      |

# Health Check Implementation Plan

**Date**: 2026-06-10 · **Estimated Total**: ~95 min · **Schema**: 0.1.0 → 0.2.0

Sorted by: **Tier** (backend enables frontend enables docs) → **Impact** (what breaks if skipped) → **Effort** (ascending).

---

## Tier 1: Data Model (blocks everything else)

| #   | Task                                                                                                                       | File       | Impact | Effort | Est  |
| --- | -------------------------------------------------------------------------------------------------------------------------- | ---------- | ------ | ------ | ---- |
| 1   | Add `EventTypeHealthCheck` to EventType const block, add `IsHealthCheck()` on Event                                        | `types.go` | HIGH   | LOW    | 3min |
| 2   | Bump `SchemaVersion` from `"0.1.0"` to `"0.2.0"`                                                                           | `types.go` | HIGH   | LOW    | 1min |
| 3   | Add 4 health fields to `ServiceInfo`: `LastHealthCheckAt`, `HealthCheckDurationMs`, `HealthCheckError`, `HealthCheckCount` | `types.go` | HIGH   | LOW    | 3min |
| 4   | Add 2 health fields to `Report`: `HealthCheckSucceeded`, `HealthCheckedServiceCount`                                       | `types.go` | MED    | LOW    | 2min |
| 5   | Add `UnhealthyServices()` convenience method on Report                                                                     | `types.go` | LOW    | LOW    | 2min |

---

## Tier 2: Core Logic (depends on Tier 1)

| #   | Task                                                                                                                                | File          | Impact | Effort | Est  |
| --- | ----------------------------------------------------------------------------------------------------------------------------------- | ------------- | ------ | ------ | ---- |
| 6   | Add 4 health fields to `serviceRecord` struct: `lastHealthCheckAt`, `healthCheckDurationMs`, `healthCheckError`, `healthCheckCount` | `recorder.go` | HIGH   | LOW    | 2min |
| 7   | Update `newServiceRecord()` to initialize new health fields (for exhaustruct)                                                       | `recorder.go` | HIGH   | LOW    | 2min |
| 8   | Add `RecordHealthCheck(scope, serviceName, err, durationMs)` method on Recorder — emits Event + updates serviceRecord               | `recorder.go` | HIGH   | MED    | 8min |
| 9   | Update `buildServicesLocked()` to populate the 4 new ServiceInfo health fields from serviceRecord                                   | `recorder.go` | HIGH   | MED    | 5min |
| 10  | Update `BuildReport()` to compute `HealthCheckSucceeded` and `HealthCheckedServiceCount`                                            | `recorder.go` | MED    | MED    | 5min |
| 11  | Add `resolveServiceScope(injector, serviceName)` helper — walks injector scopes to find matching serviceRecord                      | `recorder.go` | MED    | MED    | 8min |

---

## Tier 3: Public API (depends on Tier 2)

| #   | Task                                                                                                                                  | File        | Impact | Effort | Est   |
| --- | ------------------------------------------------------------------------------------------------------------------------------------- | ----------- | ------ | ------ | ----- |
| 12  | Add `RecordHealthCheckWithContext(ctx, injector) map[string]error` on Plugin — wrapper that times each service, delegates to recorder | `plugin.go` | HIGH   | MED    | 10min |
| 13  | Add `RecordHealthCheck(injector) map[string]error` on Plugin — calls RecordHealthCheckWithContext with Background ctx                 | `plugin.go` | HIGH   | LOW    | 3min  |
| 14  | Handle disabled plugin: both methods delegate to injector directly without recording when `Enabled: false`                            | `plugin.go` | MED    | LOW    | 2min  |

---

## Tier 4: Tests (depends on Tier 3)

| #   | Task                                                                                                       | File               | Impact | Effort | Est  |
| --- | ---------------------------------------------------------------------------------------------------------- | ------------------ | ------ | ------ | ---- |
| 15  | Add `HealthyDB` test helper that implements `do.Healthchecker` (returns nil)                               | `auditlog_test.go` | MED    | LOW    | 3min |
| 16  | Add `UnhealthyCache` test helper that implements `do.HealthcheckerWithContext` (returns error)             | `auditlog_test.go` | MED    | LOW    | 3min |
| 17  | Test: `TestPlugin_HealthCheckHealthy` — healthy service gets event + ServiceInfo fields                    | `auditlog_test.go` | HIGH   | MED    | 5min |
| 18  | Test: `TestPlugin_HealthCheckUnhealthy` — unhealthy service gets event with error + ServiceInfo fields     | `auditlog_test.go` | HIGH   | MED    | 5min |
| 19  | Test: `TestPlugin_HealthCheckMultipleServices` — 3 services, mix of healthy/unhealthy, verify all recorded | `auditlog_test.go` | HIGH   | MED    | 5min |
| 20  | Test: `TestPlugin_HealthCheckDisabled` — disabled plugin delegates to injector, no events                  | `auditlog_test.go` | MED    | LOW    | 3min |
| 21  | Test: `TestPlugin_HealthCheckCount` — call RecordHealthCheck twice, verify HealthCheckCount=2              | `auditlog_test.go` | MED    | LOW    | 3min |
| 22  | Test: `TestPlugin_HealthCheckReport` — verify Report.HealthCheckSucceeded and HealthCheckedServiceCount    | `auditlog_test.go` | MED    | LOW    | 3min |
| 23  | Test: `TestPlugin_HealthCheckWithScope` — child scope service health check resolves correctly              | `auditlog_test.go` | MED    | MED    | 5min |
| 24  | Test: `TestEvent_IsHealthCheck` — add health_check case to existing convenience method test                | `auditlog_test.go` | LOW    | LOW    | 2min |

---

## Tier 5: HTML Visualization (depends on Tier 4 verified)

| #   | Task                                                                                 | File         | Impact | Effort | Est  |
| --- | ------------------------------------------------------------------------------------ | ------------ | ------ | ------ | ---- |
| 25  | Add CSS for `health_check` event badge (color: green for healthy, red for unhealthy) | `html.templ` | MED    | LOW    | 3min |
| 26  | Add "Health" column to services table — shows ✓/✗ badge with tooltip for error       | `html.templ` | HIGH   | MED    | 5min |
| 27  | Add `health_check` to event type filter chips                                        | `html.templ` | MED    | LOW    | 2min |
| 28  | Add health check stat card to stats bar ("Health Checks: N checked, M unhealthy")    | `html.templ` | MED    | LOW    | 3min |
| 29  | Regenerate `html_templ.go` via `go generate ./...`                                   | generated    | HIGH   | LOW    | 1min |

---

## Tier 6: Documentation & Polish (depends on all above)

| #   | Task                                                                                                                       | File              | Impact | Effort | Est  |
| --- | -------------------------------------------------------------------------------------------------------------------------- | ----------------- | ------ | ------ | ---- |
| 30  | Update `doc.go` package comment to mention health check tracking                                                           | `doc.go`          | MED    | LOW    | 2min |
| 31  | Update `example/main.go` — replace direct `injector.HealthCheckWithContext()` with `plugin.RecordHealthCheckWithContext()` | `example/main.go` | MED    | MED    | 5min |
| 32  | Update `AGENTS.md` — document new API methods, health check event type, gotchas                                            | `AGENTS.md`       | HIGH   | MED    | 5min |
| 33  | Run full test suite + `go vet` to verify everything passes                                                                 | CLI               | HIGH   | LOW    | 2min |

---

## Summary

| Tier              | Tasks  | Est         |
| ----------------- | ------ | ----------- |
| 1 — Data Model    | #1–5   | 11min       |
| 2 — Core Logic    | #6–11  | 30min       |
| 3 — Public API    | #12–14 | 15min       |
| 4 — Tests         | #15–24 | 37min       |
| 5 — HTML          | #25–29 | 14min       |
| 6 — Docs & Polish | #30–33 | 14min       |
| **Total**         | **33** | **~121min** |

---

## Execution Rules

1. Execute **strictly in order** within each tier (each task depends on the previous)
2. Run `go test ./...` after **every tier** completes — fix before moving on
3. Run `go vet ./...` after Tier 3
4. Run `golangci-lint run` after Tier 6
5. If any test fails, fix immediately — do not continue to next task
6. Keep each task under 12 minutes — if it's taking longer, stop and reassess

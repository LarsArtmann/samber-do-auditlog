# Comprehensive UI/UX Improvement Plan

**Date**: 2026-06-10 · **Project**: samber-do-auditlog · **Scope**: HTML visualization + Report data model

---

## Analysis: Current State

### Data we have but DON'T show in HTML

| Field                       | In ServiceInfo | In HTML?              | Why missing                                       |
| --------------------------- | -------------- | --------------------- | ------------------------------------------------- |
| `RegisteredAt`              | ✅             | ❌                    | Not rendered                                      |
| `FirstInvokedAt`            | ✅             | ❌                    | Not rendered                                      |
| `ShutdownAt`                | ✅             | ❌                    | Not rendered                                      |
| `ShutdownDurationMs`        | ✅             | ❌                    | Not in services table                             |
| `ShutdownError`             | ✅             | ❌                    | Not in services table                             |
| `InvocationError`           | ✅             | Partial (status only) | Status says "error" but error message not visible |
| `Dependents` (reverse deps) | ✅             | ❌                    | Not shown anywhere                                |
| `Version`                   | In Report      | ❌                    | Not in header                                     |
| `ScopeTree`                 | In Report      | ❌                    | Not rendered as a visual                          |
| `ServiceCount` in scope     | In ScopeNode   | ❌                    | Only flat service list                            |

### Data samber/do v2 provides that we DON'T capture

| Data                                      | Available                         | Our gap                                                     |
| ----------------------------------------- | --------------------------------- | ----------------------------------------------------------- |
| Shutdown report (`*do.ShutdownReport`)    | Returned by `injector.Shutdown()` | We re-derive from hooks; miss total duration + success flag |
| Health check results                      | `injector.HealthCheck()`          | Not captured                                                |
| Service list (registered but not invoked) | `scope.ListProvidedServices()`    | We only show invoked services                               |
| Registration kind (lazy/transient/value)  | Not in hook API                   | Can't distinguish                                           |

### UI/UX Problems Found

| #   | Problem                                                                       | Severity |
| --- | ----------------------------------------------------------------------------- | -------- |
| 1   | No search/filter on services table                                            | HIGH     |
| 2   | No way to see error details (error message hidden)                            | HIGH     |
| 3   | No scope tree visualization                                                   | HIGH     |
| 4   | No total build time stat                                                      | MEDIUM   |
| 5   | No shutdown duration in services table                                        | MEDIUM   |
| 6   | No reverse dependencies shown                                                 | MEDIUM   |
| 7   | Events table event types are abbreviated inconsistently ("reg", "inv", "sht") | MEDIUM   |
| 8   | No responsive design for mobile                                               | MEDIUM   |
| 9   | Graph has no tooltips on hover                                                | MEDIUM   |
| 10  | Graph arrows don't account for node width (overlap into node)                 | MEDIUM   |
| 11  | No dark/light theme toggle                                                    | LOW      |
| 12  | No export button within HTML (can't re-download)                              | LOW      |
| 13  | Version not shown in header                                                   | LOW      |
| 14  | Timeline doesn't show shutdown durations                                      | LOW      |
| 15  | No "copy as JSON" button for report data                                      | LOW      |

### Report Data Model Improvements

| #   | Improvement                             | Impact                           |
| --- | --------------------------------------- | -------------------------------- |
| 1   | Add `TotalBuildDurationMs` to Report    | Sum of all first build durations |
| 2   | Add `TotalShutdownDurationMs` to Report | Wall-clock total                 |
| 3   | Add `ShutdownSucceeded bool` to Report  | Any shutdown errors?             |
| 4   | Add `ScopeCount int` to Report          | Pre-computed for stats cards     |

---

## Task List (max 12min each)

Sorted by: impact × effort (highest ROI first). Customer value = what the end-user sees.

### Tier 1: Data Model (backend, enables everything else)

| #    | Task                                                                                               | Impact | Effort | File               |
| ---- | -------------------------------------------------------------------------------------------------- | ------ | ------ | ------------------ |
| T1-1 | Add `TotalBuildDurationMs`, `TotalShutdownDurationMs`, `ShutdownSucceeded`, `ScopeCount` to Report | HIGH   | 10min  | `types.go`         |
| T1-2 | Compute new Report fields in `BuildReport()`                                                       | HIGH   | 8min   | `recorder.go`      |
| T1-3 | Add test for new Report fields                                                                     | HIGH   | 10min  | `auditlog_test.go` |

### Tier 2: Services Table Improvements

| #    | Task                                                 | Impact | Effort | File         |
| ---- | ---------------------------------------------------- | ------ | ------ | ------------ |
| T2-1 | Show shutdown duration column in services table      | HIGH   | 8min   | `html.templ` |
| T2-2 | Show error message on hover/click for error statuses | HIGH   | 10min  | `html.templ` |
| T2-3 | Show dependents (reverse deps) in services table     | MEDIUM | 8min   | `html.templ` |
| T2-4 | Add search/filter input above services table         | HIGH   | 10min  | `html.templ` |
| T2-5 | Show `RegisteredAt` and `FirstInvokedAt` timestamps  | MEDIUM | 8min   | `html.templ` |

### Tier 3: Header + Stats Improvements

| #    | Task                                                   | Impact | Effort | File         |
| ---- | ------------------------------------------------------ | ------ | ------ | ------------ |
| T3-1 | Show schema version in header                          | LOW    | 3min   | `html.templ` |
| T3-2 | Add total build duration stat card                     | MEDIUM | 5min   | `html.templ` |
| T3-3 | Add "Errors" stat card (count of services with errors) | MEDIUM | 5min   | `html.templ` |

### Tier 4: Events Table Improvements

| #    | Task                                               | Impact | Effort | File         |
| ---- | -------------------------------------------------- | ------ | ------ | ------------ |
| T4-1 | Use full event type names instead of abbreviations | MEDIUM | 5min   | `html.templ` |
| T4-2 | Color-code event type badges                       | MEDIUM | 8min   | `html.templ` |
| T4-3 | Add event type filter buttons above events table   | MEDIUM | 10min  | `html.templ` |

### Tier 5: Graph Improvements

| #    | Task                                                                          | Impact | Effort | File         |
| ---- | ----------------------------------------------------------------------------- | ------ | ------ | ------------ |
| T5-1 | Fix arrow endpoints to stop at node edges (not overlap)                       | MEDIUM | 10min  | `html.templ` |
| T5-2 | Color nodes by status (active=blue, error=red, shutdown=gray, registered=dim) | HIGH   | 8min   | `html.templ` |
| T5-3 | Add tooltip on node hover showing service details                             | MEDIUM | 10min  | `html.templ` |

### Tier 6: Scope Tree Visualization (new tab)

| #    | Task                                                   | Impact | Effort | File         |
| ---- | ------------------------------------------------------ | ------ | ------ | ------------ |
| T6-1 | Add "Scopes" tab to tab bar                            | HIGH   | 3min   | `html.templ` |
| T6-2 | Render scope tree as indented tree with service counts | HIGH   | 12min  | `html.templ` |
| T6-3 | Add scope node expand/collapse                         | MEDIUM | 10min  | `html.templ` |

### Tier 7: Timeline Improvements

| #    | Task                                                  | Impact | Effort | File         |
| ---- | ----------------------------------------------------- | ------ | ------ | ------------ |
| T7-1 | Add shutdown duration bars (second color) to timeline | MEDIUM | 10min  | `html.templ` |
| T7-2 | Show invocation order numbers on timeline             | LOW    | 5min   | `html.templ` |

### Tier 8: Global UX

| #    | Task                                        | Impact | Effort | File         |
| ---- | ------------------------------------------- | ------ | ------ | ------------ |
| T8-1 | Add responsive padding (reduce on mobile)   | MEDIUM | 5min   | `html.templ` |
| T8-2 | Add keyboard navigation (1-4 keys for tabs) | LOW    | 8min   | `html.templ` |
| T8-3 | Add footer with "Generated by do-auditlog"  | LOW    | 3min   | `html.templ` |

---

## Execution Order (recommended)

1. T1-1 → T1-2 → T1-3 (data model first — everything depends on this)
2. T2-2 → T2-1 → T2-4 (highest-value services table fixes)
3. T3-2 → T3-3 → T3-1 (stats improvements)
4. T5-2 → T5-1 → T5-3 (graph improvements)
5. T4-1 → T4-2 → T4-3 (events improvements)
6. T6-1 → T6-2 → T6-3 (scope tree — new tab)
7. T2-3 → T2-5 (remaining services table)
8. T7-1 → T7-2 (timeline)
9. T8-1 → T8-2 → T8-3 (global polish)

Total: 26 tasks × ~8min avg = ~3.5h of focused work

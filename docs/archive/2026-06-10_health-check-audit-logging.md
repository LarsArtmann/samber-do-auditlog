# Health Check Audit Logging — Research Report

**Date**: 2026-06-10 · **Status**: PROPOSAL · **Schema Impact**: 0.1.0 → 0.2.0

---

## 1. Problem Statement

The auditlog plugin captures registration, invocation, and shutdown lifecycle events from samber/do v2 containers. Health checks (`do.Healthchecker` / `do.HealthcheckerWithContext` interfaces) are a first-class lifecycle concern in samber/do, but the plugin does **not** capture them.

This means:

- A service that passes health checks but fails later has no audit trail of its health history
- Users cannot see which services implement health checks, when they were last checked, or what the results were
- The HTML visualization has no health check status — a significant gap for operational monitoring

---

## 2. Upstream API Analysis (samber/do v2@v2.0.0)

### 2.1 Hook System

samber/do provides hooks via `InjectorOpts`:

```go
type InjectorOpts struct {
    HookBeforeRegistration []func(scope *Scope, serviceName string)
    HookAfterRegistration  []func(scope *Scope, serviceName string)
    HookBeforeInvocation   []func(scope *Scope, serviceName string)
    HookAfterInvocation    []func(scope *Scope, serviceName string, err error)
    HookBeforeShutdown     []func(scope *Scope, serviceName string)
    HookAfterShutdown      []func(scope *Scope, serviceName string, err error)
    // ...
}
```

**There are no `HookBeforeHealthCheck` / `HookAfterHealthCheck` hooks.** Health checks are entirely outside the hook lifecycle.

### 2.2 Health Check Execution Path

Health checks are invoked via the `Injector` interface:

```go
type Injector interface {
    HealthCheck() map[string]error
    HealthCheckWithContext(context.Context) map[string]error
    // ...
}
```

Execution flow:

1. `Scope.HealthCheckWithContext(ctx)` iterates all services in the scope + ancestors
2. Each service is checked via `scope.serviceHealthCheck(ctx, name)` — which calls `serviceWrapper.healthcheck(ctx)`
3. Results are aggregated into `map[string]error` (nil error = healthy)
4. Parallelism and timeouts are controlled by `InjectorOpts.HealthCheckParallelism`, `HealthCheckGlobalTimeout`, `HealthCheckTimeout`

### 2.3 Healthchecker Interfaces

```go
type Healthchecker interface {
    HealthCheck() error
}

type HealthcheckerWithContext interface {
    HealthCheck(context.Context) error
}
```

Services implement one of these. The `serviceWrapper` checks at registration time whether the instance satisfies either interface and stores `isHealthchecker() bool` and `healthcheck(ctx) error` on the wrapper.

### 2.4 Service Introspection

`do.ExplainInjector()` produces `ExplainInjectorServiceOutput` which includes `IsHealthchecker bool` and `IsShutdowner bool`. However, this is a separate diagnostic API — not part of the hook or health check execution path.

### 2.5 Key Constraint

**The return value is `map[string]error` — keyed by service name only, no scope information.** When multiple scopes contain services with the same name, the child scope's service shadows the parent's. This means health check results are inherently **flat and name-based**, unlike our hook-based events which carry `*do.Scope`.

---

## 3. Design Options

### Option A: Wrapper Method on Plugin

```go
results := plugin.RecordHealthCheck(injector, ctx)
// Internally: calls injector.HealthCheckWithContext(ctx), records events, returns results
```

| Aspect                    | Assessment                                                                                   |
| ------------------------- | -------------------------------------------------------------------------------------------- |
| **Upstream dependency**   | None — works today                                                                           |
| **User ergonomics**       | Must call plugin method instead of injector directly                                         |
| **Data model fit**        | Excellent — same Event type, same Recorder, same Report                                      |
| **Scope information**     | Partial — `map[string]error` has no scope, but we can look up scope from our service records |
| **Upstream-compatible**   | Yes — if hooks are added later, just wire to same recorder                                   |
| **Implementation effort** | Low — ~50 lines core logic                                                                   |

### Option B: Upstream PR for Hook API

Add `HookBeforeHealthCheck` / `HookAfterHealthCheck` to `InjectorOpts` in samber/do.

| Aspect                  | Assessment                                                                                         |
| ----------------------- | -------------------------------------------------------------------------------------------------- |
| **Upstream dependency** | Full — requires PR review, merge, release                                                          |
| **User ergonomics**     | Seamless — automatic, like existing hooks                                                          |
| **Data model fit**      | Excellent                                                                                          |
| **Scope information**   | Excellent — hooks would receive `*do.Scope`                                                        |
| **Risk**                | May not be accepted; API design questions (parallel execution makes before/after ordering complex) |
| **Timeline**            | Weeks to months                                                                                    |

### Option C: Background Polling

Plugin starts a goroutine that periodically calls `injector.HealthCheck()`.

| Aspect              | Assessment                                                    |
| ------------------- | ------------------------------------------------------------- |
| **User ergonomics** | Automatic                                                     |
| **Surprise factor** | High — implicit goroutine, resource usage                     |
| **Timing**          | User can't control when checks happen                         |
| **Complexity**      | Lifecycle management (start/stop/pause), context cancellation |
| **Verdict**         | **Rejected** — too surprising, too complex                    |

### Option D: Post-hoc Recording

User calls `injector.HealthCheck()` normally, then passes results to plugin:

```go
results := injector.HealthCheckWithContext(ctx)
plugin.RecordHealthCheckResults(results)
```

| Aspect                 | Assessment                                                         |
| ---------------------- | ------------------------------------------------------------------ |
| **User ergonomics**    | Two-step, easy to forget                                           |
| **Scope information**  | Lost — only service names                                          |
| **Timestamp accuracy** | Lost — health check already happened                               |
| **Duration**           | Lost — no timing                                                   |
| **Verdict**            | **Rejected** — loses critical data (duration, scope, exact timing) |

---

## 4. Recommendation: Option A — Wrapper Method

### 4.1 Rationale

Option A is the only approach that:

- Works today with no upstream changes
- Captures accurate timing (duration between before/after)
- Preserves scope information (we look up scope from our service records)
- Maintains the existing event/streaming model
- Doesn't introduce surprising background behavior

The one ergonomic cost (calling plugin method instead of injector directly) is acceptable because:

- Health checks are typically called from a single place (e.g., `/health` endpoint)
- It makes the audit logging explicit and discoverable
- It's the same pattern users already follow for `plugin.Report()` / `plugin.ExportToFile()`

### 4.2 Scope Resolution Strategy

Since `injector.HealthCheckWithContext()` returns `map[string]error` keyed by service name only, we need to resolve scope information ourselves. Strategy:

1. When `RecordHealthCheck` is called, iterate the result map
2. For each service name, look up the matching `serviceRecord` in our recorder (which has scope info from registration/invocation hooks)
3. If multiple scopes have the same service name, match against the scope passed to `RecordHealthCheck`

This works because the user passes the same injector they've been using with the plugin.

---

## 5. Proposed Data Model Changes

### 5.1 New EventType

```go
const (
    EventTypeRegistration EventType = "registration"     // existing
    EventTypeInvocation   EventType = "invocation"       // existing
    EventTypeShutdown     EventType = "shutdown"          // existing
    EventTypeHealthCheck  EventType = "health_check"      // NEW
)
```

### 5.2 Event Shape

Health check events use `PhaseAfter` only (no before-phase, since we time the wrapper call internally):

```go
Event{
    Sequence:    seq,
    Timestamp:   now,
    EventType:   EventTypeHealthCheck,
    Phase:       PhaseAfter,
    ServiceRef:  ServiceRef{ScopeID, ScopeName, ServiceName},
    ContainerID: containerID,
    DurationMs:  &durationMs,     // time spent on this service's health check
    Error:       errorOrNil,      // nil if healthy
}
```

**Why `PhaseAfter` only?** Unlike invocation/shutdown where hooks fire before and after the operation, health checks are user-initiated bulk operations. We time each service's check individually and emit a single event per service after the result is known. A `PhaseBefore` would be artificial — we don't intercept before the check runs.

**Alternative considered:** Emit both phases for consistency. Rejected because:

- There's no meaningful "before" state to capture
- It would double the event count with no additional information
- The before/after pattern exists because the upstream hook API provides both callbacks; here we control the full flow

### 5.3 ServiceInfo Additions

```go
type ServiceInfo struct {
    // ... existing fields ...

    LastHealthCheckAt       *time.Time `json:"last_health_check_at,omitempty"`       // NEW
    HealthCheckDurationMs   *float64   `json:"health_check_duration_ms,omitempty"`   // NEW
    HealthCheckError        *string    `json:"health_check_error,omitempty"`          // NEW
    HealthCheckCount        int        `json:"health_check_count"`                    // NEW
}
```

**Design decisions:**

- `LastHealthCheckAt` — not `FirstHealthCheckAt` because health checks are typically called repeatedly; the last result is most relevant operationally
- `HealthCheckDurationMs` — per-service duration from the most recent check, matches the pattern of `FirstBuildDurationMs` and `ShutdownDurationMs`
- `HealthCheckError` — from the most recent check; nil means healthy
- `HealthCheckCount` — how many times this service has been health-checked (useful for tracking monitoring frequency)

### 5.4 Report Additions

```go
type Report struct {
    // ... existing fields ...

    HealthCheckSucceeded       bool `json:"health_check_succeeded"`        // NEW
    HealthCheckedServiceCount  int  `json:"health_checked_service_count"`  // NEW
}
```

**Why not `TotalHealthCheckDurationMs`?** Unlike build/shutdown durations which are per-invocation, health check durations are per-service-per-check-call. A total would be misleading (depends on parallelism). Per-service durations in `ServiceInfo` are sufficient.

### 5.5 Schema Version

Bump from `"0.1.0"` to `"0.2.0"`. This is an **additive** change (new fields, new event type) — existing consumers that ignore unknown fields will continue to work.

---

## 6. Proposed API

### 6.1 Plugin Methods

```go
// RecordHealthCheck performs health checks on all services and records the results.
// Equivalent to injector.HealthCheck() but with audit logging.
// Returns the same map[string]error as the underlying call.
func (p *Plugin) RecordHealthCheck(injector do.Injector) map[string]error

// RecordHealthCheckWithContext performs health checks with context support and records the results.
// Equivalent to injector.HealthCheckWithContext(ctx) but with audit logging.
// Returns the same map[string]error as the underlying call.
func (p *Plugin) RecordHealthCheckWithContext(ctx context.Context, injector do.Injector) map[string]error
```

**Parameter order: `ctx` before `injector`.** This follows Go convention (context is always first) and mirrors `do.HealthCheckNamedWithContext(ctx, i, name)`.

**Return value:** Same `map[string]error` as the underlying call. This makes it a true drop-in replacement.

### 6.2 Disabled Plugin Behavior

When `Enabled: false`, both methods are no-ops that delegate directly to the injector without recording anything. Consistent with how `Opts()` returns empty hooks when disabled.

### 6.3 Recorder Method

```go
// RecordHealthCheck records a single health check result.
func (r *Recorder) RecordHealthCheck(scope *do.Scope, serviceName string, err error, durationMs float64)
```

This is the internal method called by the Plugin wrapper for each service result. It:

1. Looks up the `serviceRecord` to update aggregate fields
2. Emits an `Event` with `EventTypeHealthCheck`
3. Fires the `onEvent` callback

---

## 7. Scope Resolution — Edge Cases

### 7.1 Services Not Previously Seen

If a service appears in the health check results but was never registered/invoked through our hooks (e.g., registered before plugin was attached), we create a new `serviceRecord` for it. The scope is resolved from the injector passed to `RecordHealthCheck`.

### 7.2 Same Service Name in Multiple Scopes

`injector.HealthCheckWithContext()` returns results for the calling scope AND all ancestor scopes. If the same service name exists in both parent and child scope, the child shadows the parent. We resolve scope by:

1. First checking the calling scope's services
2. Then walking up ancestors
3. Using the first match

This mirrors samber/do's own resolution behavior.

### 7.3 Services That Don't Implement Healthchecker

`injector.HealthCheckWithContext()` only returns results for services that implement `do.Healthchecker` or `do.HealthcheckerWithContext`. Services without health checks simply won't appear in the results — no events are emitted for them.

---

## 8. ServiceStatus Interaction

The existing `ServiceStatus` enum does **not** gain a health-check-specific status. Health checks are diagnostic, not lifecycle-changing. A service that fails a health check is still `active` (or whatever lifecycle state it was in). The health check error is recorded separately in `HealthCheckError`.

This is a deliberate design decision:

- `ServiceStatus` represents **lifecycle state** (registered → active → shutdown)
- `HealthCheckError` represents **runtime health** (independent of lifecycle)
- A service can be `active` and unhealthy, or `shutdown` and have been healthy until shutdown

---

## 9. Concurrency Model

Health checks are already parallelized by samber/do (controlled by `HealthCheckParallelism`). Our `RecordHealthCheck` method processes results **after** the bulk check completes, so we don't need additional synchronization beyond what the Recorder already has:

- `Recorder.mu` protects `events` and `services` — already handles concurrent writes
- Event sequence numbers use `atomic.Int64` — already race-free
- `onEvent` callback is called sequentially per event — already safe

---

## 10. Files Changed

| File               | Nature       | Description                                                                                                                  |
| ------------------ | ------------ | ---------------------------------------------------------------------------------------------------------------------------- |
| `types.go`         | Modification | `EventTypeHealthCheck`, `IsHealthCheck()`, `ServiceInfo` health fields, `Report` health fields                               |
| `recorder.go`      | Modification | `RecordHealthCheck()` method, `serviceRecord` health fields, `buildServicesLocked` updates, `computeServiceStatus` unchanged |
| `plugin.go`        | Modification | `RecordHealthCheck()`, `RecordHealthCheckWithContext()` methods                                                              |
| `html.templ`       | Modification | Health check status column in services table, health check badge                                                             |
| `auditlog_test.go` | Modification | Tests: healthy service, unhealthy service, disabled plugin, multiple checks, scope resolution                                |
| `AGENTS.md`        | Modification | Document new API and health check support                                                                                    |
| `doc.go`           | Modification | Add health check to package description                                                                                      |

Schema version: `"0.1.0"` → `"0.2.0"`

---

## 11. Future Work

- **Upstream hook PR** — If samber/do adds `HookBeforeHealthCheck`/`HookAfterHealthCheck`, the wrapper method remains valid but can also be triggered automatically via hooks
- **Health check history** — Currently stores only last check result. Could add a ring buffer of recent results for trend analysis
- **Periodic health check helper** — `plugin.StartHealthCheckLoop(injector, interval)` that calls `RecordHealthCheck` on a timer (separate from core recording logic)
- **HTML health check tab** — Dedicated tab showing health check timeline/history alongside the existing 5 tabs

---

## 12. Alternative Designs Considered and Rejected

### 12.1 Dual-Phase Events (before + after)

Emitting both `PhaseBefore` and `PhaseAfter` for each health check. Rejected because:

- We control the entire wrapper — there's no interception point before the check
- Would double event count with no additional data
- Before/after exists for other events because the upstream hook API provides both callbacks

### 12.2 Health Check as ServiceStatus

Adding `ServiceStatusHealthCheckFailed` to the status enum. Rejected because:

- Health check failure is transient and diagnostic, not a lifecycle transition
- Would conflate "broken build" with "temporarily unhealthy"
- A service can fail a health check and recover — status would oscillate confusingly

### 12.3 Storing All Health Check Results

Keeping a full history of every health check result on `ServiceInfo`. Rejected because:

- Unbounded memory growth in long-running services with frequent health checks
- Breaks the current model where `ServiceInfo` is a snapshot, not a time series
- Events stream already captures the full history — consumers can query `EventsByType(EventTypeHealthCheck)`

### 12.4 Instrumenting Individual Service Health Checks

Rather than wrapping the bulk `HealthCheckWithContext()`, intercepting individual `serviceHealthCheck()` calls. Rejected because:

- Not possible without upstream hooks
- Would require wrapping/modifying internal service wrappers
- Breaks encapsulation of samber/do internals

# Features

Honest inventory of what samber-do-auditlog actually does, verified against the code.

---

## DONE

| Feature                               | Description                                                                                                      | Verified                                                          |
| ------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| **Drop-in plugin setup**              | `New(Config)` + `Opts()` → one-line integration with `do.NewWithOpts`                                            | ✓ `plugin.go:41-81`                                               |
| **Service registration tracking**     | Captures before/after registration events with timestamps                                                        | ✓ `recorder.go:169-198`                                           |
| **Service invocation tracking**       | Captures before/after invocation events with timestamps, duration, errors                                        | ✓ `recorder.go:200-310`                                           |
| **Shutdown tracking**                 | Captures before/after shutdown events with duration and errors                                                   | ✓ `recorder.go:312-357`                                           |
| **Dependency graph inference**        | Stack-based: infers A→B when A is on-stack during B's before-hook                                                | ✓ `recorder.go:204-216`                                           |
| **Reverse dependencies**              | `Dependents` field computed at report time from forward deps                                                     | ✓ `recorder.go:467-483`                                           |
| **Scope tree**                        | Builds hierarchical scope tree with per-scope service lists                                                      | ✓ `recorder.go:485-542`                                           |
| **Scope tracking**                    | Records scope metadata (ID, name, parent) for all scopes                                                         | ✓ `recorder.go:100-117`                                           |
| **Monotonic sequence numbers**        | Per-recorder atomic counter — no global state                                                                    | ✓ `recorder.go:52-56, 96-98`                                      |
| **Build duration measurement**        | Tracks first build duration in milliseconds for each service                                                     | ✓ `recorder.go:253-271`                                           |
| **Invocation ordering**               | Global invocation order across all services                                                                      | ✓ `recorder.go:294-301`                                           |
| **Provider error capture**            | Records invocation errors as string pointers in events and service info                                          | ✓ `recorder.go:367-375`                                           |
| **JSON report export**                | Full `Report` as indented JSON to `io.Writer` or file                                                            | ✓ `plugin.go:88-120`                                              |
| **NDJSON event stream**               | Each event as a JSON line to `io.Writer` or file                                                                 | ✓ `plugin.go:102-125`                                             |
| **Self-contained HTML visualization** | Dark-themed page with services table, scopes tab, dependency graph, timeline, events, responsive layout          | ✓ `html.templ`                                                    |
| **Dependency graph visualization**    | Sugiyama layered DAG layout with barycenter crossing reduction, cubic Bézier edges, pan/zoom, click-to-highlight | ✓ `html.templ`                                                    |
| **Timeline visualization**            | Dual horizontal bar chart: build duration (blue) + shutdown duration (yellow)                                    | ✓ `html.templ`                                                    |
| **Scopes tree visualization**         | Collapsible scope tree with service counts and name chips                                                        | ✓ `html.templ`                                                    |
| **HTML search & filter**              | Live search on services table, event type filter chips, keyboard tab navigation (1-5)                            | ✓ `html.templ`                                                    |
| **Responsive HTML layout**            | Mobile-friendly with media queries, footer with schema version                                                   | ✓ `html.templ`                                                    |
| **Environment variable toggle**       | `DO_AUDITLOG_ENABLED` = true/1/yes enables without code change                                                   | ✓ `plugin.go:56-64`                                               |
| **Zero-cost disabled mode**           | `Enabled: false` → empty `InjectorOpts`, no hooks, no allocation                                                 | ✓ `plugin.go:68-71`                                               |
| **Explicit enable override**          | `Config.Enabled: true` overrides env var                                                                         | ✓ `plugin.go:46-48`                                               |
| **Container ID**                      | Human-readable identifier propagated to all events                                                               | ✓ `plugin.go:22-26`                                               |
| **Concurrent-safe recording**         | 4-lock design: RWMutex for state, Mutex for stack, ordering, shutdown                                            | ✓ `recorder.go:59-76`                                             |
| **Deterministic report output**       | Services sorted by (scope_name, service_name), scope tree sorted by scope ID                                     | ✓ `recorder.go:429-434, sortedScopesLocked`                       |
| **Transient provider support**        | Works with `do.ProvideTransient` — tracks multiple invocations                                                   | ✓ Tested: `TestPlugin_ProvideTransient`                           |
| **Value provider support**            | Works with `do.ProvideValue`                                                                                     | ✓ Tested: `TestPlugin_ProvideValue`                               |
| **Named service support**             | Works with `do.ProvideNamed` / `do.InvokeNamed`                                                                  | ✓ Tested throughout                                               |
| **Schema versioning**                 | `SchemaVersion` constant for forward compatibility                                                               | ✓ `types.go:6`                                                    |
| **Defensive copies**                  | `Events()` and `Report()` return copies, not internal slices                                                     | ✓ `recorder.go:544-550, 378-395`                                  |
| **Service lifecycle status**          | Computed `ServiceStatus` field: registered, active, invocation_error, shutdown, shutdown_error                   | ✓ `types.go:ServiceStatus`, `recorder.go:computeServiceStatus`    |
| **ServiceRef type**                   | Embedded in `Event` and `ServiceInfo` — single source of truth for service identity (renamed from DependencyRef) | ✓ `types.go:ServiceRef`                                           |
| **Event convenience methods**         | `IsRegistration()`, `IsInvocation()`, `IsShutdown()`, `IsBefore()`, `IsAfter()`                                  | ✓ `types.go:Event methods`                                        |
| **EventHandler callback**             | `Config.OnEvent func(Event)` for real-time event streaming, called outside mutex, nil = disabled                 | ✓ `plugin.go:Config.OnEvent`, `recorder.go:addEvent`              |
| **ServiceRef.String()**               | Human-readable `"scope/name"` format for compact display                                                         | ✓ `types.go:ServiceRef.String()`                                  |
| **ServiceStatus.IsError()**           | `true` for invocation_error or shutdown_error                                                                    | ✓ `types.go:ServiceStatus.IsError()`                              |
| **Report convenience methods**        | `ServiceByName(name)`, `EventsByType(t)`, `FailedServices()` for querying report data                            | ✓ `types.go:Report methods`                                       |
| **ProviderType**                      | Named type for service provider kinds (lazy, eager, transient, alias) with Icon() and String() methods           | ✓ `types.go:ProviderType`                                         |
| **Health check auditing**             | `RecordHealthCheck()` / `RecordHealthCheckWithContext()` wraps injector health checks with audit events          | ✓ `plugin.go:RecordHealthCheck*`, `recorder.go:RecordHealthCheck` |
| **Health check events**               | `EventTypeHealthCheck` with `IsHealthCheck()`, PhaseAfter only, no DurationMs (per-service timing unavailable)   | ✓ `types.go:EventTypeHealthCheck`                                 |
| **Health check service fields**       | `LastHealthCheckAt`, `HealthCheckError`, `HealthCheckCount` on ServiceInfo; `IsHealthchecker`, `IsShutdowner`    | ✓ `types.go:ServiceInfo`                                          |
| **Health check report fields**        | `HealthCheckSucceeded`, `HealthCheckedCount` on Report; `UnhealthyServices()` convenience method                 | ✓ `types.go:Report`                                               |
| **Health check scope resolution**     | `ResolveServiceScope()` handles both RootScope and child Scope ancestor lookup                                   | ✓ `recorder.go:ResolveServiceScope`                               |
| **Health check HTML visualization**   | Health column in services table, health_check event badge (amber), filter chip, conditional stat card            | ✓ `html.templ`                                                    |
| **Event.ServiceType**                 | ProviderType carried on every event, populated in newEvent/newEventFromRef                                        | ✓ `types.go:Event.ServiceType`                                    |
| **Capability detection**              | `enrichCapabilities()` in BuildReport() populates IsHealthchecker/IsShutdowner via `do.ExplainInjector`           | ✓ `recorder.go:enrichCapabilities`                                |
| **Config.Validate()**                 | Forward-compatible API placeholder for config validation                                                           | ✓ `plugin.go:Config.Validate`                                     |
| **Provider column in Events tab**     | HTML Events tab shows provider type badge per event                                                               | ✓ `html.templ`                                                    |
| **Zero golangci-lint issues**         | All 28 lint issues fixed across production code, tests, and example                                               | ✓ `.golangci.yml`                                                 |

---

## PARTIALLY DONE

| Feature              | What's Done                     | What's Missing                        |
| -------------------- | ------------------------------- | ------------------------------------- |
| **Schema migration** | `SchemaVersion` constant exists | No migration function for old exports |

---

## PLANNED

| Feature                       | Description                                                            | Priority |
| ----------------------------- | ---------------------------------------------------------------------- | -------- |
| **Report filtering**          | Functional options to filter report by service, time range, event type | P2       |
| **Additional export formats** | Mermaid diagram, PlantUML                                              | Future   |

---

## NOT PLANNED (but worth considering)

| Feature                                            | Why Not Now                                                         |
| -------------------------------------------------- | ------------------------------------------------------------------- |
| **Multi-module split**                             | Project is too small (1 package, ~2400 LOC) — revisit at 5+ packages |
| **External storage backends**                      | YAGNI — file/io.Writer exports are sufficient                       |
| **Metrics integration (Prometheus/OpenTelemetry)** | Out of scope for audit logging — use EventHandler when available    |

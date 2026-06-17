# Features

Honest inventory of what `samber-do-auditlog` actually does, verified against the code. Status labels mean exactly what they say — no "planned" item here is already implemented.

---

## FULLY FUNCTIONAL

### Core Plugin / Container Integration

| Feature                             | Description                                                                                             | Verified                                      |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------- | --------------------------------------------- |
| **Plugin constructor**              | `New(Config) (*Plugin, error)` validates config, applies env-var enablement, initializes recorder       | `plugin.go` (`New`)                           |
| **Injector options generation**     | `Opts()` returns `*do.InjectorOpts` wiring all six lifecycle hooks into samber/do v2                    | `plugin.go` (`Opts`)                          |
| **Environment-variable enablement** | `DO_AUDITLOG_ENABLED` (`true`/`1`/`yes`) enables logging without code change                            | `plugin.go` (`EnvKeyEnabled`, `envIsEnabled`) |
| **Explicit enable override**        | `Config.Enabled: true` bypasses the env-var check                                                       | `plugin.go` (`New`)                           |
| **Zero-cost disabled mode**         | When disabled, `Opts()` returns empty hooks and `RecordHealthCheck*` delegates directly to the injector | `plugin.go` (`Opts`, `RecordHealthCheck*`)    |
| **Container ID**                    | Human-readable identifier propagated to events, report, and HTML title                                  | `plugin.go` (`Config.ContainerID`)            |
| **Config validation**               | Rejects `ContainerID` values containing `/` or `\` path separators                                      | `plugin.go` (`Config.Validate`)               |
| **Real-time event callback**        | `Config.OnEvent func(Event)` streams every captured event outside the recorder lock                     | `plugin.go`, `recorder.go`                    |
| **In-memory event cap**             | `Config.MaxEvents` caps stored events and exposes a drop counter                                        | `plugin.go`, `recorder.go`                    |
| **Initial event capacity**          | `Config.InitialEventCapacity` pre-allocates the events slice                                            | `plugin.go`, `recorder.go`                    |

### Lifecycle Event Recording

| Feature                                | Description                                                                               | Verified                                                   |
| -------------------------------------- | ----------------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| **Registration events**                | `before`/`after` registration for every service                                           | `hooks.go` (`OnBeforeRegistration`, `OnAfterRegistration`) |
| **Invocation events**                  | `before`/`after` invocation with duration and errors                                      | `hooks.go` (`OnBeforeInvocation`, `OnAfterInvocation`)     |
| **Shutdown events**                    | `before`/`after` shutdown with duration and errors                                        | `hooks.go` (`OnBeforeShutdown`, `OnAfterShutdown`)         |
| **Health-check events**                | Per-service `health_check`/`after` events                                                 | `healthcheck.go` (`RecordHealthCheck`)                     |
| **Event type enum**                    | `registration`, `invocation`, `shutdown`, `health_check`                                  | `types.go` (`EventType`)                                   |
| **Phase enum**                         | `before`, `after`                                                                         | `types.go` (`Phase`)                                       |
| **Provider type enum**                 | `lazy`, `eager`, `transient`, `alias` with `String()`, `IsKnown()`, `Icon()`              | `types.go` (`ProviderType`)                                |
| **Service status enum**                | `registered`, `active`, `invocation_error`, `shutdown`, `shutdown_error` with `IsError()` | `types.go` (`ServiceStatus`)                               |
| **Service reference identity**         | `ServiceRef` embeds scope ID/name + service name; provides `String()` and `IsRoot()`      | `types.go` (`ServiceRef`)                                  |
| **Sequence numbers**                   | Per-recorder atomic counter; no global state                                              | `recorder.go`                                              |
| **Invocation ordering**                | Global invocation order counter stored per service                                        | `hooks.go`, `recorder.go`                                  |
| **Build duration tracking**            | First-build duration in milliseconds per service                                          | `hooks.go`                                                 |
| **Shutdown duration tracking**         | Shutdown duration in milliseconds per service                                             | `hooks.go`                                                 |
| **Error capture**                      | Invocation/shutdown/health errors stored as `*string` in events and service records       | `hooks.go`, `healthcheck.go`                               |
| **Dependency graph inference**         | Stack-based: if A is on-stack when B is invoked, A depends on B                           | `hooks.go`                                                 |
| **Reverse dependencies**               | `Dependents` field computed at report time from forward deps                              | `report_builder.go`                                        |
| **Scope tracking**                     | Records scope ID, name, parent ID, and reference for all scopes                           | `recorder.go` (`recordScopeLocked`)                        |
| **Capability detection**               | `IsHealthchecker`/`IsShutdowner` populated via `do.ExplainInjector`                       | `report_builder.go` (`enrichCapabilities`)                 |
| **Scope resolution for health checks** | `ResolveServiceScope` handles root scope and ancestor lookup                              | `healthcheck.go` (`ResolveServiceScope`)                   |
| **Concurrent-safe recording**          | Single `sync.RWMutex` plus atomic counters; callbacks invoked outside the lock            | `recorder.go`                                              |
| **Deterministic output**               | Services sorted by (scope_name, service_name); scope tree sorted by scope ID              | `report_builder.go`                                        |

### Report Model

| Feature                        | Description                                                                                                                                 | Verified                        |
| ------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------- |
| **Report struct**              | Consolidated snapshot with version, container ID, counts, durations, success flags, events, services, scope tree                            | `report.go`                     |
| **Schema version**             | Current report schema is `"0.2.0"`                                                                                                          | `types.go` (`SchemaVersion`)    |
| **Service info aggregate**     | Per-service rollup of status, type, timings, deps, dependents, errors, health                                                               | `service.go` (`ServiceInfo`)    |
| **Scope tree**                 | Hierarchical `ScopeNode` with services and children                                                                                         | `service.go` (`ScopeNode`)      |
| **Report validation**          | Checks denormalized counts match actual slice/tree lengths                                                                                  | `report.go` (`Report.Validate`) |
| **Report indexing**            | `Report.Index()` builds O(1) lookups by name, ref, scope, events                                                                            | `report.go` (`Index`)           |
| **Report convenience queries** | `ServiceByName`, `ServiceByRef`, `ServicesByScope`, `EventsByService`, `EventsByRef`, `EventsByType`, `FailedServices`, `UnhealthyServices` | `report.go`                     |
| **Event convenience helpers**  | `IsRegistration`, `IsInvocation`, `IsShutdown`, `IsHealthCheck`, `IsBefore`, `IsAfter`, `HasError`, `Duration`                              | `event.go`                      |
| **Service info helpers**       | `Uptime()`, `HasHealthError()`                                                                                                              | `service.go`                    |
| **Report diff**                | `Report.Diff(other)` returns added/removed/changed services and event-count delta                                                           | `diff.go`                       |

### Report Filtering

| Feature                     | Description                                                                                             | Verified    |
| --------------------------- | ------------------------------------------------------------------------------------------------------- | ----------- |
| **Filtered report**         | `Report.Filtered(opts...)` returns a new report with matching services/events and recomputed aggregates | `filter.go` |
| **Filter by service name**  | `WithServicesByName(names...)`                                                                          | `filter.go` |
| **Filter by provider type** | `WithServicesByType(providerType)`                                                                      | `filter.go` |
| **Filter by event type**    | `WithEventsByType(eventType)`                                                                           | `filter.go` |
| **Filter by time range**    | `WithTimeRange(from, to)`                                                                               | `filter.go` |
| **Filter by scope**         | `WithScope(scopeID)`                                                                                    | `filter.go` |
| **Pruned scope tree**       | Filtered reports keep only scopes with at least one matching service                                    | `filter.go` |
| **Plugin filtered report**  | `Plugin.ReportFiltered(opts...)`                                                                        | `plugin.go` |

### Export Formats

| Feature                           | Description                                                                           | Verified                    |
| --------------------------------- | ------------------------------------------------------------------------------------- | --------------------------- |
| **JSON report to writer**         | `Plugin.WriteReportJSON(writer)`                                                      | `plugin.go`                 |
| **NDJSON event stream to writer** | `Plugin.WriteEventsNDJSON(writer)`                                                    | `plugin.go`                 |
| **JSON report to file**           | `Plugin.ExportToFile(path)`                                                           | `plugin.go`                 |
| **NDJSON events to file**         | `Plugin.ExportEventsToNDJSON(path)`                                                   | `plugin.go`                 |
| **Filtered JSON report to file**  | `Plugin.ExportFilteredToFile(path, opts...)`                                          | `plugin.go`                 |
| **Report JSON writer**            | `Report.WriteJSON(writer)`                                                            | `report.go`                 |
| **Report NDJSON writer**          | `Report.WriteNDJSON(writer)`                                                          | `report.go`                 |
| **Atomic file writes**            | File exports write to temp file and rename for crash safety                           | `plugin.go` (`writeToFile`) |
| **Mermaid diagram export**        | `Report.WriteMermaid(writer)` outputs a themed flowchart                              | `mermaid.go`, `diagram.go`  |
| **PlantUML diagram export**       | `Report.WritePlantUML(writer)` outputs a styled component diagram                     | `plantuml.go`, `diagram.go` |
| **Shared diagram formatter**      | `diagramFormatter` interface drives Mermaid/PlantUML with deduplicated, sorted output | `diagram.go`                |
| **Self-contained HTML export**    | `Plugin.ExportToHTML(path)` and `Plugin.WriteHTML(w)` render a single-file report     | `html.go`, `html.templ`     |

### HTML Visualization

| Feature                     | Description                                                                                                                                         | Verified                    |
| --------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------- |
| **Dark-themed dashboard**   | Custom CSS color scheme, IBM Plex Mono / Space Grotesk typography                                                                                   | `html.templ`                |
| **Header with metadata**    | Container ID, schema version, export timestamp                                                                                                      | `html.templ`                |
| **Stats cards**             | Services, scopes, events, dependencies, total build, errors, health-check summary                                                                   | `html.templ`                |
| **Lifecycle waveform**      | Event timeline strip colored by event type, height scaled by duration                                                                               | `html.templ`                |
| **Services table**          | Columns for service, type, scope, status, order, invocations, build/shutdown ms, deps, dependents, health                                           | `html.templ`                |
| **Scopes tree**             | Collapsible tree with service counts and provider icons                                                                                             | `html.templ`                |
| **Dependency graph**        | Sugiyama layered DAG layout with barycenter crossing reduction, cubic Bézier edges, pan/zoom/fit, click-to-highlight, mouse wheel and touch support | `html.templ`                |
| **Timeline tab**            | Dual horizontal bar chart of build vs shutdown durations                                                                                            | `html.templ`                |
| **Events table**            | Event list with sequence, time, type badge, provider badge, phase, scope, service, duration, error                                                  | `html.templ`                |
| **Live service search**     | Debounced filter on service name/scope/type                                                                                                         | `html.templ`                |
| **Event type filter chips** | All / registration / invocation / shutdown / health_check buttons                                                                                   | `html.templ`                |
| **Keyboard navigation**     | Number keys 1-5 switch tabs; ignored when focus is in form controls                                                                                 | `html.templ`                |
| **Pagination**              | "Show first N" with "Show all" button for services (50) and events (100)                                                                            | `html.templ`                |
| **Error tooltips**          | Click error status badges to show the full error message                                                                                            | `html.templ`                |
| **Type metadata injection** | `BuildTypeMetadata()` JSON is injected into the page as the single source of truth for icons and labels                                             | `metadata.go`, `html.templ` |
| **Responsive layout**       | Mobile-friendly padding, media queries, reduced-motion support                                                                                      | `html.templ`                |
| **Content Security Policy** | CSP meta tag with `base-uri 'none'`, `frame-ancestors 'none'`, and inline style/script allowances                                                   | `html.templ`                |
| **XSS hardening**           | All user-controlled strings escaped via `esc()` before DOM insertion                                                                                | `html.templ`                |

### Schema Migration

| Feature                                | Description                                                                                                  | Verified                            |
| -------------------------------------- | ------------------------------------------------------------------------------------------------------------ | ----------------------------------- |
| **Report migration**                   | `MigrateReport([]byte)` upgrades older JSON reports to current schema, recomputing derived fields and status | `migration.go`                      |
| **Status derivation from legacy data** | Computes service status when missing from imported reports                                                   | `migration.go`, `report_helpers.go` |

### Health Checks

| Feature                          | Description                                                                                           | Verified         |
| -------------------------------- | ----------------------------------------------------------------------------------------------------- | ---------------- |
| **Health-check wrapper methods** | `Plugin.RecordHealthCheck(injector)` and `RecordHealthCheckWithContext(ctx, injector)`                | `plugin.go`      |
| **Health-check event recording** | `Recorder.RecordHealthCheck` emits `EventTypeHealthCheck` events and updates per-service health state | `healthcheck.go` |
| **Health-check report fields**   | `Report.HealthCheckSucceeded`, `HealthCheckedCount`, `DroppedEventCount`                              | `report.go`      |
| **Health-check service fields**  | `ServiceInfo.LastHealthCheckAt`, `HealthCheckError`, `HealthCheckCount`, `HasHealthError()`           | `service.go`     |
| **Unhealthy service lookup**     | `Report.UnhealthyServices()`                                                                          | `report.go`      |

### Testing / Infrastructure

| Feature                      | Description                                                                                        | Verified                   |
| ---------------------------- | -------------------------------------------------------------------------------------------------- | -------------------------- |
| **GitHub Actions CI**        | `go vet`, `go build`, race-detector tests, golangci-lint, govulncheck, generated-code drift checks | `.github/workflows/ci.yml` |
| **golangci-lint config**     | `.golangci.yml` defines lint rules for the project                                                 | `.golangci.yml`            |
| **Generated-code check**     | CI runs `go generate ./...` and fails on drift, ensuring `html_templ.go` stays in sync             | `.github/workflows/ci.yml` |
| **templ code generation**    | `//go:generate go tool templ generate` in `html.go` produces `html_templ.go`                       | `html.go`, `html_templ.go` |
| **Fuzz tests**               | XSS fuzz targets for HTML output (service names, error messages, dependency chains)                | `fuzz_test.go`             |
| **Benchmark tests**          | Performance benchmarks for hot paths                                                               | `benchmarks_test.go`       |
| **Example tests**            | Runnable `Example*` functions for pkg.go.dev                                                       | `example_test.go`          |
| **Defensive-copy accessors** | `Plugin.Events()` and `Recorder.Events()` return copied slices; `EventsCount()` avoids copying     | `plugin.go`, `recorder.go` |
| **Dropped-event counter**    | `Plugin.DroppedEventCount()` / `Recorder.DroppedEventCount()`                                      | `plugin.go`, `recorder.go` |

---

## PARTIALLY FUNCTIONAL

| Feature              | Description                                                                                                                    | Status            |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------ | ----------------- |
| **Test parallelism** | Only ~15% of tests use `t.Parallel()`; the rest run sequentially. Suite is ~1s, so not a bottleneck yet.                       | Could be expanded |
| **Fuzz coverage**    | All fuzz targets test HTML XSS; no fuzzing of `MigrateReport`, Mermaid/PlantUML special characters, nested scopes, or filters. | Could be expanded |
| **Metadata testing** | `BuildTypeMetadata()` is exercised indirectly by HTML tests; individual emoji/label/color values are not directly asserted.    | Could be expanded |

---

## WORTH CONSIDERING

| Feature                                            | Why Not Now                                                               |
| -------------------------------------------------- | ------------------------------------------------------------------------- |
| **External storage backends**                      | File/`io.Writer` exports are sufficient for current scope                 |
| **Prometheus / OpenTelemetry metrics integration** | Users can derive metrics via `Config.OnEvent`; OTel bridge example exists |
| **Multi-module repository split**                  | Project is one package (~3,000 LOC); revisit at 5+ packages               |
| **gosec static analysis in CI**                    | Security linting alongside existing govulncheck                           |
| **CSV / TSV export**                               | Tabular export of services or events for spreadsheets                     |
| **CLI tool**                                       | Stand-alone binary to convert/export/visualize saved reports              |
| **NDJSON import**                                  | Enable loading events back from NDJSON for diffing across time            |
| **JSON Schema file**                               | Machine-readable schema for the report format                             |
| **Property-based testing**                         | `rapid`/`gopter` tests for `Diff`, `MigrateReport`, filter round-trips    |
| **WebSocket live stream**                          | Bridge `OnEvent` to real-time dashboards                                  |
| **v0.1.0 release**                                 | Project meets `STABILITY.md` criteria for a first stable-ish tag          |

---

_Last verified against the codebase on 2026-06-17._

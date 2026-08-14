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
| **Late event-callback wiring**       | `Plugin.SetOnEvent` attaches or replaces the callback after `New()`, race-safe with recording goroutines (enables live-dashboard hubs wired after CLI flag parsing) | `plugin.go` (`SetOnEvent`), `recorder.go` (`setOnEvent`) |
| **Late enablement**                  | `Plugin.Enable` turns logging on after `New()` (idempotent; effective before `Opts()` is consumed)                                       | `plugin.go` (`Enable`)                        |
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

| Feature                        | Description                                                                                                                                              | Verified                        |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------- |
| **Report struct**              | Consolidated snapshot with version, container ID, counts, durations, success flags, events, services, scope tree                                         | `report.go`                     |
| **Schema version**             | Current report schema is `"0.3.0"`                                                                                                                       | `types.go` (`SchemaVersion`)    |
| **Service info aggregate**     | Per-service rollup of status, type, timings, deps, dependents, errors, health                                                                            | `service.go` (`ServiceInfo`)    |
| **Scope tree**                 | Hierarchical `ScopeNode` with services and children                                                                                                      | `service.go` (`ScopeNode`)      |
| **Report validation**          | Checks denormalized counts match actual slice/tree lengths                                                                                               | `report.go` (`Report.Validate`) |
| **Report indexing**            | `Report.Index()` builds O(1) lookups by name, ref, scope, events                                                                                         | `report.go` (`Index`)           |
| **Report convenience queries** | `ServiceByName`, `ServiceByRef`, `ServicesByScope`, `EventsByService`, `EventsByRef`, `EventsByType`, `FailedServices`, `UnhealthyServices`              | `report.go`                     |
| **Event convenience helpers**  | `IsRegistration`, `IsInvocation`, `IsShutdown`, `IsHealthCheck`, `IsBefore`, `IsAfter`, `HasError`, `Duration`                                           | `event.go`                      |
| **Service info helpers**       | `Uptime()`, `HasHealthError()`                                                                                                                           | `service.go`                    |
| **Report diff**                | `Report.Diff(other)` returns added/removed/changed services, event-count delta, and timing deltas (`TotalBuildDurationMsDelta`, `TotalShutdownDurationMsDelta`). `HasChanges()` / `IsEmpty()` for polarity-agnostic checks | `diff.go`                       |

### Event Streaming

|| Feature                     | Description                                                                                             | Verified               |
| --------------------------- | ------------------------------------------------------------------------------------------------------- | ---------------------- |
| **MultiWriter event fan-out** | `MultiWriter` broadcasts events to multiple `OnEvent` callbacks simultaneously. Thread-safe, ordered   | `multi_writer.go`      |
| **StreamEvents callback reader** | `StreamEvents(reader, callback)` reads NDJSON events line-by-line via `bufio.Scanner`, invoking callback per event | `ndjson.go`            |
| **Flush interval**           | `WithFlushInterval(d)` on `NDJSONStreamer` for bounded-latency time-based flushing                     | `stream.go`            |
| **RunID correlation**        | 128-bit hex branded type auto-generated via `crypto/rand`, stamped on every `Event` and `Report`        | `runid.go`, `types.go` |

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

| Feature                           | Description                                                                                 | Verified                    |
| --------------------------------- | ------------------------------------------------------------------------------------------- | --------------------------- |
| **JSON report to writer**         | `Plugin.WriteReportJSON(writer)`                                                            | `plugin.go`                 |
| **NDJSON event stream to writer** | `Plugin.WriteEventsNDJSON(writer)`                                                          | `plugin.go`                 |
| **JSON report to file**           | `Plugin.ExportToFile(path)`                                                                 | `plugin.go`                 |
| **NDJSON events to file**         | `Plugin.ExportEventsToNDJSON(path)`                                                         | `plugin.go`                 |
| **Filtered JSON report to file**  | `Plugin.ExportFilteredToFile(path, opts...)`                                                | `plugin.go`                 |
| **Plugin CSV/TSV export**         | `Plugin.WriteReportCSV/TSV(w)` and `Plugin.ExportToCSV/TSV(path)`                           | `plugin.go`                 |
| **Plugin diagram export**         | `Plugin.WriteMermaid/PlantUML/DOT/D2(w)` and `Plugin.ExportToMermaid/PlantUML/DOT/D2(path)` | `plugin.go`                 |
| **Plugin tree export**            | `Plugin.WriteTree/WriteHTMLTree(w)` and `Plugin.ExportToTree/ExportToHTMLTree(path)`        | `plugin.go`, `tree.go`      |
| **Plugin table export**           | `Plugin.WriteTable(w, format, opts)` and `Plugin.ExportToTable(path, format, opts)`         | `plugin.go`, `table.go`     |
| **Report JSON writer**            | `Report.WriteJSON(writer)`                                                                  | `report.go`                 |
| **Report NDJSON writer**          | `Report.WriteNDJSON(writer)`                                                                | `report.go`                 |
| **Atomic file writes**            | File exports write to temp file and rename for crash safety                                 | `plugin.go` (`writeToFile`) |
| **Mermaid diagram export**        | `Report.WriteMermaid(writer)` outputs a themed flowchart                                    | `mermaid.go`, `diagram.go`  |
| **PlantUML diagram export**       | `Report.WritePlantUML(writer)` outputs a styled component diagram                           | `plantuml.go`, `diagram.go` |
| **DOT diagram export**            | `Report.WriteDOT(writer)` outputs a Graphviz digraph                                        | `dot.go`, `diagram.go`      |
| **D2 diagram export**             | `Report.WriteD2(writer)` outputs a D2 diagram with per-node warm-amber styling              | `d2.go`, `diagram.go`       |
| **Shared diagram builder**        | `buildDiagramNodes`/`buildDiagramEdges` drives all four formats with deduplicated output    | `diagram.go`                |
| **ASCII tree export**             | `Report.WriteTree(writer)` outputs a dependency DAG as an ASCII tree                        | `tree.go`                   |
| **HTML tree export**              | `Report.WriteHTMLTree(writer)` outputs a dependency DAG as an HTML nested list              | `tree.go`                   |
| **Multi-format table export**     | `Report.WriteTable(writer, format, opts)` outputs service summary in 16+ formats            | `table.go`                  |
| **Self-contained HTML export**    | `Plugin.ExportToHTML(path)` and `Plugin.WriteHTML(w)` render a single-file report           | `html.go`, `html.templ`     |
| **Write\*String convenience**     | `WriteMermaidString()`, `WritePlantUMLString()`, `WriteDOTString()`, `WriteD2String()`, `WriteHTMLString()` return output as `string` | `mermaid.go` etc.           |

### HTML Visualization

| Feature                     | Description                                                                                                                                                                                                                       | Verified                    |
| --------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------- |
| **Dark-themed dashboard**   | Custom CSS color scheme, IBM Plex Mono / Space Grotesk typography                                                                                                                                                                 | `html.templ`                |
| **Header with metadata**    | Container ID, schema version, export timestamp                                                                                                                                                                                    | `html.templ`                |
| **Stats cards**             | Services, scopes, events, dependencies, total build, errors, health-check summary                                                                                                                                                 | `html.templ`                |
| **Lifecycle waveform**      | Event timeline strip colored by event type, height scaled by duration                                                                                                                                                             | `html.templ`                |
| **Services table**          | Columns for service, type, scope, status, order, invocations, build/shutdown ms, deps, dependents, health                                                                                                                         | `html.templ`                |
| **Scopes tree**             | Collapsible tree with service counts and provider icons                                                                                                                                                                           | `html.templ`                |
| **Dependency graph**        | Sugiyama layered DAG layout with barycenter crossing reduction, cubic Bézier edges, pan/zoom/fit, click-to-highlight, mouse wheel and touch support                                                                               | `html.templ`                |
| **Timeline tab**            | Dual horizontal bar chart of build vs shutdown durations                                                                                                                                                                          | `html.templ`                |
| **Events table**            | Event list with sequence, time, type badge, provider badge, phase, scope, service, duration, error                                                                                                                                | `html.templ`                |
| **Live service search**     | Debounced filter on service name/scope/type                                                                                                                                                                                       | `html.templ`                |
| **Event type filter chips** | All / registration / invocation / shutdown / health_check buttons                                                                                                                                                                 | `html.templ`                |
| **Keyboard navigation**     | Skip link, ARIA tablist (Arrow/Home/End + roving tabindex), `?` help dialog with focus trap + restoration, `/` focus search, `e` toggle errors-only, Esc closes overlays, Enter/Space on sortable column headers with `aria-sort` | `html.templ`                |
| **Pagination**              | "Show first N" with "Show all" button for services (50) and events (100)                                                                                                                                                          | `html.templ`                |
| **Error tooltips**          | Click error status badges to show the full error message                                                                                                                                                                          | `html.templ`                |
| **Type metadata injection** | `BuildTypeMetadata()` JSON is injected into the page as the single source of truth for icons and labels                                                                                                                           | `metadata.go`, `html.templ` |
| **Responsive layout**       | Mobile-friendly padding, media queries, reduced-motion support                                                                                                                                                                    | `html.templ`                |
| **Content Security Policy** | CSP meta tag with `base-uri 'none'`, `frame-ancestors 'none'`, and inline style/script allowances                                                                                                                                 | `html.templ`                |
| **XSS hardening**           | All user-controlled strings escaped via `esc()` before DOM insertion                                                                                                                                                              | `html.templ`                |

### Type Safety

| Feature                               | Description                                                                                                                                               | Verified                                                           |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| **Typed identifiers**                 | `ContainerID`, `ScopeID`, `ServiceName` are distinct named string types throughout the entire codebase                                                    | `types.go`, all signatures                                         |
| **IO-boundary `string()` convention** | External library calls (go-output, csv, fmt) wrap typed values with `string()` at the IO boundary                                                         | `csv.go`, `d2.go`, `diagram.go`                                    |
| **ServiceInfo domain split**          | 19-field struct split into four embedded structs: `ServiceIdentity`/`ServiceLifecycle`/`ServiceHealth`/`ServiceGraph`. JSON stays flat via embedding      | `service.go`                                                       |
| **Service status derivation**         | `ServiceInfo.DeriveStatus()` is the canonical entry point for computing status from underlying fields                                                     | `service.go`                                                       |
| **Exported sentinel errors**          | 11 exported sentinels (e.g. `ErrContainerIDPathSep`, `ErrReplayValidationFailed`, `ErrReport*`) matchable via `errors.Is` for programmatic error handling | `plugin.go`, `report.go`, `loader.go`, `migration.go`, `replay.go` |

### Live Dashboard (`live/` Sub-Package)

| Feature                          | Description                                                                                                                                                                                                            | Verified                                           |
| -------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| **SSE event hub**                | `Hub` is a facade over `sse.Broadcaster[sse.Event]`, broadcasting `auditlog.Event` to all connected SSE clients with 128-event buffer per client. Events are tagged with `EventID` for reconnection replay             | `live/hub.go`                                      |
| **HTTP server with prefix**      | `Server` serves dashboard, report, SSE stream, health, and export endpoints. Configurable route prefix (default `/debug/di`)                                                                                           | `live/server.go`                                   |
| **Interactive HTML dashboard**   | Self-contained 5-tab dashboard (Services/Scopes/Graph/Timeline/Events) with SSE client, reconnection, incremental rendering                                                                                            | `live/dashboard.go`, `live/dashboard.js`           |
| **CORS support**                 | Configurable `CORSAllowedOrigins` (default `*`) adds CORS headers and OPTIONS preflight to all API endpoints                                                                                                           | `live/server.go`; `TestServer_CORSHeaders`         |
| **Export endpoints**             | `GET {prefix}/api/export/ndjson` and `GET {prefix}/api/export/html` serve downloadable snapshots with `Content-Disposition`                                                                                            | `live/server.go`; `TestServer_ExportEndpoints`     |
| **Dashboard export buttons**     | Three buttons (JSON/NDJSON/HTML) in dashboard header trigger client-side downloads via the export API                                                                                                                  | `live/dashboard.go`, `live/dashboard.js`           |
| **Pagination**                   | Services table paginates at 50 rows, events at 100, with "Show all" buttons. Search and filter bypass pagination                                                                                                       | `live/dashboard.js`                                |
| **Scope tree tab**               | Recursive scope-tree renderer showing nested scopes, service counts, and provider-type icons                                                                                                                           | `live/dashboard.js`                                |
| **Warm amber CSS design**        | Full CSS design system matching the static HTML dashboard aesthetic (phosphor amber on dark charcoal)                                                                                                                  | `live/base_css.go`                                 |
| **Graceful shutdown**            | `Server.Shutdown(ctx)` stops HTTP server then drains broadcaster buffers before close                                                                                                                                  | `live/server.go`; `TestServer_GracefulShutdown`    |
| **Keyboard navigation**          | Skip link, ARIA tablist (Arrow/Home/End + roving tabindex), `?` help dialog with focus trap + restoration, `/` focus search, Esc closes overlays, collapsible scope tree (Enter/Space), `aria-pressed` on filter chips | `live/dashboard.js`                                |
| **Health endpoint**              | `GET {prefix}/api/health` returns uptime, connected client count, total events, dropped events, broadcaster draining status, and buffer size                                                                           | `live/server.go`; `TestServer_HealthEndpoint`      |
| **`http.Handler` compatibility** | `Server` implements `ServeHTTP` for `httptest` compatibility and embedding in existing mux chains                                                                                                                      | `live/server.go`; `TestServer_HandleSSE_NoFlusher` |
| **SSE connection lifecycle**     | `sse.Stream` manages headers, flush, heartbeat, disconnect detection, and write serialization. `SendJSON` eliminates manual marshal+write+flush boilerplate                                                            | `live/server.go`                                   |
| **SSE reconnection replay**      | Reconnecting clients with `Last-Event-ID` receive missed events via `sse.Replay` + `eventStore` adapter over `plugin.Events()`                                                                                         | `live/replay.go`; `TestServer_SSE_ReconnectReplay` |
| **SSE ring buffer**              | In-memory ring buffer stores recent events for reconnection replay. `ReplayBufferSize` config (default 256), `EventStore()` and `BufferedEventCount()` on `Hub`                                                       | `live/hub.go`, `live/server.go`                    |
| **go-sse full adoption**         | Uses `Stream`, `Broadcaster[T]`, `Replay`, `EventStore`, `Shutdown`, `Health` from `go-sse` v0.4.0. No hand-rolled fan-out or connection management code remains                                                       | `live/hub.go`, `live/server.go`, `live/replay.go`  |
| **Live demo application**        | `live/demo/main.go` registers services with delays, invokes them, runs health checks, serves dashboard until Ctrl+C                                                                                                    | `live/demo/main.go`                                |
| **Example `--live` flag**        | `go run ./example --live` starts the dashboard alongside the ride-sharing demo, registering 20 services across 4 scopes                                                                                                | `example/main.go`                                  |

### Health Probes (Extracted to [go-health](https://github.com/larsartmann/go-health))

The health-probe SDK has been extracted to its own standalone project: **[github.com/larsartmann/go-health](https://github.com/larsartmann/go-health)**. It depends only on `samber/do/v2` — no transitive dependency on auditlog's heavier stack (go-output, go-sse, templ, etc.). `auditlog.Plugin` satisfies go-health's `HealthRecorder` interface implicitly.

### Shared Module Delegation

| Feature                     | Description                                                                                                                                                                                                                                                        | Verified              |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------- |
| **go-ndjson loader**        | `LoadReport` and format detection delegate to `go-ndjson/loader`. Local `loader.go` re-exports the public API. go-ndjson uses stdlib `encoding/json` (migrated from json/v2)                                                                                       | `loader.go`, `go.mod` |
| **go-ndjson reader/writer** | `ReadEvents` and NDJSON writing delegate to the `go-ndjson` module. Local `ndjson.go` re-exports sentinels and types. **Note**: `GOEXPERIMENT=jsonv2` is required at build time — the requirement comes from `go-output` (via `go-branded-id`), not from go-ndjson | `ndjson.go`, `go.mod` |

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
| **Health-check report fields**   | `Report.HealthCheckSucceeded`, `HealthCheckedCount`                                                   | `report.go`      |
| **Health-check service fields**  | `ServiceInfo.LastHealthCheckAt`, `HealthCheckError`, `HealthCheckCount`, `HasHealthError()`           | `service.go`     |
| **Unhealthy service lookup**     | `Report.UnhealthyServices()`                                                                          | `report.go`      |

### Testing / Infrastructure

| Feature                       | Description                                                                                                                                                                                                                              | Verified                                                |
| ----------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| **GitHub Actions CI**         | `go vet`, `go build`, race-detector tests, golangci-lint, govulncheck, generated-code drift checks, goreleaser config check. All actions pinned to commit SHAs for supply-chain security.                                                                         | `.github/workflows/ci.yml`                              |
| **Dependabot**                | Automated dependency updates for gomod, github-actions, and pnpm ecosystems                                                                                             | `.github/dependabot.yml`                               |
| **Goreleaser**                | Release automation for CLI binary (linux/darwin amd64/arm64) with `RELEASE.md` process documentation                                                                    | `.goreleaser.yml`, `RELEASE.md`                        |
| **Exported testhelpers**      | `testhelpers/` package (moved from `internal/`) enables downstream integration testing                                                                                  | `testhelpers/`                                          |
| **golangci-lint config**      | `.golangci.yml` defines lint rules for the project (109 linters)                                                                                                                                                                         | `.golangci.yml`                                         |
| **Generated-code check**      | CI runs `go generate ./...` and fails on drift, ensuring `html_templ.go` stays in sync                                                                                                                                                   | `.github/workflows/ci.yml`                              |
| **templ code generation**     | `//go:generate go tool templ generate` in `html.go` produces `html_templ.go`                                                                                                                                                             | `html.go`, `html_templ.go`                              |
| **Fuzz tests**                | 5 targets: HTML XSS (service names, error messages, dep chains), `MigrateReport` integrity, Mermaid/PlantUML special chars, filter-option combinations, NDJSON parsing resilience                                                        | `fuzz_test.go`, `filter_fuzz_test.go`, `replay_test.go` |
| **Benchmark tests**           | 12 performance benchmarks for hot paths (invocation, disabled, registration, concurrent, BuildReport at 50/100/500 services, events copy, health check)                                                                                  | `benchmarks_test.go`                                    |
| **Example tests**             | 8 runnable `Example*` functions for pkg.go.dev                                                                                                                                                                                           | `example_test.go`                                       |
| **Defensive-copy accessors**  | `Plugin.Events()` and `Recorder.Events()` return copied slices; `EventsCount()` avoids copying                                                                                                                                           | `plugin.go`, `recorder.go`                              |
| **Dropped-event counter**     | `Plugin.DroppedEventCount()` / `Recorder.DroppedEventCount()`                                                                                                                                                                            | `plugin.go`, `recorder.go`                              |
| **Test parallelism**          | 311 `t.Parallel()` calls (~97% of eligible tests); only `t.Setenv()` env-var tests run sequentially                                                                                                                                      | all `*_test.go`                                         |
| **Type metadata assertions**  | `TestBuildTypeMetadata` directly asserts provider/status icons, labels, and colors                                                                                                                                                       | `metadata_test.go`                                      |
| **live/ sub-package tests**   | 35 tests covering dashboard HTML, health, report, 404, SSE snapshot/live/complete/fan-out/heartbeat, graceful shutdown, client count, server lifecycle, nil-plugin edge cases, CORS, export endpoints, hub lifecycle, buffer overflow    | `live/server_test.go`                                   |
| **health/ sub-package tests** | 33 tests + 4 examples + 4 benchmarks covering liveness (dep-free), readiness (critical/non-critical/shutdown/cache/live fallback), startup (latch), Evaluate (classify/warn), routes, GET-only enforcement, audit integration, lifecycle | `health/probe_test.go`, `health/example_test.go`        |

---

## PARTIALLY FUNCTIONAL

| Feature                       | Status     | Note                                                                            |
| ----------------------------- | ---------- | ------------------------------------------------------------------------------- |
| Shared CSS between dashboards | Drift risk | Static templ and live dashboards have separate CSS that can drift               |
| Coverage gate margin thin     | Headroom   | 94.1% vs 94% threshold; adding code without tests risks dropping below the gate |

---

## WORTH CONSIDERING

| Feature                                     | Why Not Now                                                                                                                  |
| ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| **Property-based testing**                  | `rapid`/`gopter` tests for filter round-trips (`Diff` and `MigrateReport` already covered)                                   |
| **NDJSON/loader extraction from go-ndjson** | Currently delegated to `go-ndjson`; further extraction (consolidating all JSON I/O) blocked by json/v1 vs json/v2 divergence |

---

_Last verified against the codebase on 2026-08-07. Coverage: 94.1% combined (root 95.0%, live 89.7%)._

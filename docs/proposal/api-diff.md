# API Diff: `go1.23-compat` vs `master`

Point-in-time: 2026-09-04, branch @ `303a280` vs master @ `59bc651`.
Purpose: input for the merge proposal — everything a reviewer needs to judge
what changes for a consumer moving from the auditlog module line to this line.

## Unchanged (the vast majority)

The entire core recording and reporting surface is byte-identical:

- `New(Config) (*Plugin, error)`, `Plugin.Opts()`, `Enable()`, `SetOnEvent()`
- `Plugin.Report()`, `Events()`, `EventsCount()`, `DroppedEventCount()`
- `Event`, `ServiceRef`, `ServiceInfo` (+ its four embedded structs), `ScopeNode`
- All enum types and their constants (`EventType*`, `Phase*`, `ProviderType*`,
  `ServiceStatus*`) including their `Label()/Icon()/Color()` methods
- `Report.Validate()` and every query method (`ServiceByName`, `EventsByType`,
  `FailedServices`, `UnhealthyServices`, …)
- `Report.Diff` / `DiffResult` / `ServiceDiff` incl. `AddedDeps`/`RemovedDeps`
- Filters: `Filtered`, every `With*` option
- `ReplayEvents`, `ReadEvents`, `LoadReport`, `MigrateReport` (same sentinels,
  same messages), `NDJSONStreamer`, `MultiWriter`, `RunID`
- JSON Schema generation (`cmd/genschema`, `schema/report.schema.json`, `0.3.0`)
- The `auditlog` CLI (`cmd/auditlog`: info/convert/diff/validate/schema)
- Health checks: `RecordHealthCheck`, `RecordHealthCheckWithContext`,
  `ResolveServiceScope`
- Diagram exports `WriteMermaid/PlantUML/DOT/D2` (+ `ExportTo*`, `*String`)
  — same signatures, same output bytes for identical reports (stdlib port of
  the go-output renderers), `WithDirection` kept
- Tree exports `WriteTree`/`WriteHTMLTree`
- `WriteReportJSON`, `WriteReportCSV/TSV`, `ExportToFile`,
  `ExportEventsToNDJSON`, `ExportFilteredToFile`, `WriteEventsNDJSON`
- `DesignTokensCSS`, `SharedComponentCSS`, `BuildTypeMetadata`, `SchemaVersion`
- All exported sentinel errors (`ErrContainerIDPathSep`, `ErrReport*`, …) —
  identities unchanged; on this line they are matched with `errors.Is` and are
  not additionally registered into `go-error-family` families (see below)

## Changed

| Surface | master | go1.23-compat | Migration |
| --- | --- | --- | --- |
| Table formats | `output.Format` (go-output type), 16 formats | `TableFormat` string type (`"table"`, `"json"`, `"csv"`, `"tsv"`, `"markdown"`); other 11 formats return `errUnsupportedTableFormat` | String literals work on both lines; typed constants rename `FormatCSV` → `TableFormatCSV` |
| Diagram direction | `WithDirection(output.Direction)` | `WithDirection(Direction)` — local type, same constants (`DirectionDown`, `DirectionRight`) | Only matters if you constructed `output.Direction` values directly |
| Table options | `WithColumns(TableColumn...)` (10 columns via go-output) | `WithColumns(TableColumn...)` retained for the columns the 5 formats render | Recheck custom column sets |
| Error classification | Sentinels additionally registered into `go-error-family` families (`classify.go`) | Sentinel identity only (`errors.Is`); family metadata absent | Only affects consumers inspecting family metadata; matching is unchanged |
| `health/` sub-package | Present (in-tree version pre-extraction) | Removed — extracted to `github.com/larsartmann/go-health` | Import go-health (same interfaces) |
| Live: `Hub.Subscribe()` | Returns `<-chan sse.Event` (external type) | Returns `<-chan sseEvent` — the element type is unexported, so consumers can range over it but cannot name it | Consumption (`for evt := range ch`) is unchanged; `Hub.OnEvent` remains the intended wiring path, so this type is opaque in practice |
| Live: `Hub.EventStore()` | Returns the `sse.EventStore` interface (for `sse.Replay`) | Renamed `Hub.ReplayStore()`, returns the concrete ring buffer | Rename only; `BufferedEventCount()` unchanged |
| Live: `Hub.Health()` | `sse.BroadcasterHealth` (external) | `broadcasterHealth` — unexported struct with exported fields `Draining`, `SubscriberCount`, `BufferSize` (master also had `Closed`) | Field access unchanged; the type name is not user-visible in practice |
| Static HTML rendering | templ (`html.templ` + generated `html_templ.go`) | `html/template` (`html.go` + `html_view.go`); same five-tab output identity, no code generation | None for consumers (API unchanged); only affects contributors |

## Toolchain & dependency deltas

| Property | master | go1.23-compat |
| --- | --- | --- |
| `go.mod` floor | 1.26.7 | **1.23** |
| `GOEXPERIMENT=jsonv2` | Required (transitive via go-output/go-ndjson) | **Not required** — consumers set no env vars |
| Runtime deps | go-output family, go-sse, go-ndjson, go-atomic-write, go-error-family, samber/do | **`github.com/samber/do/v2` only** (plus `samber/go-type-to-string` transitively) |
| Code generation | templ + genschema | genschema only (schema) |
| Toolchain motivation | Latest features | samber/do's merge floor ("1.23 means 2 years back. Seems good for a UI") |

## Deliberately NOT carried to this line

- goreleaser release job (releases remain master's job)
- `go-error-family` classification (master-only feature)
- go-output's extra 11 table formats (return a sentinel; can be added as
  hand-rolled renderers if the merge wants them)

# API Stability Promise (0.x)

> **Pre-1.0 notice.** This library is in BETA. The public API is stabilizing
> but may still change between minor releases before 1.0. This document defines
> what you can rely on and what may evolve.

## Stable API (breaking changes require a major version bump or deprecation cycle)

These surfaces are used by every consumer and follow semantic versioning within
the 0.x series:

| Surface                                                                  | Contract                                                                                                                                                                                                               |
| ------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `New(Config) (*Plugin, error)`                                           | Signature is stable. New `Config` fields may be added (zero-valued = opt-in).                                                                                                                                          |
| `Plugin.Opts() do.InjectorOpts`                                          | Stable — this is the primary integration point.                                                                                                                                                                        |
| `Plugin.Report() Report`                                                 | Stable. The `Report` struct may gain new fields but existing fields keep their JSON keys.                                                                                                                              |
| `Plugin.Events() []Event`                                                | Stable.                                                                                                                                                                                                                |
| `Plugin.EventsCount() int`                                               | Stable.                                                                                                                                                                                                                |
| `Plugin.DroppedEventCount() int64`                                       | Stable.                                                                                                                                                                                                                |
| `ExportToFile`, `ExportToHTML`, `ExportEventsToNDJSON`                   | Stable method signatures. Output format may evolve (see below).                                                                                                                                                        |
| `Plugin.RecordHealthCheck` / `RecordHealthCheckWithContext`              | Stable.                                                                                                                                                                                                                |
| `Config{Enabled, ContainerID, RunID, MaxEvents, InitialEventCapacity, OnEvent}` | All current fields are stable. New fields may be added.                                                                                                                                                          |
| Exported sentinel errors (`Err*`)                                        | Stable identity for `errors.Is` matching (e.g. `ErrContainerIDPathSep`, `ErrReport*`, `ErrReplayValidationFailed`, `ErrMigration*`, `ErrUnsupportedFormat`). The set may grow; existing sentinels keep their identity. |
| `RunID` type and auto-generation                                         | Stable. `Config.RunID` zero-value means auto-generate via `crypto/rand`. Non-zero values are respected as-is.                                                                                                      |
| `Plugin.WriteMermaid/WritePlantUML/WriteDOT/WriteD2`                     | Stable method signatures. Diagram output format may evolve between releases.                                                                                                                                       |
| `Plugin.WriteTree/WriteHTMLTree/WriteTable`                              | Stable method signatures. Output format may evolve.                                                                                                                                                                 |
| `Plugin.RecordHealthCheck/RecordHealthCheckWithContext`                  | Stable.                                                                                                                                                                                                           |

## Evolving API (may change between 0.x releases)

These surfaces are functional but their exact shape may change:

| Surface                                                              | Reason                                                                                                           |
| -------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `Report.Diff(other Report) DiffResult`                               | `DiffResult` and `ServiceDiff` field sets may grow (timing deltas added in 0.9.0).                               |
| `DiffResult.HasChanges()`                                            | New in 0.9.0. Polarity companion to the existing `IsEmpty()`.                                                   |
| `Report.WriteNDJSON`, `Report.WriteJSON`                             | Error wrapping format may change.                                                                                |
| `Report.Filtered(opts ...ReportOption)`                              | The filter option set may expand. Existing options keep their behavior.                                          |
| `Report.Write*String` (Mermaid/PlantUML/DOT/D2/HTML/Tree/Table)      | New in 0.9.0. Convenience wrappers around `Write*`; same stability as their parent methods.                      |
| `Report.WriteTable` + `TableColumn`/`WithColumns`                    | New in 0.9.0. Column set may grow; existing columns keep their indices.                                         |
| `DiagramOption` / `WithDirection`                                    | New in 0.9.0. Diagram layout options may expand.                                                                 |
| `MigrateReport(data []byte)`                                         | Handles v0.1.0 → v0.3.0. Future schema bumps add new migration logic.                                            |
| `MultiWriter` / `NewMultiWriter`                                     | New in 0.9.0. Event fan-out type. Callback ordering is stable; internals may change.                             |
| `NDJSONStreamer` / `NewNDJSONStreamer` / `CreateNDJSONStreamer`      | New in 0.9.0. Streaming NDJSON writer. Options (`WithAutoFlush`, `WithFlushInterval`, `WithStreamBufferSize`) may grow. |
| `StreamEvents(reader, validate, callback)`                           | New in 0.9.0. Callback-based NDJSON reader. Callback signature is stable.                                        |
| `Event`, `ServiceInfo`, `ServiceRef` field set                       | New fields may be added (e.g. `RunID` on `Event`/`Report` in 0.9.0). Existing JSON tags are stable.              |
| HTML report visual design                                            | The self-contained HTML output is regenerated from `html.templ` and its appearance will change between releases. |

## `live/` Sub-Package (Evolving)

The `live/` sub-package is newer than the core library. Its public API may change:

| Surface                                                            | Reason                                                                                     |
| ------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| `live.New(auditlog.Config, live.Config) (*Server, *Plugin, error)` | Convenience constructor. Signature stable, but `Config` may grow.                           |
| `live.Config{Addr, Prefix, CORSAllowedOrigins, ReplayBufferSize}`  | All current fields stable. New fields may be added.                                         |
| `live.Server.ListenAndServe()` / `Shutdown(ctx)`                   | Stable lifecycle methods.                                                                   |
| `live.Server.ServeHTTP`                                            | Stable — enables `httptest` and mux embedding.                                              |
| `live.Hub`                                                         | Evolving — internal event broadcaster. May gain metrics hooks.                              |
| `live.Hub.EventStore()` / `BufferedEventCount()`                   | New in 0.9.0. Ring buffer inspection for SSE reconnection replay.                          |

## Unstable / Internal (no stability guarantee)

- All unexported types and functions.
- The `serviceRecord`, `scopeMeta`, `svcKey` internal types.
- The `Recorder` type (construct via `New`, not directly).
- The generated `html_templ.go` file (never edit by hand).

## JSON Schema Versioning

The JSON report format has its own version (`schema_version`, currently `0.3.0`)
that is **independent** of release tags:

- Release tags: `v0.x.y` (Git/GitHub releases)
- Schema version: `0.3.0` (in the JSON `version` field)

A schema bump (e.g. `0.2.0` → `0.3.0`) does NOT require a release tag bump.
Old schemas can always be migrated forward via `MigrateReport`.

## What "breaking" means in 0.x

A **breaking change** is any of:

- Removing or renaming an exported type, function, method, or field.
- Changing a function/method signature.
- Changing the JSON tag of an existing field.

When a breaking change is necessary:

1. It is documented in `CHANGELOG.md` under a `### Breaking` section.
2. If feasible, the old surface is kept as deprecated for one release.
3. The `New()` → `(*Plugin, error)` change in v0.0.3 is an example of this policy in action.

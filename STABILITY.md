# API Stability Promise (0.x)

> **Pre-1.0 notice.** This library is in ALPHA. The public API may change
> between minor releases. This document defines what you can rely on and what
> may evolve.

## Stable API (breaking changes require a major version bump or deprecation cycle)

These surfaces are used by every consumer and follow semantic versioning within
the 0.x series:

| Surface | Contract |
|---------|----------|
| `New(Config) (*Plugin, error)` | Signature is stable. New `Config` fields may be added (zero-valued = opt-in). |
| `Plugin.Opts() do.InjectorOpts` | Stable — this is the primary integration point. |
| `Plugin.Report() Report` | Stable. The `Report` struct may gain new fields but existing fields keep their JSON keys. |
| `Plugin.Events() []Event` | Stable. |
| `Plugin.EventsCount() int` | Stable. |
| `Plugin.DroppedEventCount() int64` | Stable. |
| `ExportToFile`, `ExportToHTML`, `ExportEventsToNDJSON` | Stable method signatures. Output format may evolve (see below). |
| `Plugin.RecordHealthCheck` / `RecordHealthCheckWithContext` | Stable. |
| `Config{Enabled, ContainerID, MaxEvents, InitialEventCapacity, OnEvent}` | All current fields are stable. New fields may be added. |

## Evolving API (may change between 0.x releases)

These surfaces are functional but their exact shape may change:

| Surface | Reason |
|---------|--------|
| `Report.Diff(other Report) DiffResult` | New in unreleased. `DiffResult` and `ServiceDiff` field sets may grow. |
| `Report.WriteNDJSON`, `Report.WriteJSON` | New in unreleased. Error wrapping format may change. |
| `Report.Filtered(opts ...ReportOption)` | The filter option set may expand. Existing options keep their behavior. |
| `MigrateReport(data []byte)` | Handles v0.1.0 → v0.2.0. Future schema bumps add new migration logic. |
| `Event`, `ServiceInfo`, `ServiceRef` field set | New fields may be added. Existing JSON tags are stable. |
| HTML report visual design | The self-contained HTML output is regenerated from `html.templ` and its appearance will change between releases. |

## Unstable / Internal (no stability guarantee)

- All unexported types and functions.
- The `serviceRecord`, `scopeMeta`, `svcKey` internal types.
- The `Recorder` type (construct via `New`, not directly).
- The generated `html_templ.go` file (never edit by hand).

## JSON Schema Versioning

The JSON report format has its own version (`schema_version`, currently `0.2.0`)
that is **independent** of release tags:

- Release tags: `v0.0.x` (Git/GitHub releases)
- Schema version: `0.2.0` (in the JSON `version` field)

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

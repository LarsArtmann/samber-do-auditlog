# Domain Language

A **Unified Language** for `samber-do-auditlog` — shared across developers and AI.
Inspired by Domain-Driven Design (DDD) Ubiquitous Language.

Every term below should mean the **same thing** to everyone who reads it.

## Glossary

| Term           | Definition                                                            | Context                      |
| -------------- | --------------------------------------------------------------------- | ---------------------------- |
| Audit Log      | The complete record of DI container lifecycle events                  | What this library produces   |
| Plugin         | The top-level entry point that wires into samber/do hooks             | `auditlog.New()` creates one |
| Recorder       | The internal state machine that captures and stores events            | In-memory event capture      |
| Event          | A single timestamped observation from the DI container                | Core domain concept          |
| Service        | A named dependency registered in the DI container                     | samber/do service            |
| Scope          | A hierarchical context for service isolation                          | samber/do scope              |
| Container      | The DI container (samber/do Injector)                                 | Host for services            |
| Invocation     | The act of resolving/creating a service instance                      | Service lifecycle phase      |
| Registration   | The act of declaring a service provider to the container              | Service lifecycle phase      |
| Shutdown       | The act of cleaning up a service when the container shuts down        | Service lifecycle phase      |
| Phase          | Whether an event is the start (before) or end (after) of an operation | Event attribute              |
| Dependency     | A service that another service needs to function                      | A→B relationship             |
| Dependent      | A service that depends on another service                             | Reverse of dependency        |
| Scope Tree     | The hierarchical structure of scopes in the container                 | Visualization concept        |
| Health Check   | A diagnostic check that verifies a service is operational             | Service lifecycle phase      |
| Provider Type  | How a service was registered: lazy, eager, transient, or alias        | Service metadata             |
| Service Status | Computed lifecycle state: registered, active, invocation_error, etc.  | Service metadata             |
| Capability     | Whether a service implements Healthchecker or Shutdowner interfaces   | Service metadata             |
| Report         | A consolidated snapshot of all captured events and service metadata   | Export output                |
| Schema Version | The version of the report data format                                 | Forward compatibility        |

## Entities

| Term        | Definition                                                                           | Context                    |
| ----------- | ------------------------------------------------------------------------------------ | -------------------------- |
| Plugin      | The user-facing object that wraps a Recorder and provides export methods             | Created by `New()`         |
| Recorder    | The internal state machine: event capture, dependency inference, aggregation         | Created by `NewRecorder()` |
| Event       | A single lifecycle observation with sequence, timestamp, type, phase, scope, service | Immutable after creation   |
| ServiceInfo | Aggregated data for one service across its entire lifecycle                          | Computed at report time    |
| ScopeNode   | A node in the scope hierarchy tree                                                   | Computed at report time    |

## Value Objects

| Term         | Definition                                                                           | Context                |
| ------------ | ------------------------------------------------------------------------------------ | ---------------------- |
| Config       | Plugin configuration: Enabled, ContainerID, OnEvent, MaxEvents, InitialEventCapacity | Input to `New()`       |
| EventType    | Enum: registration, invocation, shutdown, health_check                               | Event categorization   |
| Phase        | Enum: before, after                                                                  | Event timing           |
| ServiceRef   | A lightweight reference to a service in a specific scope                             | Dependency graph edges |
| ReportOption | A functional option for filtering reports                                            | Filter input           |
| Report       | A complete, self-contained snapshot of all audit data                                | Export payload         |

## Events

| Term               | Definition                                                   | Context                             |
| ------------------ | ------------------------------------------------------------ | ----------------------------------- |
| Registration Event | Fired when a service provider is registered to the container | Before and after                    |
| Invocation Event   | Fired when a service is resolved/created                     | Before and after, includes duration |
| Shutdown Event     | Fired when a service is cleaned up during container shutdown | Before and after, includes duration |
| Health Check Event | Fired after a health check is performed on a service         | After only, no duration             |

## Commands

| Term                         | Definition                                              | Context             |
| ---------------------------- | ------------------------------------------------------- | ------------------- |
| New                          | Create a new audit log plugin                           | Entry point         |
| Opts                         | Get DI container hook options                           | Wire into samber/do |
| Report                       | Get a snapshot of all captured data                     | Read operation      |
| ReportFiltered               | Get a filtered snapshot of captured data                | Read operation      |
| WriteReportJSON              | Write indented JSON report to an io.Writer              | Export command      |
| WriteEventsNDJSON            | Write NDJSON event stream to an io.Writer               | Export command      |
| WriteHTML                    | Write self-contained HTML visualization to an io.Writer | Export command      |
| ExportToFile                 | Write JSON report to a file path                        | Export command      |
| ExportEventsToNDJSON         | Write NDJSON event stream to a file path                | Export command      |
| ExportToHTML                 | Write self-contained HTML visualization to a file path  | Export command      |
| ExportFilteredToFile         | Write a filtered JSON report to a file path             | Export command      |
| RecordHealthCheck            | Wrap injector health check with audit events            | Health command      |
| RecordHealthCheckWithContext | Same as RecordHealthCheck with context support          | Health command      |
| Events                       | Defensive copy of captured events                       | Read operation      |
| EventsCount                  | Count of captured events                                | Read operation      |
| DroppedEventCount            | Count of events dropped due to MaxEvents                | Read operation      |
| MigrateReport                | Normalize/repair any JSON report to the current schema (upgrades v0.1.0 and re-derives all denormalized fields for any input version) | Migration command   |
| Filtered                     | Apply functional filter options to a Report             | Query command       |
| Validate                     | Check report denormalized counts match actual data      | Validation command  |
| Index                        | Build O(1) lookup index for report queries              | Query command       |
| Diff                         | Compare two reports structurally                        | Query command       |
| WriteJSON                    | Write indented JSON report to an io.Writer              | Export command      |
| WriteNDJSON                  | Write NDJSON event stream to an io.Writer               | Export command      |
| WriteMermaid                 | Export dependency graph as Mermaid flowchart            | Export command      |
| WritePlantUML                | Export dependency graph as PlantUML component diagram   | Export command      |
| WriteDOT                     | Export dependency graph as Graphviz DOT digraph         | Export command      |
| WriteD2                      | Export dependency graph as D2 diagram                   | Export command      |
| WriteCSV                     | Export all services as comma-separated values           | Export command      |
| WriteTSV                     | Export all services as tab-separated values             | Export command      |
| WriteTree                    | Export dependency DAG as ASCII tree                     | Export command      |
| WriteHTMLTree                | Export dependency DAG as HTML nested-list tree          | Export command      |
| WriteTable                   | Export service summary table in 16+ formats             | Export command      |
| ReadEvents                   | Read NDJSON event stream back into memory               | Import command      |
| ReplayEvents                 | Reconstruct a Report from a flat event stream           | Import command      |
| LoadReport                   | Auto-detect format and load a report from file          | Import command      |
| NewReport                    | Construct a validated Report from core data             | Constructor         |
| JSONSchema                   | Return the canonical JSON Schema for the report format  | Schema command      |

## Bounded Contexts

| Context       | Description                                                                                                                                                                                                  |
| ------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Capture       | Hook callbacks, event recording, dependency inference (hooks.go, recorder.go)                                                                                                                                |
| Aggregation   | Building reports, scope trees, dependency graphs (report_builder.go)                                                                                                                                         |
| Export        | Formatting and writing: JSON, NDJSON, CSV, TSV, HTML, Mermaid, PlantUML, DOT, D2, Tree, Table (plugin.go, html.go, report.go, csv.go, mermaid.go, plantuml.go, dot.go, d2.go, tree.go, table.go, diagram.go) |
| Configuration | Plugin setup, environment variable handling (plugin.go)                                                                                                                                                      |

---

> **How to use this file:**
>
> - Keep terms concise — one clear sentence per definition
> - Update when new domain concepts emerge
> - Use these terms consistently in code, docs, and conversations
> - When in doubt about a word's meaning, check here first

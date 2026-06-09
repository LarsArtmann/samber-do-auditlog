# do-auditlog

An audit-log plugin for [samber/do v2](https://github.com/samber/do) that tracks every registration, invocation, and shutdown with timestamps, dependency resolution, and build durations. Built for debugging and visualization.

## Features

- **Zero-config hooks** — drop-in `*do.InjectorOpts` producer
- **Dependency graph inference** — tracks which services resolved which, without access to do's internal DAG
- **Reverse dependencies** — every service knows which services depend on it (`Dependents`)
- **Scope tree capture** — records the full scope hierarchy with per-scope service lists
- **Timing data** — build duration, invocation count, invocation order
- **Machine-readable exports** — JSON report, NDJSON event stream, or self-contained HTML
- **Minimal overhead** — ~1μs per cached invocation (see benchmarks)
- **Debug mode friendly** — toggle on/off without changing container wiring
- **Zero external dependencies** — only `samber/do/v2`

## Install

```bash
go get github.com/larsartmann/do-auditlog
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/larsartmann/do-auditlog"
    "github.com/samber/do/v2"
)

func main() {
    plugin := auditlog.New(auditlog.Config{
        Enabled:     true,
        ContainerID: "my-app",
    })

    injector := do.NewWithOpts(plugin.Opts())

    do.Provide(injector, func(i do.Injector) (*MyService, error) {
        return &MyService{}, nil
    })
    svc := do.MustInvoke[*MyService](injector)
    _ = svc

    // Export as JSON, NDJSON, or HTML
    plugin.ExportToFile("audit.json")
    plugin.ExportEventsToNDJSON("events.ndjson")
    plugin.ExportToHTML("audit.html")
}
```

## Export Formats

### JSON Report (`ExportToFile` / `WriteReportJSON`)

A single JSON file with complete event timeline, per-service summaries, and scope tree.

```json
{
  "container_id": "demo-app",
  "service_count": 3,
  "services": [
    {
      "service_name": "*main.UserService",
      "service_type": "lazy",
      "invocation_count": 1,
      "invocation_order": 2,
      "build_duration_ms": 9.079,
      "dependencies": [
        {"scope_name": "[root]", "service_name": "*main.Database"},
        {"scope_name": "[root]", "service_name": "*main.Cache"}
      ],
      "dependents": [
        {"scope_name": "[root]", "service_name": "*main.HTTPServer"}
      ]
    }
  ],
  "scope_tree": {
    "id": "...",
    "name": "[root]",
    "services": ["*main.Config", "*main.Database", "*main.Cache"],
    "children": []
  }
}
```

### NDJSON Event Stream (`ExportEventsToNDJSON` / `WriteEventsNDJSON`)

Each line is a self-contained JSON object — ideal for streaming ingestion and log aggregators.

```ndjson
{"sequence":1,"timestamp":"...","event_type":"registration","phase":"before","scope_name":"[root]","service_name":"*main.Config"}
{"sequence":2,"timestamp":"...","event_type":"registration","phase":"after","scope_name":"[root]","service_name":"*main.Config"}
{"sequence":3,"timestamp":"...","event_type":"invocation","phase":"before","scope_name":"[root]","service_name":"*main.Database"}
{"sequence":4,"timestamp":"...","event_type":"invocation","phase":"after","scope_name":"[root]","service_name":"*main.Database","duration_ms":5.196}
```

### HTML Visualization (`ExportToHTML` / `WriteHTML`)

A self-contained dark-themed HTML page with:

- **Stats cards** — services, events, scopes, dependencies
- **Services table** — sortable with type tags, invocation counts, build durations
- **Dependency graph** — force-directed SVG with color-coded service types
- **Timeline** — horizontal bars showing relative build durations
- **Events table** — full chronological event log

No external JS/CSS dependencies. Works offline.

## API

| Method | Description |
|--------|-------------|
| `New(config Config) *Plugin` | Create plugin. `ContainerID` defaults to `"default"`. |
| `Opts() *do.InjectorOpts` | Pre-configured hooks for `do.NewWithOpts`. |
| `Report() Report` | In-memory snapshot. |
| `WriteReportJSON(w io.Writer) error` | Write report as indented JSON to any writer. |
| `WriteEventsNDJSON(w io.Writer) error` | Write events as NDJSON to any writer. |
| `WriteHTML(w io.Writer) error` | Write self-contained HTML visualization to any writer. |
| `ExportToFile(path string) error` | Write JSON report to file. |
| `ExportEventsToNDJSON(path string) error` | Write NDJSON events to file. |
| `ExportToHTML(path string) error` | Write HTML visualization to file. |
| `Events() []Event` | Defensive copy of raw events. |

## How Dependency Tracking Works

do-auditlog uses a lightweight invocation stack:

1. When `HookBeforeInvocation` fires for service A, A is pushed onto a stack.
2. If A's provider calls `do.MustInvoke[B](i)`, `HookBeforeInvocation` fires for B while A is still on the stack.
3. The plugin records that **A depends on B**.
4. When `HookAfterInvocation` fires, the service is popped from the stack.

This correctly reconstructs the dependency graph even for cached services and across scopes. The reverse graph (`Dependents`) is computed at report time.

## Performance

Benchmarks on AMD Ryzen AI MAX+ 395:

| Scenario | ns/op | allocs | overhead |
|----------|-------|--------|----------|
| Cached invocation (enabled) | ~1,164 | 7 | baseline |
| Cached invocation (disabled) | ~223 | 4 | — |
| Registration (enabled) | ~20,571 | 49 | — |

Overhead: ~1μs per cached invocation with audit logging enabled.

## License

MIT

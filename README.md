# do-auditlog

An audit-log plugin for [samber/do v2](https://github.com/samber/do) that tracks every registration, invocation, and shutdown with timestamps and dependency resolution. Built for debugging and visualization.

## Features

- **Zero-config hooks** — drop-in `*do.InjectorOpts` producer
- **Dependency graph inference** — tracks which services resolved which, without access to do's internal DAG
- **Scope tree capture** — records the full scope hierarchy (root → children)
- **Timing data** — build duration, invocation count, shutdown times
- **Machine-readable exports** — indented JSON report or NDJSON event stream
- **Minimal overhead** — in-memory capture, synchronous hooks do zero I/O
- **Debug mode friendly** — toggle on/off without changing container wiring

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
    // 1. Create the plugin
    plugin := auditlog.New(auditlog.Config{
        Enabled:     true,
        ContainerID: "my-app",
    })

    // 2. Wire it into the DI container
    injector := do.NewWithOpts(plugin.Opts())

    // 3. Register and use services as usual
    do.Provide(injector, func(i do.Injector) (*MyService, error) {
        return &MyService{}, nil
    })
    svc := do.MustInvoke[*MyService](injector)
    _ = svc

    // 4. Export the audit log
    if err := plugin.ExportToFile("audit.json"); err != nil {
        log.Fatal(err)
    }
}
```

## Export Formats

### Full Report (`ExportToFile`)

A single JSON file containing:

- Complete event timeline
- Per-service summaries with dependencies
- Scope tree for visualization

```json
{
  "container_id": "demo-app",
  "event_count": 34,
  "service_count": 5,
  "services": [
    {
      "service_name": "*main.Database",
      "invocation_count": 1,
      "build_duration_ms": 5.196,
      "dependencies": ["[root]/*main.Config"]
    }
  ],
  "scope_tree": {
    "id": "...",
    "name": "[root]",
    "children": []
  }
}
```

### NDJSON Event Stream (`ExportEventsToNDJSON`)

Each line is a self-contained JSON object — ideal for streaming ingestion, log aggregators, or line-by-line parsing in a visualizer.

```ndjson
{"id":"...","timestamp":"...","event_type":"invocation","phase":"before","scope_id":"...","service_name":"*main.Database"}
{"id":"...","timestamp":"...","event_type":"invocation","phase":"after","scope_id":"...","service_name":"*main.Database","duration_ms":5.196}
```

## API

### `auditlog.New(config Config) *Plugin`

Creates a new plugin instance. `ContainerID` defaults to `"default"` if empty.

### `(*Plugin) Opts() *do.InjectorOpts`

Returns a pre-configured `InjectorOpts` with all six lifecycle hooks wired to the recorder. Pass this directly to `do.NewWithOpts`.

### `(*Plugin) Report() Report`

Returns the in-memory `Report` struct without writing to disk.

### `(*Plugin) ExportToFile(path string) error`

Marshals the full `Report` as indented JSON.

### `(*Plugin) ExportEventsToNDJSON(path string) error`

Writes every captured event as NDJSON (newline-delimited JSON).

### `(*Plugin) Events() []Event`

Returns a defensive copy of the raw event slice.

## How Dependency Tracking Works

do-auditlog does **not** have access to samber/do's internal DAG. Instead it uses a lightweight invocation stack:

1. When `HookBeforeInvocation` fires for service A, A is pushed onto a stack.
2. If A's provider calls `do.MustInvoke[B](i)`, `HookBeforeInvocation` fires for B while A is still on the stack.
3. The plugin records that **A depends on B**.
4. When `HookAfterInvocation` fires, the service is popped from the stack.

This correctly reconstructs the dependency graph even for cached services and across scopes.

## Performance

- Hooks append to a mutex-protected slice — nanoseconds to microseconds per event.
- No file I/O happens during container operation.
- Export is a single `json.Marshal` or line iteration — pay the cost only when you need the data.

## Visualization Ideas

Because the output is plain JSON/NDJSON, you can build:

- **Timeline view** — plot every `before`/`after` event on a time axis
- **Dependency graph** — use the `services[].dependencies` array to draw a directed graph (D3, Cytoscape.js)
- **Scope tree** — render the `scope_tree` as a collapsible tree
- **Flame chart** — reconstruct the invocation stack from event order and durations

## License

MIT

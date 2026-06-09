<div align="center">

# 🔍 do-auditlog

**Audit-log plugin for [samber/do v2](https://github.com/samber/do)**

Track every service registration, invocation, and shutdown.
Get timestamps, build durations, dependency graphs, and scope trees.
Export as JSON, NDJSON, or a self-contained HTML visualization.

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/samber-do-auditlog.svg)](https://pkg.go.dev/github.com/larsartmann/samber-do-auditlog)
[![Go Report Card](https://goreportcard.com/badge/github.com/larsartmann/samber-do-auditlog)](https://goreportcard.com/report/github.com/larsartmann/samber-do-auditlog)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

</div>

---

> [!CAUTION]
> ## 🚧 ALPHA — WORK IN PROGRESS 🚧
>
> This project is in **early development**. The API may change at any time without notice.
>
> **No guarantees** are made regarding:
> - Backward compatibility between versions
> - Stability of exported types and functions
> - Correctness in all edge cases
>
> **Use at your own risk.** Pin your dependency to a specific commit hash if you depend on this in production.
>
> Feedback, bug reports, and breaking-change requests are very welcome in [Issues](https://github.com/larsartmann/samber-do-auditlog/issues).

---

## Why?

samber/do v2 has lifecycle hooks but no built-in observability. You get hooks, but no recorder, no export, no visualization.

**do-auditlog** wires into those hooks in one line and gives you:

- What services exist, when they were created, and how long they took to build
- Which services depend on which — forward and reverse
- The scope tree with per-scope service lists
- A complete chronological event stream
- A self-contained HTML page you can open in any browser to explore your DI container

## Features

| Feature | Description |
|---------|-------------|
| **Drop-in setup** | `do.NewWithOpts(plugin.Opts())` — one line, zero config |
| **Dependency graph** | Infers which service resolved which, without accessing do's internal DAG |
| **Reverse dependencies** | Every service knows who depends on it |
| **Scope tree** | Full hierarchy with per-scope service lists |
| **Timing** | Build duration, invocation count, invocation order |
| **3 export formats** | JSON report · NDJSON stream · self-contained HTML |
| **~1μs overhead** | In-memory capture, no I/O during container operation |
| **Toggle on/off** | `Enabled: false` → zero hooks, zero cost |
| **Zero extra deps** | Only depends on `samber/do/v2` |

## Install

```bash
go get github.com/larsartmann/samber-do-auditlog
```

Requires Go 1.22+ and samber/do v2.

## Quick Start

```go
package main

import (
    "log"

    "github.com/larsartmann/samber-do-auditlog"
    "github.com/samber/do/v2"
)

func main() {
    // 1. Create the plugin
    plugin := auditlog.New(auditlog.Config{
        Enabled:     true,           // flip to false in production
        ContainerID: "my-app",
    })

    // 2. Pass options to the DI container
    injector := do.NewWithOpts(plugin.Opts())

    // 3. Register and use services as usual
    do.Provide(injector, func(i do.Injector) (*MyService, error) {
        return &MyService{}, nil
    })
    svc := do.MustInvoke[*MyService](injector)
    _ = svc

    // 4. Export when you're done
    plugin.ExportToFile("audit.json")              // full report
    plugin.ExportEventsToNDJSON("events.ndjson")   // streaming format
    plugin.ExportToHTML("audit.html")              // open in browser
}
```

## Export Formats

### JSON Report

Full snapshot: event timeline, service summaries, scope tree.

```json
{
  "container_id": "my-app",
  "exported_at": "2026-06-09T22:18:00Z",
  "service_count": 3,
  "event_count": 20,
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
    "name": "[root]",
    "services": ["*main.Config", "*main.Database", "*main.Cache"],
    "children": []
  }
}
```

### NDJSON Event Stream

One JSON object per line. Feed it into log aggregators, stream processors, or custom tooling.

```ndjson
{"sequence":1,"timestamp":"...","event_type":"registration","phase":"before","scope_name":"[root]","service_name":"*main.Config"}
{"sequence":2,"timestamp":"...","event_type":"registration","phase":"after","scope_name":"[root]","service_name":"*main.Config"}
{"sequence":3,"timestamp":"...","event_type":"invocation","phase":"before","scope_name":"[root]","service_name":"*main.Database"}
{"sequence":4,"timestamp":"...","event_type":"invocation","phase":"after","duration_ms":5.196,"scope_name":"[root]","service_name":"*main.Database"}
```

### HTML Visualization

A single, self-contained dark-themed HTML page. No external JS/CSS. Works offline.

**What you get:**

- **Stats cards** — services, events, scopes, dependency count
- **Services table** — name, scope, type tag, invocation order, count, build time, deps, status
- **Dependency graph** — force-directed SVG with color-coded nodes (lazy · eager · transient)
- **Timeline** — horizontal bars showing relative build durations
- **Events table** — full chronological log with sequence numbers

Open the file in any browser. No server needed.

## API Reference

| Method | Description |
|--------|-------------|
| `New(config Config) *Plugin` | Create plugin. `ContainerID` defaults to `"default"`. |
| `Opts() *do.InjectorOpts` | Hooks for `do.NewWithOpts`. No-ops when `Enabled: false`. |
| `Report() Report` | In-memory snapshot. No I/O. |
| `WriteReportJSON(w) error` | Indented JSON to any `io.Writer`. |
| `WriteEventsNDJSON(w) error` | NDJSON event stream to any `io.Writer`. |
| `WriteHTML(w) error` | Self-contained HTML visualization to any `io.Writer`. |
| `ExportToFile(path) error` | JSON report to file. |
| `ExportEventsToNDJSON(path) error` | NDJSON events to file. |
| `ExportToHTML(path) error` | HTML visualization to file. |
| `Events() []Event` | Defensive copy of raw event slice. |

## How Dependency Tracking Works

do-auditlog does **not** access samber/do's internal DAG. Instead, it uses a lightweight invocation stack:

1. `HookBeforeInvocation` fires for service A → A is pushed onto a stack
2. A's provider calls `do.MustInvoke[B](i)` → `HookBeforeInvocation` fires for B while A is still on the stack
3. The plugin records: **A depends on B**
4. `HookAfterInvocation` fires → service is popped from the stack

This correctly reconstructs the dependency graph even for:
- **Cached services** — subsequent invocations of a lazy service are near-instant but still tracked
- **Cross-scope resolution** — services inherited from parent scopes
- **Provider errors** — failed invocations are still recorded with error details

The reverse graph (`Dependents`) is computed at report time from the forward dependencies.

## Performance

Benchmarks from a real run (AMD Ryzen AI MAX+ 395):

```
BenchmarkHookOverhead_Invocation    ~1,305 ns/op    7 allocs    (enabled)
BenchmarkHookOverhead_Disabled       ~252 ns/op    4 allocs    (disabled)
BenchmarkHookOverhead_Registration  ~26,519 ns/op   49 allocs    (full container)
```

**Overhead: ~1μs per cached invocation** when enabled. Zero cost when disabled.

No file I/O happens during container operation. Export is a single `json.Marshal` or line iteration — you pay the cost only when you need the data.

## Data Model

```
Report
├── container_id        string
├── exported_at         time
├── service_count       int
├── event_count         int
├── services[]          ServiceInfo
│   ├── service_name    string
│   ├── scope_name      string
│   ├── service_type    lazy | eager | transient
│   ├── invocation_order int
│   ├── build_duration_ms float64
│   ├── dependencies[]  {scope_name, service_name}
│   └── dependents[]    {scope_name, service_name}
├── events[]            Event
│   ├── sequence        int (monotonic)
│   ├── timestamp       time
│   ├── event_type      registration | invocation | shutdown
│   ├── phase           before | after
│   ├── duration_ms     float64 (after-invocation only)
│   └── error           string (on failure only)
└── scope_tree          ScopeNode
    ├── name            string
    ├── services[]      string
    └── children[]      ScopeNode (recursive)
```

## License

[MIT](https://opensource.org/licenses/MIT)

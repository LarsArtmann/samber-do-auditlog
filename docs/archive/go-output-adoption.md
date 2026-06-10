# go-output Adoption Analysis

**Date:** 2026-06-09 · **Status:** REJECTED · **Revisit if:** users request CSV/Markdown/YAML export

## Context

Should `samber-do-auditlog` depend on [go-output](https://github.com/larsartmann/go-output) for its export formats?

## What go-output IS

A tabular data formatting library — `TableData` in, 16 formats out (JSON, CSV, Markdown, HTML, YAML, D2, Mermaid, Tree, etc.). Also provides graph/tree rendering primitives (`GraphNode`, `GraphEdge`, `TreeNode`).

## What samber-do-auditlog DOES

Captures DI lifecycle events, infers dependency graphs, and exports rich domain-specific reports: services with invocation order, build durations, error tracking, scope hierarchy, dependency graph with SVG force simulation, timeline visualization.

## Mapping: What Would Be Replaced?

| Current in auditlog                     | go-output equivalent        | Fit                                                                                      |
| --------------------------------------- | --------------------------- | ---------------------------------------------------------------------------------------- |
| `WriteReportJSON` (json.Encoder)        | `serialization.JSONWriter`  | **Overkill** — stdlib `json.Encoder` is 5 lines, already done                            |
| `WriteEventsNDJSON` (json.Encoder loop) | `serialization.JSONLWriter` | Marginal — NDJSON is trivially simple                                                    |
| `ExportToHTML` (templ template)         | `markup.HTMLRenderer`       | **Poor** — our HTML is a full interactive dashboard with tabs, SVG force graph, timeline |
| `ScopeNode` tree                        | `output.TreeNode`           | **Shape mismatch** — `ScopeNode` carries `Services []string`; `TreeNode` has `Metadata`  |
| Dependency graph (custom SVG)           | D2/Mermaid/DOT renderers    | Conceptual fit for graph text, but loses interactive SVG force simulation                |
| `Report` struct                         | `TableData`                 | **Fundamental mismatch** — `Report` has nested structs; `TableData` is `[][]string`      |

## PRO

1. **Graph exports "for free"** — Output dependency graph as D2, Mermaid, or DOT alongside JSON/HTML with ~20 lines of conversion (`ServiceInfo` → `GraphNode`/`GraphEdge`)
2. **Multi-format CLI output** — If users eventually want CSV/Markdown/YAML/TOML exports of the services table, go-output saves writing those formatters
3. **Ecosystem consistency** — Same author (`larsartmann`), same Go version (1.26.3), aligned philosophy
4. **Type-safe format enum** — `Format` with `ParseFormat` is nice for a `--format=json|csv|html|d2` CLI flag
5. **Zero-cost when unused** — Multi-module design means `go get go-output` pulls no heavy transitive deps

## CONTRA

1. **`TableData` is `[][]string` — our domain is rich structs** — Converting `ServiceInfo` (12 fields, optional pointers, nested `DependencyRef`) to `[]string` rows loses type information and requires round-trip parsing to recover. Fundamental abstraction mismatch.
2. **Our HTML is a full interactive dashboard** — Tabs, SVG force-directed graph, timeline visualization, JS-driven rendering. go-output's `HTMLRenderer` produces a static `<table>`. We'd still need templ for the dashboard, making go-output redundant for the primary use case.
3. **JSON export is already trivial** — `WriteReportJSON` is 8 lines of stdlib `json.Encoder`. Adding `go-output/serialization` saves nothing.
4. **Adds a dependency for marginal gain** — Current deps: `samber/do`, `a-h/templ`. Adding `go-output` (+ transitive: `go-branded-id`, `golang.org/x/term`) increases supply chain surface for a `[][]string` formatter we barely need.
5. **depguard must be reconfigured** — Current allowlist is `$gostd, samber, templ, samber-do-auditlog`. Adding `go-output` means opening it up.
6. **Our `ScopeNode` and `Report` types are better** — Domain-specific, carry exactly the right data, produce exactly the right output. Mapping into generic `TreeNode`/`TableData` is a lossy downgrade.
7. **YAGNI** — Zero evidence users need CSV/TSV/Markdown/YAML export of audit data. JSON + NDJSON + interactive HTML covers the real use cases (machine parsing + human inspection).

## Verdict

**Don't use go-output.** The abstraction is wrong for this domain.

- `go-output` solves: "I have tabular data and want it in 16 formats"
- `samber-do-auditlog` needs: "I have rich domain structs and want domain-specific exports"

The only plausible use — graph format exports (D2/Mermaid/DOT) — can be added in ~50 lines of targeted code when actually requested, without importing a 15-module workspace.

## Revisit Criteria

Add go-output only when a concrete user request arrives for "export the services table as CSV/Markdown/YAML" — and even then, evaluate whether `TableData` conversion is worth the dependency vs. writing a focused formatter.

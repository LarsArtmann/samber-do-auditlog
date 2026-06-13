# Status Report: Frontend Redesign & Brutal Self-Review

**Date**: 2026-06-13 11:14  
**Session Focus**: HTML frontend redesign (frontend-design skill), then brutal self-review of entire codebase  
**Branch**: master  
**Commit**: 88c3421 → (pending)

---

## a) FULLY DONE

### Frontend Redesign (This Session)

- **New palette**: Warm amber "Container Telemetry" — phosphor amber (#e8a838) on dark charcoal (#14110d). Replaces generic cold navy + sky-blue.
- **New typography**: Space Grotesk (display/body) + IBM Plex Mono (monospace). Replaces DM Sans + JetBrains Mono.
- **Signature element**: Lifecycle waveform — plots all events as colored vertical marks on a horizontal timeline strip between header and stats. Height-encoded by duration, colored by event type (amber=registration, jade=invocation, warm-gold=shutdown, teal=health, coral=error). Hover tooltips with event details.
- **All CSS variables updated**: 24 color/radius tokens replaced. All JS references inherited automatically via CSS variables.
- **Hardcoded RGBA colors updated**: Badge backgrounds, tooltip shadows, graph-info background, health-check badge color all harmonized to warm palette.
- **Reduced-motion support**: `@media (prefers-reduced-motion: reduce)` zeroes out all animations/transitions.
- **Subtle warm radial glow**: Body background has faint amber glow from top.
- **templ regenerated**: `html_templ.go` updated via `templ generate`.
- **All 140 tests pass**: 133 unit + 6 examples + 3 fuzz seeds. `go vet` clean. `go build` clean.
- **AGENTS.md updated**: HTML redesign gotcha reflects new aesthetic.

### Pre-Existing (Prior Sessions)

- **Core plugin**: Full DI lifecycle capture (registration, invocation, shutdown, health checks) with hook integration into samber/do v2.
- **5 export formats**: JSON, NDJSON, self-contained HTML, Mermaid flowchart, PlantUML component diagram.
- **Dependency graph inference**: Stack-based inference from invocation ordering.
- **Service type tracking**: lazy/eager/transient/alias auto-detected via `do.ExplainNamedService`.
- **Capability tracking**: `IsHealthchecker`/`IsShutdowner` detected via `do.ExplainInjector`.
- **Schema migration**: v0.1.0 → v0.2.0 with scope tree + service status computation.
- **Report querying**: 10 convenience methods (ServiceByName, EventsByType, FailedServices, etc.).
- **Report filtering**: 5 filter dimensions (name, type, event_type, time_range, scope).
- **Concurrency**: Single RWMutex, atomic counters, onEvent outside lock.
- **140 tests + 11 benchmarks**: ~95% statement coverage.
- **XSS safety**: 3 fuzz targets, `esc()` on all user-controlled strings.
- **Example app**: 19-feature self-checking ride-sharing demo.

---

## b) PARTIALLY DONE

### HTML Visualization

- **Visual polish**: The waveform is functional but the overall page could benefit from more intentional spacing, a stronger hero moment, and better mobile responsive treatment. The redesign changed colors and fonts but did NOT restructure the layout (tab bar, table layouts, graph container dimensions).
- **Graph tab**: SVG graph works but colors now reference warm palette CSS variables — the actual rendered colors may need verification on a real browser (no screenshot capability in this environment).
- **CSS variable consistency**: Most colors use CSS variables, but the timeline bars use inline `style="background:var(--accent)"` and `style="background:var(--warning)"` — these are correct but fragile.

### Test Coverage

- **HTML tests**: 4 tests verify structural HTML output (tabs exist, elements present, XSS safety). No tests verify the waveform rendering, CSS correctness, or visual layout.
- **Mermaid/PlantUML**: Mermaid has writer-error test; PlantUML does not.
- **Concurrency**: Concurrent invocation tested; concurrent `BuildReport` during recording is NOT.

---

## c) NOT STARTED

- **Visual regression testing**: No screenshot/snapshot tests for HTML output.
- **Mobile responsive design**: Media query reduces padding but no real mobile layout work.
- **Dark/light theme toggle**: Only dark mode exists.
- **Interactive service detail view**: No drill-down from graph/table to individual service detail panel.
- **Search across all tabs**: Only Services tab has search; Events has type filters but no text search.
- **Export comparison**: No way to diff two reports.
- **Streaming/real-time HTML**: HTML is a static snapshot; no WebSocket/SSE live view.
- **Accessibility audit**: No ARIA roles, no screen reader testing, no keyboard focus management beyond tab navigation (1-5).

---

## d) TOTALLY FUCKED UP

### Critical Issues Found

1. **`Config.Validate()` is NEVER called in `New()`** (`plugin.go:61`)

   The validation function exists, tests exist for it, the sentinel error exists — but `New()` never calls it. A user can pass `Config{ContainerID: "my/app"}` with path separators and it silently succeeds. This is a **ghost validation system**: code that looks like it works but isn't wired into the actual instantiation path.

2. **`go.sum` was modified unexpectedly**

   Running `templ generate` cleaned up stale go.sum entries (testify, go-spew, go-difflib, yaml.v3 were removed). These were transitive checksums from an older state. The build verifies fine and `go mod verify` passes, but this was an unintended side effect of the frontend redesign session.

3. **Mermaid/PlantUML are copy-paste duplicates** (~45 identical lines)

   `mermaid.go` (62 lines) and `plantuml.go` (74 lines) are near-identical files with only 4 differences (header, footer, node format, ID sanitizer). The deduplication skill was run in a prior session but apparently missed these files. The `export.go` helpers (`writeSortedLines`, `serviceLabel`, `serviceRefLabel`) were extracted but the core graph-writing loop was not.

4. **`mermaidNodeID` doesn't sanitize `*[]{}` characters**

   Service names like `*main.Database` produce Mermaid node IDs with `*` which Mermaid may misparse. PlantUML's sanitizer handles these characters; Mermaid's does not. This means the Mermaid export can produce invalid diagrams for Go type names (which almost always start with `*`).

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

| #   | Issue                                     | Impact                                                | Fix Approach                                                                                      |
| --- | ----------------------------------------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| 1   | **Wire `Validate()` into `New()`**        | High — silent acceptance of invalid configs           | Change `New()` signature to `(*Plugin, error)` or call `Validate()` and log/panic on invalid      |
| 2   | **Deduplicate Mermaid/PlantUML**          | Medium — 45 lines of copy-paste                       | Extract `writeDependencyGraph(w, graphFormat)` with format descriptor                             |
| 3   | **Fix Mermaid ID sanitization**           | Medium — invalid diagrams for `*`-prefixed names      | Port PlantUML's sanitizer or create shared `sanitizeNodeID()`                                     |
| 4   | **`ReportIndex` is a parallel query API** | Low — two ways to query same data                     | Either make `Report` use `ReportIndex` internally, or document that `Index` is the preferred path |
| 5   | **`shutdownStart` map can leak**          | Low — entries never cleaned on unmatched before/after | Add eviction or use `sync.Map` with TTL                                                           |
| 6   | **`Uptime()` uses wall-clock**            | Low — non-deterministic for exported snapshots        | Should compute from `RegisteredAt` to `ShutdownAt` or `ExportedAt`                                |

### Frontend Design

| #   | Issue                          | Impact                                                        | Fix Approach                                                                      |
| --- | ------------------------------ | ------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| 7   | **No layout restructuring**    | Medium — same tab/table structure, just recolored             | Consider card-grid for services, full-width timeline, side-panel for graph detail |
| 8   | **Waveform needs testing**     | Medium — no test verifies waveform renders                    | Add HTML test checking for `wf-event` class and `renderWaveform` function         |
| 9   | **No mobile layout**           | Low — media query only reduces padding                        | Responsive tab bar (scrollable), collapsible table columns, stacked stat cards    |
| 10  | **Graph node colors untested** | Low — SVG fill uses CSS vars, may not resolve in all browsers | Verify in real browser or use computed hex values                                 |

### Testing

| #   | Issue                                   | Impact                                        | Fix Approach                                                                                         |
| --- | --------------------------------------- | --------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| 11  | **Misnamed test**                       | Low — confusing                               | `TestPlugin_WriteReportJSONError` tests success path, rename to `TestPlugin_WriteReportJSON_Success` |
| 12  | **No PlantUML writer-error test**       | Low — asymmetric coverage                     | Add `TestWritePlantUML_WriterError`                                                                  |
| 13  | **No concurrent BuildReport test**      | Medium — RWMutex protocol untested under load | Add test with goroutines writing while reading                                                       |
| 14  | **No alias provider registration test** | Low — type method tested but not registration | Add test with `do.As` or alias provider                                                              |

### Code Quality

| #   | Issue                                             | Impact                             | Fix Approach                                          |
| --- | ------------------------------------------------- | ---------------------------------- | ----------------------------------------------------- |
| 15  | **`newServiceRecordCore` explicit nil/zero init** | Trivial — exhaustruct-driven noise | Acceptable tradeoff for lint compliance; document why |
| 16  | **`newReportFilter` explicit nil init**           | Trivial — same                     | Same                                                  |
| 17  | **Duplicate `r.services[key]` lookups**           | Trivial — minor perf               | Reuse first lookup result                             |
| 18  | **`serviceRefLabel` drops scope info**            | Low — ambiguous cross-scope labels | Include scope when non-root                           |
| 19  | **`newSequenceCounter` is trivial wrapper**       | Trivial                            | Could be `new(atomic.Int64)` but harmless             |
| 20  | **Migration silently bumps unknown versions**     | Low — future-compatibility risk    | Add warning log for unknown versions                  |

---

## f) Top 25 Things to Get Done Next

Sorted by **impact / work ratio** (highest first):

### Quick Wins (Low Work, High Impact)

1. **Wire `Config.Validate()` into `New()`** — return error on invalid ContainerID. 15 min.
2. **Fix Mermaid `mermaidNodeID` sanitization** — add `*`, `[`, `]` to replacement set. 5 min.
3. **Add waveform test** — verify `wf-event` divs and `renderWaveform` in HTML output. 10 min.
4. **Rename misnamed test** — `TestPlugin_WriteReportJSONError` → `TestPlugin_WriteReportJSON_Success`. 2 min.
5. **Add PlantUML writer-error test** — mirror Mermaid test. 5 min.
6. **Fix duplicate `r.services[key]` lookups** — reuse result. 10 min.
7. **Add `serviceRefLabel` scope prefix** — include scope when non-root. 5 min.

### Medium Effort, High Impact

8. **Deduplicate Mermaid/PlantUML** — extract `writeDependencyGraph(w, format)`. 30 min.
9. **Add concurrent `BuildReport` test** — goroutines writing while reading. 20 min.
10. **Add waveform interaction test** — verify hover events, tooltips. 15 min.
11. **Fix `Uptime()` to use deterministic timestamps** — compute from RegisteredAt to ShutdownAt/ExportedAt. 15 min.

### Medium Effort, Medium Impact

12. **Restructure Services tab** — card-grid layout option for visual scanability. 45 min.
13. **Add Events tab text search** — alongside type filter chips. 20 min.
14. **Add ARIA roles to tabs and tables** — accessibility compliance. 30 min.
15. **Add scope-prefixed labels in Mermaid/PlantUML** — for cross-scope services. 15 min.
16. **Add `ExportToHTML` invalid path test** — write to nonexistent directory. 5 min.
17. **Add alias provider registration test** — `do.As` in test container. 15 min.
18. **Document `ReportIndex` as preferred query API** — or refactor Report to use it internally. 30 min.

### Higher Effort, Strategic Impact

19. **Add visual snapshot tests** — render HTML headless, compare structure. 2 hrs.
20. **Add service detail panel** — click service row → side panel with full timeline. 1 hr.
21. **Add dark/light theme toggle** — CSS custom properties swap. 1 hr.
22. **Add streaming/real-time HTML mode** — SSE endpoint for live audit view. 4 hrs.
23. **Add report diff view** — compare two exported JSON reports. 3 hrs.
24. **Add multi-step migration pipeline** — registry pattern for v0.2.0 → v0.3.0 → ... 2 hrs.
25. **Add OpenTelemetry integration example** — `OnEvent` → OTel span exporter. 2 hrs.

---

## g) Top #1 Question

**Should `New()` return `(*Plugin, error)` to enforce validation, or should validation be a soft check (log warning, continue)?**

Changing the signature is a **breaking API change** for all existing users. The current `New(Config) *Plugin` is ergonomic — no error handling needed for the common case. Options:

- **A)** Breaking change: `New(Config) (*Plugin, error)` — strictest, cleanest, but breaks all callers.
- **B)** Non-breaking: `MustNew(Config) *Plugin` (panics on invalid) + `New(Config) (*Plugin, error)`. Standard Go pattern.
- **C)** Non-breaking: Keep `New(Config) *Plugin`, call `Validate()` internally, log a warning on invalid config but proceed. No caller changes needed.

I lean toward **B** (idiomatic Go `Must`/`New` pair) but this is a public API design decision that affects every user of the library.

---

## Metrics Summary

| Metric                             | Value                                          |
| ---------------------------------- | ---------------------------------------------- |
| Source files                       | 14 (.go + .templ)                              |
| Source LOC (excl. tests/generated) | ~2,100                                         |
| Test LOC                           | 3,748 (auditlog_test.go) + 367 (other tests)   |
| Generated LOC                      | 4,846 (html_templ.go)                          |
| Template LOC                       | 1,152 (html.templ)                             |
| Total tests                        | 140 (133 unit + 6 examples + 1 fuzz × 3 seeds) |
| Benchmarks                         | 11                                             |
| Dependencies                       | 2 direct (samber/do, a-h/templ), 1 indirect    |
| Go version                         | 1.26.3                                         |
| Schema version                     | 0.2.0                                          |
| Coverage                           | ~95%                                           |

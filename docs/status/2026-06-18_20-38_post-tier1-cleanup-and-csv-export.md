# Status Report — 2026-06-18 20:38

> Comprehensive snapshot of `samber-do-auditlog` after the Tier 1 cleanup + CSV/TSV export session.

---

## Executive Summary

**The project is in excellent shape.** All 5 CI gates pass (test 95.1%, lint 0 issues, vet clean, generate clean, mod-tidy clean). The library has 7 export formats, a replay engine, auto-detecting loader API, schema migration, report diffing, and a hardened HTML dashboard. The codebase is 3,398 production LOC with 7,553 test LOC (2.2:1 test-to-prod ratio), zero TODOs in production code, and only 3 minor clone groups at `-t 15`.

This session resolved all 5 lint failures that were blocking CI, shipped CSV/TSV export, and completed Tier 1 cleanup tasks (unused params, CHANGELOG, AGENTS.md, documentation).

---

## a) FULLY DONE ✅

### Core Plugin (stable API)

| Capability                                                            | Status | Verified                    |
| --------------------------------------------------------------------- | ------ | --------------------------- |
| Plugin constructor `New(Config) (*Plugin, error)`                     | ✅     | `plugin.go`                 |
| 6 lifecycle hooks (registration, invocation, shutdown × before/after) | ✅     | `hooks.go`                  |
| Zero-cost disabled mode (`DO_AUDITLOG_ENABLED`)                       | ✅     | `plugin.go`                 |
| Container ID validation (rejects `/` and `\`)                         | ✅     | `plugin.go:Config.Validate` |
| Real-time event callback (`Config.OnEvent`)                           | ✅     | `plugin.go`, `recorder.go`  |
| Memory-bounded event capture (`Config.MaxEvents`)                     | ✅     | `recorder.go`               |
| Health check recording (wrapper pattern)                              | ✅     | `healthcheck.go`            |

### Data Capture & Inference

| Capability                                               | Status | Verified                               |
| -------------------------------------------------------- | ------ | -------------------------------------- |
| Timestamped events with sequence numbers                 | ✅     | `event.go`, `recorder.go`              |
| Stack-based dependency graph inference                   | ✅     | `recorder.go`, `hooks.go`              |
| Reverse dependencies (dependents)                        | ✅     | `report_builder.go`                    |
| Provider type detection (lazy/eager/transient/alias)     | ✅     | `hooks.go:inferServiceType`            |
| Build/shutdown duration tracking (millisecond precision) | ✅     | `hooks.go`                             |
| Scope tree with hierarchical visualization               | ✅     | `report_builder.go`                    |
| Capability detection (IsHealthchecker, IsShutdowner)     | ✅     | `report_builder.go:enrichCapabilities` |

### Report Assembly & Queries

| Capability                                                                                                          | Status | Verified     |
| ------------------------------------------------------------------------------------------------------------------- | ------ | ------------ |
| `Report.Validate()` — 5 consistency checks                                                                          | ✅     | `report.go`  |
| `buildReportFromCore()` — single construction path                                                                  | ✅     | `report.go`  |
| `ServiceInfo.DeriveStatus()` — canonical status derivation                                                          | ✅     | `service.go` |
| Query methods: `ServiceByName`, `ServiceByRef`, `ServicesByScope`, `EventsByService`, `EventsByType`, `EventsByRef` | ✅     | `report.go`  |
| `FailedServices()`, `UnhealthyServices()`                                                                           | ✅     | `report.go`  |
| `Report.Index()` — pre-computed lookup maps                                                                         | ✅     | `report.go`  |
| `Report.Diff(other)` — structural comparison                                                                        | ✅     | `diff.go`    |
| `Report.Filtered(opts...)` — 5 filter options                                                                       | ✅     | `filter.go`  |

### Export Formats (7 total)

| Format   | Method                                       | Status                  |
| -------- | -------------------------------------------- | ----------------------- |
| JSON     | `Report.WriteJSON(writer)`                   | ✅                      |
| NDJSON   | `Report.WriteNDJSON(writer)`                 | ✅                      |
| CSV      | `Report.WriteCSV(writer)`                    | ✅ **NEW this session** |
| TSV      | `Report.WriteTSV(writer)`                    | ✅ **NEW this session** |
| HTML     | `Plugin.ExportToHTML(path)` / `WriteHTML(w)` | ✅                      |
| Mermaid  | `Report.WriteMermaid(writer)`                | ✅                      |
| PlantUML | `Report.WritePlantUML(writer)`               | ✅                      |

### Replay & Loading Pipeline

| Capability                                                                                  | Status | Verified    |
| ------------------------------------------------------------------------------------------- | ------ | ----------- |
| `ReplayEvents([]Event) (Report, error)` — inverse of recording                              | ✅     | `replay.go` |
| `ReadEvents(reader) ([]Event, error)` — NDJSON reader                                       | ✅     | `ndjson.go` |
| `LoadReport(path, opts...)` — auto-detecting loader                                         | ✅     | `loader.go` |
| `LoadReportFromReader`, `LoadReportFromBytes`, `LoadReportFromJSON`, `LoadReportFromNDJSON` | ✅     | `loader.go` |
| Format enum + `WithFormat` option                                                           | ✅     | `loader.go` |

### Schema Migration

| Capability                                                       | Status | Verified       |
| ---------------------------------------------------------------- | ------ | -------------- |
| `MigrateReport` v0.1.0 → v0.2.0                                  | ✅     | `migration.go` |
| Re-derives status from underlying fields (no stale preservation) | ✅     | `migration.go` |
| Re-derives all denormalized aggregates                           | ✅     | `migration.go` |

### HTML Visualization

| Feature                                                  | Status |
| -------------------------------------------------------- | ------ |
| 5-tab layout (Services/Scopes/Graph/Timeline/Events)     | ✅     |
| Warm amber "Container Telemetry" aesthetic               | ✅     |
| Lifecycle waveform signature element                     | ✅     |
| Sugiyama layered DAG graph with pan/zoom/touch           | ✅     |
| XSS-hardened (all user strings via `esc()`)              | ✅     |
| CSP hardened (`base-uri 'none'; frame-ancestors 'none'`) | ✅     |
| Type metadata injected via `@templ.JSONScript`           | ✅     |
| Pagination (50 services, 100 events)                     | ✅     |
| Keyboard navigation (1-5)                                | ✅     |

### CI Pipeline (5 jobs, all passing)

| Job                  | Status | Detail                                                                 |
| -------------------- | ------ | ---------------------------------------------------------------------- |
| **test**             | ✅     | `go vet`, `go build`, `go test -race`, **95.1% coverage gate**         |
| **lint**             | ✅     | `golangci-lint config verify` + `golangci-lint v2.12.2` — **0 issues** |
| **vulncheck**        | ✅     | `govulncheck` via `golang/govulncheck-action`                          |
| **mod-tidy**         | ✅     | `go mod tidy` — no drift                                               |
| **stale-generation** | ✅     | `go generate ./...` — no drift                                         |

### Testing Infrastructure

| Metric                                             | Value                                                                     |
| -------------------------------------------------- | ------------------------------------------------------------------------- |
| Test functions (Test + Benchmark + Example + Fuzz) | 225                                                                       |
| Benchmarks                                         | 11                                                                        |
| Fuzz targets                                       | 4 (`FuzzPluginHTML`, `FuzzMigrateReport`, `FuzzDiagramSpecialChars`, + 1) |
| Godoc Examples                                     | 7                                                                         |
| Coverage                                           | **95.1%** of non-example statements                                       |
| `t.Parallel()` calls                               | ~97% of eligible tests                                                    |
| Shared test helpers                                | `helpers_test.go` (providers, assertions, construction)                   |

### Documentation

| Document            | Status                                                   |
| ------------------- | -------------------------------------------------------- |
| `AGENTS.md`         | ✅ Updated this session (Architecture section + helpers) |
| `CHANGELOG.md`      | ✅ Updated this session (Unreleased section complete)    |
| `FEATURES.md`       | ✅ Honest feature inventory                              |
| `TODO_LIST.md`      | ✅ Verified against code                                 |
| `STABILITY.md`      | ✅ 0.x API stability promise                             |
| `CONTRIBUTING.md`   | ✅ Includes releasing section                            |
| `BENCHMARKS.md`     | ✅ Post-v0.0.3 baseline                                  |
| OTel bridge example | ✅ `docs/examples/otel-bridge.md`                        |

### Releases

| Tag    | Date       | Status                                              |
| ------ | ---------- | --------------------------------------------------- |
| v0.0.1 | 2026-06-10 | ✅ First alpha                                      |
| v0.0.2 | 2026-06-11 | ✅ Generated templ committed                        |
| v0.0.3 | 2026-06-17 | ✅ HTML redesign, observability, monolith split     |
| v0.0.4 | 2026-06-17 | ✅ CI pipeline, coverage gate, unified construction |

---

## b) PARTIALLY DONE 🟡

### Typed Identifiers (`ContainerID`, `ScopeID`, `ServiceName`)

**Assessed, not implemented.** Blast radius analysis complete:

- ~80 call sites across 14 map key types, 26 struct fields, ~18 function signatures
- Natural chokepoints: `ServiceRef` (public) and `svcKey` (internal)
- No existing bugs from string mixing
- **Decision: defer to v0.1.0** — introduce alongside `NewReport()` constructor (both are breaking changes)

### Deduplication Session Follow-up

- ✅ Production code: zero harmful clones
- 🟡 3 clone groups remain at `-t 15` (1.07% duplicated lines):
  1. `replay.go:265-282` vs `report_builder.go:49-67` — scope tree builder (generic shared, structurally similar call sites)
  2. `hooks.go:294-303` vs `hooks.go:151-160` — OnBeforeShutdown vs OnAfterRegistration hook bodies
  3. `hooks.go:319-329` vs `hooks.go:217-227` — OnAfterShutdown vs OnAfterInvocation hook bodies
- All 3 are structural similarity in hook method bodies (same pattern, different event types)

### `example/` Demo

- ✅ 19 samber/do v2 features demonstrated
- ✅ Self-checking feature checklist
- 🟡 No integration test that runs the example binary and validates the HTML output end-to-end

---

## c) NOT STARTED ⬜

### Architecture

| Task                                            | Effort | Impact | Blocks                                |
| ----------------------------------------------- | ------ | ------ | ------------------------------------- |
| **JSON Schema generation** from Go types        | 3h     | HIGH   | Blocks v0.1.0 release                 |
| **`NewReport()` constructor** `(Report, error)` | 2h     | HIGH   | Makes invalid reports unrepresentable |
| **Split `ServiceInfo`** into 4 concern structs  | 6h     | HIGH   | Breaking change, decide before v0.1.0 |
| **Typed identifiers**                           | 2h     | HIGH   | ~80 call sites, defer with NewReport  |

### Features

| Task                                                        | Effort | Impact |
| ----------------------------------------------------------- | ------ | ------ |
| **CLI tool** (`auditlog convert --format html report.json`) | 4h     | HIGH   |
| **WebSocket live stream** bridge for `OnEvent`              | 3h     | MED    |
| **Prometheus exporter** example                             | 2h     | MED    |
| **DOT diagram format** (3rd diagram via `go-output`)        | 3h     | LOW    |

### Testing

| Task                                                | Effort | Impact |
| --------------------------------------------------- | ------ | ------ |
| **Property-based `Diff` tests** (symmetry, inverse) | 2h     | MED    |
| **Property-based `MigrateReport` tests**            | 2h     | MED    |
| **HTML golden-file test**                           | 1h     | MED    |
| **Fuzz filter inputs**                              | 2h     | LOW    |

### Release & CI

| Task                            | Effort | Impact                                 |
| ------------------------------- | ------ | -------------------------------------- |
| **v0.0.5 release**              | 30min  | MED — 35 commits since v0.0.4          |
| **v0.1.0 release**              | —      | HIGH — blocked on JSON Schema decision |
| **`actionlint`** in CI          | 30min  | LOW                                    |
| **Flake app for coverage gate** | 1h     | LOW                                    |

---

## d) TOTALLY FUCKED UP 💥 (Issues Found & Fixed This Session)

### 1. CI-Blocking Lint Failures (FIXED ✅)

The dedup session introduced **5 lint failures** that would have failed the CI lint job:

| Issue                  | File                        | Root Cause                                                                                         | Fix                                                                            |
| ---------------------- | --------------------------- | -------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| `gochecknoglobals` × 3 | `types.go:28,75,116`        | Dedup session converted idiomatic switch-statement enum methods to package-level `var` map globals | Added `//nolint:gochecknoglobals` with justification (read-only lookup tables) |
| `varnamelen` × 2       | `report_builder.go:225,256` | Generic `buildScopeTreeFromMeta` used `id` for a variable with wide scope                          | Renamed to `scopeID` / `rootID`                                                |

**Lesson**: The dedup session's "convert switches to maps" optimization was a **false dedup win** — it traded idiomatic Go (switch with distinct cases per domain value) for a lint violation. Switch statements with one case per enum value are NOT harmful duplication; they're the idiomatic Go pattern for enum dispatch.

### 2. Unused Parameters in replay.go (FIXED ✅)

Three `apply*` methods in `replay.go` took a `key svcKey` parameter that was never used (the helper computed the key internally). Fixed by removing the parameter from `applyRegistrationAfter`, `applyInvocationAfter`, `applyHealthCheck`.

### 3. html_templ.go Import Formatting Fragility (NOTED, NOT BROKEN)

`go generate` produces single-line imports; `golangci-lint fmt` (gci formatter) reformats to grouped imports. The `.golangci.yml` excludes `_templ\.go$` from lint, but `golangci-lint fmt` doesn't respect path exclusions the same way. **Current state: committed version matches `go generate` output, CI stale-generation gate passes.** But running `golangci-lint fmt` manually will create a diff that must be reverted.

---

## e) WHAT WE SHOULD IMPROVE 🎯

### Architecture Improvements

1. **Stop calling `gochecknoglobals` violations "dedup wins"** — The map-based enum metadata in `types.go` should either stay as switches (idiomatic, no lint issue) or be documented as an explicit architectural choice. The `//nolint` directives are a band-aid.

2. **Unify the hook method pattern** — The 3 remaining clone groups in `hooks.go` are all structural similarity between hook methods (OnBeforeShutdown ≈ OnAfterRegistration, OnAfterShutdown ≈ OnAfterInvocation). A table-driven hook dispatcher could eliminate these, but the tradeoff is readability. May not be worth it.

3. **`Report` constructor validation** — Currently `Report` is a public struct that anyone can construct with invalid data. `NewReport()` should enforce `Validate()` at construction time. This is the single biggest "make impossible states unrepresentable" win.

4. **Typed identifiers** — `ContainerID`, `ScopeID`, `ServiceName` as distinct types would prevent accidental argument swaps (e.g., `ServiceByName(scopeID, name)` vs `ServiceByRef(name, scopeID)`). The blast radius is ~80 sites but mechanical.

### Process Improvements

5. **Always run `golangci-lint run` after changes** — Not just `go build` + `go test`. The lint config is extremely strict and catches issues the compiler doesn't.

6. **Never run `golangci-lint fmt`** — It reformats generated files (`html_templ.go`) in ways that conflict with `go generate`. Use `gofumpt` directly on specific files instead.

7. **Consider a pre-commit hook** — Running `go generate ./... && golangci-lint run && go test -race ./...` before every commit would prevent CI surprises.

### Documentation Improvements

8. **CHANGELOG should note the lint fix** — The `gochecknoglobals` + `varnamelen` fixes should be documented under `[Unreleased] → Fixed`.

9. **AGENTS.md should document the `html_templ.go` formatting trap** — So future contributors don't run `golangci-lint fmt` and create drift.

---

## f) Top 25 Things to Do Next

Sorted by **impact ÷ effort** (highest first).

### Tier 1 — Do Now (high impact, low effort)

| #   | Task                                                                                          | Effort | Impact |
| --- | --------------------------------------------------------------------------------------------- | ------ | ------ |
| 1   | **v0.0.5 release** — tag current state, 35 commits since v0.0.4                               | 30min  | HIGH   |
| 2   | **Add `WriteCSV`/`WriteTSV` to `Plugin` export methods** — wire through from Plugin to Report | 15min  | MED    |
| 3   | **HTML golden-file test** — deterministic report → assert output matches committed file       | 1h     | MED    |
| 4   | **Property-based `Diff` tests** — `Diff(a,a)` empty, `Diff(a,b)`/`Diff(b,a)` symmetry         | 2h     | MED    |
| 5   | **`actionlint` in CI** — validate `.github/workflows/ci.yml`                                  | 30min  | LOW    |

### Tier 2 — High Impact, Medium Effort

| #   | Task                                                                           | Effort | Impact |
| --- | ------------------------------------------------------------------------------ | ------ | ------ |
| 6   | **`NewReport()` constructor** — `(Report, error)` enforcing `Validate()`       | 2h     | HIGH   |
| 7   | **Typed identifiers** — `ContainerID`, `ScopeID`, `ServiceName` distinct types | 2h     | HIGH   |
| 8   | **JSON Schema generation** — derive `schema.json` from Go types                | 3h     | HIGH   |
| 9   | **Property-based `MigrateReport` tests** — arbitrary JSON → migrate → validate | 2h     | MED    |
| 10  | **Prometheus exporter example** — parallel to OTel example                     | 2h     | MED    |

### Tier 3 — Medium Impact, Medium Effort

| #   | Task                                                                     | Effort | Impact |
| --- | ------------------------------------------------------------------------ | ------ | ------ |
| 11  | **CLI tool** — `auditlog convert --format html report.json`              | 4h     | HIGH   |
| 12  | **WebSocket live stream** bridge — `OnEvent` → browser dashboard         | 3h     | MED    |
| 13  | **Split `ServiceInfo` into 4 structs** — identity/lifecycle/health/graph | 6h     | HIGH   |
| 14  | **Flake app for coverage gate** — replace inline shell in CI             | 1h     | LOW    |
| 15  | **Fuzz filter inputs** — arbitrary `ReportOption` combinations           | 2h     | LOW    |

### Tier 4 — Lower Priority

| #   | Task                                                            | Effort | Impact |
| --- | --------------------------------------------------------------- | ------ | ------ |
| 16  | **DOT diagram format** via `go-output` v0.12.0                  | 3h     | LOW    |
| 17  | **`go-output` adoption** — replace custom Mermaid/PlantUML      | 4h     | LOW    |
| 18  | **Pre-commit hook** — `go generate + lint + test` before commit | 30min  | LOW    |
| 19  | **Coverage gate as Nix flake check** — replace inline CI shell  | 1h     | LOW    |
| 20  | **BDD tests** for critical user journeys (via Ginkgo)           | 3h     | LOW    |

### Tier 5 — Rejected / Deferred

| #   | Task                             | Status                                            |
| --- | -------------------------------- | ------------------------------------------------- |
| 21  | **Multi-module split**           | ❌ Rejected — too small (1 package)               |
| 22  | **External storage backends**    | ❌ Rejected — file + io.Writer sufficient         |
| 23  | **`samber/lo` dependency**       | ❌ Rejected — stdlib slices/cmp sufficient        |
| 24  | **`encoding/json/v2`**           | ❌ Rejected — risk of breaking JSON format        |
| 25  | **NDJSON import (`ReadNDJSON`)** | ✅ Already done via `ReadEvents` + `ReplayEvents` |

---

## g) Top Question I Cannot Answer Myself 🤔

**"Should we ship v0.1.0 with the current `ServiceInfo` monolith (19 fields), or block on splitting it into 4 concern-specific structs first?"**

**Context**: `ServiceInfo` mixes 4 orthogonal concerns:

- **Identity** (ServiceRef: scope/service names)
- **Lifecycle** (status, type, timestamps, invocation/shutdown data, errors)
- **Health** (last check, error, count)
- **Graph** (dependencies, dependents)

**Why I can't decide**:

- **Argument for splitting now**: It's a breaking change. Every v0.0.x consumer will need to migrate. Better to break once (v0.1.0) than twice (v0.1.0 + v0.2.0). The 19-field struct is already anemic — it's a bag of fields with 3 methods.
- **Argument against splitting now**: No consumer has complained about the 19 fields. The struct works fine for JSON serialization (all fields flatten). Splitting adds indirection (consumers must navigate `svc.Lifecycle.Status` instead of `svc.Status`) for marginal type safety. YAGNI may apply.
- **The real blocker**: I don't know how many external consumers exist or how they use `ServiceInfo`. If this is a personal project with 0 external users, break freely. If it has users, the migration cost matters.

**What I need from you**: A decision on whether v0.1.0 ships the current monolith or blocks on the split. This determines whether tasks #6, #7, #13 are done together (one breaking release) or incrementally (multiple breaking releases).

---

## Metrics Snapshot

| Metric                  | Value                           | Trend                              |
| ----------------------- | ------------------------------- | ---------------------------------- |
| Total LOC (`.go` files) | 11,687                          | ↑ (CSV export +132 LOC)            |
| Production LOC          | 3,398                           | ↑ (csv.go +132 LOC)                |
| Test LOC                | 7,553                           | ↑ (csv_export_test.go +218 LOC)    |
| Test functions          | 225                             | ↑ (+6 CSV tests)                   |
| Benchmarks              | 11                              | Stable                             |
| Fuzz targets            | 4                               | Stable                             |
| Godoc Examples          | 7                               | Stable                             |
| Production functions    | 185                             | ↑ (+5 csv.go functions)            |
| Go files (total)        | 55                              | ↑ (+2: csv.go, csv_export_test.go) |
| Coverage                | **95.1%**                       | ✅ Meets 95% gate                  |
| Lint issues             | **0**                           | ✅ Was 5, now 0                    |
| Clone groups (`-t 15`)  | 3                               | Stable (1.07% duplication)         |
| TODO/FIXME in prod code | **0**                           | ✅                                 |
| CI jobs                 | 5                               | ✅ All pass                        |
| Commits since v0.0.4    | 35                              | ↑                                  |
| Direct dependencies     | 2 (`samber/do v2`, `a-h/templ`) | Stable                             |
| Go version              | 1.26.3                          | Stable                             |

---

## Session Commits (3 pushed)

| Commit    | Description                                                                                     |
| --------- | ----------------------------------------------------------------------------------------------- |
| `cd1a528` | docs: document replay/ndjson/loader additions; normalize CHANGELOG; clean up (Tier 1 tasks 1-5) |
| `49f9679` | fix: resolve 5 golangci-lint failures blocking CI (gochecknoglobals + varnamelen)               |
| `899622f` | feat: add CSV/TSV export for data analysis workflows (Report.WriteCSV/WriteTSV)                 |

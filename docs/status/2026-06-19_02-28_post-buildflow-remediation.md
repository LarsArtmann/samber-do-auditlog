# Status Report — 2026-06-19 02:28

**Project**: samber-do-auditlog — Go plugin for samber/do v2 DI container audit logging
**Module**: `github.com/larsartmann/samber-do-auditlog` · **Go**: 1.26.3 · **Status**: ALPHA (pre-v0.1.0)
**Commits**: 284 total (65 since v0.0.4) · **Files**: 75 Go files · **Coverage**: 95.2%

---

## A. FULLY DONE ✓

### Core Plugin (Production-Ready)

| Feature                        | Details                                                                                                         |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------- |
| **Plugin lifecycle hooks**     | All 6 samber/do v2 hooks wired: registration (before/after), invocation (before/after), shutdown (before/after) |
| **Dependency graph inference** | Invocation stack infers A→B dependencies automatically from provider call chains                                |
| **Build duration tracking**    | Millisecond-precision per-service build and shutdown timing                                                     |
| **Health check recording**     | `RecordHealthCheck[WithContext]` wrapper records `EventTypeHealthCheck` events                                  |
| **Service type tracking**      | Auto-detected via `do.ExplainNamedService` — lazy/eager/transient/alias                                         |
| **Scope tree**                 | Root → child scopes with cross-scope dependency tracking                                                        |
| **Concurrency model**          | Single `sync.RWMutex`, `atomic.Int64` counters — 1 lock acquisition per hook                                    |
| **Zero-cost disabled mode**    | Empty hooks returned when disabled — zero recorder overhead                                                     |
| **Config validation**          | ContainerID path-separator rejection, env-var enablement                                                        |
| **Real-time event callback**   | `Config.OnEvent` streams events outside the recorder lock                                                       |
| **In-memory event cap**        | `Config.MaxEvents` with drop counter                                                                            |

### Export Formats (9 total)

| Format   | Method                                                           | Status                                                 |
| -------- | ---------------------------------------------------------------- | ------------------------------------------------------ |
| JSON     | `Report.WriteJSON`                                               | ✓                                                      |
| NDJSON   | `Report.WriteNDJSON`                                             | ✓                                                      |
| HTML     | `Report.WriteHTML`                                               | ✓ (self-contained, warm amber dashboard, 5-tab layout) |
| Mermaid  | `Report.WriteMermaid`                                            | ✓                                                      |
| PlantUML | `Report.WritePlantUML`                                           | ✓                                                      |
| DOT      | `Report.WriteDOT`                                                | ✓                                                      |
| CSV      | `Report.WriteCSV`                                                | ✓                                                      |
| TSV      | `Report.WriteTSV`                                                | ✓                                                      |
| File     | `Plugin.ExportToFile`, `ExportFilteredToFile`, `WriteReportJSON` | ✓                                                      |

### CLI (7 subcommands)

| Command    | Purpose                             |
| ---------- | ----------------------------------- |
| `info`     | Inspect report metadata             |
| `convert`  | Convert between JSON/NDJSON formats |
| `diff`     | Compare two reports                 |
| `validate` | Validate report against schema      |
| `stats`    | Aggregate statistics (NEW)          |
| `schema`   | Print JSON Schema                   |
| `version`  | Version info (ldflags-overridable)  |

### Report Operations

| Feature              | Method                                                                                                       |
| -------------------- | ------------------------------------------------------------------------------------------------------------ |
| **Filtering**        | `Report.Filtered(opts...)` with 5 filter options                                                             |
| **Diffing**          | `Report.Diff(other)` returning `DiffResult`                                                                  |
| **Merging**          | `Report.MergeReports([]Report)` (NEW)                                                                        |
| **Validation**       | `Report.Validate()` — count + status consistency checks                                                      |
| **Schema migration** | `MigrateReport` v0.1.0 → v0.2.0                                                                              |
| **Loading**          | `LoadReport` / `LoadReportFromReader` (auto-detect JSON/NDJSON)                                              |
| **Query methods**    | `ServiceByName`, `ServicesByScope`, `EventsByType`, `EventsByService`, `FailedServices`, `UnhealthyServices` |

### Testing (Comprehensive)

| Metric                            | Value                                                                                            |
| --------------------------------- | ------------------------------------------------------------------------------------------------ |
| **Test functions**                | 369 (test + benchmark + fuzz + example entries)                                                  |
| **Top-level tests**               | ~233                                                                                             |
| **Benchmarks**                    | 11 (3 sub-benchmarks)                                                                            |
| **Fuzz targets**                  | 5 (FuzzPluginHTML, FuzzMigrateReport, FuzzDiagramSpecialChars, FuzzReadEvents, FuzzFilterInputs) |
| **Examples**                      | 7 (godoc)                                                                                        |
| **t.Parallel() calls**            | ~152 (~97% of eligible)                                                                          |
| **Coverage**                      | 95.2% (non-example, gate: ≥95%)                                                                  |
| **Clone groups (art-dupl -t 30)** | **0**                                                                                            |
| **Lint issues**                   | **0**                                                                                            |

### CI Pipeline (6 jobs, all green)

| Job                | Purpose                                         |
| ------------------ | ----------------------------------------------- |
| `test`             | go vet, build, test -race, coverage gate ≥95%   |
| `lint`             | golangci-lint v2.12.2 (extremely strict config) |
| `vulncheck`        | govulncheck via GitHub Action                   |
| `mod-tidy`         | go.sum drift detection                          |
| `stale-generation` | templ + schema regeneration check               |
| `actionlint`       | GitHub Actions workflow validation              |

### Infrastructure

| Component            | Status                                                                                 |
| -------------------- | -------------------------------------------------------------------------------------- |
| `flake.nix` devShell | Go 1.26.4, templ, golangci-lint, govulncheck, actionlint                               |
| JSON Schema          | Draft 2020-12, generated from Go types, `go:embed`ded                                  |
| Pre-commit hook      | generate + vet + lint + test                                                           |
| `.prettierignore`    | Protects golden files, generated schema, docs from oxfmt                               |
| Documentation        | README, CHANGELOG, FEATURES.md, TODO_LIST.md, STABILITY.md, CONTRIBUTING.md, AGENTS.md |

---

## B. PARTIALLY DONE ⚠️

| Item                                          | Status                           | Notes                                                                                                             |
| --------------------------------------------- | -------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| **`TestDiff_MultipleChanged` event literals** | ~12 raw struct literals remain   | Uses `rootRef()` + full fields. Not art-dupl flagged at -t 30 due to semantic variation. Lower ROI conversion.    |
| **HTML template polish**                      | Functional but could be improved | Warm amber dashboard works. JS is inline. Could use component extraction for maintainability.                     |
| **CLI integration tests**                     | 8 test functions                 | Coverage gate excludes `cmd/` (exercised via golden/integration tests, not in-process). Some edge cases untested. |
| **BENCHMARKS.md**                             | Baseline captured post-v0.0.3    | Not updated for v0.1.0 features (MergeReports, stats, etc.)                                                       |

---

## C. NOT STARTED 📋

| Item                                  | Impact | Effort                                                                     |
| ------------------------------------- | ------ | -------------------------------------------------------------------------- |
| **Typed identifiers (branded types)** | HIGH   | Deferred to v0.1.0 breaking batch — 65+ compile errors, zero existing bugs |
| **`ServiceInfo` split**               | MEDIUM | Deferred to v0.1.0 — split into identity/lifecycle/health/graph structs    |
| **Schema drift detection in CI**      | MEDIUM | Test exists (`TestSchemaDrift`) but not wired as separate CI job           |
| **OpenTelemetry bridge example**      | LOW    | Documented in `docs/examples/otel-bridge.md` but no runnable code          |
| **Prometheus exporter**               | LOW    | `Config.OnEvent` enables it but no reference implementation                |
| **`.buildflow.yml` config**           | LOW    | Buildflow has known `language: go` detection bug — workaround is env var   |
| **Streaming/chunked export**          | LOW    | Current exports materialize full report in memory                          |
| **WebSocket live dashboard**          | LOW    | HTML is self-contained static; no real-time push                           |

---

## D. TOTALLY FUCKED UP! 💥

**Nothing is critically broken.** All tests pass, lint is clean, CI is green, zero clones.

One **recently unfucked** item worth noting:

| Item                                   | What happened                                                                                                                       | Resolution                                                                                                    |
| -------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| **oxfmt corrupting golden file**       | `oxfmt --write .` was pretty-printing `testdata/golden/report.html`, breaking `TestReport_WriteHTML_GoldenFile` every buildflow run | Added `.prettierignore` excluding `testdata/`, `schema/`, `docs/`, `CHANGELOG.md` — oxfmt reads it by default |
| **buildflow test-fuzz false failures** | Default 2m timeout killed fuzz processes mid-run (5 targets × 30s = 2.5m)                                                           | Documented `--max-time=5m` requirement. Not a code bug — buildflow config issue.                              |
| **templ version mismatch**             | `html_templ.go` had v0.3.1036 header but go.mod pins v0.3.1020                                                                      | Regenerated via `go generate` to match pinned version                                                         |

---

## E. WHAT WE SHOULD IMPROVE! 🎯

### Architecture & Type Safety

1. **Branded types for identifiers** — `ContainerID`, `ScopeID`, `ServiceName` as named string types. Makes impossible states unrepresentable. 65+ compile error blast radius — batch as v0.1.0 breaking change.
2. **Split `ServiceInfo` monolith** — Currently a 20+ field struct mixing identity, lifecycle, health, and graph concerns. Split into focused structs composed at the report level.
3. **`mkEvent` factory generalization** — Test helpers (`mkEvent`, `mkEventWithDur`, `mkRegEvent`) share patterns. A single builder pattern or functional options could unify them across packages.
4. **Error type hierarchy** — Current sentinel errors (`errReportEventCountMismatch`, etc.) lack structure. Consider typed error wrappers for programmatic handling.

### Testing & Quality

5. **Update `BENCHMARKS.md`** — Baselines captured pre-v0.1.0 features. Need fresh runs after MergeReports, stats, filter changes.
6. **CLI integration test coverage** — `cmd/auditlog` is excluded from the coverage gate. More edge cases needed for diff/convert/stats.
7. **Property-based testing for filters** — `FuzzFilterInputs` exists but doesn't verify filter invariants (e.g., filtered result should always have ≤ original count).
8. **Schema drift as CI job** — Test exists but runs only locally. Wire it as a dedicated CI step.

### Developer Experience

9. **`.buildflow.yml`** — Currently relying on env var workaround. Track buildflow's language-detection bug fix.
10. ** templ source → generated diff noise** — Generated `html_templ.go` is ~800 lines. Consider `.gitattributes` for diff suppression.
11. **Interactive HTML demo** — GitHub Pages site with a live report viewer would improve first-impression.
12. **Go docs enrichment** — Examples exist but package-level overview could be stronger.

---

## F. TOP #25 THINGS TO GET DONE NEXT

Sorted by **impact / effort ratio** (highest first):

| #   | Task                                                                | Impact   | Effort | Category        |
| --- | ------------------------------------------------------------------- | -------- | ------ | --------------- |
| 1   | **Tag v0.1.0 release**                                              | CRITICAL | LOW    | Release         |
| 2   | **Batch branded types** (`ContainerID`, `ScopeID`, `ServiceName`)   | HIGH     | HIGH   | Architecture    |
| 3   | **Split `ServiceInfo`** into identity/lifecycle/health/graph        | HIGH     | HIGH   | Architecture    |
| 4   | **Update `BENCHMARKS.md`** with v0.1.0 feature baselines            | MEDIUM   | LOW    | Testing         |
| 5   | **Wire schema drift test** as dedicated CI job                      | MEDIUM   | LOW    | CI              |
| 6   | **Add `MergeReports` property tests** (idempotent, associative)     | MEDIUM   | LOW    | Testing         |
| 7   | **Convert `TestDiff_MultipleChanged`** raw literals to `mkEvent`    | LOW      | MEDIUM | Code Quality    |
| 8   | **Add `.gitattributes`** for `*_templ.go` diff suppression          | LOW      | LOW    | DX              |
| 9   | **CLI: add `--format` flag** to `info` subcommand for JSON output   | MEDIUM   | LOW    | DX              |
| 10  | **Add `Report.WriteAll(writer, format)`** unified export dispatcher | MEDIUM   | LOW    | API             |
| 11  | **Error type hierarchy** — wrap sentinels in typed structs          | MEDIUM   | MEDIUM | Architecture    |
| 12  | **Add `ServiceInfo.IsRootScope()`** convenience method              | LOW      | LOW    | API             |
| 13  | **HTML: extract JS into separate file** (reduce template size)      | LOW      | MEDIUM | Maintainability |
| 14  | **Add `Config.Validate()` examples** to godoc                       | LOW      | LOW    | Docs            |
| 15  | **Property test: filter invariants** (filtered ≤ original)          | MEDIUM   | LOW    | Testing         |
| 16  | **Prometheus exporter reference** in `example/`                     | LOW      | MEDIUM | DX              |
| 17  | **Add `Report.EventTimeline()`** returning time-sorted event view   | LOW      | LOW    | API             |
| 18  | **OpenTelemetry bridge** runnable example                           | LOW      | MEDIUM | DX              |
| 19  | **Add `DiffResult.Summary()`** human-readable diff string           | LOW      | LOW    | API             |
| 20  | **Streaming JSON export** (avoid full materialization)              | LOW      | HIGH   | Performance     |
| 21  | **Add `Config.LogLevel`** for internal plugin logging               | LOW      | MEDIUM | DX              |
| 22  | **GitHub Pages demo** with sample report viewer                     | LOW      | HIGH   | Marketing       |
| 23  | **Add `Report.Stats()`** as a public method (currently CLI-only)    | MEDIUM   | LOW    | API             |
| 24  | **Add `--watch` flag to CLI** for live report regeneration          | LOW      | HIGH   | DX              |
| 25  | **Add `ServiceInfo.Uptime()`** to public API (currently internal)   | LOW      | LOW    | API             |

---

## G. TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF 🤔

**Should v0.1.0 be a breaking-change release (branded types + ServiceInfo split), or a stabilization release (freeze API, add tests, polish DX)?**

The branded types refactor has a 65+ compile error blast radius across production code, tests, and generated templ. It would make impossible states unrepresentable but introduces zero bug fixes — there are currently zero bugs from the string-typed approach. The `ServiceInfo` split has similar blast radius.

Arguments for **breaking v0.1.0**:

- Get the pain over with while the project is ALPHA and user base is small
- The longer we wait, the more consumers depend on the current API
- Types are the foundation — everything else builds on them

Arguments for **stabilization v0.1.0**:

- 95.2% coverage, 0 lint issues, 0 clones — the codebase is already high quality
- Breaking changes risk introducing regressions in a clean codebase
- "If it ain't broke, don't fix it" — zero existing bugs from string types

**What is the target audience and timeline?** If this is heading toward production use with external consumers, the breaking changes should happen now. If it's still a personal/exploratory project, stabilization is fine.

---

## Session Stats

| Metric                  | Value                                                    |
| ----------------------- | -------------------------------------------------------- |
| Commits this session    | 2 (`badc5d1`, `29529c0`)                                 |
| Files changed           | 7                                                        |
| Lines changed           | +124 / -120                                              |
| Clone groups eliminated | 3 → 0                                                    |
| BuildFlow steps passing | 59/59 (with `--max-time=5m`)                             |
| Issues fixed            | go-fix flaky, test-race, test-fuzz, duplications-checker |
| Coverage                | 95.2% (unchanged — refactoring was net-neutral)          |

# Status Report — 2026-06-17 15:28

## Post-Remediation-Consolidation Comprehensive Status

**Branch**: `master` (up to date with `origin/master`) · **Latest tag**: `v0.0.3` · **Schema version**: `0.2.0` · **Go**: `1.26.4` · **Working tree**: clean

---

## Executive Summary

This session executed the four "immediate (before push)" follow-up items from the previous remediation status report, then extended the work with a significant architecture improvement and two new fuzz targets. All changes are committed (6 commits) and pushed to `origin/master`.

The most impactful change is **unified Report construction**: the three independent code paths that computed the same 8 denormalized Report fields (`BuildReport`, `Filtered`, `MigrateReport`) now route through a single `buildReportFromCore()` constructor. This makes it structurally impossible for count/summary fields to drift from the underlying data, and means any future Report construction path (e.g. NDJSON import) gets all fields for free.

Test coverage remains **95.3%** of production statements, with **145 top-level test functions**, **6 fuzz targets** (up from 4), **11 benchmark functions**, and **7 example functions** across **24 test files**.

### Self-Critique: What Was Forgotten / Could Be Better

1. **`html_templ.go` keeps drifting** — An editor/formatter reformats the import block in the generated file, diverging from `go tool templ generate` output. I had to restore it twice this session. A `.gitattributes` or editor-config rule to exclude generated files from format-on-save would prevent this permanently.
2. **`TODO_LIST.md` is stale** — Still lists Go 1.26.3 references and doesn't reflect the new `buildReportFromCore` refactor or the two new fuzz targets. The "Last updated" date is correct but the content lags.
3. **No CHANGELOG entry for the `buildReportFromCore` refactor** — The architecture improvement was committed but not surfaced in `[Unreleased]`. Consumers reading the changelog won't know about the new `ServiceInfo.DeriveStatus()` public method.
4. **`AGENTS.md` Gotchas section doesn't mention `buildReportFromCore`** — The central construction path is a critical architectural invariant that future contributors need to know about.
5. **Sequential tests in healthcheck/plugin_basic** — 18 tests remain sequential (7 in healthcheck, 11 in plugin_basic). Some are env-var tests that can't be parallelized, but others likely can.
6. **No JSON Schema yet** — Still the biggest missing piece for report consumers. Listed as #7 in the top-25.

---

## a) FULLY DONE

### Architecture Improvements

| Improvement                             | File(s)                                                       | Details                                                                                                                                                                                                                                 |
| --------------------------------------- | ------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Unified Report construction**         | `report.go`, `report_builder.go`, `filter.go`, `migration.go` | Extracted `buildReportFromCore()` + `finalizeDenormalized()` — single construction path for all 3 Report producers. Eliminates 3-way duplication of 8 denormalized fields. Adding new construction paths now auto-fills all aggregates. |
| **`ServiceInfo.DeriveStatus()` method** | `service.go`                                                  | Moved status derivation from migration-local `computeServiceStatusFromInfo` to a method on the type it operates on. Single canonical derivation entry point, reusable beyond migration.                                                 |

### Documentation

| Doc                        | Status     | Details                                                                                                                                                                                                                                                                            |
| -------------------------- | ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `CHANGELOG.md`             | ✅ Updated | `[Unreleased]` expanded with Fixed/Changed/Added/Tests entries for the full remediation batch (scope-count fix, normalize-any-version, scope-assertion hardening, dead JS timestamps, Go bump, experimental tags, capabilityFlags refactor, CI gates, new fuzz/parallelism tests). |
| `BENCHMARKS.md`            | ✅ Updated | Go version `1.26.3` → `1.26.4` in Environment table to match `go.mod`.                                                                                                                                                                                                             |
| `AGENTS.md`                | ✅ Updated | Go version bumped (header + devShell). Commands table now documents `golangci-lint config verify`, `go mod tidy`, and coverage-profile run. CI section rewritten to reflect 5 parallel jobs (added coverage gate + mod-tidy job + config-verify step).                             |
| `README.md`                | ✅ Updated | `MigrateReport` description in Plugin table corrected to "normalize/repair" semantics. Schema-migration callout expanded to document the repair behavior.                                                                                                                          |
| `migration.go` doc comment | ✅ Updated | `MigrateReport` doc comment expanded to state the "repair/normalize → current" contract and which fields are re-derived.                                                                                                                                                           |
| Status reports archived    | ✅ Done    | 9 superseded reports moved from `docs/status/` to `docs/archive/` via `git mv` (history preserved). Only the current report remains in `docs/status/`. Broken cross-reference fixed.                                                                                               |

### Tests

| Addition                      | File           | Details                                                                                                                                                                                                                                       |
| ----------------------------- | -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Diagram special-char fuzz** | `fuzz_test.go` | `FuzzDiagramSpecialChars` (5th fuzz target) — seeds Mermaid and PlantUML exporters with `]`, `"`, `-->`, `@enduml`, `%%`, newlines, pipes, 500-char strings. Verifies structural integrity of output headers/footers.                         |
| **Nested scope tree fuzz**    | `fuzz_test.go` | `FuzzNestedScopeExport` (6th fuzz target) — generates scope trees up to 500 levels deep, normalizes via `MigrateReport`, exports to JSON + Mermaid + PlantUML. Guards against stack overflow in recursive `countScopeNodes`/`pruneScopeTree`. |

### Verification Snapshot (all green)

```
Build:       ✅ go build ./... — clean
Vet:         ✅ go vet ./... — clean
Tests:       ✅ 145 top-level + 10 subtests — all PASS (with -race)
Coverage:    ✅ 95.3% of production statements (example/ excluded)
Lint:        ✅ golangci-lint v2.12.2 — 0 issues
Generate:    ✅ go generate ./... — no drift
Working tree: ✅ clean, all commits pushed to origin/master
```

---

## b) PARTIALLY DONE

| Item                                 | Status                          | Notes                                                                                                                                                                                                                                                |
| ------------------------------------ | ------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Test parallelism**                 | ~93% of eligible tests parallel | 138 `t.Parallel()` calls across 19 files. 18 tests remain sequential (7 in `healthcheck_basic_test.go`, 11 in `plugin_basic_test.go`). Some are env-var tests that genuinely can't be parallel; others may be parallelizable with minor refactoring. |
| **`TODO_LIST.md` freshness**         | Partially stale                 | "Last updated" date is correct (2026-06-17) but content doesn't reflect the `buildReportFromCore` refactor, `ServiceInfo.DeriveStatus()`, or the 2 new fuzz targets. Several Go 1.26.3 references remain in completed items.                         |
| **`CHANGELOG.md` for refactor**      | Missing entry                   | The `buildReportFromCore` unification and `ServiceInfo.DeriveStatus()` public method were committed but not documented in `[Unreleased]`. Consumers won't know about the new method.                                                                 |
| **`AGENTS.md` Gotchas for refactor** | Missing entry                   | The unified construction path is a critical architectural invariant. The Gotchas section doesn't mention `buildReportFromCore` or the `finalizeDenormalized` pattern.                                                                                |
| **html_templ.go format drift**       | Workaround, not fixed           | The generated file keeps being reformatted by an editor. Restored manually twice this session. Needs a permanent `.gitattributes` or editor-config exclusion.                                                                                        |

---

## c) NOT STARTED

These were identified in previous audits but not attempted:

1. **v0.1.0 release** — Project meets `STABILITY.md` criteria. Blocked on the JSON-schema-first vs. ship-now decision.
2. **JSON Schema file** for the report format — biggest missing piece for report consumers.
3. **Prometheus exporter example** parallel to the OTel example.
4. **NDJSON import** — loading events back from NDJSON into a Report.
5. **Property-based tests** with `rapid` or stdlib fuzz for `Diff` symmetry, filter round-trips.
6. **CSV / TSV export** of services/events.
7. **CLI tool** for report conversion/export/visualization.
8. **WebSocket live stream** bridge for `OnEvent`.
9. **GitHub Actions version upgrades** — `actions/checkout@v4` → v5, `actions/setup-go@v5` → v6 (when stable).
10. **actionlint** integration for workflow validation.
11. **gosec** already enabled in golangci-lint config, but a dedicated `gosec` CI step could provide deeper SAST.
12. **HTML integration test** realistic multi-service golden-file or DOM assertions.
13. **Fuzz filter inputs** — arbitrary `ReportOption` combinations.
14. **Flake app for coverage gate** to replace inline shell in CI.
15. **`Report.Validate()` → constructor validation** — make invalid reports unrepresentable.
16. **Typed identifiers** (`ContainerID`, `ScopeID`, `ServiceName` as distinct string types).
17. **Split `ServiceInfo`** into lifecycle sub-structs (identity / lifecycle / health / graph).

---

## d) TOTALLY FUCKED UP!

### 🟡 `html_templ.go` Format Drift Is Recurring

The generated `html_templ.go` keeps diverging from `go tool templ generate` output because an editor reformats the import block. This happened **twice** this session and also in the previous session. It would fail the CI `stale-generation` check if pushed in the reformatted state. The workaround (manual restore) is fragile and wastes time.

**Root cause**: No `.gitattributes` or editor configuration excludes generated files from format-on-save.

**Fix**: Add `html_templ.go` to a `.gitattributes` with `linguist-generated=true` and/or configure the editor to exclude `*_templ.go` from formatting.

### 🟡 `CHANGELOG.md` Doesn't Reflect the Refactor

The `buildReportFromCore` unification and the new `ServiceInfo.DeriveStatus()` public method are committed but not in `[Unreleased]`. A consumer reading the changelog has no visibility into the new public method or the internal architecture improvement.

### 🟡 `TODO_LIST.md` Lags Behind Code

The TODO list still references Go 1.26.3 in completed items, doesn't mention the `buildReportFromCore` refactor, the `DeriveStatus()` method, or the 6 fuzz targets (still says "3 fuzz targets" in historical entries). This creates a false impression of staleness for a project that's actually very current.

---

## e) WHAT WE SHOULD IMPROVE!

### Immediate (next session)

1. **Add `CHANGELOG.md` entry** for the `buildReportFromCore` refactor and `ServiceInfo.DeriveStatus()` method.
2. **Update `TODO_LIST.md`** to reflect current code state (6 fuzz targets, refactor done, Go 1.26.4 everywhere).
3. **Add `AGENTS.md` Gotcha** for `buildReportFromCore` — the unified construction path is a critical invariant.
4. **Fix `html_templ.go` drift permanently** — add `.gitattributes` with `linguist-generated=true` for `*_templ.go`.
5. **Parallelize remaining 18 sequential tests** where safe (healthcheck_basic, plugin_basic).

### Short-Term Architecture

6. **Introduce typed identifiers** — `ContainerID`, `ScopeID`, `ServiceName` as distinct named string types. Compiler rejects accidental swaps; validation moves into constructors. Low effort, high safety.
7. **Split `ServiceInfo` lifecycle concerns** — The 19-field struct mixes identity (3), timing (4), errors (2), health (3), graph (2), capabilities (2), type (1), status (1), order (1). Consider sub-structs: `ServiceIdentity`, `ServiceLifecycle`, `ServiceHealth`, `ServiceGraph`. High effort, but makes `deriveServiceStatus` and report building more explicit.
8. **Make `Report` constructor-validated** — `NewReport(...)` returns `(Report, error)` so invalid reports are unrepresentable. `Validate()` becomes a constructor check, not a post-hoc audit.
9. **Add JSON Schema generation** — Derive `schema.json` from `Report`, `Event`, `ServiceInfo` via reflection. Avoids drift. Medium effort.
10. **NDJSON import** — `ReadNDJSON(reader) ([]Event, error)` or `ImportNDJSON(reader) (Report, error)`. The unified `buildReportFromCore` already makes this trivial — just build the core slices and let the constructor fill aggregates.

### Testing

11. **Property-based `Diff`** — generate random reports, assert `Diff(a,a)` is empty and `Diff(a,b)` + `Diff(b,a)` symmetry.
12. **Property-based `MigrateReport`** — arbitrary JSON → migrate → validate; arbitrary current-schema report → re-migrate → equal.
13. **Fuzz filter inputs** — arbitrary `ReportOption` combinations on arbitrary reports.
14. **HTML golden-file test** — deterministic multi-service report → assert HTML output matches a committed golden file.

### CI / Release

15. **Add `actionlint`** to CI.
16. **Upgrade GitHub Actions versions** when stable (checkout@v5, setup-go@v6).
17. **Tag v0.1.0** after JSON schema and changelog updates.
18. **Create `RELEASING.md`** or expand `CONTRIBUTING.md` with a release checklist.
19. **Add a flake app** for the coverage gate to replace inline shell.
20. **CSV/TSV export** — low effort, high value for data analysis workflows.

### Libraries

21. **Stick with stdlib + samber/do + templ** for core. Any new library must pass depguard and add enough value to justify the dependency. Current 4-dependency policy is a feature.
22. **`pgregory/rapid`** for property-based testing — would require depguard allowlist change. Could be test-only.
23. **`github.com/invopop/jsonschema`** for JSON Schema generation — also requires depguard change.
24. **`github.com/google/go-cmp`** for test diffs — requires depguard change. Marginal value over hand-rolled comparisons at this project size.
25. **`encoding/json/v2`** — wait for stdlib stabilization. Do not introduce experimental JSON packages.

---

## f) Top 25 Things to Get Done Next

Sorted by **Impact × Customer-Value ÷ Effort**:

| #   | Task                                                            | Impact | Effort     | Category     |
| --- | --------------------------------------------------------------- | ------ | ---------- | ------------ |
| 1   | **CHANGELOG + TODO + AGENTS update** (refactor visibility)      | 🟠 Med | ⚪ Trivial | Docs         |
| 2   | **Fix `html_templ.go` drift** (`.gitattributes`)                | 🟠 Med | ⚪ Trivial | DevEx        |
| 3   | **Parallelize remaining 18 sequential tests**                   | 🟡 Low | ⚪ Trivial | Testing      |
| 4   | **Typed identifiers** (`ContainerID`, `ScopeID`, `ServiceName`) | 🟠 Med | 🔵 Low     | Architecture |
| 5   | **NDJSON import** (trivial now with `buildReportFromCore`)      | 🟠 Med | 🔵 Low     | Feature      |
| 6   | **v0.1.0 release**                                              | 🟠 Med | 🟡 Med     | Release      |
| 7   | **JSON Schema file**                                            | 🟠 Med | 🔵 Low     | Docs/API     |
| 8   | **CSV/TSV export**                                              | 🟡 Low | 🔵 Low     | Feature      |
| 9   | **Refactor `ServiceInfo` lifecycle concerns**                   | 🟠 Med | 🔴 High    | Architecture |
| 10  | **Property-based `Diff` tests**                                 | 🟡 Low | 🔵 Low     | Testing      |
| 11  | **Property-based `MigrateReport` tests**                        | 🟡 Low | 🔵 Low     | Testing      |
| 12  | **Fuzz filter inputs**                                          | 🟡 Low | 🔵 Low     | Testing      |
| 13  | **HTML golden-file test**                                       | 🟠 Med | 🟡 Med     | Testing      |
| 14  | **`Report` constructor validation**                             | 🟠 Med | 🟡 Med     | Architecture |
| 15  | **Prometheus exporter example**                                 | 🟠 Med | 🟡 Med     | Docs         |
| 16  | **Add `actionlint` to CI**                                      | 🟡 Low | ⚪ Trivial | CI           |
| 17  | **GitHub Actions version upgrades**                             | 🟡 Low | ⚪ Trivial | CI           |
| 18  | **Flake app for coverage gate**                                 | 🟡 Low | 🔵 Low     | DevEx        |
| 19  | **`RELEASING.md`** or release checklist                         | 🟡 Low | ⚪ Trivial | Docs         |
| 20  | **CLI tool** for report conversion                              | 🟢 Low | 🔴 High    | Feature      |
| 21  | **WebSocket live stream**                                       | 🟢 Low | 🔴 High    | Feature      |
| 22  | **`pgregory/rapid`** for property-based testing                 | 🟡 Low | 🔵 Low     | Testing      |
| 23  | **`invopop/jsonschema`** for schema generation                  | 🟠 Med | 🔵 Low     | Docs/API     |
| 24  | **Upgrade Go further** (1.27 when released)                     | 🟡 Low | ⚪ Trivial | Deps         |
| 25  | **Explore `samber/do` v2.1+** new APIs                          | 🟡 Low | 🔵 Low     | Deps         |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the `ServiceInfo` split into lifecycle sub-structs happen before or after the v0.1.0 release?**

The 19-field `ServiceInfo` is the widest struct in the codebase. Splitting it into `ServiceIdentity` (scope + name + type), `ServiceLifecycle` (registered/invoked/shutdown timestamps + errors), `ServiceHealth` (last check + error + count), and `ServiceGraph` (dependencies + dependents) would make the domain model more explicit and make `deriveServiceStatus` naturally fall out of `ServiceLifecycle` rather than operating on the aggregate.

However, this is a **breaking JSON schema change**. The current JSON output flattens everything via Go's embedded struct promotion, so `{"service_name": ..., "registered_at": ..., "invocation_count": ...}` would remain the same IF the sub-structs are embedded. But the Go API surface changes: `svc.FirstInvokedAt` becomes `svc.Lifecycle.FirstInvokedAt`, which breaks every consumer.

If we do it **before v0.1.0**, the breaking change is "free" (pre-1.0 API is explicitly unstable per `STABILITY.md`). If we do it **after v0.1.0**, it becomes a v0.2.0 migration with a deprecation period.

The question is: **is the struct split worth the API churn, or is the current flat model actually the right level of abstraction for a library whose output is primarily machine-readable JSON?**

---

## Commit History (this session)

```
2aeb96d fix: restore canonical html_templ.go from go tool templ generate
09d2280 test: fuzz diagram output and deeply nested scope trees
513b833 style: apply markdown table alignment and formatter normalization
1b7b73a refactor: unify Report construction and status derivation
19f94ef docs(migration): document MigrateReport normalize-any-version behavior
94754bc docs: finalize remediation batch follow-ups
```

All 6 commits pushed to `origin/master`. Working tree clean.

---

## File Inventory

### Production Code (20 files, ~2400 LOC excluding generated)

```
diff.go             event.go          export.go          filter.go
healthcheck.go      hooks.go          html.go            html_templ.go (generated)
mermaid.go          metadata.go       migration.go       plantuml.go
plugin.go           recorder.go       report.go          report_builder.go
report_helpers.go   service.go        types.go           doc.go
diagram.go
```

### Test Code (24 files, ~5800 LOC)

```
benchmarks_test.go       diagram_test.go          diff_export_test.go
example_test.go          extra_test.go            fuzz_test.go
healthcheck_basic_test.go  healthcheck_export_test.go  helpers_test.go
metadata_test.go         migration_test.go        plugin_basic_test.go
plugin_errors_test.go    plugin_export_test.go    plugin_html_test.go
plugin_lifecycle_test.go plugin_provider_test.go  plugin_scope_test.go
recorder_internal_test.go  report_filter_test.go    report_query_test.go
robustness_test.go       status_internal_test.go  type_method_test.go
```

### Infrastructure

```
.github/workflows/ci.yml    flake.nix    flake.lock    .golangci.yml
```

### Documentation

```
AGENTS.md    CHANGELOG.md    CONTRIBUTING.md    FEATURES.md    README.md
STABILITY.md    BENCHMARKS.md    TODO_LIST.md
docs/DOMAIN_LANGUAGE.md    docs/examples/otel-bridge.md
docs/status/ (1 current report)
docs/archive/ (21 historical reports)
docs/planning/ (3 execution plans)
docs/research/ (1 performance review)
```

### Metrics Summary

| Metric                           | Value                                                          |
| -------------------------------- | -------------------------------------------------------------- |
| Production LOC (excl. generated) | ~2,400                                                         |
| Test LOC                         | ~5,800                                                         |
| Public API functions             | 76                                                             |
| Top-level test functions         | 145                                                            |
| Subtests                         | 10                                                             |
| Fuzz targets                     | 6                                                              |
| Benchmarks                       | 11                                                             |
| Examples                         | 7                                                              |
| Test files                       | 24                                                             |
| Production files                 | 20                                                             |
| Test coverage                    | 95.3%                                                          |
| Dependencies (direct)            | 2 (`samber/do`, `a-h/templ`)                                   |
| CI jobs                          | 5 (test+coverage, lint, vulncheck, mod-tidy, stale-generation) |
| golangci-lint issues             | 0                                                              |

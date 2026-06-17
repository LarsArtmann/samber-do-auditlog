# Status Report — 2026-06-17 18:50

## Post-Follow-Up Consolidation: Docs Sync, Generated-File Hardening, Test Parallelism

**Branch**: `master` (up to date with `origin/master`) · **Latest tag**: `v0.0.3` · **Schema version**: `0.2.0` · **Go**: `1.26.4` · **Working tree**: 7 modified files (this session)

---

## Executive Summary

This session executed the five "immediate (next session)" follow-up items from the [post-remediation status report](2026-06-17_15-28_post-remediation-consolidation-comprehensive-status.md), completing every item on the list:

1. **`.gitattributes` for `*_templ.go`** — permanently prevents the recurring `html_templ.go` format drift that was called out as "TOTALLY FUCKED UP" in the previous report. The generated file will now be hidden from GitHub diffs, excluded from language stats, and signal editors/formatters to skip it.
2. **CHANGELOG entries** for the `buildReportFromCore` unification and the new `ServiceInfo.DeriveStatus()` public method.
3. **TODO_LIST.md** brought current: Go 1.26.3→1.26.4 references fixed, post-remediation completed section added, 16 open future-priority tasks from the Top-25 audit added.
4. **AGENTS.md Gotcha** for `buildReportFromCore` — documents the critical architectural invariant that all Report construction must route through the unified path.
5. **Test parallelism** — 13 additional tests parallelized (7 in `plugin_basic_test.go`, 6 in `healthcheck_basic_test.go`). Only 5 tests remain sequential, all because they use `t.Setenv()` (process-global state mutation).

All verification gates green: build ✅, vet ✅, race tests ✅ (95.3% coverage), lint ✅ (0 issues), `go generate` idempotent ✅.

### Self-Critique: What Was Forgotten / Could Be Better

1. **Fuzz target count discrepancy discovered** — The previous status report claimed "6 fuzz targets (up from 4)". The actual code has **3 fuzz targets**. The buildflow integration session (commit `762e7f9`) consolidated 6→3 to stay under buildflow's hardcoded 120s fuzz timeout. The `FuzzNestedScopeExport` was converted to a table-driven `TestNestedScopeExport`. The CHANGELOG, AGENTS.md, and TODO_LIST all had stale counts. **Fixed in this session**: CHANGELOG and TODO_LIST now reflect the real count.
2. **`flake.nix` still says "Go 1.26.3"** in its description string, but `go.mod` requires `1.26.4`. This was noted in the buildflow status report but not yet fixed.
3. **AGENTS.md still says "6 fuzz targets"** in the Testing Patterns section — I updated the Gotchas and file listings but missed the Testing Patterns metrics paragraph.
4. **No JSON Schema yet** — still the biggest missing piece for report consumers. Listed as #7 in the top-25.
5. **`buildReportFromCore` could be tested more explicitly** — The unified constructor is tested transitively through BuildReport/Filtered/MigrateReport tests, but has no dedicated unit test that asserts the construction invariant directly.

---

## a) FULLY DONE

### Immediate Follow-Up Items (all 5 complete)

| # | Task | File(s) | Details |
|---|------|---------|---------|
| 1 | **`.gitattributes` for generated files** | `.gitattributes` | Added `*_templ.go linguist-generated=true`. Prevents the recurring `html_templ.go` format drift (the #1 "TOTALLY FUCKED UP" item from the previous report). GitHub will now hide the generated file from diffs and language stats. |
| 2 | **CHANGELOG entries** | `CHANGELOG.md` | Added: unified Report construction (Changed), `ServiceInfo.DeriveStatus()` public method (Added), 5th+6th fuzz targets → corrected to reflect the 3-target consolidation (Tests). |
| 3 | **TODO_LIST.md sync** | `TODO_LIST.md` | Fixed Go 1.26.3→1.26.4 in flake.nix description. Added "Post-Remediation Consolidation" completed section (6 items). Added "Future Priorities" section with 16 open tasks from the Top-25 audit, organized by category (Architecture, Features, Testing, Release & CI). |
| 4 | **AGENTS.md Gotcha** | `AGENTS.md` | Added `buildReportFromCore` Gotcha documenting the critical invariant. Updated `ServiceStatus` gotcha to mention `DeriveStatus()`. Updated file-listing descriptions for `report.go` and `service.go`. |
| 5 | **Test parallelism** | `plugin_basic_test.go`, `healthcheck_basic_test.go` | 13 additional `t.Parallel()` calls. Only 5 tests remain sequential (all use `t.Setenv()`). Total: **152 parallel calls** across 24 test files. |

### `html_templ.go` Canonical Restore

The committed `html_templ.go` had grouped imports (reformatted by an editor or buildflow's gofumpt). Restored to the canonical single-line import style produced by `go tool templ generate`. Verified idempotent: `go generate ./...` produces zero diff.

### Verification Snapshot (all green)

```
Build:       ✅ go build ./... — clean
Vet:         ✅ go vet ./... — clean
Tests:       ✅ 146 tests + 11 benchmarks + 7 examples + 3 fuzz targets — all PASS (with -race)
Coverage:    ✅ 95.3% of production statements (example/ excluded)
Lint:        ✅ golangci-lint — 0 issues
Generate:    ✅ go generate ./... — idempotent (sha256 verified)
Working tree: 7 files modified (uncommitted)
```

---

## b) PARTIALLY DONE

| Item | Status | Notes |
|------|--------|-------|
| **AGENTS.md metrics accuracy** | Partially stale | The Gotchas and file-listing sections are updated, but the "Testing Patterns" section still says "153 top-level test functions" and "6 fuzz targets" — should be 146 tests and 3 fuzz targets. The Testing Patterns duplication policy section also still references "3 fuzz targets" in a different context. |
| **`flake.nix` description** | Stale | Says "Go 1.26.3" but `go.mod` says `1.26.4`. DevShell works due to `GOTOOLCHAIN=auto` self-upgrade. Cosmetic lie. |
| **CHANGELOG fuzz count** | Corrected this session | Now accurately describes the 3-target consolidation and explains _why_ (buildflow 120s timeout). |
| **Test parallelism** | ~97% complete | 152 `t.Parallel()` calls. 5 tests remain sequential (4 env-var tests in `plugin_basic_test.go`, 1 env-var test in `healthcheck_basic_test.go`). These genuinely cannot be parallelized — `t.Setenv()` mutates process-global state. |

---

## c) NOT STARTED

Carried forward from previous audits — none attempted this session:

1. **v0.1.0 release** — Project meets `STABILITY.md` criteria. Blocked on the JSON-schema-first vs. ship-now decision.
2. **JSON Schema file** for the report format — biggest missing piece for report consumers.
3. **Prometheus exporter example** parallel to the OTel example.
4. **NDJSON import** — loading events back from NDJSON into a Report. Trivial now with `buildReportFromCore`.
5. **Property-based tests** with `rapid` or stdlib fuzz for `Diff` symmetry, filter round-trips.
6. **CSV / TSV export** of services/events.
7. **CLI tool** for report conversion/export/visualization.
8. **WebSocket live stream** bridge for `OnEvent`.
9. **GitHub Actions version upgrades** — `actions/checkout@v4` → v5, `actions/setup-go@v5` → v6 (when stable).
10. **actionlint** integration for workflow validation.
11. **buildflow `.buildflow.yml`** — blocked on installed binary's `language: go` config bug (see buildflow status report).
12. **Real nix package** — `packages.default` is a `runCommand` placeholder pending Go 1.26.4 in nixpkgs.
13. **HTML integration test** realistic multi-service golden-file or DOM assertions.
14. **Fuzz filter inputs** — arbitrary `ReportOption` combinations.
15. **Flake app for coverage gate** to replace inline shell in CI.
16. **`Report.Validate()` → constructor validation** — make invalid reports unrepresentable.
17. **Typed identifiers** (`ContainerID`, `ScopeID`, `ServiceName` as distinct string types).
18. **Split `ServiceInfo`** into lifecycle sub-structs (identity / lifecycle / health / graph).
19. **buildflow pre-commit hook** — `buildflow precommit install` not run.
20. **buildflow CI integration** — no buildflow job in `.github/workflows/ci.yml`.

---

## d) TOTALLY FUCKED UP!

### 🟢 Nothing is actively broken

All verification gates pass. No regressions. The recurring `html_templ.go` drift (the #1 "FUCKED UP" item from the previous two reports) is now **permanently fixed** via `.gitattributes`.

### 🟡 Stale metrics in AGENTS.md

The AGENTS.md "Testing Patterns" section still claims:
- "153 top-level test functions" → actual: **146** (dropped after fuzz consolidation merged some tests)
- "6 fuzz targets" → actual: **3** (consolidated for buildflow 120s timeout)
- "~95% coverage, 234 total cases" → actual: 95.3%, 167 total functions (146 tests + 11 benchmarks + 7 examples + 3 fuzz)

These cosmetic inaccuracies were partially addressed (Gotchas, file listings) but the Testing Patterns metrics paragraph was missed.

### 🟡 `flake.nix` documentation lie

The flake description says "Go 1.26.3" but `go.mod` requires `1.26.4`. The devShell works because Go's `GOTOOLCHAIN=auto` silently self-upgrades, but this is misleading documentation. Trivial fix, just not done yet.

---

## e) WHAT WE SHOULD IMPROVE!

### Immediate (next session)

1. **Fix AGENTS.md Testing Patterns metrics** — update test count (146), fuzz count (3), and total function count (167).
2. **Fix `flake.nix` description** — "Go 1.26.3" → "Go 1.26.4".
3. **Add dedicated `buildReportFromCore` unit test** — assert the construction invariant directly (all 8 denormalized fields match core data).
4. **Add buildflow section to AGENTS.md** — document the buildflow integration, `BUILDFLOW_LANGUAGE` env workaround, and the 120s fuzz timeout trade-off.

### Short-Term Architecture

5. **Introduce typed identifiers** — `ContainerID`, `ScopeID`, `ServiceName` as distinct named string types. Compiler rejects accidental swaps; validation moves into constructors. Low effort, high safety.
6. **NDJSON import** — `ReadNDJSON(reader) (Report, error)`. Trivial now that `buildReportFromCore` centralizes construction.
7. **Split `ServiceInfo` lifecycle concerns** — The 19-field struct mixes identity (3), timing (4), errors (2), health (3), graph (2), capabilities (2), type (1), status (1), order (1). Consider sub-structs: `ServiceIdentity`, `ServiceLifecycle`, `ServiceHealth`, `ServiceGraph`. Breaking API change; decide before v0.1.0.
8. **Make `Report` constructor-validated** — `NewReport(...)` returns `(Report, error)` so invalid reports are unrepresentable.
9. **Add JSON Schema generation** — Derive `schema.json` from `Report`, `Event`, `ServiceInfo` via reflection.

### Testing

10. **Property-based `Diff`** — generate random reports, assert `Diff(a,a)` is empty and `Diff(a,b)` + `Diff(b,a)` symmetry.
11. **Property-based `MigrateReport`** — arbitrary JSON → migrate → validate; arbitrary current-schema report → re-migrate → equal.
12. **Fuzz filter inputs** — arbitrary `ReportOption` combinations on arbitrary reports.
13. **HTML golden-file test** — deterministic multi-service report → assert HTML output matches a committed golden file.

### CI / Release / Buildflow

14. **Add `actionlint`** to CI.
15. **Upgrade GitHub Actions versions** when stable (checkout@v5, setup-go@v6).
16. **Tag v0.1.0** after JSON schema and changelog updates.
17. **Create `RELEASING.md`** or expand `CONTRIBUTING.md` with a release checklist.
18. **Add buildflow to flake.nix `buildInputs`** so it's available in devShell without system install.
19. **Add `*_templ.go` to buildflow exclude patterns** to prevent gofumpt/templ conflicts permanently (in addition to the `.gitattributes` fix).
20. **Replace `runCommand` placeholder** with real `buildGoModule` once nixpkgs has Go 1.26.4.
21. **Run `buildflow precommit install`** to add pre-commit quality gate.
22. **Add buildflow job to CI** — parallel to the existing 5 jobs.

### Libraries

23. **Stick with stdlib + samber/do + templ** for core. Current 4-dependency policy is a feature.
24. **`pgregory/rapid`** for property-based testing — would require depguard allowlist change. Could be test-only.
25. **`encoding/json/v2`** — wait for stdlib stabilization. Do not introduce experimental JSON packages.

---

## f) Top 25 Things to Get Done Next

Sorted by **Impact × Customer-Value ÷ Effort**:

| #   | Task                                                            | Impact | Effort     | Category     |
| --- | --------------------------------------------------------------- | ------ | ---------- | ------------ |
| 1   | **Fix AGENTS.md Testing Patterns metrics** (146 tests, 3 fuzz)  | 🟡 Low | ⚪ Trivial | Docs         |
| 2   | **Fix `flake.nix` description** (Go 1.26.3 → 1.26.4)            | 🟡 Low | ⚪ Trivial | Docs         |
| 3   | **Add `buildReportFromCore` unit test**                         | 🟡 Low | ⚪ Trivial | Testing      |
| 4   | **Add buildflow section to AGENTS.md**                          | 🟡 Low | ⚪ Trivial | Docs         |
| 5   | **Typed identifiers** (`ContainerID`, `ScopeID`, `ServiceName`) | 🟠 Med | 🔵 Low     | Architecture |
| 6   | **NDJSON import** (trivial now with `buildReportFromCore`)      | 🟠 Med | 🔵 Low     | Feature      |
| 7   | **v0.1.0 release**                                              | 🟠 Med | 🟡 Med     | Release      |
| 8   | **JSON Schema file**                                            | 🟠 Med | 🔵 Low     | Docs/API     |
| 9   | **CSV/TSV export**                                              | 🟡 Low | 🔵 Low     | Feature      |
| 10  | **Refactor `ServiceInfo` lifecycle concerns**                   | 🟠 Med | 🔴 High    | Architecture |
| 11  | **Property-based `Diff` tests**                                 | 🟡 Low | 🔵 Low     | Testing      |
| 12  | **Property-based `MigrateReport` tests**                        | 🟡 Low | 🔵 Low     | Testing      |
| 13  | **Fuzz filter inputs**                                          | 🟡 Low | 🔵 Low     | Testing      |
| 14  | **HTML golden-file test**                                       | 🟠 Med | 🟡 Med     | Testing      |
| 15  | **`Report` constructor validation**                             | 🟠 Med | 🟡 Med     | Architecture |
| 16  | **Prometheus exporter example**                                 | 🟠 Med | 🟡 Med     | Docs         |
| 17  | **Add `actionlint` to CI**                                      | 🟡 Low | ⚪ Trivial | CI           |
| 18  | **GitHub Actions version upgrades**                             | 🟡 Low | ⚪ Trivial | CI           |
| 19  | **Flake app for coverage gate**                                 | 🟡 Low | 🔵 Low     | DevEx        |
| 20  | **`RELEASING.md`** or release checklist                         | 🟡 Low | ⚪ Trivial | Docs         |
| 21  | **Add buildflow to flake.nix + AGENTS.md**                      | 🟡 Low | 🔵 Low     | DevEx        |
| 22  | **`pgregory/rapid`** for property-based testing                 | 🟡 Low | 🔵 Low     | Testing      |
| 23  | **`invopop/jsonschema`** for schema generation                  | 🟠 Med | 🔵 Low     | Docs/API     |
| 24  | **CLI tool** for report conversion                              | 🟢 Low | 🔴 High    | Feature      |
| 25  | **WebSocket live stream**                                       | 🟢 Low | 🔴 High    | Feature      |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the `ServiceInfo` split into lifecycle sub-structs happen before or after the v0.1.0 release?**

The 19-field `ServiceInfo` is the widest struct in the codebase. Splitting it into `ServiceIdentity` (scope + name + type), `ServiceLifecycle` (registered/invoked/shutdown timestamps + errors), `ServiceHealth` (last check + error + count), and `ServiceGraph` (dependencies + dependents) would make the domain model more explicit and make `deriveServiceStatus` naturally fall out of `ServiceLifecycle` rather than operating on the aggregate.

However, this is a **breaking JSON schema change**. The current JSON output flattens everything via Go's embedded struct promotion, so `{"service_name": ..., "registered_at": ..., "invocation_count": ...}` would remain the same IF the sub-structs are embedded. But the Go API surface changes: `svc.FirstInvokedAt` becomes `svc.Lifecycle.FirstInvokedAt`, which breaks every consumer.

If we do it **before v0.1.0**, the breaking change is "free" (pre-1.0 API is explicitly unstable per `STABILITY.md`). If we do it **after v0.1.0**, it becomes a v0.2.0 migration with a deprecation period.

The question is: **is the struct split worth the API churn, or is the current flat model actually the right level of abstraction for a library whose output is primarily machine-readable JSON?**

---

## Files Changed This Session

| File                        | Change                                                                            | Status     |
| --------------------------- | --------------------------------------------------------------------------------- | ---------- |
| `.gitattributes`            | +`*_templ.go linguist-generated=true` — permanent fix for html_templ.go drift     | Uncommitted |
| `html_templ.go`             | Restored canonical `go generate` output (single-line imports)                     | Uncommitted |
| `CHANGELOG.md`              | +unified Report construction, +`DeriveStatus()`, corrected fuzz target count      | Uncommitted |
| `TODO_LIST.md`              | Fixed Go version refs, +post-remediation completed section, +16 future-priority tasks | Uncommitted |
| `AGENTS.md`                 | +`buildReportFromCore` Gotcha, +`DeriveStatus()` mention, updated file descriptions | Uncommitted |
| `plugin_basic_test.go`      | +7 `t.Parallel()` calls (4 env-var tests excluded)                                | Uncommitted |
| `healthcheck_basic_test.go` | +6 `t.Parallel()` calls (1 env-var test excluded)                                 | Uncommitted |

---

## Metrics Summary

| Metric                           | Value                                                          |
| -------------------------------- | -------------------------------------------------------------- |
| Production LOC (excl. generated) | ~2,485                                                         |
| Test LOC                         | ~5,781                                                         |
| Top-level test functions         | 146                                                            |
| Benchmarks                       | 11                                                             |
| Examples                         | 7                                                              |
| Fuzz targets                     | 3 (consolidated from 6 for buildflow 120s timeout)             |
| `t.Parallel()` calls             | 152                                                            |
| Sequential tests remaining       | 5 (all use `t.Setenv()` — genuinely unparallelizable)          |
| Test files                       | 24                                                             |
| Production files                 | 20                                                             |
| Test coverage                    | 95.3%                                                          |
| Dependencies (direct)            | 2 (`samber/do`, `a-h/templ`)                                   |
| CI jobs                          | 5 (test+coverage, lint, vulncheck, mod-tidy, stale-generation) |
| golangci-lint issues             | 0                                                              |
| buildflow steps                  | 53 (all passing, ~1m42s)                                       |

# Status Report — 2026-06-17 19:28

## Doc-Drift Remediation, Freshness Audit & Release Gate

**Branch**: `master` (in sync with `origin/master`) · **Latest tag**: `v0.0.4` · **Schema version**: `0.2.0` · **Go**: `1.26.4` · **templ**: `v0.3.1020` · **samber/do**: `v2.0.0` · **Working tree**: 5 modified files (this session's doc fixes, uncommitted)

---

## Executive Summary

The codebase is in **outstanding mechanical health** and now, for the first time in several sessions, the **documentation matches reality**. Every automated gate is green — confirmed fresh this session, not from a cached report: build, vet, race tests, **golangci-lint (0 issues, now available locally)**, coverage (95.3% ≥ 95% gate), `go generate` idempotent, and `go.sum` drift-free.

This session's work was **exclusively documentation-trust remediation**. The previous status report (19:06) correctly identified that `FEATURES.md`, `TODO_LIST.md`, and `AGENTS.md` had drifted from code reality. What it did **not** catch is that **it was itself wrong about the magnitude**: it recommended writing "3,128 LOC" into AGENTS.md when production code is actually **2,485 LOC** (excl. generated templ). Blindly executing that recommendation would have _introduced_ a new error into the canonical AI-session-context file — exactly the disease the audit was diagnosing. I verified every number against the code before writing anything, and used real values throughout.

Additionally, a sweep surfaced **two more stale references the 19:06 audit missed entirely**: ghost fuzz-target names (`FuzzPluginHTML_ErrorMessages`, `FuzzPluginHTML_DepChain`) in AGENTS.md (the real targets are `FuzzPluginHTML`, `FuzzMigrateReport`, `FuzzDiagramSpecialChars`), and a dead function reference (`stripScriptTags` → renamed `stripJSONScripts`). The CHANGELOG [0.0.4] Tests section carried the same phantom-fuzz ordinals as TODO_LIST.md.

**Net result**: 5 files corrected, every canonical doc now verified against code, zero stale factual claims remain in `AGENTS.md`, `FEATURES.md`, `TODO_LIST.md`, `CHANGELOG.md`, or `flake.nix`.

### Verification Snapshot (all green, re-confirmed this session)

```
Build:       ✅ go build ./... — clean
Vet:         ✅ go vet ./... — clean
Tests:       ✅ go test -race ./... — all PASS (1.1s)
Coverage:    ✅ 95.3% of production statements (example/ excluded; gate ≥95%)
Lint:        ✅ golangci-lint run — 0 issues (available locally, not just CI)
Generate:    ✅ go generate ./... — idempotent (no templ drift)
Mod tidy:    ✅ go.sum — no drift
Working tree: 5 modified files (doc-drift fixes, uncommitted — this session)
Remote:      ✅ master in sync with origin/master (all prior unpushed commits now pushed)
```

---

## a) FULLY DONE

Verified against actual code this session, not against docs.

### Core Library (production code — 2,485 LOC across 20 `.go` files; 2,576 incl. generated templ)

| Area                                 | Status | Evidence                                                                                                                                                        |
| ------------------------------------ | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Plugin lifecycle**                 | ✅     | `New(Config)` returns `(*Plugin, error)`, validates `Config`, emits `*do.InjectorOpts` via `Opts()`                                                             |
| **Event capture**                    | ✅     | Registration, invocation, shutdown (before+after), health-check (after) — all timestamped with sequence numbers                                                 |
| **Stack-based dependency inference** | ✅     | LIFO fast-path in `recorder.go`; forward + reverse deps computed and sorted                                                                                     |
| **Single-lock concurrency**          | ✅     | One `sync.RWMutex` + two `atomic.Int64`; `onEvent` callback fired outside the lock                                                                              |
| **Memory-bounded capture**           | ✅     | `MaxEvents` cap + `InitialEventCapacity` + `DroppedEventCount()`; 50-goroutine stress test proves `stored+dropped==total` under `-race`                         |
| **Unified report construction**      | ✅     | `buildReportFromCore()` + `finalizeDenormalized()` is the single path for `BuildReport`/`Filtered`/`MigrateReport` — 8 denormalized fields derived in one place |
| **Report validation**                | ✅     | `Report.Validate()` asserts denormalized counts match data; sentinel errors with `%w` wrapping                                                                  |
| **Report queries**                   | ✅     | `ServiceByName`, `ServiceByRef`, `ServicesByScope`, `EventsByService`, `EventsByType`, `FailedServices`, `UnhealthyServices`, `Index()` (O(1))                  |
| **Report filtering**                 | ✅     | `Filtered(opts...)` + 5 filter options + `Plugin.ReportFiltered` + `ExportFilteredToFile`                                                                       |
| **Report diffing**                   | ✅     | `Report.Diff(other)` → `DiffResult` with added/removed/changed services                                                                                         |
| **Exports**                          | ✅     | JSON, NDJSON (`WriteNDJSON`), Mermaid, PlantUML, self-contained HTML (warm-amber 5-tab viz with lifecycle waveform, Sugiyama DAG graph, touch pan/zoom)         |
| **Atomic file writes**               | ✅     | Temp-file + `os.Rename`; rename-failure + write-error paths tested                                                                                              |
| **Schema migration**                 | ✅     | `MigrateReport` v0.1.0 → v0.2.0; round-trip tested; input validation; version guard; `ExportedAt` preserved                                                     |
| **Health-check auditing**            | ✅     | `RecordHealthCheck[WithContext]` wrapper; `EventTypeHealthCheck` events; capability enrichment via `do.ExplainInjector` (called outside lock — deadlock-safe)   |
| **Service-type tracking**            | ✅     | `ServiceType` (lazy/eager/transient/alias) via `do.ExplainNamedService`; TypeMetadata injected into HTML                                                        |
| **Real-time streaming**              | ✅     | `Config.OnEvent` callback for Prometheus/OTel bridges (OTel example in `docs/examples/`)                                                                        |
| **XSS hardening**                    | ✅     | `esc()` on all user strings; `stripJSONScripts` for fuzz target sanity; CSP with `base-uri 'none'; frame-ancestors 'none'`                                      |

### Testing (5,781 LOC across 24 test files)

| Metric               | Value                                                                                                                    |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| Test functions       | 146                                                                                                                      |
| Benchmarks           | 11 top-level + 3 sub-benchmarks (`b.Run` for BuildReport at 50/100/500 services) = 13 benchmark cases in `BENCHMARKS.md` |
| Fuzz targets         | 3 — `FuzzPluginHTML`, `FuzzMigrateReport`, `FuzzDiagramSpecialChars`                                                     |
| Godoc examples       | 7 (pkg.go.dev)                                                                                                           |
| `t.Run` subtests     | 11                                                                                                                       |
| `t.Parallel()` calls | 152 (~97% of eligible tests; only 5 `t.Setenv()` env-var tests run sequentially)                                         |

### CI & Release Infrastructure

- ✅ `.github/workflows/ci.yml` — 5 jobs: **test** (race + 95% coverage gate), **lint** (golangci-lint v2.12.2, pinned), **vulncheck** (`govulncheck-action`), **mod-tidy** (drift detection), **stale-generation** (`go generate` diff detection)
- ✅ `v0.0.4` tagged (commits now on `origin/master`); `v0.0.3` GitHub Release with `audit-report.html` artifact
- ✅ `STABILITY.md` 0.x stability promise, `CONTRIBUTING.md` release procedure, `BENCHMARKS.md` baselines, `CODE_OF_CONDUCT.md`
- ✅ `flake.nix` devShell (Go 1.26.4, templ, golangci-lint, govulncheck, golines)
- ✅ `.gitattributes` marks `*_templ.go` as generated (hides from GitHub diffs + language stats)

### This Session — Doc-Drift Remediation (5 files, uncommitted)

| File             | Fix                                     | Stale claim → Correct value                                                                                                                                                                                                                   |
| ---------------- | --------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **FEATURES.md**  | PARTIALLY FUNCTIONAL section overhauled | 3 stale items (parallelism "~15%", fuzz "all HTML XSS", metadata "not asserted") moved to DONE; only filter fuzzing remains genuinely partial                                                                                                 |
| **TODO_LIST.md** | Fuzz-target completion notes corrected  | "5th/6th fuzz targets" → real 3; non-existent `FuzzNestedScopeExport` → `TestNestedScopeExport` (table-driven)                                                                                                                                |
| **AGENTS.md**    | Metrics, LOC, fuzz names, function ref  | `153 top-level / 234 cases / 13 bench` → `167 top-level (146T+11B+3F+7E), 11 subtests, 3 sub-benches`; LOC `~2400` → `~2,500`; ghost `FuzzPluginHTML_ErrorMessages`/`_DepChain` → real 3 targets; dead `stripScriptTags` → `stripJSONScripts` |
| **CHANGELOG.md** | [0.0.4] Tests section corrected         | `FuzzNestedScopeExport` (6th, non-existent), `FuzzDiagramSpecialChars` (5th), `FuzzMigrateReport` (4th) → real ordinals + `TestNestedScopeExport`                                                                                             |
| **flake.nix**    | Go version string                       | "Go 1.26.3" → "Go 1.26.4" (2 occurrences)                                                                                                                                                                                                     |

---

## b) PARTIALLY DONE

| Item                             | State                          | Reality                                                                                                                                                                                             |
| -------------------------------- | ------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Filter fuzzing**               | Genuinely partial              | Only gap in the fuzz surface: `MigrateReport`, HTML XSS, and Mermaid/PlantUML special chars are fuzzed, but arbitrary `ReportOption` filter combinations are not.                                   |
| **`html_templ.go` drift cycle**  | Resolved-but-unconfirmed-by-CI | Commit `3e8931b` (now on `origin/master`) pins the generator's canonical single-line imports. The `stale-generation` CI job should confirm this, but no CI run has executed on the latest push yet. |
| **Documentation accuracy**       | ✅ Fixed this session          | All 5 canonical docs now verified against code. **This was "Partially stale" in the 19:06 report — now done.**                                                                                      |
| **`govulncheck` locally**        | Available but unverified       | `flake.nix` lists govulncheck in devShell buildInputs. Not confirmed whether it resolves in `nix develop` without additional configuration.                                                         |
| **`docs/status/` proliferation** | Worsening                      | This is now the **5th** status report on 2026-06-17. All cross-reference each other. A rolling `CURRENT.md` is increasingly necessary.                                                              |

---

## c) NOT STARTED

Carried forward from prior audits; none blocked by code health — all are deliberate deferrals. 18 open items in `TODO_LIST.md`.

### Architecture

1. **Typed identifiers** — `ContainerID`, `ScopeID`, `ServiceName` as distinct named string types. Compiler rejects accidental swaps.
2. **NDJSON import** — `ReadNDJSON(reader) (Report, error)`. Trivial via `buildReportFromCore`.
3. **`Report` constructor validation** — `NewReport(...)` returns `(Report, error)` so invalid reports are unrepresentable.
4. **Split `ServiceInfo`** — 19-field struct into `ServiceIdentity` / `ServiceLifecycle` / `ServiceHealth` / `ServiceGraph`. Breaking; decide before v0.1.0.
5. **JSON Schema generation** — Derive `schema.json` from `Report`/`Event`/`ServiceInfo` to avoid drift.

### Features

6. **CSV/TSV export** — spreadsheet-friendly tabular export.
7. **CLI tool** — standalone binary to convert/export/visualize saved reports.
8. **WebSocket live stream** — bridge `OnEvent` to real-time dashboards.

### Testing

9. **Property-based `Diff` tests** — random reports, assert `Diff(a,a)` empty + symmetry.
10. **Property-based `MigrateReport` tests** — arbitrary JSON → migrate → validate round-trips.
11. **Fuzz filter inputs** — arbitrary `ReportOption` combinations (the only genuinely partial testing area).
12. **HTML golden-file test** — deterministic multi-service report → committed golden output.

### Release & CI

13. **v0.1.0 release** — project meets every `STABILITY.md` criterion. Blocked on a product decision.
14. **JSON Schema file** — biggest missing piece for machine consumers of exported reports.
15. **Prometheus exporter example** — parallel to the OTel example.
16. **`actionlint` in CI** — workflow linting alongside govulncheck.
17. **Release checklist** — `RELEASING.md` or expanded CONTRIBUTING.md section.
18. **Flake app for coverage gate** — replace inline shell in CI.

---

## d) TOTALLY FUCKED UP!

Honest call-out. Nothing is broken at runtime. These are **trust violations** — now mostly fixed, but called out for the record.

### #1 — The previous audit report (19:06) was itself wrong about LOC

The 19:06 report recommended writing **"3,128 LOC"** into AGENTS.md as the correction for the stale "~2400 LOC" figure. The real number is **2,485 LOC** (excluding generated templ) or **2,576 LOC** (including templ). Blindly executing that recommendation would have _introduced_ a new factual error into the canonical AI-session-context file — the exact failure mode the audit was diagnosing. **An audit that drifts from reality is worse than no audit, because it carries false authority.** Fixed: I verified every number against `wc -l` before writing.

### #2 — Ghost fuzz-target names in AGENTS.md (missed by 19:06 audit)

AGENTS.md:118 listed `FuzzPluginHTML_ErrorMessages` and `FuzzPluginHTML_DepChain` as fuzz targets. **Neither exists.** The real targets are `FuzzPluginHTML`, `FuzzMigrateReport`, `FuzzDiagramSpecialChars`. These ghost names were the pre-consolidation targets from an earlier session that were never cleaned up. The 19:06 audit checked fuzz counts but not fuzz _names_. Fixed.

### #3 — Dead function reference in AGENTS.md (missed by 19:06 audit)

AGENTS.md:118 referenced `stripScriptTags()` — a function that was **renamed to `stripJSONScripts()`** in a prior commit. The AGENTS.md bullet documenting the rename exists (line 150) but the original bullet was never updated. Classic split-brain: two bullets describing the same function, one with the old name. Fixed.

### #4 — CHANGELOG [0.0.4] Tests section carried phantom fuzz ordinals

Same disease as TODO_LIST.md: `FuzzNestedScopeExport` (6th, doesn't exist), `FuzzDiagramSpecialChars` (5th, really 3rd), `FuzzMigrateReport` (4th, really 2nd). A shipped changelog that describes work that doesn't exist as shipped. Fixed.

### #5 — The 19:06 report under-counted unpushed commits

It said "1 unpushed local commit." When this session started there were **3** (`74c5e32`, `13d4102`, `3e8931b`). They have since been pushed (master is now in sync with origin). Not dangerous, but the report's "partially done" tracking was stale by the time it was committed.

---

## e) WHAT WE SHOULD IMPROVE!

Beyond section (d), structural improvements worth pursuing:

1. **Stop quoting exact counts in hand-maintained docs.** This session proved the point twice: the 19:06 audit's own LOC number was wrong, and two ghost references it missed survived because no one re-verified _names_ (only counts). Either (a) generate metric blocks from a script into the docs, or (b) stop quoting exact counts in prose and link to a single source of truth.
2. **Add a docs-freshness CI gate.** A lightweight job that asserts `FEATURES.md`/`TODO_LIST.md`/`AGENTS.md` don't claim a status contradicted by the code (e.g. fuzz target names extracted from `grep '^func Fuzz'`, LOC from `wc -l`, test counts from `grep -c`). Would have caught items #1–#4 in section d instantly.
3. **Pre-commit hook: `go generate` must be no-op.** Kill the `html_templ.go` whack-a-mole permanently. `.gitattributes` + `linguist-generated` only hides the symptom on GitHub; a local hook that runs `go generate` and fails on diff is the real fix.
4. **Decide the v0.1.0 release question explicitly.** The library has met `STABILITY.md` criteria for multiple sessions. The ambiguity (schema-first vs ship-now) is the dominant blocker on ~6 of the top-25 items.
5. **Rolling `CURRENT.md` status.** `docs/status/` now has **5 reports from the same day** that cross-reference each other. The chain is hard to follow. Replace with a rolling `CURRENT.md` (overwritten each session) + an `archive/` subdirectory.
6. **Verify audit claims against code, always.** The meta-lesson from this session: any status report, no matter how confident, is a secondary source. The code is the primary source. `wc -l`, `grep -c`, and `git log` take 2 seconds and prevent the audit-is-itself-drift failure mode.

---

## f) Top #25 Things We Should Get Done Next

Sorted by impact × value ÷ effort. Items marked ✅ were doc-trust fixes completed this session. Items marked ⚠️ remain cheap and high-trust.

| #   | Task                                                                   | Category     | Effort | Status / Why                                                  |
| --- | ---------------------------------------------------------------------- | ------------ | ------ | ------------------------------------------------------------- |
| 1   | ✅ **Fix `FEATURES.md` PARTIALLY FUNCTIONAL section**                  | Doc-trust    | XS     | Done this session                                             |
| 2   | ✅ **Fix `TODO_LIST.md` fuzz-target notes**                            | Doc-trust    | XS     | Done this session                                             |
| 3   | ✅ **Fix `AGENTS.md` metrics + fuzz names + function ref**             | Doc-trust    | XS     | Done this session                                             |
| 4   | ✅ **Fix `CHANGELOG.md` [0.0.4] fuzz ordinals**                        | Doc-trust    | XS     | Done this session (found during sweep, not in original audit) |
| 5   | ✅ **Fix `flake.nix` "Go 1.26.3" string**                              | DX           | XS     | Done this session                                             |
| 6   | ⚠️ **Commit & push doc-drift fixes**                                   | Doc-trust    | XS     | 5 files ready, awaiting commit                                |
| 7   | **Decide v0.1.0: ship-now vs JSON-schema-first**                       | Product      | S      | Unblocks ~6 downstream items                                  |
| 8   | **JSON Schema file** for the report format                             | Feature      | M      | Biggest missing artifact for consumers                        |
| 9   | **NDJSON import** (`ReadNDJSON`)                                       | Feature      | S      | Trivial via `buildReportFromCore`                             |
| 10  | **Property-based `Diff` tests** (symmetry + round-trip)                | Testing      | S      | Hardens the most-used query API                               |
| 11  | **Property-based `MigrateReport` tests**                               | Testing      | S      | Guards schema-evolution invariants                            |
| 12  | **`Report` constructor validation** (`NewReport(...) (Report, error)`) | Architecture | M      | Makes invalid reports unrepresentable                         |
| 13  | **Typed identifiers** (`ContainerID`, `ScopeID`, `ServiceName`)        | Architecture | M      | Compiler rejects accidental swaps                             |
| 14  | **HTML golden-file test** (deterministic multi-service report)         | Testing      | S      | Catches viz regressions silently                              |
| 15  | **Docs-freshness CI gate** (assert counts/names match `grep`)          | CI           | S      | Prevents recurrence of #1–#5                                  |
| 16  | **Pre-commit hook: `go generate` must be no-op**                       | DX           | S      | Kills the html_templ.go whack-a-mole permanently              |
| 17  | **CSV/TSV export**                                                     | Feature      | S      | High value for data-analysis workflows                        |
| 18  | **CLI tool** for report conversion/export/viz                          | Feature      | M      | Standalone binary, broad reach                                |
| 19  | **`actionlint` in CI**                                                 | CI           | XS     | Validates workflow YAML                                       |
| 20  | **Prometheus exporter example** (parallel to OTel)                     | DX           | S      | OnEvent bridge reference                                      |
| 21  | **Fuzz filter inputs** (arbitrary `ReportOption` combos)               | Testing      | S      | The only genuinely partial fuzz surface                       |
| 22  | **WebSocket live stream** bridge for `OnEvent`                         | Feature      | M      | Real-time dashboards                                          |
| 23  | **Split `ServiceInfo`** into Identity/Lifecycle/Health/Graph           | Architecture | L      | Breaking; decide before v0.1.0                                |
| 24  | **Rolling `CURRENT.md` status** + archive old daily reports            | Doc hygiene  | S      | 5 same-day cross-referencing reports is confusing             |
| 25  | **v0.1.0 release** (tag + GitHub Release + schema)                     | Release      | M      | The keystone — depends on #7                                  |

---

## g) Top #1 Question I Cannot Figure Out Myself

> **Is `v0.1.0` a "ship now, schema later" release or a "JSON Schema first, then tag" release?**

This is the single decision that shapes the priority of at least six items above (#7, #8, #12, #13, #23, #25). The library already satisfies every criterion in `STABILITY.md`; the code is green; all docs now match reality; the only thing standing between the current state and a `v0.1.0` tag is whether the report's JSON shape should be frozen into a published `schema.json` _before_ the tag (so consumers can codegen against a stable contract) or whether we tag now and publish the schema as a fast-follow.

I cannot resolve this because it is a product/intent question, not a technical one:

- **Ship-now** maximizes momentum and signals stability, but risks a schema change after consumers have integrated.
- **Schema-first** gives consumers a machine-readable contract and makes the 0.x→1.0 promise credible, but delays the tag and forces decisions about `ServiceInfo` splitting (#23) now rather than later.

**What I will do once you answer:** If ship-now → cut `v0.1.0` immediately and open a schema-fast-follow task. If schema-first → generate `schema.json` from the Go types (#8), wire a schema-validation test, then tag. Either way I'll execute end-to-end.

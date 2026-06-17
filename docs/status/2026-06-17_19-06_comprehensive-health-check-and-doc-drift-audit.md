# Status Report — 2026-06-17 19:06

## Comprehensive Health Check, Doc-Drift Audit & Release Readiness

**Branch**: `master` (1 unpushed local commit) · **Latest tag**: `v0.0.3` · **Schema version**: `0.2.0` · **Go**: `1.26.4` · **templ**: `v0.3.1020` · **Working tree**: clean

---

## Executive Summary

The codebase is in **excellent mechanical health**. Every automated gate is green: build, vet, race tests, lint (0 issues), coverage (95.3% ≥ 95% gate), `go generate` idempotent, and `go.sum` drift-free. There is **no broken code, no failing test, no lint regression**. The architecture is sound — single-package, single-lock concurrency, unified report construction, memory-bounded event capture.

What surfaced this session is **not a code problem, it is a documentation-trust problem.** Three documentation files have drifted from reality in ways that actively mislead: `FEATURES.md` claims test parallelism is "~15%" when it is ~97% and claims "all fuzz targets test HTML XSS" when 2 of 3 fuzz targets now exercise migration and diagrams; `TODO_LIST.md` references non-existent "5th" and "6th" fuzz targets; `AGENTS.md` under-counts tests and under-reports LOC. None of these break anything at runtime, but they erode the credibility of the docs — and credibility is the whole point of an audit-log library.

Additionally, a prior session produced a legitimate, unpushed commit (`3e8931b`) that finally pins `html_templ.go` to the generator's canonical single-line import output, breaking the recurring multi-line-vs-single-line drift cycle that has burned 4+ commits. It is not yet on `origin/master`.

### Verification Snapshot (all green, re-confirmed this session)

```
Build:       ✅ go build ./... — clean
Vet:         ✅ go vet ./... — clean
Tests:       ✅ 146 Test + 11 Benchmark + 3 Fuzz + 7 Example = 167 top-level — all PASS (-race)
Coverage:    ✅ 95.3% of production statements (example/ excluded; gate ≥95%)
Lint:        ✅ golangci-lint — 0 issues
Generate:    ✅ go generate ./... — idempotent (sha256 verified)
Mod tidy:    ✅ go.sum — no drift
Working tree: ✅ clean
```

---

## a) FULLY DONE

Verified against actual code this session, not against docs.

### Core Library (production code — 3,128 LOC across 20 `.go` files)

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

| Metric               | Value                                                                                                                           |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Test functions       | 146                                                                                                                             |
| Benchmarks           | 11 (Invocation, Disabled, Registration, ConcurrentInvocation, BuildReport 50/100/500, EventsCopy, OnEventCallback, HealthCheck) |
| Fuzz targets         | 3 — `FuzzPluginHTML`, `FuzzMigrateReport`, `FuzzDiagramSpecialChars`                                                            |
| Godoc examples       | 7 (pkg.go.dev)                                                                                                                  |
| `t.Parallel()` calls | 152 (~97% of eligible tests; only 5 remain sequential — all `t.Setenv()` env-var tests)                                         |

### CI & Release Infrastructure

- ✅ `.github/workflows/ci.yml` — 5 jobs: **test** (race + 95% coverage gate), **lint** (golangci-lint v2.12.2, pinned), **vulncheck** (`govulncheck-action`), **mod-tidy** (drift detection), **stale-generation** (`go generate` diff detection)
- ✅ `v0.0.3` tagged + GitHub Release with `audit-report.html` artifact
- ✅ `STABILITY.md` 0.x stability promise, `CONTRIBUTING.md` release procedure, `BENCHMARKS.md` baselines, `CODE_OF_CONDUCT.md`
- ✅ `flake.nix` devShell (Go 1.26.4, templ, golangci-lint, govulncheck, golines)
- ✅ `.gitattributes` marks `*_templ.go` as generated (hides from GitHub diffs + language stats)

---

## b) PARTIALLY DONE

| Item                               | State              | Reality                                                                                                                                                                                                                                                    |
| ---------------------------------- | ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **`html_templ.go` drift cycle**    | Fixed-but-unpushed | Commit `3e8931b` (prior session) commits the generator's canonical single-line imports, matching what the `stale-generation` CI job produces. This should break the recurring drift, **but it is not yet on `origin/master`**, so CI has not confirmed it. |
| **Documentation accuracy**         | Partially stale    | `AGENTS.md`, `FEATURES.md`, and `TODO_LIST.md` each carry factual inaccuracies (see section d). The _code_ is correct; the _docs describing the code_ have not kept pace.                                                                                  |
| **`flake.nix` description string** | Stale (cosmetic)   | Says "Go 1.26.3" in a comment; `go.mod` requires `1.26.4`. DevShell self-upgrades via `GOTOOLCHAIN=auto`, so it works — but the string lies.                                                                                                               |
| **govulncheck locally**            | CI-only            | `govulncheck` is not installed in this devShell (only in CI). Local security scans are not possible without `nix develop`.                                                                                                                                 |

---

## c) NOT STARTED

Carried forward from prior audits; none blocked by code health — all are deliberate deferrals.

1. **v0.1.0 release** — project meets every `STABILITY.md` criterion. The only blocker is a product decision: ship-now vs. JSON-schema-first.
2. **JSON Schema file** (`schema.json`) for the report format — the single biggest missing piece for machine consumers of exported reports.
3. **NDJSON import** — `ReadNDJSON(reader) (Report, error)`. Trivial now that `buildReportFromCore` centralizes construction.
4. **Property-based tests** — `rapid`/stdlib fuzz for `Diff` symmetry, filter round-trips, migration invariants.
5. **CSV/TSV export** — spreadsheet-friendly tabular export.
6. **CLI tool** — standalone binary to convert/export/visualize saved reports.
7. **WebSocket live stream** — bridge `OnEvent` to real-time dashboards.
8. **Prometheus exporter example** — parallel to the OTel example.
9. **`actionlint`** in CI — workflow linting alongside govulncheck.
10. **HTML golden-file test** — deterministic multi-service report → committed golden output.

---

## d) TOTALLY FUCKED UP!

Honest call-out. Nothing here is catastrophic or broken — but these are **trust violations** that a careful reader would catch and lose confidence over. Ranked by severity.

### #1 — `FEATURES.md` "PARTIALLY FUNCTIONAL" section is badly, verifiably stale

`FEATURES.md:155-156` claims:

> **Test parallelism** | Only ~15% of tests use `t.Parallel()`; the rest run sequentially.
> **Fuzz coverage** | All fuzz targets test HTML XSS; no fuzzing of `MigrateReport`, Mermaid/PlantUML special characters, nested scopes, or filters.

**Both statements are false:**

- Test parallelism is **~97%** — 152 `t.Parallel()` calls; only 5 sequential tests remain (all `t.Setenv()`).
- Fuzz coverage now includes `FuzzMigrateReport` and `FuzzDiagramSpecialChars`. Only 1 of 3 fuzz targets is HTML-XSS-only.

A "PARTIALLY FUNCTIONAL" section that lists things as not-done-when-they-are-done defeats the entire purpose of an honest feature inventory. **This is the worst doc issue in the repo right now** because FEATURES.md is explicitly positioned as "verified against the code."

### #2 — `TODO_LIST.md` references non-existent fuzz targets

`TODO_LIST.md:87-88` (in the "Completed — Post-Remediation Consolidation" section) claims completion of:

> - [x] **Diagram special-char fuzz** — `FuzzDiagramSpecialChars` (**5th fuzz target**)
> - [x] **Nested scope tree fuzz** — `FuzzNestedScopeExport` (**6th fuzz target**): 500-level-deep scope trees

Reality: there are only **3 fuzz targets**, and `FuzzNestedScopeExport` does not exist — it was converted to a table-driven `TestNestedScopeExport` during the buildflow 6→3 consolidation. A prior status report claimed this was fixed in CHANGELOG and TODO_LIST, but the TODO_LIST completion notes still carry the wrong counts. **A "completed" checklist that lists work that doesn't exist as shipped is a lie in the most actionable section of the doc.**

### #3 — `AGENTS.md` metrics under-report reality

`AGENTS.md` Testing Patterns claims:

> 153 top-level test functions (234 total cases including subtests): 143 unit tests, 7 examples, 3 fuzz targets, 13 benchmark cases.

Actual: **167 top-level** (146 Test + 11 Benchmark + 3 Fuzz + 7 Example). And the architecture section says "~2400 LOC" when production code is **3,128 LOC**. These are the canonical "AI session context" numbers — if they're wrong, every fresh session starts with a wrong mental model.

### #4 — The `html_templ.go` import-format whack-a-mole

This file's import block (single-line vs grouped) has now been "fixed" in **at least 4 commits** (`c68005f`, `69be806`, `3e8931b`, and earlier). Each "fix" gets reverted by the next formatter run or generator invocation. The `.gitattributes` marker mitigates GitHub noise but does not stop local editors from reformatting. Commit `3e8931b` claims to have found the truly canonical output; until CI confirms it green on `origin/master`, I am not declaring victory.

### #5 — One unpushed commit sitting on master

`3e8931b` (the html_templ.go fix above) is a legitimate commit by a prior session that has never reached `origin/master`. Combined with the earlier `git town sync` push failure (a benign server-side CAS race, already resolved), it means the remote is one commit behind the intended state. Not dangerous — but it should not linger.

---

## e) WHAT WE SHOULD IMPROVE!

Beyond fixing section (d), structural improvements worth pursuing:

1. **Treat docs as code under test.** The recurring drift (FEATURES, TODO, AGENTS metrics) shows that hand-maintained numbers always rot. Either (a) generate metric blocks from a script into the docs, or (b) stop quoting exact counts in prose and link to a single source of truth.
2. **Add a docs-freshness CI gate.** A lightweight job that asserts `FEATURES.md`/`TODO_LIST.md` don't claim a status contradicted by the code (e.g. counts derived from `grep`) would have caught items #1–#3 in section d instantly.
3. **Pin a single formatter config and a pre-commit hook for `html_templ.go`.** Stop humans/formatters from touching generated files at all. `.gitattributes` + a local hook that runs `go generate` and fails on diff is the real fix — `linguist-generated` only hides the symptom on GitHub.
4. **Decide the v0.1.0 release question explicitly.** The library has been "meets STABILITY criteria" for multiple sessions. The ambiguity (schema-first vs ship-now) is now the dominant blocker on ~6 of the top-25 items. A decision unblocks more than any single feature.
5. **JSON Schema generation from Go types.** This would eliminate the manual drift risk for report consumers and is the highest-leverage missing artifact.
6. **Reduce the number of status reports that reference other status reports.** `docs/status/` now has 3 reports from the same day that cross-reference each other. The chain is getting hard to follow; a rolling `CURRENT.md` (or just the latest) plus an archive would be cleaner.

---

## f) Top #25 Things We Should Get Done Next

Sorted by impact × value ÷ effort. Items marked ⚠️ are doc-trust fixes from section (d) — they are cheap and high-trust.

| #   | Task                                                                                                      | Category     | Effort | Why                                               |
| --- | --------------------------------------------------------------------------------------------------------- | ------------ | ------ | ------------------------------------------------- |
| 1   | ⚠️ **Fix `FEATURES.md` PARTIALLY FUNCTIONAL section** — move parallelism + fuzz to DONE with real numbers | Doc-trust    | XS     | Worst credibility issue in repo; verifiably false |
| 2   | ⚠️ **Fix `TODO_LIST.md` fuzz-target completion notes** — 5th/6th → actual 3                               | Doc-trust    | XS     | Checklist lists non-existent shipped work         |
| 3   | ⚠️ **Fix `AGENTS.md` metrics** — 167 tests, 3128 LOC                                                      | Doc-trust    | XS     | Wrong numbers seed wrong mental models            |
| 4   | ⚠️ **Push `3e8931b` to origin/master** (after user approval)                                              | Release      | XS     | Remote is behind intended state                   |
| 5   | **Decide v0.1.0: ship-now vs JSON-schema-first**                                                          | Product      | S      | Unblocks ~6 downstream items                      |
| 6   | **JSON Schema file** for the report format                                                                | Feature      | M      | Biggest missing artifact for consumers            |
| 7   | **NDJSON import** (`ReadNDJSON`)                                                                          | Feature      | S      | Trivial via `buildReportFromCore`                 |
| 8   | **Property-based `Diff` tests** (symmetry + round-trip)                                                   | Testing      | S      | Hardens the most-used query API                   |
| 9   | **Property-based `MigrateReport` tests**                                                                  | Testing      | S      | Guards schema-evolution invariants                |
| 10  | **`Report` constructor validation** (`NewReport(...) (Report, error)`)                                    | Architecture | M      | Makes invalid reports unrepresentable             |
| 11  | **Typed identifiers** (`ContainerID`, `ScopeID`, `ServiceName`)                                           | Architecture | M      | Compiler rejects accidental swaps                 |
| 12  | **HTML golden-file test** (deterministic multi-service report)                                            | Testing      | S      | Catches viz regressions silently                  |
| 13  | **Docs-freshness CI gate** (assert counts match `grep`)                                                   | CI           | S      | Prevents recurrence of #1–#3                      |
| 14  | **Pre-commit hook: `go generate` must be no-op**                                                          | DX           | S      | Kills the html_templ.go whack-a-mole permanently  |
| 15  | **Fix `flake.nix` "Go 1.26.3" description string**                                                        | DX           | XS     | Cosmetic lie                                      |
| 16  | **CSV/TSV export**                                                                                        | Feature      | S      | High value for data-analysis workflows            |
| 17  | **CLI tool** for report conversion/export/viz                                                             | Feature      | M      | Standalone binary, broad reach                    |
| 18  | **`actionlint` in CI**                                                                                    | CI           | XS     | Validates workflow YAML                           |
| 19  | **Prometheus exporter example** (parallel to OTel)                                                        | DX           | S      | OnEvent bridge reference                          |
| 20  | **Fuzz filter inputs** (arbitrary `ReportOption` combos)                                                  | Testing      | S      | Untested combinatorial surface                    |
| 21  | **WebSocket live stream** bridge for `OnEvent`                                                            | Feature      | M      | Real-time dashboards                              |
| 22  | **Split `ServiceInfo`** into Identity/Lifecycle/Health/Graph                                              | Architecture | L      | Breaking; decide before v0.1.0                    |
| 23  | **`govulncheck` in local devShell** (not only CI)                                                         | DX           | XS     | Local security scanning                           |
| 24  | **Rolling `CURRENT.md` status** + archive old daily reports                                               | Doc hygiene  | S      | 3 same-day cross-referencing reports is confusing |
| 25  | **v0.1.0 release** (tag + GitHub Release + schema)                                                        | Release      | M      | The keystone — depends on #5                      |

---

## g) Top #1 Question I Cannot Figure Out Myself

> **Is `v0.1.0` a "ship now, schema later" release or a "JSON Schema first, then tag" release?**

This is the single decision that shapes the priority of at least six items above (#5, #6, #10, #11, #22, #25). The library already satisfies every criterion in `STABILITY.md`; the code is green; the only thing standing between the current state and a `v0.1.0` tag is whether the report's JSON shape should be frozen into a published `schema.json` _before_ the tag (so consumers can codegen against a stable contract) or whether we tag now and publish the schema as a fast-follow.

I cannot resolve this because it is a product/intent question, not a technical one:

- **Ship-now** maximizes momentum and signals stability, but risks a schema change after consumers have integrated.
- **Schema-first** gives consumers a machine-readable contract and makes the 0.x→1.0 promise credible, but delays the tag and forces decisions about `ServiceInfo` splitting (#22) now rather than later.

**What I will do once you answer:** If ship-now → cut `v0.1.0` immediately and open a schema-fast-follow task. If schema-first → generate `schema.json` from the Go types (#6), wire a schema-validation test, then tag. Either way I'll execute end-to-end.

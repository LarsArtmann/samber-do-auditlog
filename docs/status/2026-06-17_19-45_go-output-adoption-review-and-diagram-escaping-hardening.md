# Comprehensive Status Report
**Date:** 2026-06-17 19:45
**Branch:** master
**Release:** v0.0.4 (latest tag)
**Schema:** 0.2.0
**Session focus:** `go-output` adoption review → diagram escaping hardening

---

## Project Snapshot

| Metric | Value |
|---|---|
| Source LOC (non-test, non-generated, non-example) | 2,534 |
| Test LOC | 5,853 |
| Go files (excl. example) | 45 |
| Tests (Test + Benchmark + Fuzz + Example) | 169 |
| Statement coverage (non-example) | 95.3% |
| golangci-lint issues | 0 |
| `go test -race ./...` | PASS (1.3s) |
| `go vet ./...` | PASS |
| External dependencies | `samber/do/v2`, `a-h/templ` |
| Fuzz targets | 3 |
| Go version | 1.26.4 |

---

## a) FULLY DONE

### This Session

1. **`go-output` adoption review** — Comprehensive, evidence-based evaluation of adopting `github.com/larsartmann/go-output/...` for the export pipeline. Prototyped both renderers, measured the exact dependency tree (11 external modules for graph+plantuml), captured output-format diffs proving a swap would be **breaking** (markdown fence, pink theme vs warm-amber `%%{init}%%`, no edge dedup). **Verdict: declined wholesale adoption.** Documented at `docs/research/go-output-adoption-review.md`.

2. **Diagram escaping bug — discovered and fixed.** The review surfaced a real correctness bug: service names containing `]`, `"`, `[` produced **malformed Mermaid/PlantUML output** (broken `id[label]` syntax). Fixed in `diagram.go`:
   - Unified ID sanitization: `sanitizeDiagramID` replaces the two divergent `diagramNodeID` replacers (Mermaid had 4-char, PlantUML had 7-char). Now both use a single 7-char replacer + rune allowlist filter, returning `"node"` on empty result.
   - Mermaid label escaping: `mermaidLabel` (`"`→`'`, `[]`/`{}`→`()`, `\n`→`<br>`).
   - PlantUML label escaping: `plantumlLabel` (`"`→`'`).
   - 2 regression tests: `TestWriteMermaid_EscapesSpecialChars`, `TestWritePlantUML_EscapesSpecialChars`.

3. **AGENTS.md updated** — Documented the escaping design (`sanitizeDiagramID` / `mermaidLabel` / `plantumlLabel`) and the declined `go-output` adoption with pointer to the review doc.

### Pre-Existing (verified green this session)

4. **Full test suite** — 169 tests, all pass with `-race`. Coverage gate (≥95%) met.
5. **Lint** — `golangci-lint run ./...` = 0 issues. Nearly every linter enabled.
6. **CI pipeline** — 5 parallel jobs: test (+coverage gate), lint, vulncheck, mod-tidy, stale-generation.
7. **All export formats functional** — JSON, NDJSON, Mermaid, PlantUML, self-contained HTML (5-tab interactive visualization).
8. **Report model** — `Validate()`, `Index()`, `Diff()`, `Filtered()`, convenience queries (`ServiceByName`, `EventsByType`, `FailedServices`, `UnhealthyServices`, etc.).
9. **Schema migration** — `MigrateReport` v0.1.0 → v0.2.0 with round-trip tests.
10. **Health check auditing** — `RecordHealthCheck[WithContext]` wrapper, `EventTypeHealthCheck`, per-service health fields.
11. **Concurrency model** — Single `sync.RWMutex` + 2 atomics. Callbacks outside the lock.
12. **Fuzz testing** — 3 targets: `FuzzPluginHTML` (XSS), `FuzzMigrateReport` (schema integrity), `FuzzDiagramSpecialChars` (diagram structural integrity).
13. **Developer experience** — `flake.nix` devShell, `BENCHMARKS.md` baselines, 7 godoc examples, `STABILITY.md` promise, OTel bridge example.

---

## b) PARTIALLY DONE

1. **v0.1.0 release readiness** — The project meets `STABILITY.md` criteria and has comprehensive features. **Blocked on:** JSON-schema-first decision (should the report format have a committed `schema.json` before semver stability promise?).
2. **CSV/TSV export** — Not started but `go-output` review confirmed it's low-effort if prioritized (can be done locally with `encoding/csv`, no dependency needed).
3. **DOT graph export** — `go-output` has a DOT renderer; the review noted this could be a future format. Not started, dep cost disproportionate for now.
4. **Report filtering completeness** — 5 filter options exist (`WithServicesByName`, `WithServicesByScope`, etc.). Property-based fuzz testing of arbitrary filter combinations is TODO.

---

## c) NOT STARTED

1. **Typed identifiers** — `ContainerID`, `ScopeID`, `ServiceName` as distinct named string types (compiler rejects accidental swaps). High value, low effort.
2. **NDJSON import** — `ReadNDJSON(reader) (Report, error)`. Trivial now that `buildReportFromCore` centralizes construction.
3. **`NewReport(...)` constructor** — Returns `(Report, error)` so invalid reports are unrepresentable.
4. **JSON Schema file** — Derive `schema.json` from Go types. Biggest missing piece for report consumers.
5. **CLI tool** — Report conversion/export/visualization binary.
6. **WebSocket live stream** — Bridge `OnEvent` to a WebSocket for live dashboards.
7. **HTML golden-file test** — Deterministic multi-service report → assert output matches committed golden file.
8. **Property-based `Diff` tests** — Random reports, assert `Diff(a,a)` empty + symmetry.
9. **Property-based `MigrateReport` tests** — Arbitrary JSON → migrate → validate round-trips.
10. **Prometheus exporter example** — Parallel to the OTel example.
11. **`actionlint` in CI** — Workflow validation.
12. **Flake app for coverage gate** — Replace inline shell in CI.

---

## d) TOTALLY FUCKED UP

**Nothing.** No broken state, no failing tests, no lint issues, no security holes found and left open. The one bug found this session (diagram escaping) was fixed immediately with regression tests.

The closest thing to a concern: the **stale LSP diagnostics** for `zz_dump_test.go` (a throwaway file deleted during the session). The LSP cache hadn't invalidated, but the file is gone and `golangci-lint run ./...` confirms 0 issues.

---

## e) WHAT WE SHOULD IMPROVE

1. **Schema-first design gap** — The JSON report format has no committed `schema.json`. Consumers can't validate reports programmatically. This is the single biggest architectural gap before v0.1.0.
2. **String-typed identity** — `ScopeID`, `ServiceName`, `ContainerID` are all bare `string`. Accidental swaps compile fine. Named types would make impossible states unrepresentable.
3. **`ServiceInfo` is a 19-field god struct** — Mixes identity, lifecycle, health, and graph concerns. Should split into `ServiceIdentity` / `ServiceLifecycle` / `ServiceHealth` / `ServiceGraph` before v0.1.0 (breaking change, better now than later).
4. **No HTML golden-file test** — The HTML output is complex (5 tabs, JS, CSS) and tested only via string-contains assertions. A golden-file test would catch visual regressions deterministically.
5. **Property-based test coverage** — `Diff`, `MigrateReport`, and filter combinations lack property-based / fuzz testing. Table-driven tests cover known cases but not the combinatorial space.
6. **`example/` has 0% coverage** — CI gate excludes it, but the demo could silently break.
7. **`docs/status/` is growing** — 5 status reports in `docs/status/`, 12+ in `docs/archive/`. Consider a retention policy (keep last 5, archive rest).

---

## f) Top 25 Things to Get Done Next

Sorted by impact × value ÷ effort:

| # | Task | Impact | Effort | Notes |
|---|---|---|---|---|
| 1 | **Add `CHANGELOG.md` entry** for the diagram escaping fix | High | Trivial | `[Unreleased]` section is empty |
| 2 | **JSON Schema file** (`schema.json`) for report format | High | Medium | Biggest gap for consumers; blocks v0.1.0 |
| 3 | **Typed identifiers** (`ScopeID`, `ServiceName`, `ContainerID`) | High | Low | Compiler-enforced safety |
| 4 | **NDJSON import** (`ReadNDJSON`) | Medium | Low | Symmetry with export; trivial via `buildReportFromCore` |
| 5 | **CSV/TSV export** | Medium | Low | `encoding/csv`, no dep; high value for data analysis |
| 6 | **HTML golden-file test** | Medium | Medium | Deterministic multi-service → golden file |
| 7 | **Split `ServiceInfo`** into identity/lifecycle/health/graph | High | High | Breaking change; decide before v0.1.0 |
| 8 | **`NewReport(...)` constructor** returning `(Report, error)` | Medium | Low | Makes invalid reports unrepresentable |
| 9 | **Property-based `Diff` tests** | Medium | Low | Random reports, symmetry assertions |
| 10 | **Property-based `MigrateReport` tests** | Medium | Low | Arbitrary JSON → migrate → validate |
| 11 | **Fuzz filter inputs** | Low | Low | Arbitrary `ReportOption` combinations |
| 12 | **Prometheus exporter example** | Medium | Low | Parallel to OTel bridge doc |
| 13 | **`actionlint` in CI** | Low | Trivial | Workflow validation |
| 14 | **Flake app for coverage gate** | Low | Low | Replace inline CI shell |
| 15 | **DOT graph export** | Low | Medium | New diagram format; local impl preferred over dep |
| 16 | **CLI tool** (`auditlog-convert`) | Medium | High | Report conversion/export binary |
| 17 | **WebSocket live stream** bridge for `OnEvent` | Medium | High | Live dashboards |
| 18 | **v0.1.0 release** | High | Medium | Blocked on #2 (schema) and decision on #7 (split) |
| 19 | **`RELEASING.md`** or release checklist | Low | Trivial | In CONTRIBUTING.md |
| 20 | **`example/` smoke test** | Low | Low | At least a basic integration test |
| 21 | **`docs/status/` retention policy** | Low | Trivial | Keep last 5, archive rest |
| 22 | **Add diagram escaping to fuzz target** | Medium | Low | Extend `FuzzDiagramSpecialChars` with bracket/quote corpus |
| 23 | **Coverage gate as separate CI step** | Low | Low | Clearer failure messages |
| 24 | **Review `ServiceStatus` priority** for completeness | Low | Low | Is there a missing state (e.g. "health_error")? |
| 25 | **Benchmark the escaping functions** | Low | Trivial | Ensure no hot-path regression from `sanitizeDiagramID` |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should we commit to a JSON Schema (`schema.json`) and cut v0.1.0, or keep iterating on the API surface (e.g., split `ServiceInfo`, add typed identifiers) first?**

The project is feature-complete and stable enough for v0.1.0 by every objective metric (95% coverage, 0 lint issues, 3 fuzz targets, stability promise documented). But two architectural improvements (typed identifiers and `ServiceInfo` split) would be **breaking changes** that are cheaper to do before a stability promise than after. The `go-output` review confirmed the external dependency surface should stay lean — but the internal type design is the real open question. This is a product/strategy decision, not a technical one: **stability-first (ship v0.1.0 now, iterate in v0.2) vs. correctness-first (break now, stabilize later).**

---

## Verification Evidence

```
go test -race ./...           → PASS (1.3s), coverage 95.3%
go vet ./...                  → PASS
golangci-lint run ./...       → 0 issues
git status                    → 3 modified, 1 untracked (review doc)
```

## Uncommitted Changes

| File | Change |
|---|---|
| `diagram.go` | Hardened ID sanitization + label escaping (+89/-20 lines) |
| `diagram_test.go` | 2 regression tests for escaping (+72 lines) |
| `AGENTS.md` | Documented escaping design + declined adoption (+1 line) |
| `docs/research/go-output-adoption-review.md` | New: full adoption review (untracked) |

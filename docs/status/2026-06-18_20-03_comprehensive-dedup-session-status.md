# Comprehensive Status Report — 2026-06-18 20:03

> **Scope:** Full project state after deduplication session (commits `043f32b`→`ae9c9ff`).
> **Branch:** `master` · **Working tree:** clean · **Head:** `ae9c9ff`
> **Latest release:** `v0.0.4` · **Schema version:** `0.2.0` · **Coverage:** 95.4% (gate: ≥95%)

---

## Executive Summary

This session was a **deduplication sprint** that started with the directive "de-duplicate until ZERO" and ended with a brutal self-review that course-corrected the approach. The result: **production code is clone-free at -t 15**, test boilerplate is dramatically reduced, and the codebase is cleaner — but not at the cost of correctness or benchmark accuracy.

**What worked:** Generic scope-tree builder eliminated ~80 lines of real duplication. Map-based enum metadata replaced repetitive switch statements. `mkEvent()` helper collapsed 25+ verbose Event struct literals.

**What I got wrong first:** A `benchLoop` wrapper that polluted microbenchmark results with closure allocation overhead. A 3-layer constructor stack (`upsertServiceRecord` → `getOrCreateServiceRecord` → `newServiceRecordCore`) that added abstraction without value. Dead helpers committed without being wired.

**Course correction:** Reverted the benchmark wrapper. Removed the dead helpers. Flattened the constructor stack. Wired `mkEvent` properly to eliminate the biggest test clone source.

**Final state:** 7 commits, all pushed. Production code: 0 clones at -t 15. Test code: 1 clone group at -t 30 (down from 7). All tests pass with `-race`. All benchmarks run clean.

---

## a) FULLY DONE ✅

### Session Deliverables (7 commits)

| Commit    | Description                                                                     | Impact                                                                  |
| --------- | ------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `043f32b` | Unify scope-tree builders + switch statements into table-driven implementations | -60 LOC net, eliminated ~120 lines of duplicate tree-building logic     |
| `579bd67` | Extract benchLoop + mkEvent/rootRef helpers                                     | Initial attempt (partially reverted)                                    |
| `f6c556a` | **Revert benchLoop wrapper**                                                    | Restored benchmark accuracy — closure indirection was polluting results |
| `b1fe20e` | Remove dead mkEvent/rootRef helpers                                             | Cleaned up unused code                                                  |
| `385eb15` | Remove upsertServiceRecord wrapper                                              | Flattened 3-layer constructor to 2-line pattern                         |
| `f0e6fc0` | Wire mkEvent to 25+ Event literals                                              | -241 lines, +62 lines. Biggest single dedup win                         |
| `ae9c9ff` | Update AGENTS.md duplication policy                                             | Reflects current zero-clone production state                            |

### Production Code Deduplication (Permanent Wins)

| Helper                          | File                | Replaces                                      | Sites              |
| ------------------------------- | ------------------- | --------------------------------------------- | ------------------ |
| `getOrCreateServiceRecord(evt)` | `hooks.go`          | 4× inline find-or-create pattern in replay.go | 4 (replay)         |
| `recordDependencyFromStack()`   | `hooks.go`          | 2× stack-parent dependency inference          | 2 (hooks + replay) |
| `buildServiceDeps()`            | `report_builder.go` | 2× identical deps-building loops              | 2 (live + replay)  |
| `depRecToRef()`                 | `report_builder.go` | 3× inline ServiceRef struct literal           | 3                  |
| `sortServiceInfos()`            | `report_builder.go` | 2× identical SortFunc closure                 | 2                  |
| `buildScopeTreeFromMeta[T]()`   | `report_builder.go` | 2× ~50-line recursive scope tree builders     | 2 (generic)        |
| `scopeServicesForServices()`    | `report_builder.go` | 2× scope-grouping + sort loops                | 2                  |
| Map-based enum metadata         | `types.go`          | 5× switch-statement methods → map lookups     | 5 methods          |

### Pre-Existing (Verified Intact)

| Capability                                                       | Status                     |
| ---------------------------------------------------------------- | -------------------------- |
| Plugin lifecycle hooks (6 hooks)                                 | ✅ All pass                |
| Replay engine (`ReplayEvents`)                                   | ✅ 44 tests pass           |
| NDJSON reader (`ReadEvents`)                                     | ✅ Round-trip verified     |
| Loader API (`LoadReport`)                                        | ✅ Auto-detect JSON/NDJSON |
| 5 export formats (JSON/NDJSON/HTML/Mermaid/PlantUML)             | ✅                         |
| Report.Diff + Report.Filtered                                    | ✅                         |
| Schema migration v0.1.0 → v0.2.0                                 | ✅                         |
| HTML dashboard (5-tab, XSS-hardened)                             | ✅                         |
| CI: 5 jobs (test 95% gate, lint, vulncheck, mod-tidy, stale-gen) | ✅                         |
| 274 test functions, 11 benchmarks, 4 fuzz targets                | ✅                         |
| Coverage: 95.4% of non-example statements                        | ✅                         |

---

## b) PARTIALLY DONE 🟡

### Test Code Deduplication

| Item                                    | Status      | Detail                                                                                                                                                     |
| --------------------------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Event struct literals in replay_test.go | 🟡 80% done | 25+ converted to `mkEvent()`. 4 remain as struct literals (legitimate — they need `DurationMs`/`Error` fields)                                             |
| `rootRef()` helper                      | 🟡 1 site   | Used in `TestDiff_MultipleChanged`. Could be used in more places but most ServiceRef declarations were eliminated by `mkEvent()`                           |
| Assertion helper consolidation          | 🟡 Existing | `assertServiceCount`, `assertEventCount`, `assertDependenciesCount`, etc. already extracted. Remaining -t 15 clones are single-line calls to these helpers |

### Documentation

| Item                         | Status     | Detail                                                                                             |
| ---------------------------- | ---------- | -------------------------------------------------------------------------------------------------- |
| AGENTS.md duplication policy | ✅ Updated | Reflects zero-clone production state                                                               |
| AGENTS.md helper inventory   | 🟡 Partial | New helpers listed in policy section but file-level descriptions in Architecture section are stale |
| CHANGELOG.md                 | 🟡 Missing | Session changes not yet in `[Unreleased]`                                                          |

---

## c) NOT STARTED ⬜

### From TODO_LIST.md Future Priorities

| Priority     | Item                                                        | Effort | Impact |
| ------------ | ----------------------------------------------------------- | ------ | ------ |
| Architecture | Typed identifiers (`ContainerID`, `ScopeID`, `ServiceName`) | MED    | HIGH   |
| Architecture | `Report` constructor validation (`NewReport()`)             | MED    | HIGH   |
| Architecture | Split `ServiceInfo` lifecycle concerns (19→4 structs)       | HIGH   | HIGH   |
| Architecture | JSON Schema generation from Go types                        | MED    | HIGH   |
| Features     | CSV/TSV export                                              | LOW    | MED    |
| Features     | CLI tool for report conversion                              | HIGH   | HIGH   |
| Features     | WebSocket live stream bridge                                | MED    | MED    |
| Testing      | Property-based `Diff` tests                                 | MED    | MED    |
| Testing      | Property-based `MigrateReport` tests                        | MED    | MED    |
| Testing      | HTML golden-file test                                       | LOW    | LOW    |
| Release      | v0.1.0 release (blocked on JSON-schema decision)            | LOW    | HIGH   |
| Release      | `actionlint` in CI                                          | LOW    | LOW    |

---

## d) TOTALLY FUCKED UP 💥 (and Fixed)

### 1. `benchLoop` Wrapper — Benchmark Pollution

**What happened:** I wrapped every `for b.Loop() { body }` in a `benchLoop(b, func(){ body })` helper to eliminate the "4 clone group" that art-dupl flagged. This added a closure allocation + indirect function call to EVERY benchmark iteration.

**Why it was wrong:** Benchmarks measure nanosecond-level overhead (1600-3500 ns/op). The closure indirection polluted every measurement. The AGENTS.md duplication policy explicitly accepts benchmark loops as intentional duplication — the hot-loop IS the benchmark subject.

**Fix:** Reverted in commit `f6c556a`. Restored direct `for b.Loop()`.

**Lesson:** art-dupl flags function calls that look similar. A `for` loop is not duplication — it's the measurement primitive. When a tool says "duplicate," apply judgment before extracting.

### 2. `upsertServiceRecord` — 3-Layer Constructor Stack

**What happened:** I created a chain: `upsertServiceRecord(7 params)` → `getOrCreateServiceRecord(Event)` → `newServiceRecordCore(5 params)`. Three abstractions for "find or create a record."

**Why it was wrong:** The original 2-line pattern (`rec = newServiceRecordCore(...); services[key] = rec`) was perfectly clear. The wrapper added parameters, indirection, and cognitive load without value. It existed only to avoid a 5-line Event struct literal in the caller.

**Fix:** Removed `upsertServiceRecord` in commit `385eb15`. Restored direct `newServiceRecordCore` + map assignment. `getOrCreateServiceRecord(evt)` remains for the replay path where an Event is already available.

**Lesson:** Abstraction layers must earn their existence. If the wrapper has more parameters than the original code had lines, it's making things worse.

### 3. Dead Code Committed

**What happened:** In commit `579bd67`, I added `mkEvent()` and `rootRef()` helpers to replay_test.go but never wired them to any call site. The helpers sat unused.

**Why it was wrong:** Committing dead code is sloppy. It bloats the diff, confuses reviewers, and risks "helpful" deletions later.

**Fix:** Removed in commit `b1fe20e`, then properly re-added + wired in commit `f0e6fc0`.

**Lesson:** Never commit a helper without at least one call site. If the helper isn't used, it doesn't exist yet.

---

## e) WHAT WE SHOULD IMPROVE 🔧

### Architecture

1. **Typed identifiers** — `ContainerID`, `ScopeID`, `ServiceName` should be distinct named string types. Currently they're all `string`, so the compiler can't catch `ContainerID` being passed where `ServiceName` is expected. This is the single biggest type-safety win available.

2. **`ServiceInfo` is a 19-field god struct** — It mixes identity (`ServiceRef`), lifecycle (`RegisteredAt`, `FirstInvokedAt`, `ShutdownAt`), health (`HealthCheckCount`, `HealthCheckError`), and graph (`Dependencies`, `Dependents`) concerns. Splitting into `ServiceIdentity` / `ServiceLifecycle` / `ServiceHealth` / `ServiceGraph` would make the model more honest. Breaking change — decide before v0.1.0.

3. **`Report` has no constructor** — Consumers can construct invalid reports by hand. `NewReport(...)` returning `(Report, error)` would make invalid states unrepresentable. `Validate()` becomes a constructor check, not a post-hoc audit.

4. **JSON Schema is still missing** — The biggest gap for report consumers. A `schema.json` derived from Go types would enable validation in any language.

### Code Quality

5. **`replay.go` unused parameters** — `applyRegistrationAfter`, `applyInvocationBefore`, `applyHealthCheck` all take a `key svcKey` parameter that's now unused (the helper computes the key internally). Either remove the parameter or use it.

6. **LSP diagnostics are stale** — gopls reports warnings for `upsertServiceRecord` and `benchLoop` that no longer exist. A `lsp_restart` or editor refresh would clear these.

7. **`example/register.go` demo duplication** — `do.ProvideValue` vs `do.OverrideValue` blocks are structurally identical (different values). This is intentional (showcases paired APIs) but art-dupl flags it. Document or add `//nolint` comment.

### Testing

8. **Property-based tests missing** — `Diff` and `MigrateReport` would benefit from property-based testing (random inputs, invariant assertions). Currently only example-based.

9. **HTML golden-file test** — No deterministic output test for the HTML visualization. Changes to `html.templ` could break rendering without detection.

### Process

10. **CHANGELOG.md not updated** — This session's changes (dedup helpers, scope-tree unification) should be in `[Unreleased]`.

11. **No release since v0.0.4** — Multiple feature commits since the last tag. Consider v0.0.5.

---

## f) TOP 25 THINGS TO DO NEXT

Sorted by impact ÷ effort (highest first).

### Tier 1 — High Impact, Low Effort (Do Now)

| #   | Task                                                   | Effort | Impact | Why                                                                  |
| --- | ------------------------------------------------------ | ------ | ------ | -------------------------------------------------------------------- |
| 1   | **Fix `replay.go` unused `key` parameters**            | 10min  | MED    | LSP warns on 3 unused params. Clean up or use them.                  |
| 2   | **Update CHANGELOG.md `[Unreleased]`**                 | 15min  | LOW    | Document dedup session changes.                                      |
| 3   | **Update AGENTS.md Architecture section**              | 15min  | MED    | File descriptions reference stale helper names.                      |
| 4   | **Add `//nolint` or comment on `example/register.go`** | 5min   | LOW    | Document that ProvideValue/OverrideValue duplication is intentional. |
| 5   | **Restart LSP to clear stale diagnostics**             | 1min   | LOW    | gopls shows warnings for deleted code.                               |

### Tier 2 — High Impact, Medium Effort

| #   | Task                                                            | Effort | Impact | Why                                                       |
| --- | --------------------------------------------------------------- | ------ | ------ | --------------------------------------------------------- |
| 6   | **Typed identifiers (`ContainerID`, `ScopeID`, `ServiceName`)** | 2h     | HIGH   | Compiler-enforced type safety. Biggest architectural win. |
| 7   | **JSON Schema generation from Go types**                        | 3h     | HIGH   | Enables cross-language validation. Blocks v0.1.0.         |
| 8   | **`Report` constructor (`NewReport() → (Report, error)`)**      | 2h     | HIGH   | Makes invalid reports unrepresentable.                    |
| 9   | **CSV/TSV export**                                              | 1h     | MED    | High value for data analysis workflows. Low effort.       |
| 10  | **Property-based `Diff` tests**                                 | 2h     | MED    | Catches symmetry/inverse bugs.                            |
| 11  | **HTML golden-file test**                                       | 1h     | MED    | Catches templ rendering regressions.                      |
| 12  | **v0.0.5 release**                                              | 30min  | MED    | Tag current state. Multiple commits since v0.0.4.         |

### Tier 3 — Medium Impact, Medium Effort

| #   | Task                                     | Effort | Impact | Why                                                             |
| --- | ---------------------------------------- | ------ | ------ | --------------------------------------------------------------- |
| 13  | **CLI tool for report conversion**       | 4h     | HIGH   | Standalone binary: `auditlog convert --format html report.json` |
| 14  | **Property-based `MigrateReport` tests** | 2h     | MED    | Validates schema migration robustness.                          |
| 15  | **`actionlint` in CI**                   | 30min  | LOW    | Validates `.github/workflows/ci.yml` syntax.                    |
| 16  | **Split `ServiceInfo` into 4 structs**   | 6h     | HIGH   | Breaking change. Decide before v0.1.0.                          |
| 17  | **Flake app for coverage gate**          | 1h     | LOW    | Replaces inline shell in CI.                                    |
| 18  | **WebSocket live stream example**        | 3h     | MED    | Bridges `OnEvent` to browser dashboards.                        |
| 19  | **Prometheus exporter example**          | 2h     | MED    | Parallel to existing OTel example.                              |

### Tier 4 — Lower Priority

| #   | Task                             | Effort | Impact | Why                                                         |
| --- | -------------------------------- | ------ | ------ | ----------------------------------------------------------- |
| 20  | **Fuzz filter inputs**           | 2h     | LOW    | Arbitrary `ReportOption` combinations.                      |
| 21  | **DOT diagram format**           | 3h     | LOW    | 3rd diagram format. `go-output` now viable.                 |
| 22  | **`go-output` adoption**         | 4h     | LOW    | Replaces custom Mermaid/PlantUML.                           |
| 23  | **NDJSON import (`ReadNDJSON`)** | 1h     | LOW    | Already effectively done via `ReadEvents` + `ReplayEvents`. |
| 24  | **RELEASING.md checklist**       | 30min  | LOW    | Already in CONTRIBUTING.md.                                 |
| 25  | **Multi-module split**           | —      | —      | Explicitly rejected. Too small.                             |

---

## g) TOP QUESTION I CANNOT FIGURE OUT MYSELF 🤔

### "Should we split `ServiceInfo` into 4 structs before v0.1.0, or ship the monolith and split in v0.2.0?"

**Context:** `ServiceInfo` has 19 fields spanning 4 concerns:

- **Identity:** `ServiceRef` (ScopeID, ScopeName, ServiceName)
- **Lifecycle:** `Status`, `ServiceType`, `RegisteredAt`, `FirstInvokedAt`, `InvocationCount`, `InvocationOrder`, `FirstBuildDurationMs`, `ShutdownAt`, `ShutdownDurationMs`, `ShutdownError`, `InvocationError`
- **Health:** `LastHealthCheckAt`, `HealthCheckError`, `HealthCheckCount`
- **Graph:** `Dependencies`, `Dependents`, `IsHealthchecker`, `IsShutdowner`

**The dilemma:**

- **Split now (v0.1.0):** Cleaner model, but breaking change. Anyone using `svc.HealthCheckCount` would need `svc.Health.Count`. Forces a decision before the API is "frozen."
- **Ship monolith (v0.1.0), split later (v0.2.0):** Gets to a stable release faster. But then v0.1.0 consumers have a worse API, and the migration is harder once people depend on the flat struct.

**What I'd decide if forced to choose:** Ship the monolith in v0.1.0. The 19-field struct is ugly but functional. The split is a breaking change that deserves its own release cycle with migration tooling. Get to "stable" first, then improve.

**What I need from you:** Do you agree? Or should we bite the bullet and split before v0.1.0?

---

## Metrics Snapshot

| Metric                    | Value  | Trend                                      |
| ------------------------- | ------ | ------------------------------------------ |
| Total LOC (`.go` files)   | 10,685 | Stable                                     |
| Test LOC                  | 7,335  | ↓ (dedup removed ~180 lines)               |
| Production LOC            | 3,350  | ↓ (dedup removed ~60 lines)                |
| Test functions            | 274    | Stable                                     |
| Benchmarks                | 11     | Stable                                     |
| Fuzz targets              | 4      | Stable                                     |
| Coverage                  | 95.4%  | ✅ Meets 95% gate                          |
| Clone groups (-t 30)      | 1      | ↓ from 7                                   |
| Clone groups (-t 15)      | 29     | ↓ from 37 (all are helper calls or idioms) |
| Production clones (-t 15) | 0      | ✅ Zero                                    |
| CI jobs                   | 5      | ✅ All pass                                |
| Commits this session      | 7      | All pushed                                 |

# Status Report: Post-T1-T4 Implementation — 2026-06-18 14:04

> **Scope:** Full project state after shipping the replay engine, NDJSON reader, and loader API.
> **Branch:** `master` · **Working tree:** clean · **Head:** `39ef88a`
> **Latest release:** `v0.0.4` · **Schema version:** `0.2.0` · **Coverage:** 95.3% (gate: ≥95%)

---

## Executive Summary

This session delivered the **keystone of the CLI/NDJSON/Schema plan**: the replay engine that reconstructs a full `Report` from a flat event stream. With `ReplayEvents`, `ReadEvents`, and `LoadReport` now in place, the library can round-trip: capture → export NDJSON → read back → reconstruct → re-render in any format — all without a live container.

**Two commits shipped:**
1. `b8c592e` — feat: add replay engine, NDJSON reader, and loader API (T1-T4)
2. `39ef88a` — fix: consolidate test files, fix stack overflow, reach 95% coverage gate

**Critical fix in commit 2:** The initial implementation had a **stack overflow bug** in `buildReplayScopeTree` — empty scope IDs caused infinite recursion. The fuzz target `FuzzReadEvents` discovered it. This validates the fuzz-first approach.

**All CI gates now pass:** 0 lint issues, 95.3% coverage, race-clean, 4 fuzz targets (3 existing + 1 new).

**Verdict:** The hardest technical nut is cracked. The remaining plan tasks (CLI, JSON Schema, diff extensions) are all lower-risk wiring over proven APIs.

---

## a) FULLY DONE ✅

### New This Session (T1-T4 + fixes)

| Task | File | Lines | Status |
| --- | --- | --- | --- |
| **T1+T2: Replay engine** | `replay.go` | 414 | ✅ `ReplayEvents([]Event) (Report, error)` — reconstructs Services + ScopeTree from events, uses `buildReportFromCore` finalizer, sets `Reconstructed=true` |
| **T3: NDJSON reader** | `ndjson.go` | 85 | ✅ `ReadEvents(io.Reader)` — bufio.Scanner, 1MB cap, blank-line skip, per-line error wrapping, sentinel errors |
| **T4: Loader API** | `loader.go` | 176 | ✅ `LoadReport(path)` auto-detects JSON vs NDJSON by inspecting first line for `version` vs `event_type` key |
| **Report.Reconstructed field** | `report.go` | +13 | ✅ Additive `bool` field (JSON: `reconstructed,omitempty`) — lets consumers detect capability-flag absence |
| **Stack overflow fix** | `replay.go` | +2 | ✅ Self-reference guard `meta.id != parentID` in recursive `build()` |
| **Fuzz target** | `replay_test.go` | +15 | ✅ `FuzzReadEvents` — validates no-panic on arbitrary input |
| **Test suite** | `replay_test.go` | 1477 | ✅ 44 test functions + 1 fuzz target in a single consolidated file |

### Pre-Existing (verified intact, all gates pass)

| Capability | Evidence |
| --- | --- |
| Plugin lifecycle hooks (registration/invocation/shutdown/healthcheck) | `hooks.go` (324 LOC) |
| Stack-based dependency graph inference | `hooks.go`, `report_builder.go` |
| Concurrent-safe recorder (single RWMutex + 2 atomics) | `recorder.go` |
| 5 export formats (JSON, NDJSON, HTML, Mermaid, PlantUML) | `plugin.go`, `report.go`, `html.go`, `mermaid.go`, `plantuml.go` |
| Report model with 16 fields + Validate() | `report.go` |
| Report.Diff directional diff | `diff.go` |
| Report.Filtered with 5 filter options | `filter.go` |
| Schema migration v0.1.0 → v0.2.0 | `migration.go` |
| HTML dashboard (5-tab, Sugiyama graph, waveform, XSS-hardened) | `html.templ` |
| CI: 5 jobs (test 95% gate, lint, vulncheck, mod-tidy, stale-generation) | `.github/workflows/ci.yml` |

### Metrics

| Metric | Value |
| --- | --- |
| Test functions | 197 |
| Benchmarks | 11 |
| Fuzz targets | 4 |
| Godoc examples | 7 |
| Source + test LOC | 10,831 |
| Coverage | 95.3% |
| Lint issues | 0 |
| Source files (non-test) | 24 |
| Test files | 25 |

### Planning Artifacts

| Artifact | Commit | Purpose |
| --- | --- | --- |
| Pareto plan HTML + D2 | `e6e7d52`, `c3a380d` | 26 medium tasks, 95 fine tasks, ~25h total |
| Pre-execution status report | `07fde6d` | Honest gap analysis before execution |

---

## b) PARTIALLY DONE ⚠️

### Replay engine — functional but with documented limitations

`ReplayEvents` works for all event types but has two honest gaps:

1. **`IsHealthchecker`/`IsShutdowner` always false** — these require `do.ExplainInjector` on a live `*do.Scope`. A replayed Report has no live container. `Report.Reconstructed=true` signals this.

2. **Scope tree hierarchy is flattened** — events carry `scope_id`/`scope_name` but not `parent_id`. The first-seen scope becomes root; all others are its direct children. This is a data limitation, not a code bug.

### Diff capability — works but doc still lies

`diff.go:43` still claims dependency edges are compared. `compareService` only checks Status, InvocationCount, HealthCheckCount, and error-transition. The fix (T10) is not yet implemented.

### NDJSON format — write side complete, read side new

Write: `WriteEventsNDJSON` (existing). Read: `ReadEvents` (new). But there is no in-band schema version marker in NDJSON output — an importer cannot detect future-format files.

---

## c) NOT STARTED ❌

From the Pareto plan, remaining tasks:

| Task | Plan ID | Effort | Notes |
| --- | --- | --- | --- |
| CLI skeleton (`cmd/auditlog` + cobra) | T5 | 60m | Zero scaffolding exists |
| CLI `import` subcommand | T6 | 75m | Depends on T4 (done) + T5 |
| CLI `export` subcommand | T7 | 60m | Depends on T4 (done) + T5 |
| JSON Schema file (`schema/report.schema.json`) | T8 | 75m | No machine-readable contract exists |
| Schema validation (`ValidateAgainstSchema`) | T9 | 60m | Depends on T8 |
| Diff: dependency edges | T10 | 75m | Fixes `diff.go:43` doc lie |
| Diff: scope tree | T11 | 60m | New `ScopeDiff` type |
| CLI `validate` subcommand | T12 | 45m | Depends on T9 |
| CLI `diff` subcommand | T13 | 60m | Depends on T10, T11 |
| CLI `info` subcommand | T14 | 30m | Summary stats |
| `nix build .#auditlog` binary | T18 | 45m | Replaces README stub in `flake.nix` |
| samber/ro reactive adapter | T26 | 75m | `EventsAsObservable` via BehaviorSubject |
| CI cross-compile matrix | T19 | 60m | linux/darwin/windows × amd64/arm64 |
| CI smoke test | T20 | 30m | Build + `--version` + `import` fixture |
| README CLI section | T21 | 45m | Install + examples |
| AGENTS.md update | T22 | 30m | File inventory + replay caveats |
| FEATURES.md + TODO_LIST.md | T23 | 30m | Flip items to DONE |
| cli-workflow.md example | T24 | 45m | Round-trip tutorial |
| CHANGELOG | T25 | 30m | Unreleased section |

---

## d) TOTALLY FUCKED UP 💥

### D1. `diff.go:43` doc comment still lies — integrity bug

Still unfixed. The doc claims dependency edges are diffed; `compareService` doesn't touch Dependencies/Dependents. T10 will fix both the code and the doc.

### D2. NDJSON has no schema version marker — format trap

Still unfixed. Each NDJSON line is bare JSON with no header. Deferred to roadmap (requires format change).

### D3. Capability flags unrecoverable from events — honesty gap

Still present, but now **mitigated**: `Report.Reconstructed=true` lets consumers detect this state. The flag is set by `ReplayEvents`.

### D4. Scope tree is flattened on replay — data limitation

Events don't carry parent scope IDs. The replay engine infers a flat tree (first scope = root, rest = children). Documented in `ReplayEvents` doc comment. Not fixable without changing the Event schema.

### D5. Pre-existing lint issue in helpers_test.go (now fixed)

The `//nolint:unparam` directive was on the wrong function (`provideUserServiceWithDB` instead of `provideUserServiceWithDeps`). Fixed in `39ef88a`.

---

## e) WHAT WE SHOULD IMPROVE 🛠️

### Architecture

1. **Typed identifiers** (TODO:54): `ContainerID`, `ScopeID`, `ServiceName` as distinct named string types. Currently all bare `string` — `ServiceRef{ScopeID: serviceName, ServiceName: scopeID}` compiles. Should be done before v0.1.0.

2. **Split `ServiceInfo`** (TODO:57): 21-field struct mixing identity, lifecycle, graph, health. Split into composed sub-structs before v0.1.0.

3. **`NewReport(...)` constructor** (TODO:56): Make invalid Reports unrepresentable at construction time.

4. **Replay logic duplicates Recorder hooks**: `replay.go` mirrors `hooks.go` state-machine logic. Documented as Risk D in the plan. Acceptable for now (~2500 LOC project), but a future refactor should extract shared `applyEvent` logic.

### Testing

5. **Property-based Diff tests**: `Diff(a,a)` should be empty; `Diff(a,b).Added == Diff(b,a).Removed`. Would have caught the doc/code drift.

6. **HTML golden-file test** (TODO:71): Currently tested via substring assertions. A committed golden file would catch visual regressions.

7. **Replay round-trip golden test**: Capture example/demo output, assert `ReplayEvents(ndjson) ≈ report` modulo capability flags. Currently tested ad-hoc, not as a committed fixture.

### Library Leverage

8. **samber/ro adapter (T26)**: Fills the live-streaming gap with Rx operators. Depguard already allows `samber/*`.

9. **santhosh-tekuri/jsonschema (T9)**: Pure Go JSON Schema validator for T8/T9. Zero transitive deps.

10. **spf13/cobra (T5)**: CLI framework matching samber ecosystem convention. Reversible decision — can upgrade to `charm.land/fang/v2` later.

---

## f) Top #25 things to get done next 🎯

Sorted by **impact × value ÷ effort** (descending).

| # | Task | Impact | Effort | Status |
| --- | --- | --- | --- | --- |
| 1 | **T10: Diff dependency edges** — implement dep comparison + fix `diff.go:43` doc lie | 🔴 High | 75m | Not started |
| 2 | **T8: JSON Schema file** — `schema/report.schema.json` for v0.2.0 | 🔴 High | 75m | Not started |
| 3 | **T5: CLI skeleton** — `cmd/auditlog` with cobra, `--version` | 🟠 Medium | 60m | Not started |
| 4 | **T6: CLI `import`** — `auditlog import <file> -o report.html` | 🟠 Medium | 75m | Not started |
| 5 | **T7: CLI `export`** — 5 formats via library APIs | 🟠 Medium | 60m | Not started |
| 6 | **T9: Schema validation** — embedded validator via santhosh-tekuri | 🟠 Medium | 60m | Not started |
| 7 | **T11: Diff scope tree** — flatten ScopeNode, set-diff scopes | 🟠 Medium | 60m | Not started |
| 8 | **T26: samber/ro adapter** — `EventsAsObservable()` | 🟠 Medium | 75m | Not started |
| 9 | **T13: CLI `diff`** — text + JSON output, exit 3 on non-empty | 🟡 Low | 60m | Not started |
| 10 | **T12: CLI `validate`** — `Report.Validate()` + schema check | 🟡 Low | 45m | Not started |
| 11 | **T14: CLI `info`** — summary stats | 🟡 Low | 30m | Not started |
| 12 | **T18: `nix build` binary** — replace README stub | 🟡 Low | 45m | Not started |
| 13 | **T17: CLI golden tests** — per-subcommand assertions | 🟡 Low | 75m | Not started |
| 14 | **Typed identifiers** — distinct string types for IDs | 🟡 Low | 60m | Not started |
| 15 | **`NewReport` constructor** — invalid states unrepresentable | 🟡 Low | 45m | Not started |
| 16 | **T19: CI cross-compile** — 6-target matrix | 🟡 Low | 60m | Not started |
| 17 | **T20: CI smoke test** — build + `--version` + import | 🟡 Low | 30m | Not started |
| 18 | **CSV/TSV export** — tabular export for spreadsheets | 🟡 Low | 60m | Not started |
| 19 | **Property-based Diff tests** — symmetry + identity | 🟡 Low | 60m | Not started |
| 20 | **HTML golden-file test** — deterministic fixture | 🟡 Low | 45m | Not started |
| 21 | **T21: README CLI section** — install + examples | 🟡 Low | 45m | Not started |
| 22 | **T22: AGENTS.md update** — file inventory + caveats | 🟡 Low | 30m | Not started |
| 23 | **T23: FEATURES + TODO sync** — flip to DONE | 🟡 Low | 30m | Not started |
| 24 | **T24: cli-workflow.md** — round-trip tutorial | 🟡 Low | 45m | Not started |
| 25 | **T25: CHANGELOG** — Unreleased section | 🟡 Low | 30m | Not started |

---

## g) Top #1 question I cannot figure out myself 🤔

### Q1: Should I build the CLI (T5-T7) next, or fix the diff doc lie (T10) first?

**The tension:**
- **T10 (diff fix)** is the highest-integrity issue — `diff.go:43` actively lies to consumers. It's a 75m fix that restores trust. But it doesn't unlock new user-facing capability.
- **T5-T7 (CLI)** is the highest user-value next step — it turns the replay engine into an installable tool. But it's 3-4h of work before any user sees benefit.

**My recommendation:** Fix T10 first (integrity before features), then immediately start T5-T7. The diff fix is small, self-contained, and removes a known lie from the codebase. The CLI can follow in the next session.

**But I cannot decide without you:** Do you want me to prioritize integrity (fix the lying doc + implement dep diff) or user value (ship the CLI so `auditlog import events.ndjson -o report.html` works end-to-end)?

---

_Generated 2026-06-18 14:04 against master @ `39ef88a`. All claims verified via `go test -race`, `golangci-lint run`, and coverage analysis._

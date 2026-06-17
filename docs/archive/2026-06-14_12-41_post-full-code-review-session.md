# Comprehensive Status Report — 2026-06-14

**Project**: `github.com/larsartmann/samber-do-auditlog`
**Date**: 2026-06-14 12:41
**Branch**: `master` (pushed to origin)
**Go**: 1.26.3 · **Schema**: v0.2.0 · **Status**: ALPHA

---

## Quick Metrics

| Metric                                 | Value                           |
| -------------------------------------- | ------------------------------- |
| Source files (main pkg)                | 19                              |
| Test files                             | 19                              |
| Example files                          | 4                               |
| Total `.go` files                      | 42                              |
| Source LOC (main pkg)                  | ~2,145                          |
| Total LOC (incl. tests)                | ~6,974                          |
| Test functions (Test + Example + Fuzz) | 130 pass, 0 fail                |
| Benchmarks                             | 11                              |
| Exported functions/methods             | 62                              |
| Exported types                         | 14                              |
| Lint issues                            | 0 (`golangci-lint run` clean)   |
| `go vet`                               | clean                           |
| `go test -race`                        | pass                            |
| go.mod direct dependencies             | 2 (`samber/do/v2`, `a-h/templ`) |
| golangci-lint linters enabled          | 108                             |
| TODO/FIXME/HACK comments               | 0                               |
| Features (DONE)                        | 72                              |
| Features (PLANNED)                     | 0                               |
| Features (NOT PLANNED)                 | 3 (rejected with rationale)     |

---

## a) FULLY DONE ✓

### Session Work (2026-06-14 — 4 commits)

| Commit    | What                                                                                                                                                                                           |
| --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `f14b94e` | **Bug fixes**: example eventLog slice capture bug, all-healthy health-check test, 6 thelper warnings, 5 HTML XSS vectors, diagram code deduplication, test sleep removal, float comparison fix |
| `2d2704c` | **Dead code cleanup**: `var _ = time.Now` hack, orphaned comments, reportFilter nil ceremony, AGENTS.md architecture listing (19 files documented)                                             |
| `9b09e04` | **Test quality**: vacuous assertion fix (TestEvent_HasError), filter type assertion strengthened, PlantUML writer-error test added, diagram service name assertions                            |
| `c1d8c4b` | **Accessibility**: ARIA tablist/tab/tabpanel roles, aria-selected sync, aria-label on graph buttons and search input, sr-only CSS class                                                        |

### Core Library — All Complete

- **Plugin lifecycle**: New(), Opts(), Report(), Export*(), Events(), RecordHealthCheck*
- **Event capture**: Registration, invocation, shutdown, health check (all with before/after phases)
- **Dependency inference**: Stack-based, LIFO fast path, O(1) common case
- **Report assembly**: ServiceInfo with forward + reverse deps, scope tree, event stream
- **5 export formats**: JSON, NDJSON, self-contained HTML, Mermaid, PlantUML
- **Report filtering**: By name, type, event type, time range, scope (with scope tree pruning)
- **Health check auditing**: Event recording, service health fields, report-level aggregates
- **Provider type tracking**: lazy/eager/transient/alias via `do.ExplainNamedService`
- **Capability detection**: IsHealthchecker, IsShutdowner via `do.ExplainInjector`
- **Schema migration**: v0.1.0 → v0.2.0 with round-trip preservation
- **HTML visualization**: 5 tabs (Services, Scopes, Graph, Timeline, Events), Sugiyama DAG, lifecycle waveform, stat cards
- **Zero-cost disabled mode**: Empty hooks, no recorder calls
- **Thread safety**: Single `sync.RWMutex`, atomic counters, onEvent outside lock
- **Config**: Env var toggle (`DO_AUDITLOG_ENABLED`), ContainerID validation, OnEvent callback

### Test Coverage

- **130 tests pass** (120 unit + 7 examples + 3 fuzz), 0 failures
- **11 benchmarks** covering hot paths
- **~95% statement coverage**
- External test package (`auditlog_test`) — tests only public API
- Shared helpers in `helpers_test.go` (providers, assertions, plugin construction)
- Fuzz tests: 3 targets, 6+ XSS vectors checked

---

## b) PARTIALLY DONE ⚠️

| Item                             | Status          | What Remains                                                                                                                                                                    |
| -------------------------------- | --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **HTML accessibility**           | ~60% done       | ARIA roles on tabs and labels done. Missing: `aria-pressed` on event filter chips, `scope="col"` on table headers, keyboard nav modifier key exclusion (TEXTAREA/SELECT/BUTTON) |
| **HTML XSS hardening**           | ~90% done       | All known vectors from review fixed. Fuzz tests pass. Remaining: `stripScriptTags` helper in fuzz tests is hand-rolled and case-sensitive                                       |
| **JS/Go constant deduplication** | 0% — documented | `ProviderType`/`ServiceStatus`/`EventType` string values hardcoded in both Go and JS. No single source of truth.                                                                |
| **AGENTS.md accuracy**           | ~85%            | Architecture listing updated. Some Gotchas entries reference old function names (`mermaidNodeID` renamed to `diagramNodeID`).                                                   |

---

## c) NOT STARTED

| Item                                                                                                                                        | Impact                                  | Effort        |
| ------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------- | ------------- |
| **Inject Go enum metadata into HTML** — eliminate JS/Go split brain by passing a `type_metadata` JSON block from Go into the templ template | High — eliminates 4+ hardcoded mappings | Medium (2-3h) |
| **Virtual scrolling for large reports** — DOM renders ALL services/events at once                                                           | Medium — degrades with 500+ services    | High (4-6h)   |
| **Debounced service search** — currently filters on every keystroke                                                                         | Low — fine for typical use              | Low (15 min)  |
| **`scope="col"` on all table headers**                                                                                                      | Low — a11y polish                       | Low (10 min)  |
| **`aria-pressed` on event filter chips**                                                                                                    | Low — a11y polish                       | Low (10 min)  |
| **Touch event support for graph pan/zoom**                                                                                                  | Low — mobile only                       | Medium (1h)   |
| **CSP `base-uri`/`frame-ancestors` directives**                                                                                             | Low — defense in depth                  | Low (5 min)   |
| **Debounce graph rendering** — `renderGraph` called on every tab switch but bails if SVG exists                                             | Low — already cached                    | —             |
| **README.md update** — reflect Mermaid/PlantUML/filtering additions                                                                         | Medium — user-facing docs               | Low (30 min)  |

---

## d) TOTALLY FUCKED UP! 💥

Nothing. All critical bugs from the review were fixed and verified. No regressions introduced. All tests pass, lint is clean, race detector is clean.

---

## e) WHAT WE SHOULD IMPROVE!

### Architecture

1. **JS/Go constant split brain** — The single biggest architectural debt. ProviderType emojis, ServiceStatus icons, EventType labels, and CSS color classes are duplicated across Go types and JS objects in `html.templ`. Adding a new enum value requires hunting through 4+ places. **Fix**: Inject a JSON metadata block from Go into the templ template so JS reads from data, not hardcoded objects.

2. **Report struct has no constructor** — All fields are exported. Anyone can write `Report{EventCount: 999, Events: nil}`. A `Validate()` method or constructor would enforce invariants. Currently relies on discipline.

3. **`Config.Validate()` is decoupled from `New()`** — The validation exists but `New()` doesn't call it. Users must remember to call `Validate()` separately. This is documented but fragile.

4. **`reportFilter` struct is unexported but all logic is in `filter.go`** — This is actually fine for a functional-options pattern. No change needed.

### Testing

5. **No integration test for the full HTML output** — Fuzz tests check XSS vectors, but no test verifies the complete HTML renders correctly for a realistic multi-service container. The existing HTML tests only check substring presence.

6. **Benchmarks don't track regressions** — No baseline recorded. Benchmarks exist but there's no CI integration to detect performance regressions.

7. **`stripScriptTags` in fuzz tests is fragile** — Hand-rolled, case-sensitive parser. Could produce false positives/negatives. Consider using a proper HTML parser or testing the escaped output directly.

### Process

8. **FEATURES.md and TODO_LIST.md are stale** — Last updated 2026-06-10. Don't reflect the diagram refactoring, accessibility work, or test quality improvements from this session.

9. **34 archived docs in `docs/archive/`** — Historical noise. Some reference old function names and deleted code. Consider periodic cleanup.

10. **LSP diagnostics show stale `diagram.go` warnings** — The golangci-lint language server cached pre-refactoring state. Running `golangci-lint run` fresh shows 0 issues, but IDE diagnostics are misleading. Restart the LSP to fix.

---

## f) Top 25 Things to Get Done Next

### High Impact / Low Effort (do first)

| #   | Task                                                                      | Impact             | Effort |
| --- | ------------------------------------------------------------------------- | ------------------ | ------ |
| 1   | Add `aria-pressed` on event filter chips + `scope="col"` on table headers | a11y completeness  | 15 min |
| 2   | Add CSP `base-uri 'none'; frame-ancestors 'none'`                         | security hardening | 5 min  |
| 3   | Update FEATURES.md with diagram, a11y, and test quality work              | docs freshness     | 20 min |
| 4   | Update TODO_LIST.md — mark all session items done                         | docs freshness     | 10 min |
| 5   | Update CHANGELOG.md with session commits                                  | release tracking   | 10 min |
| 6   | Update README.md with Mermaid/PlantUML/filtering API docs                 | user-facing docs   | 30 min |
| 7   | Clean up stale Gotchas in AGENTS.md referencing old function names        | accuracy           | 15 min |
| 8   | Add keyboard nav exclusion for TEXTAREA/SELECT/BUTTON                     | a11y correctness   | 10 min |

### High Impact / Medium Effort

| #   | Task                                                                     | Impact                    | Effort |
| --- | ------------------------------------------------------------------------ | ------------------------- | ------ |
| 9   | Inject Go enum metadata into HTML template (eliminate JS/Go split brain) | architecture              | 2-3h   |
| 10  | Add realistic multi-service HTML integration test                        | test confidence           | 1h     |
| 11  | Add `Report.Validate()` method to enforce count consistency              | type safety               | 30 min |
| 12  | Record benchmark baselines in CI                                         | perf regression detection | 1h     |
| 13  | Replace `stripScriptTags` with proper HTML escaping test                 | test robustness           | 30 min |
| 14  | Add debounce to service search input                                     | UX polish                 | 15 min |
| 15  | Add empty-state messages ("No services", "No events") to HTML tables     | UX polish                 | 30 min |

### Medium Impact / Low Effort

| #   | Task                                                              | Impact          | Effort |
| --- | ----------------------------------------------------------------- | --------------- | ------ |
| 16  | Remove unused `errConnectionRefused` sentinel (or document it)    | dead code       | 5 min  |
| 17  | Clean up 34 archived docs in `docs/archive/`                      | repo hygiene    | 30 min |
| 18  | Pin `go.mod` to `go 1.26` (remove patch number)                   | consumer compat | 5 min  |
| 19  | Add `@startuml skinparam` directives for better PlantUML defaults | diagram polish  | 15 min |
| 20  | Add Mermaid theme styling                                         | diagram polish  | 15 min |

### Medium Impact / Higher Effort

| #   | Task                                                       | Impact           | Effort |
| --- | ---------------------------------------------------------- | ---------------- | ------ |
| 21  | Add virtual scrolling/pagination for 500+ service reports  | scalability      | 4-6h   |
| 22  | Add touch event support for graph pan/zoom                 | mobile support   | 1h     |
| 23  | Add `go report card` badge and fix any issues              | community        | 30 min |
| 24  | Add `gosec` + `govulncheck` to CI pipeline                 | security CI      | 1h     |
| 25  | Consider `samber/lo` for filter/find boilerplate reduction | code conciseness | 1h     |

---

## g) Top Question I Cannot Figure Out Myself

**Should `New()` call `Config.Validate()` and return an error?**

Currently `New(config Config) *Plugin` is a single-return-value constructor that never fails. `Config.Validate()` exists as a separate method and is tested via godoc examples, but `New()` doesn't call it.

- **If we change `New()` to return `(*Plugin, error)`**: This is a breaking API change affecting ~100 call sites across all test files, example code, and any external consumers. It's the "correct" Go pattern (constructors validate), but the blast radius is large.
- **If we keep it as-is**: The validation is a ghost system — exists, tested, documented, but never automatically enforced. Users must remember to call `Validate()` before `New()`.

I initially tried to make it return an error but reverted because of the 100+ call site breakage. Should I:

1. Do the breaking change and update all call sites?
2. Add a `MustNew(config Config) *Plugin` that panics on invalid config?
3. Leave it as documented "call Validate() first" pattern?

This is a **public API design decision** that affects all consumers. I cannot make it unilaterally.

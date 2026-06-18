# Status Report: Pre-Execution Snapshot — 2026-06-18 10:32

> **Scope:** Full project state review before kicking off the CLI + NDJSON Import + JSON Schema execution plan.
> **Branch:** `master` · **Working tree:** clean · **Head:** `c3a380d` (docs(planning): update CLI/NDJSON architecture plan with ro-adapter and CSS polish)
> **Latest release:** `v0.0.4` (2026-06-17) · **Schema version:** `0.2.0` · **Coverage gate:** ≥ 95%
> **Format:** Markdown per explicit user request (status-report skill defaults to HTML; user override applies)

---

## Executive Summary

`samber-do-auditlog` is a **healthy, mature library** at the end of its v0.0.x hardening phase. Core plugin, lifecycle recording, health checks, five export formats, HTML dashboard, schema migration, and 95.5% coverage are all genuinely done and CI-gated. The project has reached a **feature plateau**: every "easy" feature is shipped; what remains is a coherent **round-trip story** (save → reload → diff → re-render) that turns the library into a telemetry tool rather than just a capture library.

This session produced **two planning artifacts** (Pareto plan + D2 graph, both committed) and **zero code changes**. The plan is ready to execute. The library is in a strong position to start that execution.

**Verdict:** Ready to begin Tier 1 (keystone: `ReplayEvents` engine) of the CLI/NDJSON/Schema plan. No blockers.

---

## a) FULLY DONE ✅

Verified against source, tests, and CI gates. These work and are protected by automated checks.

### Core Library (production-grade)

| Capability | Evidence |
| --- | --- |
| Plugin lifecycle hooks (registration/invocation/shutdown/healthcheck) | `hooks.go` (324 LOC), 6 hook methods + healthcheck wrapper |
| Stack-based dependency graph inference + reverse deps | `hooks.go`, `report_builder.go` |
| Concurrent-safe recorder (single `sync.RWMutex` + 2 atomics) | `recorder.go` (191 LOC) |
| Deterministic output (services sorted, scope tree sorted) | `report_builder.go` |
| Event cap (`MaxEvents`) + dropped-event counter | `plugin.go`, `recorder.go` |
| Zero-cost disabled mode | `plugin.go` (`Opts()` returns empty hooks) |
| Real-time `OnEvent` callback (outside the lock) | `plugin.go:27`, all 7 hook fire sites |
| Config validation (`ContainerID` path-sep rejection) | `plugin.go` (`Config.Validate`) |

### Report Model & Queries

| Capability | Evidence |
| --- | --- |
| `Report` struct with 15 fields + 8 denormalized aggregates | `report.go:20-41` |
| **Single construction path** (`buildReportFromCore` + `finalizeDenormalized`) | `report.go:103-123` — eliminates count drift |
| `Report.Validate()` — count + status consistency checks | `report.go:46-81`, 5 sentinel errors |
| `Report.Index()` O(1) lookups | `report.go` |
| Convenience queries: `ServiceByName`, `ServiceByRef`, `ServicesByScope`, `EventsByService`, `EventsByRef`, `EventsByType`, `FailedServices`, `UnhealthyServices` | `report.go` |
| `Report.Diff(other)` directional diff (added/removed/changed + event-count delta) | `diff.go` |
| `Report.Filtered(opts...)` with 5 filter options + pruned scope tree | `filter.go` |
| Schema migration v0.1.0 → v0.2.0 (version-agnostic re-derivation) | `migration.go` |

### Export Formats (5/5 done)

| Format | API | Notes |
| --- | --- | --- |
| JSON | `Report.WriteJSON`, `Plugin.WriteReportJSON`, `Plugin.ExportToFile` | Single encoding path via `Report.WriteJSON` |
| NDJSON | `Report.WriteNDJSON`, `Plugin.WriteEventsNDJSON`, `Plugin.ExportEventsToNDJSON` | Per-line `json.Encode`, incremental writes |
| HTML | `Plugin.WriteHTML`, `Plugin.ExportToHTML` | Self-contained, CSP-hardened, warm amber dashboard |
| Mermaid | `Report.WriteMermaid` | Themed flowchart via shared `diagramFormatter` |
| PlantUML | `Report.WritePlantUML` | Styled component diagram via shared `diagramFormatter` |
| Atomic file writes | `writeToFile` helper | Temp file + `os.Rename`, 64KB buffered |

### HTML Visualization (production-grade observability dashboard)

5-tab layout (Services/Scopes/Graph/Timeline/Events), Sugiyama layered DAG with pan/zoom/touch, lifecycle waveform, type badges, keyboard nav (ARIA tabs), table sorting, error tooltips, pagination, responsive breakpoints, XSS hardening (`esc()` + CSP). See prior status report `2026-06-18_00-04_html-ux-overhaul-comprehensive-review.md` for the full list.

### CI / Infrastructure

| Gate | Status |
| --- | --- |
| `go vet`, `go build`, `go test -race` | ✅ PASS |
| Coverage gate ≥ 95% (excluding `example/`) | ✅ 95.5% |
| golangci-lint v2.12.2 (extremely strict config) | ✅ 0 issues |
| govulncheck | ✅ PASS |
| `go mod tidy` drift check | ✅ PASS |
| `go generate ./...` stale-generation check | ✅ PASS |
| Nix devShell (Go 1.26.4, templ, golangci-lint, govulncheck) | ✅ `flake.nix` |

### Testing

| Metric | Value |
| --- | --- |
| Test functions | 147 (`^func Test`) |
| Benchmarks | 11 |
| Fuzz targets | 3 (HTML XSS, MigrateReport, Diagram special chars) |
| Godoc examples | 7 |
| `t.Parallel()` calls | 152 (~97% of eligible tests) |
| Test files | 22 (split by feature area) |

### Releases

`v0.0.1` (2026-06-10), `v0.0.2` (2026-06-11), `v0.0.3` (2026-06-17), `v0.0.4` (2026-06-17). `STABILITY.md` documents the 0.x stability promise.

### This Session (planning only, no code)

| Artifact | Commit | Purpose |
| --- | --- | --- |
| `docs/planning/2026-06-18_09-18-cli-ndjson-import-json-schema.html` | `e6e7d52`, updated `c3a380d` | Pareto plan: 26 medium tasks, 95 fine tasks, ~25h |
| `docs/planning/2026-06-18_09-18-cli-ndjson-import-json-schema.d2` | `e6e7d52`, updated `c3a380d` | D2 source for execution graph (regenerates byte-identical SVG) |

---

## b) PARTIALLY DONE ⚠️

### Diff capability

`Report.Diff` exists and works, but `compareService` (`diff.go:83-98`) inspects only Status, InvocationCount, HealthCheckCount, and an error-transition boolean. The doc comment at `diff.go:43` claims dependency edges are compared — **they are not**. Scope tree is also not diffed.

### NDJSON format

Write side is complete (`writeEventsNDJSON`). **Read side does not exist** — there is no `ReadEvents(io.Reader)` or `LoadReport(path)`. The library is currently write-only for NDJSON.

### Live event streaming

`Config.OnEvent` callback works, but is a push-only callback. There is no reactive/observable wrapper, no channel, no `iter.Seq`, no SSE/WebSocket. Consumers must roll their own pipeline.

### Schema validation

`Report.Validate()` checks internal count consistency. There is **no machine-readable JSON Schema file** and no external schema validator. Hand-edited or third-party JSON cannot be shape-validated.

---

## c) NOT STARTED ❌

These appear in FEATURES.md "WORTH CONSIDERING" or TODO_LIST.md "Future Priorities" with zero implementation.

| Feature | Source | Notes |
| --- | --- | --- |
| **CLI tool** (`cmd/auditlog`) | FEATURES.md:170, TODO:63 | Zero scaffolding exists; `flake.nix packages.default` is a README stub |
| **NDJSON import** (`ReadEvents` / `LoadReport`) | FEATURES.md:171, TODO:55 | Hardest piece: reconstruct Services + ScopeTree from event stream |
| **Replay engine** (`ReplayEvents([]Event) → Report`) | TODO:55 (implied) | Keystone blocker for NDJSON import + diff across time |
| **JSON Schema file** (`schema/report.schema.json`) | FEATURES.md:172, TODO:76 | Biggest missing piece for report consumers per TODO |
| **samber/ro reactive adapter** (`EventsAsObservable`) | New (T26 in plan) | Fills live-streaming gap with Rx operators; depguard already allows `samber/*` |
| **CSV/TSV export** | FEATURES.md:169, TODO:62 | Low effort, high value for spreadsheet workflows |
| **WebSocket live stream** | FEATURES.md:174, TODO:64 | Rejected for now; T26 (ro adapter) covers the live use case better |
| **Property-based tests** (`rapid`/`gopter`) | FEATURES.md:173, TODO:68-70 | Diff symmetry, MigrateReport round-trips, filter fuzzing |
| **HTML golden-file test** | TODO:71 | Deterministic multi-service report → committed golden |
| **Typed identifiers** (`ContainerID`, `ScopeID`, `ServiceName` distinct types) | TODO:54 | Low effort, high safety; breaking change |
| **`NewReport(...)` constructor validation** | TODO:56 | Make invalid reports unrepresentable |
| **Split `ServiceInfo` lifecycle concerns** (19→4 structs) | TODO:57 | Breaking; decide before v0.1.0 |
| **Prometheus exporter example** | TODO:77 | Parallel to OTel example |
| **`actionlint` in CI** | TODO:78 | Workflow validation |
| **Flake app for coverage gate** | TODO:80 | Replace inline CI shell |
| **v0.1.0 release** | TODO:75 | Blocked on JSON-schema-first decision |
| **gosec static analysis in CI** | FEATURES.md:168 | Alongside govulncheck |

---

## d) TOTALLY FUCKED UP 💥

Honest problems. None are catastrophic; all are fixable. Ordered by severity.

### D1. `diff.go` doc comment lies about its own behavior — **integrity bug**

`diff.go:43` claims:

> *"The comparison key is (scope_id, service_name). Timestamps and durations are intentionally ignored — only structural changes (added/removed services, **dependency edges**, status transitions, error appearances) are reported."*

The bolded phrase is **false**. `compareService` (`diff.go:83-98`) never touches `Dependencies` or `Dependents`. This is the kind of doc/code drift that destroys trust in a library — a consumer reading the doc will build on a false premise. The plan's T10 both **implements** dep-edge diffing and **fixes the doc**.

### D2. NDJSON has no schema version marker — **format trap**

Every Event line is bare JSON. There is no header, no `version` field, no sidecar. An importer cannot detect "this NDJSON came from a future v0.3.0 with new Event fields". `omitempty` field additions will silently round-trip; field removals will fail loudly with no context. This is documented as Risk C in the Pareto plan but **not fixable without a format change** — deferred to roadmap.

### D3. Capability flags (`IsHealthchecker`/`IsShutdowner`) are unrecoverable from events — **honesty gap**

These are populated by `enrichCapabilities()` calling `do.ExplainInjector` on a **live** `*do.Scope` reference. A replayed/imported Report has no live container, so these will silently be `false`. The plan's T2.5 adds a `Report.Reconstructed bool` flag so consumers can detect this — but until then, any future importer will produce subtly misleading Reports.

### D4. No CLI binary in a "library plus CLI" world — **distribution gap**

The repo declares `meta.mainProgram = "samber-do-auditlog"` in `flake.nix:50` but the `packages.default` derivation (lines 44-59) writes a README stub, not a binary. Anyone running `nix run .` or `nix build .` gets nothing useful. The comment admits this. The plan's T18 fixes it properly with `buildGoModule { subPackages = ["cmd/auditlog"]; }`.

### D5. No machine-readable contract for the report format — **integration blocker**

Consumers (the planned CLI, third-party tools, future dashboard rebuilds) have no JSON Schema to validate against. `Report.Validate()` only checks internal arithmetic; it cannot catch a hand-edited report with `event_type: "potato"`. This is the single biggest missing piece per TODO_LIST.md:76. The plan's T8 + T9 resolve it.

---

## e) WHAT WE SHOULD IMPROVE 🛠️

### Architecture / Type Models

1. **Typed identifiers** (TODO:54): `ContainerID`, `ScopeID`, `ServiceName` as distinct named string types. Today they are all bare `string`, so `ServiceRef{ScopeID: serviceName, ServiceName: scopeID}` compiles. This is a classic "stringly-typed" anti-pattern. **Low effort, high safety.** Should be done before v0.1.0.

2. **`NewReport(...)` constructor** (TODO:56): Today `Report` is a public struct anyone can instantiate with zero values. `Validate()` then catches problems at runtime. A `NewReport` returning `(Report, error)` would make invalid states unrepresentable — the same pattern already used for `Plugin.New`. **Low effort, aligns with existing codebase style.**

3. **Split `ServiceInfo`** (TODO:57): The 21-field `ServiceInfo` is a god object mixing identity, lifecycle, graph, and health concerns. Splitting into `ServiceIdentity` / `ServiceLifecycle` / `ServiceHealth` / `ServiceGraph` (composed in `ServiceInfo`) would clarify each axis. **Breaking change — must decide before v0.1.0.**

4. **`Reconstructed` flag on Report**: Once T2 (ReplayEvents) lands, this bool should be added so consumers can detect capability-flag absence. Additive, non-breaking.

5. **Reuse `MigrateReport` for JSON loading**: The existing `MigrateReport` already does version-agnostic re-derivation. The plan's T4.3 routes `LoadReportFromJSON` through it — **no new logic needed** for v0.1.0 → v0.2.0 upgrade on import.

### Library Leverage (don't reinvent wheels)

6. **`samber/lo`**: TODO_LIST.md:42 explicitly rejected `samber/lo` ("stdlib `slices`/`cmp` is sufficient"). This is **correct for the current codebase** and should not be reversed for existing code. However, the new CLI code (cmd/) will benefit from `lo` for terminal output helpers — reconsider the rejection **scoped to cmd/** only.

7. **`samber/ro` for live streaming (T26)**: Depguard already allows `github.com/samber/*`. A `ro.Observable[Event]` adapter gives filter/debounce/window/zip for free — strictly better than bespoke `chan`/`iter.Seq` plumbing. Fills the documented live-streaming gap.

8. **`santhosh-tekuri/jsonschema/v6` for schema validation (T9)**: Pure Go, zero transitive deps, draft 2020-12 compliant. Avoids pulling in `xeipuuv/gojsonschema` (unmaintained) or `go-playground/validator` (banned by `how-to-golang`).

9. **`spf13/cobra` for CLI (T5)**: Matches samber's own `do-template-cli` boilerplate and his `golang-cli` skill recommendation. Not the `how-to-golang` preferred `charm.land/fang/v2`, but fang wraps cobra — adopting cobra now keeps the fang upgrade path open. **Decision needed — see Question 1.**

10. **Reuse `buildReportFromCore` for replay**: The plan's T2.4 routes the replay engine's output through the existing finalizer. This means **counts cannot drift** between live and replayed Reports — the invariant is preserved by construction.

### Testing

11. **Property-based tests for Diff symmetry**: `Diff(a,a)` should be empty; `Diff(a,b).Added == Diff(b,a).Removed`. Currently only golden-style tests exist. PBT would have caught the doc/code drift in D1.

12. **HTML golden-file test** (TODO:71): The HTML output is currently tested via assertions on substrings (`assertHTMLContains`). A committed golden file would catch visual regressions deterministically.

13. **Filter fuzzing** (FEATURES.md:157): The only "partially functional" item. `MigrateReport`, HTML XSS, and diagram escaping are fuzzed; arbitrary `ReportOption` combinations are not.

### Process

14. **`actionlint` in CI** (TODO:78): The workflow at `.github/workflows/ci.yml` is hand-edited YAML. `actionlint` would catch syntax errors and deprecated action versions before they hit a PR.

15. **Coverage gate as flake app** (TODO:80): The inline shell at `ci.yml:27-36` is duplicated logic. A `nix run .#coverage-gate` would be reusable locally and in CI.

---

## f) Top #25 things to get done next 🎯

Sorted by **impact × value ÷ effort** (descending). Tier labels refer to the Pareto plan at `docs/planning/2026-06-18_09-18-cli-ndjson-import-json-schema.html`.

| # | Task | Impact | Effort | Source |
| --- | --- | --- | --- | --- |
| 1 | **T1+T2: Replay engine** — extract event-application logic from Recorder hooks into pure `ReplayEvents([]Event) → Report` | 🔴 Critical | L (100m) | Plan T1+T2 |
| 2 | **T3: NDJSON reader** — `ReadEvents(io.Reader) ([]Event, error)` with per-line errors | 🔴 Critical | S (45m) | Plan T3 |
| 3 | **T4: Loader API** — `LoadReport(path)` auto-detects JSON vs NDJSON | 🔴 Critical | M (60m) | Plan T4 |
| 4 | **D1 fix: Implement dependency-edge diffing + correct `diff.go:43` doc lie** | 🔴 High | M (75m) | Plan T10, integrity bug |
| 5 | **T15: Replay golden tests** — capture demo fixture, assert `ReplayEvents(ndjson) ≈ report` modulo capability flags | 🔴 High | L (90m) | Plan T15 |
| 6 | **T8: JSON Schema file** (`schema/report.schema.json`) for v0.2.0 — all 4 enums, RFC3339 timestamps, omitempty-aware required lists | 🔴 High | M (75m) | Plan T8, TODO:76 |
| 7 | **T9: Embedded schema validator** — `ValidateAgainstSchema(Report)` via `santhosh-tekuri/jsonschema` | 🟠 Medium | M (60m) | Plan T9 |
| 8 | **T5: CLI skeleton** — `cmd/auditlog` with cobra, `--version`, persistent flags | 🟠 Medium | M (60m) | Plan T5, FEATURES:170 |
| 9 | **T6: CLI `import`** — `auditlog import <file> -o report.html` round-trip works | 🟠 Medium | M (75m) | Plan T6 |
| 10 | **T7: CLI `export`** — 5 formats via library APIs | 🟠 Medium | S (60m) | Plan T7 |
| 11 | **T11: Diff scope tree** — flatten ScopeNode by path, set-diff added/removed scopes | 🟠 Medium | M (60m) | Plan T11 |
| 12 | **T26: samber/ro reactive adapter** — `EventsAsObservable()` via BehaviorSubject | 🟠 Medium | M (75m) | Plan T26 |
| 13 | **T2.5: Add `Report.Reconstructed` field** — lets consumers detect capability-flag absence | 🟡 Low | XS (10m) | Plan, Risk A mitigation |
| 14 | **T12: CLI `validate`** — `auditlog validate <file>` runs `Report.Validate()` + schema check | 🟡 Low | S (45m) | Plan T12 |
| 15 | **T13: CLI `diff`** — `auditlog diff <a> <b>` text + JSON output, exit 3 on non-empty | 🟡 Low | S (60m) | Plan T13 |
| 16 | **T14: CLI `info`** — summary stats, `--json` output | 🟡 Low | XS (30m) | Plan T14 |
| 17 | **T16+T17: NDJSON reader + CLI golden tests** — roundtrip, fuzz, exit codes | 🟡 Low | M (135m) | Plan T16+T17 |
| 18 | **Typed identifiers** — `ContainerID`, `ScopeID`, `ServiceName` distinct types; breaking change | 🟡 Low | S (60m) | TODO:54 |
| 19 | **`NewReport(...)` constructor** — invalid Reports unrepresentable | 🟡 Low | S (45m) | TODO:56 |
| 20 | **T18: `nix build .#auditlog` binary** — replace README stub with real `buildGoModule` | 🟡 Low | S (45m) | Plan T18, D4 fix |
| 21 | **T19: CI cross-compile matrix** — linux/darwin/windows × amd64/arm64, ldflags version injection, artifact upload | 🟡 Low | M (60m) | Plan T19 |
| 22 | **CSV/TSV export** — tabular export of services/events for spreadsheets | 🟡 Low | S (60m) | FEATURES:169, TODO:62 |
| 23 | **Property-based Diff tests** — `rapid`/`gopter`, assert symmetry + identity | 🟡 Low | S (60m) | TODO:68 |
| 24 | **HTML golden-file test** — deterministic multi-service report → committed golden | 🟡 Low | S (45m) | TODO:71 |
| 25 | **T21-T25: Docs sync** — README CLI section, AGENTS.md update, FEATURES/TODO flip to DONE, cli-workflow.md, CHANGELOG | 🟡 Low | L (~3h total) | Plan T21-T25 |

**Out of scope for next iteration** (deferred to roadmap): WebSocket live stream, multi-module split, Prometheus dep, `encoding/json/v2` migration, HTML diff visualization, NDJSON sidecar metadata.

---

## g) Top #1 question I cannot figure out myself 🤔

### Q1: CLI framework choice — `spf13/cobra` vs `charm.land/fang/v2`?

The Pareto plan T5 specifies **`spf13/cobra`**. The `how-to-golang` required-libraries table specifies **`charm.land/fang/v2`** for CLI. These disagree, and the choice cascades through T5, T6, T7, T12, T13, T14, T17 (every CLI task).

**The facts as I understand them:**
- `fang/v2` is a **wrapper around cobra** that adds automatic help formatting, color, and manpage generation. Adopting cobra now keeps the fang upgrade path open.
- `how-to-golang` bans `urfave/cli` but does **not** ban cobra. `fang` is listed as the required CLI lib, but cobra is its foundation.
- The samber ecosystem (which this project lives in) uses **cobra** in `samber/do-template-cli`.
- This project currently uses **no CLI framework** and has no TUI dependencies. Adding fang pulls in the entire charm v2 stack (lipgloss, bubbletea, huh) — substantial transitive dep surface for a tool that just loads files and prints JSON/HTML.
- cobra alone is ~3 dependencies; fang pulls ~15+.

**What I cannot decide without you:**
Do we follow the `how-to-golang` policy strictly (fang/v2) and accept the larger dependency surface for a richer UX (colorized help, manpages)? Or do we scope a policy exception for this project (cobra only) because the CLI is a thin loader/exporter, not an interactive TUI?

My **recommendation** if forced to choose: **cobra only for v1**, with a comment in `cmd/auditlog/main.go` documenting the fang upgrade path. The CLI's job is file I/O and format conversion — colorized help is nice-to-have, not worth ~15 transitive deps. But this is a policy call, not a technical one, and I will not make policy calls for you.

---

_Generated 2026-06-18 10:32 against master @ `c3a380d`. All claims cross-referenced against source files, FEATURES.md, TODO_LIST.md, CHANGELOG.md, and the Pareto plan._

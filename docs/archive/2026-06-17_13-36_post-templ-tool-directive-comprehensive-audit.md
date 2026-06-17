# Status Report — 2026-06-17 13:36

## Post-Templ-Tool-Directive-Migration Comprehensive Audit

**Branch**: `master` · **Latest tag**: `v0.0.3` · **Working tree**: clean (after fix commit) · **CI**: 1 job failing (being fixed)

---

## Executive Summary

The project is in **strong ALPHA** state. All 17 original TODO items are complete. The library is feature-complete for its current scope: recording DI lifecycle events, building reports, and exporting to JSON/NDJSON/HTML/Mermaid/PlantUML. Test coverage is excellent at **95.3%** with **153 top-level test functions** (234 total including subtests).

However, this audit discovered **1 real latent bug** (scope-counting inconsistency breaking `Validate()` on migrated empty reports), **1 dead JavaScript feature** (offset timestamp formatting in HTML), **1 still-failing CI job** (stale generated code — now fixed in this session), and significant **documentation drift** (FEATURES.md is almost entirely stale, README has non-compiling example, AGENTS.md has duplicate bullets and stale references).

The templ CLI was just migrated to Go's `tool` directive this session, eliminating the systemic version-drift problem that plagued v0.0.3. This was the #1 item from the previous status report and is now resolved.

---

## A) FULLY DONE

### Production Code

| Feature                    | Status      | Details                                                                                     |
| -------------------------- | ----------- | ------------------------------------------------------------------------------------------- |
| Core plugin lifecycle      | ✅ Complete | `New(Config) (*Plugin, error)`, `Opts()`, `Report()`, `Events()`, `EventsCount()`           |
| Event recording            | ✅ Complete | Registration, invocation, shutdown, health-check events with timestamps, phases, durations  |
| Dependency graph inference | ✅ Complete | Invocation-stack-based dependency detection (A→B if A on-stack when B starts)               |
| Service type tracking      | ✅ Complete | Provider types (lazy/eager/transient/alias) via `do.ExplainNamedService`                    |
| Capability tracking        | ✅ Complete | `IsHealthchecker`/`IsShutdowner` via `do.ExplainInjector` (outside mutex to avoid deadlock) |
| Service status computation | ✅ Complete | Priority: invocation_error > shutdown_error > shutdown > active > registered                |
| Scope tree                 | ✅ Complete | Root + child scopes with deterministic sorting                                              |
| Health check recording     | ✅ Complete | Wrapper pattern (`RecordHealthCheck[WithContext]`), `EventTypeHealthCheck` events           |
| Config: MaxEvents          | ✅ Complete | Ring-buffer event dropping with `DroppedEventCount()`                                       |
| Config: OnEvent callback   | ✅ Complete | Real-time event streaming, called outside mutex                                             |
| Config: Validate()         | ✅ Complete | ContainerID path-separator check, sentinel errors                                           |

### Report & Export

| Feature                            | Status      | Details                                                                                                                               |
| ---------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| Report assembly                    | ✅ Complete | `BuildReport()` with services, scope tree, capabilities, event stream                                                                 |
| Report.Validate()                  | ✅ Complete | Denormalized count checks with sentinel errors                                                                                        |
| Report query methods               | ✅ Complete | `ServiceByName`, `ServiceByRef`, `ServicesByScope`, `EventsByService`, `EventsByType`, `FailedServices`, `UnhealthyServices`, `Index` |
| Report filtering                   | ✅ Complete | `Filtered(opts...)` with 5 filter options (by name, scope, type, status, service-type)                                                |
| Report.Diff()                      | ✅ Complete | Structural comparison (added/removed/changed services, event count delta)                                                             |
| Report.WriteJSON()                 | ✅ Complete | Indented JSON to writer                                                                                                               |
| Report.WriteNDJSON()               | ✅ Complete | Streaming events to writer                                                                                                            |
| Report.WriteMermaid()              | ✅ Complete | Flowchart diagram with theme                                                                                                          |
| Report.WritePlantUML()             | ✅ Complete | Component diagram with skinparams                                                                                                     |
| ExportToHTML / WriteHTML           | ✅ Complete | Self-contained HTML with 5-tab layout, CSP-hardened                                                                                   |
| ExportToJSON / WriteJSON           | ✅ Complete |                                                                                                                                       |
| ExportToNDJSON / WriteEventsNDJSON | ✅ Complete |                                                                                                                                       |
| ExportFilteredToFile               | ✅ Complete | Filtered export to file                                                                                                               |
| Schema migration                   | ✅ Complete | `MigrateReport` v0.1.0 → v0.2.0 with validation                                                                                       |

### Infrastructure

| Feature              | Status      | Details                                                         |
| -------------------- | ----------- | --------------------------------------------------------------- |
| CI pipeline          | ✅ Complete | 4 jobs: test (-race), lint, vulncheck, stale-generation         |
| Templ tool directive | ✅ Complete | `go tool templ generate` via go.mod — no external binary needed |
| flake.nix devShell   | ✅ Complete | Go 1.26.3, golangci-lint, govulncheck, golines                  |
| BENCHMARKS.md        | ✅ Complete | 13 benchmark cases baselined (3 runs each, median)              |
| STABILITY.md         | ✅ Complete | 0.x API stability promise (stable/evolving/internal surfaces)   |
| CONTRIBUTING.md      | ✅ Complete | Setup, testing, code style, releasing guide                     |
| OTel bridge example  | ✅ Complete | `docs/examples/otel-bridge.md` reference implementation         |
| fuzz tests           | ✅ Complete | 3 XSS fuzz targets for HTML export                              |

### Testing

| Metric                            | Value                   |
| --------------------------------- | ----------------------- |
| Top-level test functions          | 153                     |
| Total test cases (incl. subtests) | 234                     |
| Test files                        | 22                      |
| Coverage                          | **95.3%** of statements |
| Benchmark cases                   | 13 (11 functions)       |
| Fuzz targets                      | 3                       |
| Example functions                 | 7                       |
| Race detector                     | ✅ CI uses `-race` flag |

---

## B) PARTIALLY DONE

### 1. Documentation Freshness — **~60% accurate**

| Document                  | Accuracy | Key Issues                                                                                                     |
| ------------------------- | -------- | -------------------------------------------------------------------------------------------------------------- |
| `CHANGELOG.md`            | ✅ 95%   | Accurate and well-maintained. Minor stale test count in [0.0.1]                                                |
| `CONTRIBUTING.md`         | ✅ 95%   | Accurate after templ tool migration update                                                                     |
| `STABILITY.md`            | ✅ 95%   | Correct, marks new APIs as "evolving"                                                                          |
| `BENCHMARKS.md`           | ✅ 90%   | Internally accurate, but conflicts with README's performance section                                           |
| `AGENTS.md`               | 🟡 70%   | Duplicate bullet points (6 items listed twice), stale `newServiceRecordFromMeta()` ref, wrong test breakdown   |
| `README.md`               | 🟡 60%   | **Non-compiling example** (`New()` return value not handled), stale benchmark numbers, missing new API methods |
| `FEATURES.md`             | 🔴 25%   | **Almost entirely stale** — 8 of 9 "planned" items are done; missing many completed features; wrong line refs  |
| `TODO_LIST.md`            | 🟡 70%   | All items ticked, but stale LOC count, inconsistent benchmark count                                            |
| `docs/DOMAIN_LANGUAGE.md` | 🟡 50%   | References split-apart `recorder.go`, missing new commands                                                     |

### 2. Test Parallelism — **~15% of tests use t.Parallel()**

Only `status_internal_test.go`, `diff_export_test.go`, `plugin_provider_test.go` (partial), and `robustness_test.go` use `t.Parallel()`. The rest run sequentially. The test suite takes ~1s — not a bottleneck now, but parallelism would help as it grows.

### 3. Fuzz Coverage — **HTML-only, 3 targets**

All fuzz targets test XSS in HTML export. No fuzzing of:

- `MigrateReport([]byte)` — arbitrary JSON input (prime fuzz target)
- Service names in Mermaid/PlantUML output (special characters)
- Deeply nested scope trees
- Filter inputs

### 4. Metadata Testing — **Indirect only**

`BuildTypeMetadata()` in `metadata.go` is never directly tested. HTML tests only assert that section labels appear in output. Individual emoji values, labels, and colors are unverified.

---

## C) NOT STARTED

These are features and improvements that have been discussed but never attempted:

1. **Coverage gate in CI** — No minimum coverage threshold enforced. Coverage is 95.3% but could silently regress.
2. **`go mod tidy` check in CI** — No check that go.sum is in sync with go.mod.
3. **`golangci-lint config verify` in CI** — Catches schema issues that `golangci-lint run` silently skips. This caused a CI failure earlier this session.
4. **Pin Go version in go.mod** — go.mod says `go 1.26.3` but CI uses `1.26.4` (CVE fix). Could set `go 1.26.4` for auto-toolchain download.
5. **Action version upgrades** — `actions/checkout@v4` → v5, `actions/setup-go@v5` → v6 (when stable).
6. **Migrate to `actionlint`** — Validate GitHub Actions workflow syntax in CI.
7. **Property-based testing** — No `rapid`/`gopter` property tests. Would be valuable for `Diff`, `MigrateReport`, filter round-trips.
8. **NDJSON import** — Can export NDJSON but can't import it back. Would enable diffing across time.
9. **WebSocket live stream** — `OnEvent` callback exists but no WebSocket bridge for real-time dashboards.
10. **Prometheus exporter** — `OnEvent` → Prometheus metrics bridge. OTel example exists, no Prometheus one.
11. **JSON Schema file** — No machine-readable JSON schema for the report format.
12. **v0.1.0 release** — Project is stable enough for a v0.1.0 tag per STABILITY.md criteria.

---

## D) TOTALLY FUCKED UP!

### 🔴 1. CI Was Failing on Every Push (3 distinct bugs, now fixed)

**Root causes (all fixed this session):**

1. `tagliatelle` config used v1 schema (`rules.json: snake_case`) instead of v2 (`case.rules.json: snake`) — `golangci-lint run` silently accepted invalid config.
2. `golangci-lint-action@v6` incompatible with golangci-lint v2 — needs v7.
3. `//go:generate templ generate` used system PATH binary (Nix v0.3.1036) instead of go.mod-pinned v0.3.1020.

**Lesson**: After creating CI, ALWAYS push and wait for the first run to complete. The CI was green on paper but red on GitHub for 6 consecutive pushes.

### 🔴 2. CI STILL Failing After "Fix" (fixed in THIS session)

The templ tool-directive migration commit (30a4fac) still failed the stale-generation check. The committed `html_templ.go` had parenthesized imports (from a stale `~/go/bin/templ` binary), but `go tool templ generate` produces single-line imports. Fixed by regenerating with `go tool templ generate` (commit 53f4faf).

**Root cause**: Even with the `tool` directive in go.mod, the previously-committed file was generated by a different binary. The `go tool` mechanism correctly produces consistent output, but the committed artifact didn't match.

### 🔴 3. Latent Bug: Scope Count Mismatch in `MigrateReport`

`countUniqueScopes` in `migration.go:67` always returns ≥1 (even for an empty tree), but `countScopeNodes` in `report.go:77` returns 0 for the same empty tree. This means `MigrateReport([]byte({"version":"0.1.0"}))` sets `ScopeCount=1` while the tree has 0 nodes → `report.Validate()` fails with `errReportScopeCountMismatch`.

**Why it's latent**: `TestMigrateReport_EmptyReport` never calls `Validate()` on the migrated report.

### 🔴 4. Dead JavaScript Feature: Offset Timestamps in HTML

`html.templ:735-736` references `s.registered_offset_ns` and `s.first_invoked_offset_ns` in `formatNs()`, but the Go `ServiceInfo` struct never serializes these fields (it emits `registered_at`/`first_invoked_at` as ISO timestamps). Result: the "Registered"/"Invoked" tooltip in the services table is always empty. Half-implemented feature — relative offset timestamps were planned but never added to the backend.

### 🟡 5. FEATURES.md Is Almost Entirely Fiction

The PLANNED section lists 8 of 9 items as "not started" when they are actually **completed**. The file references `recorder.go` line numbers for functions that were split into separate files months ago. It's actively misleading.

---

## E) WHAT WE SHOULD IMPROVE!

### Architecture & Code Quality

1. **Unify scope-counting functions** — `countUniqueScopes` (migration.go) and `countScopeNodes` (report.go) do the same thing with different empty-tree semantics. This is the root cause of bug #3 above. Consolidate into one function.

2. **Remove dead JS** — Either implement the offset-timestamp feature in the Go backend or remove the `formatNs()` references from `html.templ`. Currently it's dead code that confuses readers.

3. **Fix fragile type assertion in `ResolveServiceScope`** — `healthcheck.go:58` does `injector.(*do.Scope)` without checking the fallback. If the injector is `*do.RootScope` or any wrapper, the ancestor walk is silently skipped.

4. **Replace magic `[2]bool` tuple** — `report_builder.go:252-254` uses `caps[0]`=healthchecker, `caps[1]`=shutdowner. A named struct (`struct{ healthchecker, shutdowner bool }`) would be clearer and self-documenting.

5. **Remove duplicate bullet points from AGENTS.md** — 6 items are listed twice (Benchmark suite, Disabled path, inferServiceType, newServiceRecordCore, Stack pop, serviceKey).

6. **Remove stale `newServiceRecordFromMeta()` reference** — AGENTS.md:135 documents a function that no longer exists (consolidated into `newServiceRecordCore`).

7. **Fix README non-compiling example** — Line 281: `plugin := auditlog.New(...)` ignores the error return value. `New()` returns `(*Plugin, error)`.

8. **Rebuild FEATURES.md from scratch** — Current file is worse than no file (actively misleading). Use the features-audit skill to regenerate from actual code.

### Testing

9. **Add `Validate()` call to `TestMigrateReport_EmptyReport`** — Would catch bug #3 immediately.

10. **Add `MigrateReport` fuzz target** — Takes `[]byte`, perfect for fuzzing with arbitrary JSON.

11. **Add direct tests for `BuildTypeMetadata()`** — Verify every emoji, label, and color value.

12. **Add `t.Parallel()` to all independent tests** — Most tests create their own Plugin + Injector, no shared state. Safe to parallelize.

13. **Test `NewRecorder()` directly** — Exported constructor never tested in isolation.

### CI/DevX

14. **Add `golangci-lint config verify` as a CI step** — Catches schema issues that `golangci-lint run` silently ignores.

15. **Add coverage gate** — `go test -coverprofile=cover.out ./...` + fail if < 95%.

16. **Add `go mod tidy` check** — Ensure go.sum stays in sync.

17. **Remove `~/go/bin/templ`** — Stale globally-installed binary caused the latest CI failure. The `tool` directive makes it unnecessary.

18. **Review experimental build tags** — `.golangci.yml` enables `goexperiment.jsonv2`, `goexperiment.arenas`, `goexperiment.simd`, etc. These lint under experimental features most users don't run. `goexperiment.jsonv2` especially contradicts the documented "explicitly rejected" stance.

### Documentation

19. **Sync README benchmark numbers with BENCHMARKS.md** — They currently disagree (Invocation: README says 1,305ns/7 allocs, BENCHMARKS.md says 1,658ns/6 allocs).

20. **Add `WriteNDJSON`, `WriteJSON`, `Diff` to README API tables** — These are implemented and tested but missing from documentation.

21. **Update `docs/DOMAIN_LANGUAGE.md`** — References `recorder.go` for functions that now live in other files. Missing `Diff`, `WriteNDJSON`, `WriteJSON` from Commands section.

---

## F) Top 25 Things to Get Done Next

Sorted by `Impact × Customer-Value ÷ Effort`:

| #   | Task                                                                                                            | Impact  | Effort     | Category     |
| --- | --------------------------------------------------------------------------------------------------------------- | ------- | ---------- | ------------ |
| 1   | **Fix scope-counting bug** — unify `countUniqueScopes`/`countScopeNodes`, add `Validate()` to empty-report test | 🔴 High | 🔵 Low     | Bug fix      |
| 2   | **Remove dead `formatNs()` JS** or implement offset timestamps in backend                                       | 🔴 High | 🔵 Low     | Bug fix      |
| 3   | **Remove stale `~/go/bin/templ`** binary — eliminates future drift                                              | 🟠 Med  | ⚪ Trivial | DevX         |
| 4   | **Rebuild FEATURES.md** from actual code (features-audit skill)                                                 | 🟠 Med  | 🔵 Low     | Docs         |
| 5   | **Fix README non-compiling example** — `New()` error handling                                                   | 🟠 Med  | ⚪ Trivial | Docs         |
| 6   | **Deduplicate AGENTS.md** — remove 6 duplicate bullets, stale refs                                              | 🟡 Low  | ⚪ Trivial | Docs         |
| 7   | **Add `golangci-lint config verify`** to CI and CONTRIBUTING.md                                                 | 🟠 Med  | ⚪ Trivial | CI           |
| 8   | **Add coverage gate to CI** — fail if < 95%                                                                     | 🟠 Med  | ⚪ Trivial | CI           |
| 9   | **Add `go mod tidy` check to CI**                                                                               | 🟡 Low  | ⚪ Trivial | CI           |
| 10  | **Fix fragile type assertion** in `ResolveServiceScope` (healthcheck.go:58)                                     | 🟠 Med  | 🔵 Low     | Bug fix      |
| 11  | **Sync README benchmark numbers** with BENCHMARKS.md                                                            | 🟡 Low  | ⚪ Trivial | Docs         |
| 12  | **Add `MigrateReport` fuzz target**                                                                             | 🟠 Med  | 🔵 Low     | Testing      |
| 13  | **Add direct `BuildTypeMetadata` tests**                                                                        | 🟡 Low  | 🔵 Low     | Testing      |
| 14  | **Replace magic `[2]bool`** with named struct in capability enrichment                                          | 🟡 Low  | 🔵 Low     | Code quality |
| 15  | **Add `t.Parallel()` to independent tests**                                                                     | 🟡 Low  | 🟡 Med     | Testing      |
| 16  | **Update `docs/DOMAIN_LANGUAGE.md`** — fix file refs, add new commands                                          | 🟡 Low  | 🔵 Low     | Docs         |
| 17  | **Set `go 1.26.4` in go.mod** — patched version via auto-toolchain                                              | 🟡 Low  | ⚪ Trivial | DevX         |
| 18  | **Add missing API methods to README tables** (WriteNDJSON, WriteJSON, Diff)                                     | 🟡 Low  | ⚪ Trivial | Docs         |
| 19  | **Review experimental build tags** in .golangci.yml                                                             | 🟡 Low  | 🔵 Low     | Config       |
| 20  | **Test `NewRecorder()` directly**                                                                               | 🟢 Low  | 🔵 Low     | Testing      |
| 21  | **Add Prometheus exporter example** (parallel to OTel example)                                                  | 🟢 Low  | 🟡 Med     | Docs         |
| 22  | **JSON Schema file** for the report format                                                                      | 🟡 Low  | 🟡 Med     | Docs         |
| 23  | **Property-based tests** for Diff, MigrateReport, filter round-trips                                            | 🟢 Low  | 🟠 High    | Testing      |
| 24  | **NDJSON import** — enable loading events back from NDJSON                                                      | 🟢 Low  | 🔴 High    | Feature      |
| 25  | **v0.1.0 release** — project meets STABILITY.md criteria                                                        | 🟠 Med  | 🟡 Med     | Release      |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should we rebuild FEATURES.md now, or is it better to delete it until the project stabilizes past v0.1.0?**

FEATURES.md is currently 75% fiction — it lists completed work as "planned" and references file locations that haven't existed since the `recorder.go` split. Every status report has noted this, but nobody has fixed it. The question is whether to:

- **Option A**: Regenerate it from actual code using the features-audit skill (effort: ~15 min, produces accurate inventory)
- **Option B**: Delete it entirely until the API stabilizes (the CHANGELOG + STABILITY.md already serve as the source of truth for what exists and what's stable)
- **Option C**: Replace with a simple "see CHANGELOG.md" pointer (avoids drift, preserves the file for future use)

The risk with Option A is that it will drift again unless someone owns it. The risk with Option B is losing a useful navigational document for new contributors. I can't determine which tradeoff the project owner prefers.

---

## Verification Snapshot

```
Build:       ✅ go build ./... — clean
Vet:         ✅ go vet ./... — clean
Tests:       ✅ 153 functions, 234 cases — all PASS (with -race)
Coverage:    ✅ 95.3% of statements
Lint:        ✅ golangci-lint v2.12.2 — 0 issues
Generate:    ✅ go generate ./... — no drift (after fix)
Govulncheck: ⚠️ Not installed locally (CI runs it via action)
CI:          🔴 stale-generation job failing on commit 30a4fac → FIXED in 53f4faf (pending CI run)
```

---

## File Inventory

### Production Code (20 files, 3,087 LOC)

```
diff.go             event.go          export.go          filter.go
healthcheck.go      hooks.go          html.go            html_templ.go (generated)
mermaid.go          metadata.go       migration.go       plantuml.go
plugin.go           recorder.go       report.go          report_builder.go
report_helpers.go   service.go        types.go           doc.go
```

### Test Code (22 files, 5,229 LOC)

```
benchmarks_test.go       diagram_test.go          diff_export_test.go
example_test.go          extra_test.go            fuzz_test.go
healthcheck_basic_test.go  healthcheck_export_test.go  helpers_test.go
migration_test.go        plugin_basic_test.go     plugin_errors_test.go
plugin_export_test.go    plugin_html_test.go      plugin_lifecycle_test.go
plugin_provider_test.go  plugin_scope_test.go     report_filter_test.go
report_query_test.go     robustness_test.go       status_internal_test.go
type_method_test.go
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
docs/status/ (multiple reports)
docs/archive/ (historical)
docs/planning/ (historical)
```

# Status Report: Post-Deduplication & Architecture Cleanup

**Date:** 2026-06-09 23:32
**Branch:** master (ahead of origin by 6 commits)
**Total commits:** 28
**Test coverage:** 92.9% (production code)
**Clone groups:** 0 at t=50, 8 at t=15 (all acceptable idioms)

---

## A) FULLY DONE

| Item | Detail |
|------|--------|
| README data model | Fixed stale fields (`service_type` removed, `build_duration_ms` renamed, `version`/`container_id`/`shutdown_duration_ms` added) — commit `6533ee9` |
| ScopeID on DependencyRef | Added for unambiguous machine consumption — commit `da3d5c1` |
| Event constructor consolidation | 3 constructors (`newRegistrationEvent`, `newInvocationEvent`, `newShutdownEvent`) → single `newEvent(eventType, ...)` — eliminated ~60 lines — commit `27559d8` |
| `scopeKey` helper | Extracted canonical `scope.ID() + "/" + serviceName` key builder used 6x — commit `27559d8` |
| Tests for shutdown duration, ContainerID, Version, ScopeID | 4 new test assertions — commit `710b81a` |
| Deterministic output | Dependencies, dependents, scope services all sorted — commit `afed0ff` |
| Benchmark modernization | `for range b.N` → `for b.Loop()` in all 3 benchmarks — commit `189f4cb` |
| Test helper extraction | `provideDB()` (13 callsites), `findServiceByName()` (6 callsites) — eliminated largest clone groups |
| Lint config | `.golangci.yml` with 90+ linters, 0 issues on production code |
| HTML visualizer | Self-contained templ-based dark page with stats, services table, dependency graph, timeline, events table |
| Plugin API | `New()`, `Opts()`, `Report()`, `Events()`, 3x writer methods, 3x file export methods, `ExportToHTML()` |
| Env var toggle | `DO_AUDITLOG_ENABLED` with 7 value tests + explicit override |
| Example app | 5-service demo with `Enabled: true`, JSON/NDJSON/HTML export |

### Session Summary (6 commits since origin/master)

```
afed0ff Sort dependencies, dependents, and scope services for deterministic output
710b81a Add tests for shutdown duration, ContainerID, Version, ScopeID
189f4cb Minor cleanups: benchmark loop style, trailing whitespace, gitattributes
27559d8 Consolidate 3 event constructors into single newEvent, extract scopeKey
da3d5c1 Add ScopeID to DependencyRef for unambiguous machine consumption
6533ee9 Fix README: update data model, JSON examples, deps, Go version
```

---

## B) PARTIALLY DONE

| Item | What's Done | What's Missing |
|------|-------------|----------------|
| Test deduplication | `provideDB` and `findServiceByName` helpers extracted, 19 callsites replaced | 8 clone groups remain at t=15 (all acceptable Go idioms: assertion variants, benchmark bodies, example code) |
| Clone elimination | 0 clones at t=50 (industry standard), down from unknown baseline | t=15 aggressive threshold shows 8 groups — all `💠 idiom` / `🟢 low` priority |

---

## C) NOT STARTED

| Item | Impact | Effort |
|------|--------|--------|
| Add `go:generate` directive to `html.templ` | MED — eliminates manual `templ generate` | S |
| Add `ProvideOverride` test | LOW — edge case | S |
| Test multi-scope dependency tracking (child invoking parent service) | MED | M |
| Add CI GitHub Action (`go test`, `golangci-lint run`) | MED | M |
| Add `go test -cover` baseline to CI | MED | S |
| Add `t.Parallel()` to all safe tests | LOW | S |
| Fix example `Connect()`/`Start()` to not return always-nil errors | LOW | S |
| Investigate `html/template` import in `html.go` (potentially dead) | MED | S |
| Add DOT/Mermaid graph export | HIGH | M |
| Add OpenTelemetry bridge | HIGH | H |
| Add configurable event filtering | MED | M |
| Thread-safe benchmark for concurrent invocations | LOW | M |

---

## D) TOTALLY FUCKED UP

Nothing is broken right now. All green:

- `go test ./...` — PASS (17 tests)
- `go vet ./...` — clean
- `go build ./...` — clean
- `golangci-lint run` — 0 issues on production code
- `art-dupl -t 50` — **0 clone groups**

Previous issues that were fixed this session:

| What Was Fucked | Fixed In | How |
|-----------------|----------|-----|
| README data model was stale (lying to users) | `6533ee9` | Updated all fields, JSON examples, Go version |
| 3 duplicated event constructors (~60 lines) | `27559d8` | Consolidated into single `newEvent` with `EventType` param |
| `scope.ID() + "/" + serviceName` repeated 6x | `27559d8` | Extracted `scopeKey` helper |
| DependencyRef ambiguous without ScopeID | `da3d5c1` | Added `ScopeID` field |
| No tests for shutdown duration, ContainerID, Version | `710b81a` | Added 4 new test assertions |
| Non-deterministic JSON output | `afed0ff` | Sorted all slice fields in `buildServicesLocked` |
| Benchmarks using deprecated `for range b.N` | `189f4cb` | Modernized to `for b.Loop()` |
| Test code had massive duplication (47 clones at t=15) | this session | Extracted `provideDB` + `findServiceByName`, reduced to 8 groups |

---

## E) WHAT WE SHOULD IMPROVE

### Code Quality

1. **Extract `DurationMs` custom type** — `*float64` is a primitive obsession smell; a `DurationMs float64` type with `String()` and `Microseconds()` methods would be self-documenting
2. **Add `go:generate` to `html.templ`** — running `templ generate` manually is error-prone; a `//go:generate templ generate` header ensures it's always fresh
3. **Document lock ordering** — `Recorder` has 4 mutexes (`mu`, `stackMu`, `invocationMu`, `shutdownMu`); the ordering contract should be explicit in a comment
4. **Consider `Recorder` interface** — would allow testing `Plugin` without the full `Recorder` implementation
5. **Investigate dead `html/template` import in `html.go`** — LSP may be right that it's unused after templ migration

### Testing

6. **Add test coverage baseline** — 92.9% is good but not tracked; CI should fail on regression
7. **Test multi-scope dependency tracking** — child scope invoking parent service
8. **Add `t.Parallel()` to safe tests** — free speedup, catches race conditions

### Architecture

9. **Parent scopeID bug was lurking** — `parentKey` was incorrectly using `scope.ID()` instead of `parent.scopeID` before commit `27559d8`; the refactoring to `scopeKey` fixed this by making the pattern explicit
10. **Event streaming** — `Events() []Event` copies the whole slice; a channel-based API would be more memory-efficient for long-running containers

---

## F) TOP 25 NEXT ACTIONS (sorted by impact/effort)

| # | Action | Impact | Effort | Category |
|---|--------|--------|--------|----------|
| 1 | Add `go:generate templ generate` to `html.templ` header | MED | S | DX |
| 2 | Investigate dead `html/template` import in `html.go` | MED | S | Cleanup |
| 3 | Document lock ordering in `Recorder` (4 mutexes) | MED | S | Quality |
| 4 | Add `t.Parallel()` to all safe tests | LOW | S | Testing |
| 5 | Fix example `Connect()`/`Start()` always-nil error returns | LOW | S | Example |
| 6 | Add CI GitHub Action: `go test`, `golangci-lint run` | MED | M | DX |
| 7 | Add `go test -cover` baseline and CI gate | MED | S | Testing |
| 8 | Test multi-scope dependency tracking (child→parent) | MED | M | Testing |
| 9 | Add `ProvideOverride` test | LOW | S | Testing |
| 10 | Extract `DurationMs` custom type to replace `*float64` | MED | M | Type Model |
| 11 | Add DOT/Mermaid graph export | HIGH | M | Features |
| 12 | Add OpenTelemetry bridge | HIGH | H | Features |
| 13 | Add configurable event filtering (by service name, event type) | MED | M | Features |
| 14 | Thread-safe benchmark for concurrent invocations | LOW | M | Testing |
| 15 | Add context.Context to export methods | MED | M | API |
| 16 | Consider `Recorder` interface for testability | MED | M | Architecture |
| 17 | Event streaming channel API (`Events() <-chan Event`) | MED | H | API |
| 18 | Add Prometheus metrics endpoint export | MED | M | Features |
| 19 | Structured logging integration (slog) | LOW | M | Features |
| 20 | Add `sampler` interface for high-volume event sampling | LOW | H | Features |
| 21 | Track benchmark results over time in CI | MED | M | DX |
| 22 | HTML: click-to-highlight on dependency graph nodes | MED | M | UX |
| 23 | HTML: responsive design for mobile | MED | M | UX |
| 24 | Add PProf endpoints to HTML visualization | LOW | S | DX |
| 25 | Resolve `containerID` on ServiceInfo — add or document why only on Event | LOW | S | Type Model |

---

## G) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**Should this library target `samber/do/v2` only, or also support `v1`?**

The module is named `samber-do-auditlog` and depends on `github.com/samber/do/v2`. If `v1` users exist, we'd need either a v2-specific module path or a compatibility shim. The README doesn't address this. This is a product/ownership decision — I can't determine the user base or the maintainer's intent from code alone.

---

## Metrics Snapshot

| Metric | Value |
|--------|-------|
| Total lines (Go) | 1,757 |
| Production lines | 1,044 |
| Test lines | 713 |
| Example lines | 175 |
| Test coverage | 92.9% |
| Clone groups (t=50) | 0 |
| Clone groups (t=15) | 8 (all idioms) |
| Production clones (t=15) | 2 (function signatures) |
| Lint issues | 0 |
| `go vet` issues | 0 |
| Commits ahead of origin | 6 |

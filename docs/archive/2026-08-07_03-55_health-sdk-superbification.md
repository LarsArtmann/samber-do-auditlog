# Status Report: `health/` SDK Superbification

**Date:** 2026-08-07 03:55
**Session scope:** Turn the working-but-incomplete `health/` SDK into a "superb" SDK by fixing the 3 critical gaps (StatusWarn phantom, no examples, no benchmarks) and adding production hardening features.
**Prior session:** See `docs/status/2026-08-07_03-34_health-sub-package-creation.md` for the original creation report.
**Coverage:** 96.4% (up from 96.1%) | **Tests:** 33 + 4 examples + 4 benchmarks (all pass, -race clean, -count=3 clean) | **Lint:** 0 issues | **Lines:** 2,854 total (1,747 source + test, up from 1,368)

---

## a) FULLY DONE

### StatusWarn Phantom — FIXED (the #1 bug from prior session)

The prior session exported `StatusWarn = "warn"` but never produced it. This session:

- Converted `buildChecks` from a package-level function to a `*Probe` method
- Non-critical service failures now produce `StatusWarn` (degraded but functional)
- Critical service failures still produce `StatusFail` (take pod out of rotation)
- Updated all call sites: `Evaluate` in `probe.go`, `buildStartupResponse` in `handlers.go`
- Updated 2 existing tests that asserted `StatusFail` for non-critical → now assert `StatusWarn`
- Added 2 new regression tests: `TestEvaluate_MixedFailures_CriticalFailNonCriticalWarn`, `TestEvaluate_AllNonCriticalFailures_RollupStaysPass`

| File                       | Change                                                                          |
| -------------------------- | ------------------------------------------------------------------------------- |
| `health/probe.go:295`      | `buildChecks` now a method, uses `p.critical[name]` to distinguish fail vs warn |
| `health/probe.go:233`      | `Evaluate` call site: `p.buildChecks(results)`                                  |
| `health/handlers.go:147`   | `buildStartupResponse` call site: `p.buildChecks(results)`                      |
| `health/probe_test.go:289` | `TestReadiness_NonCriticalFailure`: asserts `StatusWarn`                        |
| `health/probe_test.go:513` | `TestEvaluate_ReturnsCorrectClassification`: asserts `StatusWarn`               |

### Runnable Examples — CREATED

`health/example_test.go` (131 lines) with 4 testable Go examples, all with `// Output:` directives:

1. **`ExampleNew`** — creates a Probe, registers a critical service, evaluates health
2. **`ExampleProbe_LivenessHandler`** — shows the dep-free 200 response
3. **`ExampleProbe_ReadinessHandler`** — shows readiness checking critical services
4. **`ExampleProbe_RegisterRoutes`** — shows the one-liner `RegisterRoutes` for `/healthz`, `/readyz`, `/startupz`

All 4 pass `testableexamples` lint and execute during `go test`.

### Performance Benchmarks — CREATED

`health/probe_test.go` now includes 4 benchmarks:

| Benchmark                            | ns/op | B/op | allocs/op | What it proves                      |
| ------------------------------------ | ----- | ---- | --------- | ----------------------------------- |
| `BenchmarkLivenessHandler`           | ~850  | 1316 | 15        | Liveness is sub-microsecond         |
| `BenchmarkReadinessHandler_CacheHit` | ~1009 | 1346 | 15        | Cache delivers ~1µs responses       |
| `BenchmarkReadinessHandler_LiveEval` | ~4716 | 3690 | 49        | Live eval is 4.7x slower than cache |
| `BenchmarkEvaluate`                  | ~3259 | 2312 | 38        | Raw health-check batch cost         |

The cache hit benchmark proves the background-caching architecture delivers on its O(1) promise.

### GET-Only Enforcement — CREATED (new feature, not in prior session)

| Deliverable            | Status | Details                                                                                                             |
| ---------------------- | ------ | ------------------------------------------------------------------------------------------------------------------- |
| `WithGETOnly()` option | DONE   | Wraps all handlers via `Probe.guard()` middleware                                                                   |
| `Probe.guard()` method | DONE   | Returns 405 + `Allow: GET` header on non-GET; passes through when `getOnly` is false (zero overhead)                |
| 3 tests                | DONE   | `TestGETOnly_RejectsNonGET` (POST/PUT/DELETE/HEAD), `TestGETOnly_AllowsGET`, `TestDefault_AllowsNonGETWithoutGuard` |

All three handlers (`LivenessHandler`, `ReadinessHandler`, `StartupHandler`) are wrapped via `p.guard()`. The guard runs at construction time, so there is zero runtime overhead when `WithGETOnly` is not set.

### Documentation — UPDATED

| File            | Changes                                                                                                                       |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `README.md`     | New "Health Probes" section (parallel to "Live Dashboard") with code example, endpoint descriptions, feature list, guide link |
| `FEATURES.md`   | New "Health Probes (`health/` Sub-Package)" table with 13 feature rows + test entry row                                       |
| `CHANGELOG.md`  | Updated `[Unreleased]` health section: added GETOnly bullet, examples bullet, benchmarks bullet, corrected test count         |
| `AGENTS.md`     | Updated file table (added `example_test.go`, updated line counts), added GET-only to design decisions, added `warn` mention   |
| `health/doc.go` | Added "Design Rationale" section with cross-reference to the guide                                                            |

### Verification

| Check                                 | Result                        |
| ------------------------------------- | ----------------------------- |
| `go test -race ./health/... -count=1` | 33 tests + 4 examples PASS    |
| `go test -race ./health/... -count=3` | PASS (no timing flakes)       |
| `go test -race ./... -count=1`        | Full project PASS             |
| `golangci-lint run ./health/...`      | 0 issues                      |
| `golangci-lint fmt ./health/...`      | Clean (no fmt/lint conflicts) |
| `go vet ./health/...`                 | Clean                         |
| Coverage                              | 96.4% of statements           |

---

## b) PARTIALLY DONE

| ~~Item~~                       | ~~What exists~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**                                                                                     | ~~What's missing~~                                                                                                                                                                                                                                                                        |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~GETOnly test coverage~~      | ~~`TestGETOnly_RejectsNonGET` tests POST/PUT/DELETE/HEAD against `LivenessHandler` only~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**           | ~~Does NOT verify `ReadinessHandler` or `StartupHandler` reject non-GET. The `guard()` wrapper is the same for all three, but there's no test proving all three handlers are actually wrapped. A regression where someone adds a 4th handler and forgets `guard()` would not be caught.~~ |
| ~~Benchmark numbers in docs~~  | ~~FEATURES.md and CHANGELOG.md cite specific numbers (3.3µs, 9.6µs)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**                               | ~~These are from the FIRST benchmark run. The second run showed different numbers (~1µs, ~4.7µs). Benchmark results vary by machine and run — hardcoding them in docs is fragile and already stale. Should either remove specific numbers or note they're indicative.~~                   |
| ~~Example compile-time guard~~ | ~~`probe_test.go` has `var _ do.HealthcheckerWithContext = (*T)(nil)` on all 4 test service types~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** | ~~`example_test.go`'s `exampleDB` type does NOT have the compile-time guard. The guide Step 1 says "always add the guard." Examples should model best practices.~~                                                                                                                        |

---

## c) NOT STARTED

| ~~Item~~                              | ~~Why it matters~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**                                                                                                                                                                                |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~**BENCHMARKS.md not updated**~~     | ~~The project maintains `BENCHMARKS.md` with baseline numbers for regression detection. The new health benchmarks are not listed there. CI doesn't enforce this, but it's a project convention.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**Stress test**~~                   | ~~No test for 1000 concurrent requests to cached readiness handler. The `atomic.Pointer[Response]` should handle this, but it's unproven under load.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**                                            |
| ~~**`Probe.Validate()` method**~~     | ~~No validation that timeout > 0, refresh interval >= 0. Invalid values would cause panics or hangs at runtime.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**                                                                                 |
| ~~**Fuzz test for `writeResponse`**~~ | ~~The project has fuzz tests for HTML XSS, NDJSON parsing, etc. `writeResponse` marshals arbitrary `Response` values to JSON — no fuzz test verifies it can't panic on edge-case inputs.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**        |
| ~~**`WithGracePeriod` option**~~      | ~~The guide Step 7 shows `time.Sleep(gracePeriod)` between `MarkShuttingDown` and resource close. The SDK has `MarkShuttingDown` but no built-in sleep mechanism.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**                               |
| ~~**`LivenessChecker` interface**~~   | ~~The guide Step 6 mentions "optionally check a deadlock watchdog." No pluggable hook exists for goroutine-starvation or deadlock detection in the liveness handler.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**                            |
| ~~**slog integration**~~              | ~~No structured logging of slow checks, failures, or state transitions (shutdown marked, startup latched).~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**                                                                                      |
| ~~**Indented JSON option**~~          | ~~No `WithIndentJSON()` for human-readable responses during development. Compact JSON only.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**                                                                                                     |
| ~~**Integration test with `live/`**~~ | ~~No test verifying `health/` and `live/` routes don't conflict when both are mounted on the same mux.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**                                                                                          |
| ~~**Restart test**~~                  | ~~No test for `Shutdown` followed by `Start` (restart scenario). The current `Start` is no-op-safe for double-call, but restart after full shutdown is untested.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**                                |

---

## d) TOTALLY FUCKED UP

Nothing is catastrophically broken, but there are real quality issues:

### 1. Hardcoded benchmark numbers in docs are already stale

I wrote `3.3µs` and `9.6µs` in FEATURES.md and CHANGELOG.md based on the first benchmark run. The second run (same machine, same binary) showed `~1µs` and `~4.7µs`. That's a 3x difference between runs. Hardcoding benchmark numbers in documentation that isn't `BENCHMARKS.md` (which has a clear "re-run with" header) is a maintenance trap. Anyone reading FEATURES.md in 3 months will see wrong numbers and either trust them (bad) or have to re-run to verify (wasted time).

**Fix:** Remove specific ns/op numbers from FEATURES.md and CHANGELOG.md. Replace with qualitative claims ("sub-microsecond", "multiple times faster than live evaluation") or remove entirely. Keep precise numbers only in BENCHMARKS.md where they belong.

### 2. GETOnly test only covers LivenessHandler

`TestGETOnly_RejectsNonGET` tests the guard on `LivenessHandler` only. It assumes `ReadinessHandler` and `StartupHandler` are wrapped identically. They are (I can see it in the code), but a test that doesn't verify all three handlers is a gap. If someone refactors and forgets to wrap one handler, the test won't catch it.

### 3. `exampleDB` in example_test.go lacks compile-time guard

The guide explicitly says in Step 1: "Add a compile-time guard." All 4 test service types in `probe_test.go` have `var _ do.HealthcheckerWithContext = (*T)(nil)`. But `exampleDB` in `example_test.go` does not. Examples should demonstrate best practices, not skip them.

---

## e) WHAT WE SHOULD IMPROVE

### High Priority

1. ~~**Remove hardcoded benchmark numbers from FEATURES.md and CHANGELOG.md** — replace with qualitative claims. Precise numbers belong in BENCHMARKS.md only.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
2. ~~**Add GETOnly test for all three handlers** — iterate over `LivenessHandler`, `ReadinessHandler`, `StartupHandler` and verify each rejects non-GET.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
3. ~~**Add compile-time guard to `exampleDB`** in `example_test.go`.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
4. ~~**Add health benchmarks to BENCHMARKS.md** — the project convention for regression baselines.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Medium Priority

5. ~~**Add `Probe.Validate()` method** — validate timeout > 0, refresh interval >= 0, version string constraints.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
6. ~~**Add fuzz test for `writeResponse`** — verify no panic on edge-case `Response` values (empty maps, nil checks, very long strings).~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
7. ~~**Add stress test** — 1000 concurrent requests to cached readiness handler, verify no race, no panic.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
8. ~~**Add restart test** — `Shutdown` then `Start` again, verify background loop restarts.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
9. ~~**Add `WithGracePeriod(d)` option** — sleep between `MarkShuttingDown` and `Shutdown` for two-phase graceful drain.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
10. ~~**Add `LivenessChecker` interface** — pluggable hook for deadlock/goroutine-starvation detection.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
11. ~~**Add integration test** — `health/` + `live/` on same mux, no route conflicts.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Low Priority

12. ~~**Add `WithIndentJSON()` option** — human-readable JSON for development.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
13. ~~**Add slog integration** — log slow checks (>threshold), state transitions.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
14. ~~**Add Prometheus metrics** — latency histogram, fail counter per service.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
15. ~~**Add per-service latency in `Check` struct** — `LatencyMs int64` field.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
16. ~~**Add stale cache detection** — `WithMaxCacheAge(d)` to re-evaluate if cache is too old.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
17. ~~**Add combined `/health` endpoint option** — for legacy systems that want one URL.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

---

## f) Up to 50 Things We Should Get Done Next

### Correctness Fixes (must do)

1. ~~Remove hardcoded benchmark ns/op numbers from FEATURES.md — use qualitative claims~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
2. ~~Remove hardcoded benchmark ns/op numbers from CHANGELOG.md — use qualitative claims~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
3. ~~Add `var _ do.HealthcheckerWithContext = (*exampleDB)(nil)` to `example_test.go`~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
4. ~~Add GETOnly rejection test for `ReadinessHandler`~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
5. ~~Add GETOnly rejection test for `StartupHandler`~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
6. ~~Consider table-testing all three handlers in one GETOnly test instead of handler-specific~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Benchmarks & Performance (must do)

7. ~~Add health benchmarks to `BENCHMARKS.md` with proper environment header~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
8. ~~Add `BenchmarkStartupHandler_Latched` — should be near-zero (just atomic load + JSON)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
9. ~~Add `BenchmarkStartupHandler_Evaluating` — startup before latch~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
10. ~~Add allocation analysis — 15 allocs/op for liveness is high; can `time.Since().String()` allocation be avoided?~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
11. ~~Consider `sync.Pool` for `Response` structs in hot paths~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
12. ~~Add `-benchmem` assertions or documentation~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Test Hardening

13. ~~Add stress test: 1000 concurrent requests to cached readiness handler~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
14. ~~Add restart test: `Shutdown` then `Start` again~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
15. ~~Add test for `Evaluate` with shutdown flag set~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
16. ~~Add test for concurrent `Evaluate` calls (race detector)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
17. ~~Add test for `readinessResponse` cache miss → live fallback with cancelled context~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
18. ~~Add fuzz test for `writeResponse` — edge-case `Response` values~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
19. ~~Add `-count=10` race test run for all cache/timing tests~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
20. ~~Add test verifying `RegisterRoutes` doesn't panic on duplicate paths~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
21. ~~Add test for `MarkShuttingDown` then `Shutdown` (two-phase)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### API Hardening

22. ~~Add `Probe.Validate()` method — timeout > 0, refresh interval >= 0~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
23. ~~Add `WithGracePeriod(d time.Duration)` option~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
24. ~~Add `LivenessChecker` interface for pluggable deadlock detection~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
25. ~~Add `WithMaxCacheAge(d)` for stale cache detection~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
26. ~~Add `WithIndentJSON()` for development~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
27. ~~Consider `WithCORS` middleware option~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
28. ~~Add `LatencyMs int64` field to `Check` struct for per-service timing~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Observability

29. ~~Add `WithLogger(logger *slog.Logger)` option~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
30. ~~Log slow health checks (> configurable threshold)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
31. ~~Log state transitions (shutdown marked, startup latched)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
32. ~~Add Prometheus metrics endpoint option~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
33. ~~Add per-service latency tracking~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Integration

34. ~~Add integration test: `health/` + `live/` on same mux~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
35. ~~Add integration test: `health/` with scoped injectors~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
36. ~~Add integration test: `health/` with `WithRefreshInterval(0)` under load~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
37. ~~Verify `depguard` rules allow `health/` imports correctly~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Documentation

38. ~~Add `health/` section to `STABILITY.md` if applicable~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
39. ~~Add `health/` to `ROADMAP.md` as completed~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
40. ~~Add `health/` to `TODO_LIST.md` as completed~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
41. ~~Consider standalone `example/health/main.go` demo~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
42. ~~Add `health/` to CLI `info` output if applicable~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### CI

43. ~~Verify `go generate ./...` doesn't need changes for health/~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
44. ~~Verify `go mod tidy` doesn't drift (no new deps)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
45. ~~Verify CI coverage gate passes with health/ included (96.4% > 94% threshold)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
46. ~~Run full `golangci-lint run ./...` and verify no health/ issues (done — 0 issues)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
47. ~~Add health/ to CI benchmark job if one exists~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Polish

48. ~~Consider adding `Probe.Status()` method returning current cached status (for external monitoring)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
49. ~~Consider adding `Probe.LastResponse()` method returning the last evaluated Response~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
50. ~~Consider adding `WithOnStateChange(fn func(old, new Status))` callback for external alerting~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

---

## g) Questions (cannot figure out myself)

> **Resolution (2026-09-11):** superseded — `health/` was extracted to [github.com/larsartmann/go-health](https://github.com/larsartmann/go-health) as its own repository with its own go.mod; `*Plugin` satisfies its `HealthRecorder` interface implicitly.

### 1. Should benchmark numbers be in FEATURES.md/CHANGELOG.md at all?

I hardcoded specific ns/op values (`3.3µs`, `9.6µs`) in FEATURES.md and CHANGELOG.md, but these are already stale (second run showed different numbers). Should I (a) remove all specific numbers and use qualitative claims only, (b) keep numbers but add "indicative, see BENCHMARKS.md" disclaimer, or (c) move all numbers to BENCHMARKS.md only? The project convention in BENCHMARKS.md uses a dated environment header for regression baselines — I'm unsure if FEATURES.md should have any performance claims at all.

### 2. Should the GETOnly guard wrap at construction time or per-request?

Currently `guard()` wraps the handler at construction time (when `LivenessHandler()` is called), so the `if !p.getOnly` check happens once and the returned `http.HandlerFunc` is either the raw handler or the guarded one. This is zero-overhead when disabled. An alternative is checking `r.Method` inside each handler. I chose construction-time because it's faster, but it means the `getOnly` field is read at handler construction, not at request time — if someone sets it after calling `LivenessHandler()`, the guard won't apply. Should this be documented, or should I make it request-time for safety?

### 3. Should non-critical failures affect the roll-up status as `warn`?

Currently, when only non-critical services fail, the overall `Response.Status` stays `pass` (correct — we don't want 503). But should the roll-up be `warn` instead of `pass` to signal degradation? The guide says "still 200 with a warn-ish check" — it's ambiguous whether the roll-up should also be warn. Currently individual checks are `warn` but the roll-up is `pass`. Some monitoring systems look at the roll-up status, not individual checks. Should the roll-up reflect `warn` when any non-critical check is `warn`?

# Status Report: `health/` Sub-Package (Health-Probe SDK)

**Date:** 2026-08-07 03:34\
**Session scope:** Creation of `health/` sub-package from the superb health endpoint guide\
**Coverage:** 96.1% | **Tests:** 29 (all pass, -race clean) | **Lint:** 0 issues | **Lines:** 1,368 total

---

## a) FULLY DONE

### Core SDK — Complete and Working

| Deliverable                            | Status | Details                                                                                                                     |
| -------------------------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------- |
| `health/types.go`                      | DONE   | `Status` enum (pass/fail/warn), `Check`, `Response` with JSON tags                                                          |
| `health/doc.go`                        | DONE   | Package doc with quick start, three-probe rationale, caching/shutdown/audit sections                                        |
| `health/probe.go`                      | DONE   | `Probe` struct, 6 functional options, `New()`, `Start`/`Shutdown`/`MarkShuttingDown`, `Evaluate`, classify, evaluateStartup |
| `health/handlers.go`                   | DONE   | `LivenessHandler`, `ReadinessHandler`, `StartupHandler`, `RegisterRoutes`, `Routes`, `DefaultRoutes`, `writeResponse`       |
| `health/probe_test.go`                 | DONE   | 29 tests: liveness (3), readiness (7), startup (4), Evaluate (4), routes (2), format (2), audit (2), lifecycle (5)          |
| `.golangci.yml` exhaustruct exclusions | DONE   | `health.Response`, `health.Check`, `health.Routes`, `health.Probe` added                                                    |
| `AGENTS.md` updated                    | DONE   | Architecture section + `health/` sub-package file table + key design decisions                                              |
| `CHANGELOG.md` updated                 | DONE   | Full `[Unreleased]` section with all features                                                                               |
| Full project builds                    | DONE   | `GOEXPERIMENT=jsonv2 go build ./...` clean                                                                                  |
| Full project tests pass                | DONE   | `go test -race ./...` — all packages pass                                                                                   |
| Health package lint clean              | DONE   | 0 issues across ~70 linters                                                                                                 |

### Design Decisions Implemented

- Three-probe separation (liveness / readiness / startup)
- Critical vs non-critical service classification via `WithCriticalServices`
- Background caching via `atomic.Pointer[Response]` (1s default refresh)
- Shutdown-aware readiness (503 on shutdown, liveness stays 200)
- Startup latch (one-way, all-critical-pass → permanently 200)
- Nil-safe auditlog integration (`WithPlugin` optional)
- `RegisterRoutes` convenience for standard or custom paths
- Live evaluation fallback when cache is nil (before `Start` called or `RefreshInterval(0)`)

---

## b) PARTIALLY DONE

| ~~Item~~                    | ~~What exists~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** | ~~What's missing~~ |
| ----------------------- | -------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~`StatusWarn` enum value~~ | ~~Defined in `types.go`~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** | ~~**Never used anywhere.** Non-critical failures are marked `StatusFail` in individual checks, not `warn`. The guide (Step 5) says non-critical failures should be "warn-ish". The roll-up stays `pass` (correct), but individual non-critical check entries show `fail` instead of `warn`.~~ |
| ~~Shutdown grace period~~   | ~~`MarkShuttingDown()` exists for two-phase shutdown~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** | ~~No `WithGracePeriod` option. The guide Step 7 shows `time.Sleep(gracePeriod)` between marking and resource close. No built-in sleep/delay mechanism.~~ |
| ~~HTTP method handling~~    | ~~Handlers accept any method~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** | ~~No GET-only enforcement. Kubernetes only uses GET, but POST/PUT/etc. are not rejected.~~ |

---

## c) NOT STARTED

| ~~Item~~                                                                    | ~~Why it matters~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~**Example file (`example/health/main.go` or `health/example_test.go`)**~~ | ~~The project has `example/` and `live/demo/`. The guide is literally a tutorial. A runnable example is the #1 missing deliverable.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**Benchmarks**~~                                                          | ~~Project has `BENCHMARKS.md` and `benchmarks_test.go`. No benchmarks for handler latency, cache hit/miss, Evaluate throughput, or allocation profiling.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**README.md update**~~                                                    | ~~README documents `live/` sub-package (line 274) but says nothing about `health/`.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**FEATURES.md update**~~                                                  | ~~FEATURES.md has a full `live/` section (lines 137-158). No `health/` sub-package section exists.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**Coverage gate verification**~~                                          | ~~Coverage is 96.1% but the CI gate is 94%. Not verified whether the health package is included in the coverage gate script or excluded like `example/` and `cmd/`.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**`WithHealthCheckTimeouts` option**~~                                    | ~~The guide Step 2 shows configuring `InjectorOpts.HealthCheckTimeout`, `HealthCheckGlobalTimeout`, `HealthCheckParallelism`. The SDK doesn't help configure these — user must set them manually on `do.InjectorOpts` before creating the injector.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**Internal-only middleware**~~                                            | ~~Guide Step 9 says "gate detail internally". No `InternalOnly()` middleware or CIDR-based access control provided.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**Deadlock watchdog hook**~~                                              | ~~Guide Step 6 mentions "Optionally: check a deadlock watchdog". No pluggable liveness checker interface.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**Fuzz tests**~~                                                          | ~~Project has fuzz tests (`fuzz_test.go`, `filter_fuzz_test.go`). No fuzz tests for health response serialization or edge cases.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**JSON output format options**~~                                          | ~~No option for indented JSON (human-readable) vs compact JSON. Compact only.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**Combined `/health` endpoint**~~                                         | ~~Some legacy systems want one endpoint. Not provided (the guide argues against it, but users may want the option).~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**slog integration**~~                                                    | ~~No structured logging of slow checks, failures, or state transitions.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**Prometheus metrics**~~                                                  | ~~No health-check metrics exposition (latency histogram, fail counter).~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |
| ~~**Context-aware Start**~~                                                 | ~~`Start(ctx)` takes a context but the test uses `t.Context()` — fine for tests but production users need to understand the lifecycle. No doc example of wiring `Start` to a signal handler.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.** |

---

## d) TOTALLY FUCKED UP

Nothing is catastrophically broken, but there are real quality gaps:

### 1. `StatusWarn` is a phantom type — defined, exported, never used

This is the most embarrassing issue. I defined `StatusWarn = "warn"` in `types.go`, exported it, documented it ("Used for non-critical service failures"), and then **never once use it in any code path**. Non-critical service failures are marked `StatusFail` in `buildChecks()`. This means:

- The type lies about the SDK's capability
- A user importing `health.StatusWarn` gets a value that the SDK itself never produces
- The response JSON for a non-critical failure shows `"status": "fail"` when the guide says it should be `"warn"`

**Fix:** `buildChecks` should accept the critical set and mark non-critical failures as `StatusWarn`. Or remove `StatusWarn` entirely if we decide all individual failures are `fail`.

### 2. No example — this is derived from a GUIDE

The entire `health/` package exists because of `docs/guides/superb-health-endpoint-with-samber-do.md` — a step-by-step tutorial. Shipping a "nice SDK" derived from a guide **without a runnable example** is shipping half the value. The user explicitly asked for "a nice SDK" — an SDK without examples is not nice.

### 3. No benchmarks in a performance-critical package

Health probes are the most frequently-called endpoints in production (kubelet polls every 1-2 seconds, load balancers even more). The whole point of the background cache (guide Step 8) is performance. Shipping this without a single benchmark to prove the cache actually delivers O(1) response time is irresponsible.

---

## e) WHAT WE SHOULD IMPROVE

### High Priority

1. ~~**Wire `StatusWarn` into `buildChecks`** — non-critical failures should be `warn`, not `fail`. This is a 5-line fix in `probe.go` but requires passing the critical set to `buildChecks`.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
2. ~~**Create `health/example_test.go`** — at minimum, a testable Go example (`ExampleNew`, `ExampleProbe_ReadinessHandler`) with `// Output:` directives. Ideally also `example/health/main.go`.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
3. ~~**Add benchmarks** — `BenchmarkReadinessHandler_CacheHit`, `BenchmarkReadinessHandler_LiveEval`, `BenchmarkEvaluate`, `BenchmarkLivenessHandler`, `BenchmarkWriteResponse`.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
4. ~~**Update README.md** — add a `health/` section parallel to the `live/` section.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
5. ~~**Update FEATURES.md** — add a `Health Probe SDK (health/ Sub-Package)` section.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
6. ~~**Verify coverage gate** — check `scripts/coverage-gate.sh` includes or excludes `health/`.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Medium Priority

7. ~~**Add `WithHealthCheckTimeouts`** — convenience option that sets `InjectorOpts.HealthCheckTimeout`, `HealthCheckGlobalTimeout`, `HealthCheckParallelism` on the probe's injector. Currently the user has to figure this out themselves.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
8. ~~**Add GET-only enforcement** — health probes should return 405 on non-GET methods.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
9. ~~**Add `WithGracePeriod`** — sleep between `MarkShuttingDown` and actual background loop stop in `Shutdown`.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
10. ~~**Add `LivenessChecker` interface** — pluggable hook for deadlock/goroutine-starvation detection in the liveness handler.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
11. ~~**Add fuzz test for `writeResponse`** — verify no panic on edge-case `Response` values.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
12. ~~**Add indented JSON option** — `WithIndentJSON()` for human-readable responses in development.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Low Priority

13. ~~**Add slog integration** — log slow checks (>threshold), state transitions (shutdown marked, startup latched).~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
14. ~~**Add Prometheus metrics** — latency histogram, fail counter per service.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
15. ~~**Add combined `/health` endpoint option** — for legacy systems that want one URL.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
16. ~~**Add `Probe.Validate()`** — check timeout > 0, refresh interval >= 0.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
17. ~~**Add integration test with `live/`** — verify no route conflicts when both are mounted.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
18. ~~**Add stale cache detection** — if cache is older than N seconds, return warn status or re-evaluate synchronously.~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

---

## f) Up to 50 Things We Should Get Done Next

### Correctness (must do)

1. ~~Wire `StatusWarn` into `buildChecks` for non-critical failures~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
2. ~~Add test verifying non-critical failure produces `warn` status~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
3. ~~Add test verifying `StatusWarn` is actually produced by the SDK~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
4. ~~Consider whether `Evaluate` should also use `warn` for non-critical roll-up~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Examples & Docs (must do)

5. ~~Create `health/example_test.go` with `ExampleNew` and `// Output:` directive~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
6. ~~Create `health/example_test.go` with `ExampleProbe_LivenessHandler`~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
7. ~~Create `health/example_test.go` with `ExampleProbe_ReadinessHandler`~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
8. ~~Consider `example/health/main.go` — standalone runnable demo~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
9. ~~Update `README.md` with `health/` section (parallel to `live/` section)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
10. ~~Update `FEATURES.md` with health probe SDK feature table~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
11. ~~Update `STABILITY.md` if it tracks sub-package stability~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
12. ~~Add `health/` to `ROADMAP.md` if applicable~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
13. ~~Add `health/` to `TODO_LIST.md` as completed item~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
14. ~~Add guide cross-reference in `health/doc.go` (link to the guide file)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Performance (must do)

15. ~~Add `BenchmarkLivenessHandler` — should be <1us / 0 allocs~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
16. ~~Add `BenchmarkReadinessHandler_CacheHit` — should be <1us / 0 allocs~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
17. ~~Add `BenchmarkReadinessHandler_LiveEval` — measure raw injector check cost~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
18. ~~Add `BenchmarkEvaluate` — full health-check batch~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
19. ~~Add `BenchmarkWriteResponse` — JSON serialization~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
20. ~~Add allocation profiling (`-benchmem`) assertions~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
21. ~~Add `health/` to `BENCHMARKS.md`~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Test Coverage

22. ~~Verify coverage gate script (`scripts/coverage-gate.sh`) includes `health/`~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
23. ~~Add test for `writeResponse` marshal error path (even if hard to trigger)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
24. ~~Add test for concurrent `Start` calls (race detector)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
25. ~~Add test for `Shutdown` followed by `Start` (restart scenario)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
26. ~~Add test for `Evaluate` with shutdown flag set (should return `StatusFail`)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
27. ~~Add test for `readinessResponse` cache miss → live fallback with context cancellation~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
28. ~~Add test for `StartupHandler` after `Shutdown` (should return latched value)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
29. ~~Add stress test: 1000 concurrent requests to cached readiness handler~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
30. ~~Add `-count=10` race test run for timing-dependent cache tests~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### API Hardening

31. ~~Add GET-only method enforcement (405 on POST/PUT/DELETE)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
32. ~~Add `WithHealthCheckTimeouts(perService, global, parallelism)` option~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
33. ~~Add `WithGracePeriod(d)` for shutdown sleep~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
34. ~~Add `LivenessChecker` interface for pluggable deadlock detection~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
35. ~~Add `Probe.Validate()` method~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
36. ~~Add stale cache detection (`WithMaxCacheAge`)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
37. ~~Consider `WithIndentJSON` for development~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
38. ~~Consider `WithCORS` middleware option~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Observability

39. ~~Add `WithLogger(logger)` for slog integration~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
40. ~~Log slow health checks (> threshold)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
41. ~~Log state transitions (shutdown, startup latch)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
42. ~~Add Prometheus metrics endpoint option~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
43. ~~Add per-service latency tracking in `Check` struct (`LatencyMs int64`)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### Integration

44. ~~Add integration test: `health/` + `live/` on same mux~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
45. ~~Add integration test: `health/` with disabled auditlog plugin~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
46. ~~Add integration test: `health/` with scoped injectors (child scope health checks)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
47. ~~Verify `depguard` rules work for external consumers importing `health/`~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

### CI

48. ~~Verify `go generate ./...` doesn't need changes~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
49. ~~Verify `go mod tidy` doesn't drift (no new deps added)~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**
50. ~~Verify CI coverage gate passes with `health/` included~~ **Won't implement — superseded — health/ extracted to github.com/larsartmann/go-health.**

---

## g) Questions (cannot figure out myself)

### 1. Should `health/` be a sub-module (`go.mod`) or stay as a sub-package?

The `live/` sub-package is part of the main module. But `health/` only depends on `samber/do` and the parent `auditlog` package — it could be a separate module with fewer dependencies (no go-output, no templ, no go-sse). This matters because users who only want health probes shouldn't have to pull in the entire auditlog dependency tree. Should we split it, or keep it simple as a sub-package?

### 2. Should non-critical failures be `warn` or `fail` in individual checks?

The guide says non-critical failures should be "warn-ish" (Step 5: "still 200 with a `warn`-ish check"). But `StatusWarn` is defined and never used. Should I wire it in (non-critical failures → `warn` in individual checks, roll-up stays `pass`), or remove `StatusWarn` entirely and keep all failures as `fail`? This changes the JSON contract.

### 3. Should the health SDK be promoted to its own repo (`go-health-probes`) or stay in `samber-do-auditlog`?

The SDK is tightly coupled to `samber/do` (it holds a `do.Injector` reference) but only optionally coupled to `auditlog` (the plugin is nil-safe). It could live in its own repo with just `samber/do` as a dependency, making it usable by anyone — not just auditlog users. Or it stays here as the "official health SDK companion to the auditlog plugin." Which direction?

---

## Session Metrics

| Metric              | Value                                                                |
| ------------------- | -------------------------------------------------------------------- |
| Files created       | 5 (`doc.go`, `types.go`, `probe.go`, `handlers.go`, `probe_test.go`) |
| Files modified      | 3 (`.golangci.yml`, `AGENTS.md`, `CHANGELOG.md`)                     |
| Total lines written | 1,368 (596 source + 769 test + 3 doc/config)                         |
| Test functions      | 29                                                                   |
| Test coverage       | 96.1%                                                                |
| Lint issues         | 0                                                                    |
| Race detector       | Clean                                                                |
| Build               | Clean (`GOEXPERIMENT=jsonv2 go build ./...`)                         |
| Full suite          | All packages pass                                                    |
| Time to build       | <1s                                                                  |
| Benchmark files     | 0                                                                    |
| Example files       | 0                                                                    |
| Fuzz tests          | 0                                                                    |

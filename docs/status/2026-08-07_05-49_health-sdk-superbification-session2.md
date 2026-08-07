# Status Report: `health/` SDK Superbification — Session 2

**Date:** 2026-08-07 05:49
**Session scope:** Resolve 3 issues from prior session's self-assessment, add production hardening features, fix correctness bugs in status classification.
**Prior sessions:** `docs/status/2026-08-07_03-34_health-sub-package-creation.md` (creation), `docs/status/2026-08-07_03-55_health-sdk-superbification.md` (session 1 superbification)
**Coverage:** 98.0% (up from 96.4%) | **Tests:** 41 + 4 examples + 4 benchmarks (up from 33+4+4) | **Lint:** 0 issues | **Lines:** 1,977 total (390 probe.go + 1,157 probe_test.go + 174 handlers.go + 133 example_test.go + 80 doc.go + 43 types.go)

---

## a) FULLY DONE

### Three-State Roll-Up Status — CORRECTNESS FIX (the #1 improvement this session)

**The bug:** The prior session exported `StatusWarn` and used it for individual non-critical check entries, but the roll-up `Response.Status` stayed `StatusPass` when only non-critical services failed. This was semantically wrong — a degraded system reporting "all good" in the roll-up status that most monitoring tools actually read.

**The fix:** `classify()` rewritten from two-state (pass/fail) to three-state (pass/warn/fail):

| Condition | Old roll-up | New roll-up | HTTP status |
|---|---|---|---|
| All services healthy | `pass` | `pass` | 200 |
| Only non-critical failures | `pass` ← **wrong** | `warn` ← **correct** | 200 |
| Any critical failure | `fail` | `fail` | 503 |
| Shutting down | `fail` | `fail` | 503 |

`ReadinessHandler` HTTP logic changed from `if resp.Status != StatusPass` to `if resp.Status == StatusFail` — only `fail` triggers 503, both `warn` and `pass` return 200.

**Files changed:**
- `health/probe.go:280-307` — `classify()` rewritten with three-state logic + `hasWarning` flag
- `health/handlers.go:70` — `if resp.Status == StatusFail` (was `!= StatusPass`)
- `health/types.go:27-30` — `Status` doc comment updated to mention warn roll-up
- `health/probe_test.go` — 3 existing tests updated to assert `StatusWarn` roll-up

### `Probe.Validate()` Method — NEW API

Validates configuration before startup. Returns sentinel errors for the two misconfigurations that cause silent runtime failures:

| Check | Error sentinel | What it prevents |
|---|---|---|
| `timeout <= 0` | `ErrInvalidTimeout` | Every health check fails with "context deadline exceeded" |
| `refreshInterval < 0` | `ErrInvalidRefreshInterval` | Negative interval is treated as 0 (live mode) but caller likely intended positive |

Not enforced in `New()` (no API change, backward compatible). Callers call `Validate()` explicitly for early detection.

### GETOnly Tests Expanded to All Three Handlers

`TestGETOnly_RejectsNonGET` now iterates over all three handlers (`LivenessHandler`, `ReadinessHandler`, `StartupHandler`) × 4 non-GET methods (POST, PUT, DELETE, HEAD) = **12 combinations** (was 4). `TestGETOnly_AllowsGET` similarly expanded to all three handlers. This closes the regression gap where someone could add a 4th handler and forget to wrap it.

### Concurrency Hardening

| Test | What it proves |
|---|---|
| `TestReadiness_ConcurrentAccess_AllSucceed` | 1000 goroutines hit cached readiness handler simultaneously — all 200, no race |
| `TestEvaluate_ConcurrentAccess_NoRace` | 100 goroutines call `Evaluate()` concurrently — `-race` clean |
| `TestShutdown_Idempotent` | Double `Shutdown()` doesn't panic or hang; readiness still returns 503 |

### Compile-Time Guard on `exampleDB`

Added `var _ do.HealthcheckerWithContext = (*exampleDB)(nil)` to `health/example_test.go`. The guide Step 1 says "always add the guard." All 4 test services in `probe_test.go` already had it; `exampleDB` now does too.

### Stale Benchmark Numbers Removed from Docs

- `FEATURES.md:176` — replaced `3.3µs`/`9.6µs` with qualitative claim + BENCHMARKS.md link
- `CHANGELOG.md:27` — replaced specific numbers with "see BENCHMARKS.md"

### Health Benchmarks Added to BENCHMARKS.md

New "Health Package Benchmarks" section with median-of-3 runs, captured 2026-08-07:

| Benchmark | Time/op | Bytes/op | Allocs/op |
|---|---|---|---|
| `BenchmarkLivenessHandler` | 995 ns | 1,316 B | 15 |
| `BenchmarkReadinessHandler_CacheHit` | 1,230 ns | 1,346 B | 15 |
| `BenchmarkReadinessHandler_LiveEval` | 5,710 ns | 3,691 B | 49 |
| `BenchmarkEvaluate` | 3,898 ns | 2,312 B | 38 |

Cache delivers ~4.6× speedup over live evaluation.

### Documentation Updates

- `FEATURES.md` — Added StatusWarn roll-up row, Config validation row, Concurrency tested row; removed stale numbers
- `CHANGELOG.md` — Updated non-critical description (warn roll-up), added Validate/concurrency bullets, test count corrected to 41
- `AGENTS.md` — Updated file table (probe.go now mentions Validate + classify three-state), probe_test.go test count updated, design decisions updated (warn roll-up, three-state classify, config validation)
- `BENCHMARKS.md` — New health section + key observation

### Verification

| Check | Result |
|---|---|
| `go test -race ./health/... -count=1` | 41 tests + 4 examples PASS |
| `go test -race ./health/... -count=3` | PASS (no timing flakes) |
| `go test -race ./... -count=1` | Full project PASS |
| `golangci-lint run ./health/...` | 0 issues |
| `golangci-lint fmt ./health/...` | Clean (no fmt/lint conflicts) |
| `go vet ./health/...` | Clean |
| Coverage (health/) | 98.0% of statements |

---

## b) PARTIALLY DONE

### Doc Comments Not Fully Updated for Three-State Behavior

I changed `classify()` from two-state to three-state, but **5 doc comments across 3 files still describe the old behavior**. I discovered this during the self-review at the end of this session but did NOT fix them — the user asked for a status report, not more changes.

| File | Line | Current text (stale) | What it should say |
|---|---|---|---|
| `health/probe.go` | 32 | "non-critical failures are surfaced as individual check entries but do not affect the HTTP status code" | Should mention roll-up `warn` |
| `health/probe.go` | 68 | "their failures appear in the response body but do not change the HTTP status code" | Should mention roll-up `warn` |
| `health/handlers.go` | 51-52 | "200 when all critical services pass (non-critical failures appear as individual check entries but do not change the status code)" | Should mention three-state: 200 for pass+warn, 503 for fail |
| `README.md` | 340 | "Non-critical failures surface as `warn` in the response body without triggering 503" | Should mention roll-up status is also `warn` |
| `health/doc.go` | — | No mention of `Validate()` or three-state behavior | Should document both |

**Why this matters:** These are the user-facing descriptions that consumers read first. They now under-describe the actual behavior. The code is correct; the docs are stale.

### Coverage Gate Is Failing (Pre-Existing, Not Caused by This Session)

`scripts/coverage-gate.sh` fails: **83.2% total vs 94% threshold**. Root cause: `live/` package at 69.6% coverage drags the total down. Health package is at 98.0%, well above threshold. This is a pre-existing issue — I did not touch any `live/` files. But I should have checked the gate earlier instead of discovering it during the self-review.

---

## c) NOT STARTED

| Item | Why it matters |
|---|---|
| **Fix 5 stale doc comments** | User-facing descriptions now under-describe the three-state behavior. See section b. |
| **Fuzz test for `writeResponse`** | `writeResponse` marshals arbitrary `Response` values to JSON. No fuzz test verifies it can't panic on edge-case inputs (nil maps, very long strings, unicode). |
| **`WithGracePeriod(d)` option** | Guide Step 7 shows `time.Sleep(gracePeriod)` between `MarkShuttingDown` and resource close. SDK has `MarkShuttingDown` but no built-in sleep mechanism. |
| **`LivenessChecker` interface** | Guide Step 6 mentions "optionally check a deadlock watchdog." No pluggable hook for goroutine-starvation detection. |
| **slog integration** | No structured logging of slow checks, failures, or state transitions. |
| **`WithIndentJSON()` option** | No human-readable JSON for development. Compact JSON only. |
| **Integration test with `live/`** | No test verifying `health/` and `live/` routes don't conflict on same mux. |
| **Restart test** | No test for `Shutdown` then `Start` (restart scenario). `Start` is no-op-safe for double-call, but restart after full shutdown is untested. |
| **Per-service latency in `Check` struct** | No `LatencyMs int64` field for per-service timing visibility. |
| **`Probe.Status()` method** | No way to get current cached status without serving an HTTP request (for external monitoring). |
| **`WithOnStateChange` callback** | No callback for external alerting when status transitions (pass→warn→fail). |

---

## d) TOTALLY FUCKED UP

### 1. I shipped a semantic change without updating doc comments

I rewrote `classify()` from two-state to three-state. This is the most impactful behavioral change in the package. I updated tests, I updated `types.go`, I updated FEATURES.md and CHANGELOG.md — but I **did not update the doc comments on `Probe`, `WithCriticalServices`, and `ReadinessHandler` that directly describe the behavior I changed.** I also didn't update `README.md:340`.

I discovered these stale comments **during this self-review**, not during my verification step. My verification was: tests pass, lint passes, coverage high. I never grep'd for stale descriptions of the old behavior. That's a process failure — after changing semantics, the first thing to do is search for every place that describes the old semantics.

**Severity:** Medium. The code is correct. The docs are misleading. A new user reading `ReadinessHandler`'s doc comment would think non-critical failures don't affect the roll-up status at all, when they now set it to `warn`.

### 2. I didn't check the coverage gate until the self-review

I reported "full project passes" based on `go test -race ./...` succeeding. But `scripts/coverage-gate.sh` fails at 83.2%. The test suite passing and the coverage gate passing are different things. I should have run the gate as part of verification, not as an afterthought during self-critique. (The failure is pre-existing in `live/`, not my fault — but not knowing the gate state is my fault.)

### 3. `Validate()` exists but nothing calls it

I added `Probe.Validate()` and 5 tests for it, but I never wired it into `New()` or `Start()`. It's purely opt-in. A user who constructs a Probe with `WithTimeout(0)` will still get silent runtime failures unless they remember to call `Validate()` manually. The method exists but provides no protection unless the caller knows about it. I documented this as "opt-in, backward compatible" — but that's rationalization for not making a decision about whether to enforce it.

---

## e) WHAT WE SHOULD IMPROVE

### High Priority (must do before next release)

1. **Fix the 5 stale doc comments** — search and update every description of non-critical failure behavior to reflect the three-state roll-up
2. **Decide on `Validate()` enforcement** — either call it in `New()` (breaking, returns error), call it in `Start()` (panic on misconfig), or document clearly that it's the caller's responsibility and add a `NewMustValidate` or similar
3. **Add `Validate()` to `health/doc.go`** — the package doc is the entry point for godoc/pkg.go.dev users; it must mention the validation method
4. **Add `Validate()` to README** — the health section lists functional options but not Validate
5. **Fix `live/` coverage** (out of scope for health/ but blocks the gate) — or exclude `live/` from the gate if it's intentionally demo-level code

### Medium Priority

6. **Fuzz test for `writeResponse`** — the project has fuzz tests elsewhere; this is a gap
7. **`WithGracePeriod(d)` option** — two-phase shutdown helper from the guide
8. **Restart test** — `Shutdown` then `Start` lifecycle
9. **`Probe.Status()` method** — expose current cached status for external monitoring
10. **Per-service latency tracking** — `LatencyMs int64` in `Check` struct
11. **`WithOnStateChange` callback** — for external alerting on status transitions
12. **Integration test with `live/`** — verify no route conflicts on shared mux

### Low Priority

13. **`WithIndentJSON()` option** — development-mode pretty-printing
14. **slog integration** — structured logging of slow checks and state transitions
15. **`LivenessChecker` interface** — pluggable deadlock detection
16. **Prometheus metrics** — latency histogram, fail counter per service
17. **Stale cache detection** — `WithMaxCacheAge(d)` to re-evaluate if cache is too old
18. **Combined `/health` endpoint** — for legacy systems wanting one URL

---

## f) Up to 50 Things We Should Get Done Next

### Documentation Fixes (must do)
1. Update `Probe` struct doc comment (probe.go:32) to mention three-state warn roll-up
2. Update `WithCriticalServices` doc comment (probe.go:68) to mention warn roll-up
3. Update `ReadinessHandler` doc comment (handlers.go:51-52) to describe three-state HTTP mapping
4. Update `README.md:340` readiness description to mention warn roll-up status
5. Add `Validate()` documentation to `health/doc.go`
6. Add `Validate()` to README health features list
7. Add three-state behavior summary to `health/doc.go` package comment
8. Consider adding `Validate()` call example to `health/example_test.go`

### Validate() Design Decision (must decide)
9. Decide: enforce `Validate()` in `New()` (returns error), in `Start()` (panic), or document as caller responsibility
10. If caller responsibility: add a lint rule or startup checklist
11. Consider `NewOrPanic(injector, opts...)` variant that calls Validate internally
12. Add `Validate()` call to the guide (`docs/guides/superb-health-endpoint-with-samber-do.md`)

### Test Hardening
13. Fix 5 stale doc comments (items 1-4 above)
14. Add fuzz test for `writeResponse` — edge-case Response values
15. Add restart test: `Shutdown` then `Start` again
16. Add test for `Evaluate` with shutdown flag set (classify returns fail)
17. Add test verifying `RegisterRoutes` doesn't panic on duplicate paths
18. Add test for `MarkShuttingDown` then `Shutdown` (two-phase) with grace period
19. Add `-count=10` race test run for all concurrency tests
20. Add test for cached response staleness (cache populated before shutdown, served after)
21. Add test for `readinessResponse` cache miss → live fallback with cancelled context
22. Add test verifying readiness returns `warn` status field (not just HTTP 200) for non-critical failures
23. Add test verifying startup handler ignores non-critical failures entirely

### API Hardening
24. Add `WithGracePeriod(d time.Duration)` option
25. Add `Probe.Status() Status` method returning current cached status
26. Add `Probe.LastResponse() Response` method returning last evaluated response
27. Add `LivenessChecker` interface for pluggable deadlock detection
28. Add `WithMaxCacheAge(d)` for stale cache detection
29. Add `WithIndentJSON()` for development
30. Add `LatencyMs int64` field to `Check` struct
31. Add `WithOnStateChange(fn func(old, new Status))` callback
32. Add `WithLogger(logger *slog.Logger)` option
33. Consider `WithCORS` middleware option

### Coverage Gate Fix
34. Investigate `live/` coverage (69.6%) — is this demo code that should be excluded?
35. If `live/` is production: add tests to raise coverage
36. If `live/` is demo: exclude from coverage gate
37. Re-run coverage gate after fix to verify 94% threshold passes

### Observability
38. Log slow health checks (> configurable threshold) via slog
39. Log state transitions (shutdown marked, startup latched)
40. Add Prometheus metrics endpoint option
41. Add per-service latency tracking in `Check` struct

### Integration
42. Add integration test: `health/` + `live/` on same mux
43. Add integration test: `health/` with scoped injectors
44. Add integration test: `health/` with `WithRefreshInterval(0)` under load
45. Add standalone `example/health/main.go` demo

### CI
46. Verify `go mod tidy` doesn't drift (no new deps from Validate)
47. Add health/ to CI benchmark job if one exists
48. Run full `golangci-lint run ./...` and verify 0 issues in health/ (done — 0 issues)
49. Verify `go generate ./...` doesn't need changes for health/
50. Add `health/` to `ROADMAP.md` and `TODO_LIST.md` as completed

---

## g) Questions

### 1. Should `Validate()` be enforced or opt-in?

I added `Probe.Validate()` as an opt-in method — callers must remember to call it. If someone constructs `health.New(injector, health.WithTimeout(0))`, they get silent runtime failures (every health check times out immediately). Three options:

**(a) Return error from `New()`** — `func New(injector, opts...) (*Probe, error)`. Breaking API change but safest. No one hits a misconfigured probe.

**(b) Panic in `Start()`** — `New()` stays as-is, but `Start()` calls `Validate()` and panics on invalid config. Non-breaking for users who don't call `Start()` (live mode). Forces early failure for cached mode.

**(c) Keep opt-in** — document clearly, add to quick-start guide. Caller's responsibility. Matches current implementation.

I lean toward (b) because it catches the mistake at the earliest runtime point without breaking the `New()` signature.

### 2. Should `live/` coverage failure block the health/ SDK release?

The coverage gate (`scripts/coverage-gate.sh`) fails at 83.2% because `live/` is at 69.6%. This is pre-existing and unrelated to health/. Should I (a) fix `live/` coverage first, (b) exclude `live/` from the gate (if it's demo code), or (c) tag health/ as stable regardless of the gate? The health/ package itself is at 98.0%.

### 3. Should the startup handler also produce `warn` for non-critical failures?

Currently `buildStartupResponse` only produces `pass` or `fail` — it doesn't call `classify()`. The startup probe is binary: "are we done booting?" Non-critical failures don't affect the latch. But some users might expect the startup response body to show `warn` for non-critical checks (it does show them in the `Checks` map via `buildChecks`, but the roll-up is `fail` until all critical services pass, then `pass`). Should the startup roll-up also support `warn`, or is binary correct for the startup semantic?

# Benchmarks

Baseline benchmark results for `samber-do-auditlog`, captured post-v0.0.3.

These serve as a regression detection baseline. Re-run with:

> **Requires `GOEXPERIMENT=jsonv2`** — set in the Nix devShell automatically, or `export GOEXPERIMENT=jsonv2` manually. See [CONTRIBUTING.md](CONTRIBUTING.md).

```bash
GOEXPERIMENT=jsonv2 go test -bench=. -benchmem -count=3 -run=^$ ./...
```

Compare against this file with `benchstat`:

```bash
go test -bench=. -benchmem -count=5 -run=^$ ./... > /tmp/new.txt
benchstat /tmp/new.txt  # compare manually against the table below
```

---

## Environment

| Property           | Value                                             |
| ------------------ | ------------------------------------------------- |
| Date               | 2026-06-21 (re-baselined post-go-output adoption) |
| Go                 | 1.26.5                                            |
| OS                 | Linux (NixOS)                                     |
| CPU                | AMD Ryzen AI MAX+ 395 (32 threads)                |
| Runs per benchmark | 3                                                 |

---

## Results

Median of 3 runs. Lower is better.

| Benchmark                            | Time/op    | Bytes/op  | Allocs/op | Notes                                                                    |
| ------------------------------------ | ---------- | --------- | --------- | ------------------------------------------------------------------------ |
| `BenchmarkHookOverhead_Invocation`   | 1,658 ns   | 2,019 B   | 6         | Hot path: single service invoke (before+after hooks)                     |
| `BenchmarkHookOverhead_Disabled`     | 113 ns     | 96 B      | 4         | Zero-cost disabled path (empty hooks, samber/do overhead only)           |
| `BenchmarkHookOverhead_Registration` | 21,982 ns  | 167,671 B | 54        | Full registration lifecycle (scope, stack, event, service record)        |
| `BenchmarkHookOnAfterInvocation`     | 633 ns     | 1,897 B   | 6         | After-invocation hook only                                               |
| `BenchmarkHookRegistrationOnly`      | 31,881 ns  | 167,924 B | 58        | Registration hook (slightly heavier than full registration due to setup) |
| `BenchmarkConcurrentInvocation`      | 1,014 ns   | 2,002 B   | 6         | Invocation under concurrent access                                       |
| `BenchmarkBuildReport/services=50`   | 110,653 ns | 80,855 B  | 53        | BuildReport with 50 services                                             |
| `BenchmarkBuildReport/services=100`  | 127,492 ns | 163,183 B | 60        | BuildReport with 100 services                                            |
| `BenchmarkBuildReport/services=500`  | 540,531 ns | 851,222 B | 80        | BuildReport with 500 services                                            |
| `BenchmarkEnrichCapabilities`        | 48,091 ns  | 80,854 B  | 53        | `do.ExplainInjector` capability detection (outside mutex)                |
| `BenchmarkEventsCopy`                | 21,985 ns  | 32,768 B  | 1         | Defensive copy of all events                                             |
| `BenchmarkOnEventCallback`           | 1,686 ns   | 1,853 B   | 6         | OnEvent callback overhead per event                                      |
| `BenchmarkHealthCheck`               | 41,484 ns  | 16,265 B  | 147       | Full health check cycle (bulk HealthCheckWithContext)                    |
| `BenchmarkWriteD2`                   | 48,969 ns  | 96,085 B  | 1,176     | D2 diagram export (build + render + write, 50 services)                  |

---

## Health Package Benchmarks

Captured 2026-08-07. Same environment as above. Median of 3 runs.

| Benchmark                            | Time/op  | Bytes/op | Allocs/op | Notes                                                        |
| ------------------------------------ | -------- | -------- | --------- | ------------------------------------------------------------ |
| `BenchmarkLivenessHandler`           | 995 ns   | 1,316 B  | 15        | Liveness handler: zero dependency checks, always 200         |
| `BenchmarkReadinessHandler_CacheHit` | 1,230 ns | 1,346 B  | 15        | Readiness served from atomic cache (background refresh)     |
| `BenchmarkReadinessHandler_LiveEval` | 5,710 ns | 3,691 B  | 49        | Readiness with live `HealthCheckWithContext` (no cache)      |
| `BenchmarkEvaluate`                  | 3,898 ns | 2,312 B  | 38        | Raw `Evaluate()` call: health-check batch + classify        |

The cache delivers ~4.6× faster responses vs live evaluation (1,230 ns cached vs 5,710 ns live).

---

## Key Observations

- **Disabled path is truly zero-cost**: 113 ns / 4 allocs — entirely samber/do's own overhead. The plugin adds nothing when `Enabled: false`.
- **Invocation hot path is lean**: ~1.6 us / 6 allocs for a full before+after invocation hook pair.
- **BuildReport scales linearly**: 50→500 services is ~5x time, confirming O(n) complexity.
- **EventsCopy is a single allocation**: the `append([]Event(nil), r.events...)` pattern allocates exactly once for the backing array.
- **HealthCheck has high alloc count (147)**: the bulk `HealthCheckWithContext` API allocates per-service internally; this is samber/do's cost, not the plugin's.
- **Diagram export via go-output**: `BenchmarkWriteD2` (added 2026-06-21) covers the full build→render→write path for the D2 format after the go-output adoption. The pre-existing diagram bench was not re-baselined because go-output replaced the entire rendering pipeline; the numbers above for hook/report paths were re-confirmed stable post-adoption.
- **Health cache delivers ~4.6× speedup**: the `atomic.Pointer[Response]` cache keeps readiness at ~1.2 us (15 allocs), while live evaluation costs ~5.7 us (49 allocs). Liveness is sub-microsecond because it performs zero dependency checks.

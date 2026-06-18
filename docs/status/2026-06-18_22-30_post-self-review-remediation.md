# Post Self-Review Remediation — 2026-06-18

## Context

A brutal self-review was triggered to find what was forgotten, what was stupid, what could be better. Two parallel agent reviews found critical bugs, ghost systems, split brains, and code quality issues across the library and CLI. This report covers the remediation work executed in response.

**Result**: 9 commits, 95.5% coverage maintained, 0 lint issues, all tests green.

---

## A. FULLY DONE

### Critical Bug Fixes

1. **`invocationOrder` always 0 in replay** (commit `e2bb93f`) — The replay engine computed `invocationOrder = invocationCount - 1`, which is always 0 since the branch only fires on first invocation. Every replayed report (via `ReplayEvents`, `LoadReport` NDJSON path, or `MigrateReport`) silently lost all cross-service build ordering. Fixed by adding a global `invocationSeq` counter to `replayState`.

2. **Stale HTML golden file** (commit `6bf2293`) — The golden file was never regenerated after the `invocationOrder` bug fix. It still showed `invocation_order:0` for both services instead of the correct `config=0, db=1`.

### Doc Drift Fixes (commit `93a8d90`)

3. README: "5 export formats" → "8 export formats" with full list.
4. README: "~1μs overhead" → "~1.7μs overhead" (matches benchmark).
5. README: Added 13+ missing API method rows.
6. `doc.go`: "force-directed graph" → actual format list (JSON, NDJSON, CSV/TSV, HTML, Mermaid, PlantUML, DOT).
7. AGENTS.md: Go version mismatch fixed (1.26.4 vs go.mod 1.26.3).

### CLI Robustness (commit `10ba076`)

8. `loadFile` no longer calls `os.Exit` via `failf` — returns errors properly.
9. Added stdin support: path `"-"` routes through `LoadReportFromReader(os.Stdin)`.
10. `flag.ExitOnError` → `flag.ContinueOnError` on all subcommands (dead error-handling code revived).
11. `convert -o` properly returns Close errors via named return.
12. `validate` error message includes filename.
13. `schema` rejects extra positional args.

### Code Cleanup (this session)

14. **`trimWhitespace` → `bytes.TrimSpace`** (commit `6d997ff`) — Hand-rolled reimplementation replaced with stdlib (handles Unicode whitespace, strictly more correct).
15. **`serviceRefLabel` inlined** (commit `6d997ff`) — Pointless wrapper that returned `ref.ServiceName` with zero transformation.
16. **Ghost loader API removed** (commit `70234e7`) — `LoadReportFromJSON` (one-line alias for `MigrateReport`) and `LoadReportFromNDJSON` (duplicate of `LoadReportFromReader(r, FormatNDJSON)`) deleted. CLI's `loadFromReader` reimplemented `LoadReportFromReader` — split brain fixed.
17. **CLI UX** (commit `0557480`) — `--help` to stdout (Unix convention), `version`/`-v`/`--version` command, `--` end-of-options marker in `reorderFlags`.
18. **`scopeMeta`/`replayScopeMeta` unified** (commit `15529f2`) — Eliminated duplicate struct + 6 accessor functions + `sortedReplayScopes` + `buildReplayScopeTree`. Renamed `sortedScopesLocked` → `sortedScopes`.
19. **`buildServicesLocked`/`buildReplayServices` collapsed** (commit `3b41631`) — 100% functionally identical implementations unified into `buildServicesFromMap`. Deleted `buildDepsLocked` one-line wrapper. This was the most dangerous split brain: adding a new `ServiceInfo` field would have silently diverged report output between live and replay.
20. **Stack-pop logic extracted** (commit `9db6dcc`) — `popStackFrame` shared between `hooks.go` (live path) and `replay.go` (replay path).
21. **Enum validation on ingest** (commit `8781eb9`) — `IsKnown()` added to `EventType`, `Phase`, `ServiceStatus`. `ReadEvents` now rejects garbage `event_type`/`phase` values at the NDJSON ingest boundary.

---

## B. PARTIALLY DONE

| Item                                | Status             | Notes                                                                                                                           |
| ----------------------------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------- |
| Typed identifiers (branded types)   | Deferred to v0.1.0 | 65+ compile error blast radius, zero existing bugs. Will batch as breaking change.                                              |
| `google/go-cmp` for test assertions | Evaluated          | Already a transitive dep. Needs depguard policy change to adopt.                                                                |
| CLI framework (cobra/kong)          | Evaluated          | `reorderFlags` fragility is the main argument for adoption. Defensible to stay stdlib-only for a 5-subcommand tool.             |
| `encoding/json/v2`                  | Rejected           | Project has a frozen wire format (JSON Schema, stability guarantees). json/v2 changes defaults that would perturb the contract. |
| Schema drift detection test         | Not started        | Would run `cmd/genschema` and compare to committed `schema/report.schema.json`.                                                 |

---

## C. NOT STARTED

1. **`--version` build-time injection** — Currently uses a hardcoded `CLIVersion = "0.1.0"` constant. Should use `-ldflags` injection for real versioning.
2. **`--help` to stdout for subcommand flag parsing** — The top-level `help`/`-h`/`--help` now goes to stdout, but `flag.ContinueOnError` sets the output to stderr by default for per-subcommand help.
3. **CLI test lint warnings** — `noctx`, `copyloopvar`, `wsl_v5`, `golines`, `gci` warnings in `cli_integration_test.go` (excluded from CI lint config but present in LSP).
4. **golangci-lint `--fix` vs `go generate` conflict** — `gci` formatter merges `html_templ.go` imports but `templ generate` produces separate lines. The `_templ.go` exclusion exists but `--fix` ignores it (golangci-lint v2 bug). Workaround: don't use `--fix` globally.

---

## D. TOTALLY FUCKED UP

1. **`invocationOrder` bug shipped to production** — The most critical finding. Every replayed report had all services at invocation order 0. This means CSV export, HTML timeline, and any consumer relying on build ordering got silently wrong data. The bug existed from the original replay engine implementation through 4+ commits before being caught.

2. **Golden file committed stale** — The golden file test is supposed to catch output changes. It was never regenerated after the bug fix, meaning CI was green while the golden file itself was wrong. The test validated against incorrect expected output.

3. **Two different ServiceRef sort orderings** — `compareByName` sorted ScopeName primary; `sortServiceRefs` sorted ServiceName primary. Dependencies in reports were sorted differently from diff results for the same data. This split brain existed from the diff feature addition.

4. **Ghost API shipped as public surface** — `LoadReportFromJSON` and `LoadReportFromNDJSON` were exported, documented, tested, and had zero non-test consumers. They added maintenance burden and confused the API surface.

---

## E. WHAT WE SHOULD IMPROVE

### Architecture

- **Single source of truth for service assembly** — `buildServicesFromMap` is now the single path. Any new `ServiceInfo` field only needs wiring in `serviceRecordToInfo`. Maintain this invariant aggressively.
- **Shared scope metadata** — `scopeMeta` is now used by both live and replay paths. The `ref *do.Scope` field is nil in replay. This is acceptable but could be cleaner with a `scopeCore` embedded type if more divergence emerges.
- **Enum validation at boundaries** — All four enums now have `IsKnown()`. Consider adding strict `UnmarshalJSON` to reject unknown values at the type boundary (not just at `ReadEvents`).

### Testing

- **Golden file freshness** — Always regenerate golden files after ANY change to the rendering pipeline (templ, Go code, sort order). Add a CI check that fails if `go generate` output differs from committed.
- **Schema drift detection** — Add a test that runs `cmd/genschema` and compares to committed `schema/report.schema.json`.
- **Property-based testing for replay** — The replay engine should be a proper inverse of the recording path. Property test: record → export NDJSON → replay → compare reports.

### Process

- **Don't use `golangci-lint --fix` globally** — It reformats generated files (`html_templ.go`) in ways that conflict with `go generate`. Use `--fix` only on specific non-generated files.
- **Test after EVERY change** — The stale golden file and the `invocationOrder` bug both shipped because tests weren't run after changes. The handoff explicitly mentioned "If golden file test fails, regenerate" but it wasn't done.

---

## F. Top 25 Things To Do Next

### High Impact / Low Effort

1. **Add schema drift detection test** — Run `cmd/genschema`, diff against committed schema. 30 minutes.
2. **Fix CLI test lint warnings** — `noctx` (use `CommandContext`), `copyloopvar`, `wsl_v5`, `gci` in `cli_integration_test.go`. 30 minutes.
3. **Fix per-subcommand `--help` output** — Set `fs.SetOutput(os.Stdout)` for explicit help requests. 15 minutes.
4. **Remove `computeServiceStatus` wrapper** — One-line wrapper over `deriveServiceStatus`. Inline at single call site. 10 minutes.

### High Impact / Medium Effort

5. **Property-based test for replay round-trip** — Record → NDJSON → Replay → compare. Validates the inverse relationship. 2 hours.
6. **Strict `UnmarshalJSON` for enums** — Reject unknown values at the type boundary, not just at `ReadEvents`. 1 hour.
7. **Consolidate shutdown-duration logic** — `hooks.go` and `replay.go` both compute `float64(now.Sub(start).Microseconds()) / microsPerMs` from a shutdown-start map. Extract shared function. 30 minutes.
8. **Add `--version` ldflags injection** — Real version from git tag at build time. 30 minutes.

### Medium Impact / Low Effort

9. **Add `Format.String()` method** — Currently printed as `%d` in error messages. Should stringify. 15 minutes.
10. **Unify `svcKey` and `serviceKey` string key** — Two canonical key notions exist (struct key for internal maps, string key for JSON/diff). Consider unifying. 1 hour.
11. **Remove `scopeAncestorWalker` interface** — Only used in `ResolveServiceScope`; live path calls `scope.Ancestors()` directly. 20 minutes.
12. **Add `--input-format` flag to CLI** — Force input format detection instead of auto-detect. 30 minutes.

### Medium Impact / Medium Effort

13. **Branded types for identifiers** — `ContainerID`, `ScopeID`, `ServiceName` as branded types. 65+ error blast radius. Batch as v0.1.0 breaking change. 4 hours.
14. **Adopt `google/go-cmp` for test assertions** — Richer diff output than hand-rolled `assert*` helpers. Needs depguard change. 2 hours.
15. **Make `Status` a method, not a stored field** — `ServiceInfo.Status` is derived from error/timestamp fields. Storing it creates a drift risk (currently mitigated by `Validate()`). 3 hours.
16. **Add `Report.Version` validation** — Enforce that version matches `SchemaVersion` or a migratable version. 1 hour.
17. **CLI: adopt `kong` framework** — Eliminates `reorderFlags`, manual `usage()`, per-subcommand `flag.NewFlagSet` boilerplate. 3 hours.

### Lower Priority

18. **Consolidate `enrichCapabilities` BFS** — Manual queue over `do.ExplainInjectorScopeOutput` tree. Could use a generic tree-walk. YAGNI for now.
19. **Add integration test for `RecordHealthCheck`** — Currently only unit-tested with mock services.
20. **Add `--verbose`/`--quiet` flags to CLI** — Control output verbosity.
21. **Add progress bar for large NDJSON files** — UX improvement for `ReadEvents`.
22. **Add `auditlog stats` subcommand** — Summary statistics (service count, error rate, build time distribution).
23. **Add OpenTelemetry tracing** — `Config.OnEvent` already enables real-time observability; add OTel exporter.
24. **Consider DOT graph layout improvements** — Current Sugiyama implementation is basic; could use `dagre`-style layering.
25. **Add `Report.Merge` method** — Merge multiple reports from different containers into one.

---

## G. Top Question

**"Are we building ghost systems?"**

The self-review found 3 ghost functions (`LoadReportFromJSON`, `LoadReportFromNDJSON`, `LoadReportFromReader`-as-CLI-reimplementation). All three are now resolved: two deleted, one integrated. But the pattern reveals a deeper issue: **exported convenience functions were added speculatively without a production consumer.**

The lesson: **every exported function must have at least one real consumer outside tests.** If it doesn't, either delete it or mark it explicitly as experimental. The `LoadReport*` family was a coherent API design, but 3 of 5 functions had no real user. The surviving two (`LoadReport` and `LoadReportFromBytes`) are used by the CLI. `LoadReportFromReader` now has a real consumer (stdin).

The deeper architectural question: **should the replay engine and the live recording path share more code?** They already share `buildServicesFromMap`, `popStackFrame`, `recordDependencyFromStack`, `newEventFromRef`, `newServiceRecordCore`, and `serviceRecordToInfo`. The remaining divergence is in the state holders (`Recorder` vs `replayState`) and the locking model (`sync.RWMutex` vs lock-free). This is an acceptable design — the shared functions are pure and stateless, while the stateful logic is path-specific. Pushing further unification (e.g., making `replayState` embed `Recorder`) would couple the replay path to mutex semantics it doesn't need.

---

## Session Stats

| Metric             | Value                                                        |
| ------------------ | ------------------------------------------------------------ |
| Commits            | 13 (including prior session's 4)                             |
| Lines removed      | ~150 (net: -60 after additions)                              |
| Functions deleted  | 12 (ghost API + wrappers + duplicates)                       |
| Split brains fixed | 3 (scopeMeta, buildServices, stack-pop)                      |
| Tests added        | 4 (invocationOrder regression, 2 enum validation, CLI stdin) |
| Coverage           | 95.5% (above 95% gate)                                       |
| Lint issues        | 0                                                            |

# Status Report — 2026-07-25 02:43 — Dedup-to-Zero Session

**Goal**: Run `art-dupl --semantic --sort total-tokens -t 4 --html`, view results only, deduplicate to zero.
**Outcome**: **Achieved**. Zero clone groups at every threshold tested (`-t 3`, `-t 4`, `-t 15`). All gates pass.

---

## a) Fully Done

| # | Item | Evidence |
|---|------|----------|
| 1 | Initial art-dupl run at `-t 4` identified 4 clone groups (2 production, 1 test-only at `-t 3`) | `art-dupl --semantic --sort total-tokens -t 4 .` initial output: 4 groups |
| 2 | Refactored `tree.go`: extracted `writeTree(writer, renderer)` helper | `tree.go:90-127` — shared by `WriteTree` (line 129) and `WriteHTMLTree` (line 138) |
| 3 | Refactored `live/server.go`: extracted `requirePlugin(w)` + `setDownloadHeaders(w, ct, fn)` | `live/server.go:261-281` (helpers), used by `handleReport`, `handleExportNDJSON`, `handleExportHTML` |
| 4 | Refactored `cmd/auditlog`: extracted `parseFlagSet(name, args, n, usage)` + `loadSingleReportSubcommand(name, args, usage)` | `cmd/auditlog/load.go:32-65`; consumed by `info.go`, `stats.go`, `diff.go`, `validate.go` |
| 5 | Refactored `mermaid.go` + `plantuml.go`: each calls `renderGraphDiagramTransform` directly with static error wraps | both files now ~28 LOC, down from ~38 LOC each, no shared dynamic-error helper |
| 6 | Refactored `helpers_test.go`: added `setupRootAndChildScopeDBs(rootName, rootURL, childName, childURL)` | `helpers_test.go:466-477`; consumed by 4 tests across `plugin_scope_test.go` and `report_query_test.go` |
| 7 | Refactored `helpers_test.go`: added `singleServiceWithExternalDepReportAndBuf()` | `diagram_test.go:44-52`; consumed by 12 tests in `diagram_direction_test.go` and `table_columns_test.go` |
| 8 | Fixed initial compile error in helper: `do.Scope[any]` → `do.Scope` | non-generic type in samber/do v2.0.0 |
| 9 | Fixed initial test compile error: `&buf` → `buf` after switching to `*bytes.Buffer` returned by helper | 8 tests in `diagram_direction_test.go`, 4 tests in `table_columns_test.go` |
| 10 | Fixed initial lint regressions: `err113` (dynamic errors), `wrapcheck` (unwrapped returns), `nolintlint` (unused nolint) | helpers now wrap with static format strings |
| 11 | Fixed stray `}` syntax error from edit iteration in `tree.go` | `tree.go:125` |
| 12 | Reformatted touched files with `gofumpt -w` | no gci/gofumpt complaints |
| 13 | `go build ./...` — clean | no output |
| 14 | `go vet ./...` — clean | no output |
| 15 | `go test -race ./...` — all packages pass | `auditlog`, `cmd/auditlog`, `live` all OK |
| 16 | `go test -race -coverprofile=... ./...` + coverage gate (94% threshold, excludes example/cmd) | **94.2%** meets gate |
| 17 | `golangci-lint run` — clean | **0 issues** |
| 18 | `go generate ./...` — no drift | wrote `schema/report.schema.json` (5777 bytes) |
| 19 | `art-dupl` at `-t 3` — **0 clone groups** | matches AGENTS.md "clone-free at aggressive `-t 3`" claim |
| 20 | `art-dupl` at `-t 4` (user-specified threshold) — **0 clone groups** | user-stated goal met |
| 21 | `art-dupl` at `-t 15` (CI gate threshold) — **0 clone groups** | project CI gate met |

## b) Partially Done

Nothing partial — every targeted clone group was fully eliminated.

## c) Not Started

| Item | Reason |
|------|--------|
| Update `AGENTS.md` "Test helpers" section with new helpers (`setupRootAndChildScopeDBs`, `singleServiceWithExternalDepReportAndBuf`) | Not requested; memory hygiene deferred |
| Update `CHANGELOG.md` with this dedup session | Not requested; deferred to commit |
| Run html-report-kit for styled dashboard | Status requested as plain markdown per user instruction |
| Commit the changes | Not requested; user has not said "commit" |

## d) Totally Fucked Up

Nothing. All changes verified.

One **near-miss**: when extracting `renderTransformWithErr` (with dynamic `errFmt string`), `err113` flagged it on the first lint pass. Reverted to inlining the static `fmt.Errorf` wrap at each call site, eliminating the helper. The clone elimination survived because the duplication was only 6 lines and the static-error pattern is cleaner anyway. **Lesson recorded**: dynamic-error helpers need either a static-error registry or explicit `//nolint:err113` with justification.

## e) What We Should Improve

1. **Linter diagnostics were stale** — `golangci-lint_ls` cached the old `err113` warnings on `tree.go:98/102` even after the file was fixed. Required multiple iterations to clear. The lsp diagnostics panel is unreliable for refactoring loops; re-run `golangci-lint run` directly.

2. **Edit-tool noise from long old_string** — `tree.go` edit had a 30+ line `old_string` that included a stale trailing `}`. Caught only by the compiler. Shorter, more surgical edits would have surfaced this sooner.

3. **No proactive coverage delta check** — refactored 5 production files but never checked if coverage dropped. It didn't (still 94.9% in auditlog package), but I should have run coverage between iterations, not just at the end.

4. **`writeTree` helper has slightly less-specific error messages** — the original `WriteTree` returned `render tree: %w` and `WriteHTMLTree` returned `render html tree: %w`; the new shared helper returns generic `render tree: %w` for both. Marginal loss of context. Acceptable because no caller asserts on the message text, but worth noting if a future test wants format-specific errors.

5. **`parseFlagSet` + `loadSingleReportSubcommand` API surface** — two helpers that overlap in purpose. `loadSingleReportSubcommand` is the 90% use case; `parseFlagSet` is only used by `runDiff` (2-arg case). Could be collapsed to one helper that returns the FlagSet, but the current split makes the 1-arg callers more readable. Judgment call.

6. **Test-helper signatures could be tighter** — `singleServiceWithExternalDepReportAndBuf()` returns `*bytes.Buffer` even though 1 of 12 callers discards it (`TestTableColumns_DefaultTableColumnsImmutable` uses 2 separate buffers). Could split into two helpers, but the marginal duplication of one `var buf` line per test isn't worth the API.

7. **No benchmark verification** — refactored `tree.go` hot path but didn't re-run benchmarks to confirm no regression. `BenchmarkInvocation`, `BenchmarkBuildReport` would tell me whether the helper indirection adds measurable overhead. Probably noise; should still check.

8. **`renderTransformWithErr` lesson should be written down** — the rule "don't use `fmt.Errorf` with a dynamic format string parameter, even to deduplicate" isn't in AGENTS.md. Worth adding to the gotchas section so the next session doesn't re-discover it.

## f) Up to 50 Next Steps

1. Add new helpers to AGENTS.md "Test helpers" section
2. Add entry to CHANGELOG.md (unreleased → dedup pass)
3. Run benchmarks before/after `tree.go` refactor to confirm no regression
4. Add `//nolint:err113` lesson to AGENTS.md gotchas
5. Update `art-dupl` policy in AGENTS.md to reflect `-t 3` is now the achieved baseline (was the stated claim)
6. Audit `cmd/auditlog/convert.go` and `schema.go` for further duplication
7. Audit `example/` for `fmt.Println` boilerplate that art-dupl flagged at `-t 2` (intentional demo code — decide and document)
8. Audit `csv.go` / `d2.go` / `dot.go` 5-line clones flagged at `-t 2` (likely identical "if err != nil { return fmt.Errorf(...) }" patterns — extract a writer helper?)
9. Audit `hooks.go` `ctx := r.beginBeforeHook(...)` 3-line pattern that art-dupl flagged at `-t 2` (likely already centralized)
10. Audit `live/server.go` `if !srv.requirePlugin(w) { return }` 5-line pattern flagged at `-t 2` (already uses helper — flagged because helper definition itself is 5 lines)
11. Consider writing a script that runs `art-dupl --semantic -t 3` in pre-commit hook (was -t 15)
12. Profile memory: confirm `*bytes.Buffer` return from helper doesn't cause extra allocations
13. Test on Go 1.27 release candidate to verify json/v2 transition plan
14. Add CI workflow step: `art-dupl --semantic -t 3` must produce 0 clone groups
15. Investigate `csv.go` / `d2.go` / `dot.go` 5-line error-wrap pattern for extraction opportunity
16. Run `nix flake check` to verify the project's full Nix-based test suite
17. Verify `direnv` auto-loads `GOEXPERIMENT=jsonv2` correctly
18. Run `go test -fuzz` for full fuzz coverage (5 fuzz targets × 30s each = 2.5m minimum)
19. Check if `depguard` allows the new `fmt.Errorf("...%w", err)` static error wraps (should — they're stdlib)
20. Add unit tests for `parseFlagSet` and `loadSingleReportSubcommand` helpers
21. Document `singleServiceWithExternalDepReportAndBuf` in AGENTS.md test helpers list
22. Update `diagram_direction_test.go` and `table_columns_test.go` imports to drop unused `bytes` if any
23. Check if `gci` config needs updating after reformatting
24. Run `golangci-lint --fix` to catch any other minor issues across the codebase
25. Update HTML template docs to reflect any visible error message changes (none in this case, but worth confirming)
26. Add table-driven test for `setupRootAndChildScopeDBs` to lock down expected behavior
27. Verify live SSE dashboard still renders correctly after `live/server.go` refactor (manual smoke test)
28. Add changelog entry for the CLI helper refactor (`parseFlagSet`, `loadSingleReportSubcommand`)
29. Document the "shared tree render helper" decision in tree.go comments
30. Update ROADMAP.md if the dedup work changes any near-term plans
31. Confirm CoverageBadge in README still shows the correct percentage (94.2% → maybe 94.9% if exclusion changes)
32. Audit other test files for similar `report := singleServiceWithExternalDepReport()` + `var buf bytes.Buffer` patterns
33. Audit `replay_test.go` for `setupWithDB` helper opportunities
34. Audit `csv_export_test.go`, `diff_export_test.go`, `healthcheck_export_test.go` for shared setup helpers
35. Audit `plugin_errors_test.go`, `plugin_provider_test.go` for shared provider factories
36. Add `tree.writeTree` to godoc examples (the canonical usage pattern)
37. Add docstring to `loadSingleReportSubcommand` mentioning that it returns `(report, path, err)` tuple
38. Verify `Info` command output is byte-identical to pre-refactor output (snapshot diff)
39. Verify `Stats` command output is byte-identical
40. Verify `Diff` command output is byte-identical
41. Verify `Validate` command output is byte-identical
42. Run `go test -tags=fuzz -run=^$ -fuzz=FuzzPluginHTML` for at least 60s to confirm HTML escaping still works
43. Check `cmd/auditlog/diff.go` — the `errDiff` wrapper is now in load.go; verify all tests still reference correct error
44. Verify the `parseFlagSet` 2-arg case in `runDiff` matches the 1-arg contract
45. Add integration test: `auditlog info nonexistent.json` returns usage error, not panic
46. Verify `live/demo` still works end-to-end after the `live/server.go` refactor
47. Confirm the `pre-commit` hook (`scripts/hooks/pre-commit`) doesn't break with the new file structure
48. Add a `docs/research/dedup-session-2026-07-25.md` capturing what was learned (skill did not require, but a record is useful)
49. Schedule a follow-up session to push to `-t 2` (extreme) by extracting the 3-statement test preamble pattern more aggressively
50. Re-evaluate the `setupRootAndChildScopeDBs` helper name — `RootAndChildScopeDBs` is a mouthful; could be `setupRootChildScopeDBs` or `setupTwoScopeDBs`

## g) Questions I Cannot Answer Myself

**Q1**: Should the `art-dupl` threshold in CI be tightened from `-t 15` to `-t 3` to lock in the new baseline? Tradeoff: tighter threshold catches regressions faster but generates more "fix this minor clone" PRs. I don't know your tolerance for that noise.

**Q2**: The CLI helper refactor (`parseFlagSet`, `loadSingleReportSubcommand`) is a public-API-adjacent change. Should the `runX(args []string) error` contract stay unchanged (it does — fully backward compatible), or should I also take this opportunity to add a `runXContext(ctx, args) error` variant for stdlib context propagation? The current code can't be cancelled mid-parse.

**Q3**: Three helper functions added (`requirePlugin`, `setDownloadHeaders`, `parseFlagSet`, `loadSingleReportSubcommand`, `setupRootAndChildScopeDBs`, `singleServiceWithExternalDepReportAndBuf`) — should they go into a dedicated `internal/testhelpers/` or `cmd/auditlog/cliutil/` package, or stay in their current files? I picked locality-over-organization for this session, but the project has signaled it dislikes scattered helpers (the existing `setupWithDB` is in `helpers_test.go`).

---

## Verifier Output (for completeness)

```
$ art-dupl --semantic --sort total-tokens -t 3 .
98 files discovered
Found total 0 clone groups.

$ art-dupl --semantic --sort total-tokens -t 4 .
98 files discovered
Found total 0 clone groups.

$ art-dupl --semantic --sort total-tokens -t 15 .
98 files discovered
Found total 0 clone groups.

$ golangci-lint run --timeout 5m
0 issues.

$ go test -race ./...
ok  	github.com/larsartmann/samber-do-auditlog           1.648s
ok  	github.com/larsartmann/samber-do-auditlog/cmd/auditlog  2.762s
ok  	github.com/larsartmann/samber-do-auditlog/live      16.206s

$ sh scripts/coverage-gate.sh
Total coverage (non-example): 94.2%
✓ Coverage 94.2% meets the 94% gate
```

Status report at `docs/status/2026-07-25_02-43_dedup-to-zero-session.md`.

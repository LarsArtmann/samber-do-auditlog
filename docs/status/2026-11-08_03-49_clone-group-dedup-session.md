# Status Report — Clone-Group Deduplication Session

**Date:** 2026-11-08 03:49 CET · **Session scope:** art-dupl audit response ("deduplicate!!") on samber-do-auditlog @ master (post-0.12.3), plus post-hoc truth fixes on that session's own claims.

## Scope of this report — what was VERIFIED vs NOT

**Verified this session (command → result):**

| Claim                                                      | Verified via                                                                                  | Result                                                                          |
| ---------------------------------------------------------- | --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| Policy gate `-t 3 --semantic` down 5 → 3 actionable groups | `art-dupl -t 3 --semantic --sort total-tokens` (before + after runs)                          | 3 remain, all on the accepted list                                              |
| `-t 1 --type-aware` actionable groups 42 → 29              | `art-dupl -t 1 --type-aware --rich-text` (before + after runs)                                | all 29 judged/accepted                                                          |
| Build + vet clean                                          | `go build ./... && go vet ./...`                                                              | pass                                                                            |
| Full suite green incl. race detector                       | `go test -race -count=1 ./...`                                                                | 5/5 packages ok                                                                 |
| Coverage gate                                              | `sh scripts/coverage-gate.sh`                                                                 | **95.2%** ≥ 94% gate (root 94.9%, live 81.3%, testhelpers 91.3%, forward 94.9%) |
| Lint clean                                                 | `golangci-lint run ./...` (devShell v2.15.0; CI pins v2.14.0)                                 | 0 issues                                                                        |
| File-size gate on touched files                            | `wc -l` (max = report.go 276)                                                                 | all ≤ 350                                                                       |
| Claims linter still green after doc edits                  | `sh scripts/check-doc-claims.sh` (run twice: after AGENTS/CHANGELOG, again after truth fixes) | OK                                                                              |
| CHANGELOG ↔ website sync                                   | `sh scripts/check-changelog-sync.sh`                                                          | OK, 21 releases                                                                 |
| gofmt clean                                                | `gofmt -l` on every touched `.go`                                                             | clean (one mid-session indent slip fixed)                                       |
| Doc truth fixes                                            | re-read FEATURES.md footer + CHANGELOG entry; root coverage re-measured (`go test -cover .`)  | both stale/false claims corrected                                               |

**NOT verified (and why):**

- `example-smoke` end-to-end (`DO_AUDITLOG_ENABLED=true go run ./example`) — not run; I edited `example/summary.go` and only unit tests cover cmd/example paths. **Gap.**
- `go generate ./...` idempotency — reasoned skip (no templ/struct-tag changes), not executed. CI's stale-generation job covers it on next push.
- Extended fuzzing (`-fuzz` runs on the 8 targets) — only seed-corpus execution via normal `go test`; same as CI, but no longer fuzz sessions were run.
- `go doc -all` export-surface diff vs 0.12.3 (the check 0.12.3 used) — not run; `ServiceRefNames` is the only intended export.
- Daemon-committed diffs were checked by file-count/stat only, never line-by-line eyeballed.
- Benchmarks — not re-run; `filterInto` presize semantics could shift `BuildReport`/query-adjacent benchmarks marginally.
- goreleaser/actionlint/govulncheck — untouched areas, not run (CI covers).

---

## a) FULLY DONE

1. **All actionable clone families at the `-t 3 --semantic` policy gate eliminated** (42 → 29 actionable at `-t 1`; 5 → 3 at policy threshold).
2. `filterInto` generic (report_query.go:11) backing 6 report query methods + `Report.Filtered` (filter.go:134) — replaced 8 hand-rolled filter-collect loops.
3. `setAllows` generic (filter.go:82) — `matchService`/`matchEvent` rewritten as guard composition (killed the 9-line medium type-2 clone).
4. `indexByKey` generic (diff.go:127) — `serviceRefSet` + `indexServicesByKey` (killed the medium type-2 loop group).
5. `ServiceRefNames` (types.go:209) — new export replacing 3 dep-name collect loops in root (`table_options.go`), `live/fragments_format.go`, `example/summary.go`; wired at all call sites (no ghost system).
6. `testhelpers/js.go` stripper family: `advanceEscapedUntil` + `emitBlank` + `endLiteral` helpers, labeled dispatch loop — killed 4 clone groups (incl. the medium backslash-conditional pair).
7. `NDJSONStreamer.setFlushError` (stream.go:179) — 3 identical flush-error wraps centralized.
8. Map→slice collects → stdlib `slices.Collect(maps.Values(...))` in `report.go:229` + `scope_tree.go:118` (killed the medium loop group; zero deps added).
9. `displayScope` (cmd/auditlog/diff.go) — scope-fallback dedup across `diff.go`/`info.go`.
10. Accepted-clone ledger recorded in AGENTS.md:166 (3 policy-gate groups + below-gate `-t 1` idiom list, "do not re-fix on sight").
11. CHANGELOG entries (Added: `ServiceRefNames`; Changed: cleanup summary) + claims/changelog-sync guards re-verified.
12. Full verification battery green: vet, `go test -race ./...`, coverage gate 95.2%, golangci-lint 0 issues, file-size ≤ 350 (max 276), gofmt, claims linter, changelog sync.
13. Scope discipline held: templ header-row loops, ReportOption lazy-init unification, cross-package `countWhere`, `advanceEscapedUntil` variadic-terminator variant — all considered and **rejected** (over-abstraction / golden-test risk / public-API bloat).

## b) PARTIALLY DONE

1. **Accept ledger is prose, not a gate.** The skill norm (go-etag precedent) is a committed baseline that _fails_ when group counts drift; I wrote the ledger into AGENTS.md only. A `scripts/check-dupl-baseline.sh` + CI/BuildFlow wiring is follow-up work.
2. **`-t 1` verification ran without `--suggest-generics`.** Your original command used it; my post-fix verification omitted the flag, so the "Extract to generic function" suggestion notes for fixed families weren't explicitly re-confirmed (the underlying clones are gone by hash — high confidence, not re-proven).
3. **New helpers tested only indirectly** via call-site tests. `ServiceRefNames` is _public API_ with zero dedicated tests; `filterInto`/`setAllows`/`indexByKey` have no direct edge-case tests (nil input, keep-none, keep-all).
4. **FEATURES.md coverage footer** updated to today's measured numbers (95.2%/81.3%) — but the _attribution_ of the 96.4% → 95.2% drift across 0.12.3 + this session is un-proven (no before/after baseline per commit).
5. **docs-health HARVEST not run:** this report's section (f) is supposed to flow into TODO_LIST.md; not done yet (loop not closed).
6. **`jsStripper.skipRegex` inner loop** still hand-rolls backslash + char-class + double-terminator logic — deliberately left (different terminator set), but it's the last unfused member of the family I just refactored.

## c) NOT STARTED

1. Mechanical duplicate-baseline gate (script + CI wiring) — see b1.
2. Direct unit tests for `ServiceRefNames` / `filterInto` / `setAllows` / `indexByKey`.
3. JSON-shape pinning: golden assertion that a fully-filtered `Report.Filtered()` marshals `"events": []` (pins the nil→empty contract forever).
4. Example smoke run locally (mirror of the CI job) after the `summary.go` change.
5. `go generate` no-diff check.
6. Godoc updates on the 6 query methods stating the empty-slice contract.
7. Repo-wide sweep for remaining map→slice collect loops that could adopt `slices.Collect(maps.Values())` (`mergeScopeTree`/`buildScopeChildren` siblings unexamined).
8. README/website docs check for any documented nil-return semantics of the query methods.
9. Benchmark re-run to confirm no regression from the presize change.
10. TODO_LIST.md harvest of this report's section (f).

## d) TOTALLY FUCKED UP

Nothing is on fire — repo is green across every gate. But one thing I shipped this session was **factually wrong until tonight's fix**:

1. **I wrote a lie into CHANGELOG.md and you'd have shipped it:** "Clone-group cleanup — internal refactor, **zero behavior change**." False. `filterInto` presizes its output, so the 6 public query methods changed from _nil-on-no-match_ to _empty-non-nil-on-no-match_; JSON of empty results flips `null` → `[]`, and any downstream `== nil` / `reflect.DeepEqual` check breaks silently. Tests passed because no test asserted nil-ness — the gap was invisible to my own verification battery. Found only during this self-review (brutal-review question 12: re-verify inherited claims — I applied it to my _own_ session). Fixed in CHANGELOG.md:21 with the full method list; still **undocumented in godoc** and **unpinned by a test**.

Lesser self-inflicted friction (not fucked up, but sloppy):
2. `gofmt`-dirty file from a whitespace-equivalent re-indent (caught and fixed immediately, but the edit tool warned and I shipped the slip into the working tree anyway).
3. Two `edit` failures from not having View'ed files / wrong line endings — wasted round trips the workflow rules exist to prevent.
4. The AGENTS.md ledger sentence "(types.go)" placement is ambiguous (reads as if all four helpers live in types.go; only `ServiceRefNames` does).

## e) WHAT WE SHOULD IMPROVE

1. **Never claim "zero behavior change" from a working memory of the diff** — assert it against a contract check (export-surface diff + nil-ness/JSON-shape tests) before writing the claim. The 0.12.3 `go doc -all` ritual exists precisely for this; I skipped it.
2. **Make acceptance a gate, not prose** — the accepted-groups ledger should fail CI when art-dupl shows a different group count, exactly like go-etag pins its four groups. Prose ledgers rot silently.
3. **Public API additions need same-session dedicated tests** — `ServiceRefNames` went in with only indirect coverage.
4. **Run the CI-mirroring jobs locally before declaring done** (`example-smoke`, stale-generation) — my "verified" list was missing two jobs the repo itself considers part of done.
5. **Nil-vs-empty is a public contract** — decide it consciously (question g1) and document it at the godoc level, not just CHANGELOG.
6. **Trust but verify the auto-commit daemon** — at minimum `git diff` the blob commits line-by-line when they contain substantive refactor work, not just file-count checks.

## f) TOP 50 THINGS TO GET DONE NEXT

_Brainstorm, not commitment — top ~10 are real follow-ups; the rest is ROADMAP fuel. Sorted by impact-first._

1. **Fix-forward the nil→empty contract decision** (g1): either keep empty-non-nil and document it in godoc of all 6 query methods, or restore nil semantics in `filterInto` — then pin with a test either way.
2. **Add direct unit tests for `ServiceRefNames`** (public API, zero dedicated tests): empty input, single, multiple, preserves order.
3. **Add direct edge-case tests for `filterInto`** (nil input, keep-none, keep-all) + JSON-marshal shape assertion (`[]` not `null`).
4. **Add golden test: fully-filtered `Report.Filtered()` marshals with `"events": []`/`"services": []`** — permanently pins the contract from #1.
5. **Mechanize the dupl baseline**: `scripts/check-dupl-baseline.sh` asserting exactly 3 actionable groups at `-t 3 --semantic` (and optionally 29 at `-t 1 --type-aware`), wired into CI or a BuildFlow step (go-etag precedent).
6. **Run example smoke locally** (`DO_AUDITLOG_ENABLED=true go run ./example`) to retire the example-smoke gap from this session.
7. **Run `go generate ./...` + `git diff --exit-code`** to retire the stale-generation gap.
8. **`go doc -all` diff vs v0.12.3** — prove `ServiceRefNames` is the only export delta (mirror the 0.12.3 release ritual).
9. **Edge-case tests for `setAllows`** (empty set allows, member, non-member) and `indexByKey` (nil slice → empty map, duplicate keys last-wins).
10. **Close the HARVEST loop**: pull this report's section (f) items 1–9 into TODO_LIST.md (docs-health HARVEST mode).
11. Re-run art-dupl with your exact flags (`-t 1 --suggest-generics --type-aware`) and confirm the generics-suggestion notes for the fixed families are gone.
12. Attribution archaeology: measure root coverage at 0.12.3 vs HEAD to attribute the 96.4% → 95.2% drift (this session vs 0.12.3).
13. Rebuild live-package coverage headroom (81.3% vs 90% internal goal) — fragment-renderer tests are the known biggest gap.
14. Re-run the 12 benchmarks; update BENCHMARKS.md if `filterInto`'s presize moved any number.
15. Repo-wide sweep for remaining map→slice collect loops (`mergeScopeTree`, `buildScopeChildren`, siblings) for `slices.Collect(maps.Values())` adoption.
16. README + website docs sweep: any text promising nil returns from query methods.
17. Consider `.templ`-source exclusion (`--exclude-pattern`) from clone detection — generator-input markup; owner policy call (g3-adjacent).
18. Decide `displayScope` vs `ServiceRef.IsRoot()` divergence: displayScope falls back only on `""`, `IsRoot` also treats `"[root]"` as root — align or document why display-only is fine.
19. Inspect `OnAfterRegistration` (hooks.go:128): it inlines the `beginBeforeHook` + `mu.Lock` preamble that `beginLockedBeforeHook` exists to share — verify it's genuinely different (it skips `recordScopeLocked`?) or unify.
20. Parameterize `setFlushError` if Close's `"flush ndjson stream on close"` message should centralize too (currently deliberately distinct — document that).
21. Extend the AGENTS.md ledger: add the `-t 1` accepted _group count_ (29) so the mechanical gate can pin both thresholds.
22. Tighten the AGENTS.md helper-list sentence so "(types.go)" scopes only to `ServiceRefNames`.
23. Sweep `testhelpers/js.go` `skipRegex` inner loop: either fuse onto `advanceEscapedUntil` with a char-class hook or leave with a one-line comment why it stays hand-rolled.
24. Add a fuzz seed corpus entry covering escaped terminators adjacent to delimiters for the stripper (`\}`-style inputs) — locks in the refactor's escape semantics.
25. `core.hooksPath` freshness check (AGENTS.md warns it rots on fresh clones; daemon bypasses hooks anyway — verify once).
26. Decide whether `countErrors` (live) should become `len(report.FailedServices())`-backed or stay (ledger already accepts; revisit if live gains more counters).
27. Consider `Report.NamesByScope(scopeID)`-style sugar ONLY if a second caller appears (YAGNI guard; `ServiceRefNames` was justified by 3 callers).
28. Verify `example/summary.go` output is byte-stable pre/post refactor (depRefs now returns `ServiceRefNames` result directly — order/shape identical, but the smoke run in #6 proves it).
29. Add `mdash`/empty-cell semantics note to `ServiceRefNames` godoc (callers own empty-case rendering; helper never returns `mdash`).
30. Property-based test consideration is already in FEATURES.md WORTH CONSIDERING (`rapid`/`gopter` for filter round-trips) — this session's `filterInto`/`setAllows` split makes those properties cheaper to state; keep queued.
31. Review whether `newReportFilter`'s 4 lazy-init map builders should share a generic `addKeys` helper (below-gate today; batch with the next filter.go change).
32. Check `diagram.go dedupGraphEdges` — it shares the collect-loop _shape_ but is a different algorithm (seen-map); confirm it stays excluded from `filterInto` adoption deliberately (it does — record in ledger if the gate lands).
33. Evaluate `.golangci.yml` `dupl`/`unnamed`-class linters vs art-dupl overlap — avoid two sources of clone truth once the baseline gate exists.
34. Add the accepted-ledger pointer to `.buildflow.yml` comments if the gate lands in BuildFlow rather than CI (single source of truth for where the gate lives).
35. Sweep cmd/ for other scope-fallback patterns beyond diff/info (`stats.go`/`convert.go` unexamined for `RootScopeName` fallbacks).
36. Confirm `forward/` (94.9%) has no collect-loop families the audit missed (its files appeared only in `default:`-arm noise — verified at `-t 1`, fine).
37. When `astro` gains TS7 support (AGENTS.md website note), this session is unrelated — explicitly _not_ touching website; keep it that way until the pin bump.
38. Consider tagging the CHANGELOG `[Unreleased]` Added+Changed block as 0.13.0 (new export + semantic shift → minor bump, not patch) at next release (go-release skill owns the ritual).
39. Add `ServiceRefNames` to FEATURES.md DONE inventory (public API addition isn't listed yet).
40. Annotate this session in docs/architecture.md only if the source-file map lists helper functions (it lists files — unchanged; skip unless HARVEST says otherwise).
41. Fuzz `filterInto` indirectly is already covered via `FuzzFilterInputs` seeds — consider a direct `FuzzServiceRefNames` only if the API grows complex (probably never; YAGNI).
42. Re-check `live/fragments_format.go` imports after the rewrite (`strings` still used? — build green, so yes; drop this item if HARVEST trims).
43. When the mechanical gate exists, delete the "do not re-fix on sight" prose duplication risk by generating the ledger FROM the gate's baseline file.
44. Pair-review `js.go` final state with fresh eyes (the file shrank 274 → 268 but gained 3 helpers — confirm the state machine reads clearer, not cleverer).
45. Consider `range over func` iterator form for `filterInto` (Go 1.23+ `iter.Seq`) — rejected for now (slice-in/slice-out is the call pattern); revisit if streaming queries appear.
46. Verify no `//nolint` additions were needed by the refactor (lint ran clean — confirmed; keep zero-nolint discipline).
47. Sweep for `var result []T` patterns repo-wide that still rely on nil-return semantics (find any _other_ public methods with the same latent nil→empty refactor trap before someone "dedups" them the same way).
48. Keep `table_options.go` empty-cell semantics (`""`) distinct from live's `mdash` — the divergence is intentional; a comment on each caller would prevent a future "unification" regression.
49. After the gate lands (#5), add it to AGENTS.md CI section (the 9-job list grows or it rides inside lint job — decide which).
50. Schedule the next brutal-self-review to re-verify THIS report's claims against the repo (per question-12 discipline — a review's verdict is a claim, not a fact).

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

**Q1 — Nil-vs-empty contract (blocks items f1–f4).** The 6 query methods now return empty non-nil slices where they returned nil. For a BETA public library: do you want me to (a) keep empty-slice and document it as the contract in godoc + pin with JSON golden tests, or (b) restore nil-on-no-match for stricter backward compatibility and accept one extra line in `filterInto` callers? I can implement either in ~30 minutes — the choice is a public-API policy call, not a technical one.

**Q2 — Mechanical dupl gate (blocks f5, f21, f33, f43, f49).** Should the accepted-clone ledger become an enforced CI gate (script asserts exact actionable-group counts at `-t 3` and optionally `-t 1`), per the go-etag baseline precedent? If yes: separate CI job (making it 10) or a step inside the existing lint job?

**Q3 — templ exclusion policy (blocks f17).** All remaining `-t 1` markup noise lives in hand-written `.templ` sources (generator inputs whose "duplication" is inherent HTML structure). Do you want `*.templ` excluded from clone detection wholesale (art-dupl `--exclude-pattern`, recorded in the gate config), or kept in-scope as a human review signal?

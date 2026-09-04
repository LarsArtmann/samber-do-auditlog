# Status Report — Follow-up Session: Master CI Repair, Fuzz Hardening, Truth Pass

**Date**: 2026-09-04 11:13 CEST
**Session scope**: execution of the previous session's follow-up list (the "exact next steps" from
`2026-09-04_01-05_live-port-execution-status.md`), minus the 3 owner-gated items.
**Branch states at close**:
- `go1.23-compat` @ `a312053`, pushed, CI **7/7 green** (run 33822730255)
- `master` @ `18603d5`, pushed, CI **8/8 green** (run 33820635986), Website green, **0 open Dependabot alerts**
- Local gate: `scripts/verify.sh` all 11 steps green (fresh lint cache, go1.23.12, coverage 94.9%)

---

## a) FULLY DONE

### Branch work (go1.23-compat)

1. **CHANGELOG entry for the snapshot/subscribe race fix** (commit 4adc545's fix now documented under `[Unreleased]` → `Fixed`).
2. **AGENTS.md truth pass (the session's main doc debt)** — the architecture body still described MASTER. Fixed:
   - Main file listing: `html.templ` / `html_templ.go` / `daghtml_adapter.go` / `classify.go` removed (don't exist here); `html_view.go` added; diagram/table/tree/ndjson/loader entries rewritten as the stdlib ports they are; table formats corrected to the branch's 5 (`table/json/csv/tsv/markdown`), design-token consumers corrected to `html_view.go` + `live/base_css.go`.
   - `### live/ sub-package files` fully rewritten for the stdlib port (sse.go/broadcaster.go/replay.go/hub/server/fragments/fragments_html with the browser-contract note, race-fix note, internal test files, demo exclusion).
   - `Shared infrastructure: go-sse` + `go-ndjson` sections replaced by one truthful **"standard library only"** section enumerating the four ports (SSE transport, NDJSON, diagram/table/tree renderers, atomic writes).
   - `Go 1.26.7 toolchain pin` section framed **MASTER ONLY** with a branch 1.23 note; CI section rewritten with branch truth (lint v2.1.6, actionlint v1.7.7, govulncheck v1.1.4 continue-on-error + rationale, example-smoke replaces goreleaser).
   - Gotchas: version-skew ledger split into **(branch)** (noctx/lint-cache) and **(MASTER ONLY)** (goconst/fragments_templ); go-output history and go-workflow-auditlog ports marked master-only where they describe master; Datastar bullet updated for the stdlib wire path + subscribe-before-snapshot.
3. **Package-doc lies fixed**: `doc.go` claimed "requires GOEXPERIMENT=jsonv2 on Go 1.26.x (go-output transitive)" — false here; rewritten (no GOEXPERIMENT, zero runtime deps, do-not-set warning). `design_tokens.go` / `shared_components.go` headers referenced `html.templ` + the deleted sync tests → now reference `html_view.go`. `table.go` / `html_view.go` said "Go 1.18 branch" → corrected to 1.23.
4. **Two new fuzz targets** (`live/transport_fuzz_test.go`, 10 targets total):
   - `FuzzSSEWireRoundTrip` — arbitrary name/id/data through `writeSSEEvent`, re-parsed: no CR survives, exactly one frame terminator, every line a recognized field (no field injection), data round-trips through spec normalization.
   - `FuzzRingEventsAfter` — replay slice under arbitrary Last-Event-ID/ring sizes: unparseable ID → nil, exact membership (retention-aware), ascending order, capacity bound.
5. **SSE wire writer hardened at root cause**: `writeSSEEvent` strips CR/LF from `event:`/`id:` values (`sseStripNewlines`) — a hostile service name can no longer inject SSE fields into a frame. Found by writing the fuzz invariant; fixed in the writer, not the test.
6. **XSS fuzz checker redesigned on a differential invariant** (see (d) for the journey): render with hostile input vs a same-length benign twin; raw `< > " '` counts must be equal. `assertNoRawXSS` static vectors now run on the JSON-stripped HTML portion only. The corpus entry that exposed the old checker's false positive (`9284ed3413ebcad7`) is kept as a regression seed.
7. **`scripts/verify.sh`** — 11-step hook+CI parity runner: go-version guard → doc claims → build → generate drift → `go mod tidy -diff` drift → vet → golangci-lint (config verify warn-only offline + run) → `go test -race` → coverage gate → fuzz seed corpus → per-target live fuzzing (`VERIFY_FUZZ_SECS`, 0=off). Executed end-to-end green.
8. **depguard allow-lists scrubbed** — templ/go-output/go-atomic-write/go-error-family/go-ndjson/go-sse removed from the example/main/tests rules (depguard is settings-only here; the lists were dead config that lied about deps).
9. **Dead API deleted**: `live.HealthInfo` (exported, unused on BOTH branch and master) removed from `live/server.go`.
10. **README fuzz claims synced** (8→10 in both places; verified CONTRIBUTING/FEATURES/STABILITY carry no fuzz-count claims — grep-verified, no other stale counts).
11. **TODO_LIST truthed**: backport item closed (race fix IS on master now), Dependabot item closed (4 alerts, all fast-uri, closed), master-red-CI item closed with full root-cause chain; version-skew ledger item updated (goconst nolint retired at root cause on master). One **wrong claim I had written was corrected** (see (d) #8).
12. **CI verified as truth**: workflow_dispatch on the branch → all 7 jobs green after push.

### Master work (done FROM a worktree, master checkout untouched)

13. **Master red CI root-caused to four causes, ALL daemon-auto-commit fallout on `59bc651`**:
    - (1) A formatter rewrote `.golangci.yml` 2-space→4-space; `scripts/check-go-version.sh` parsed `run.go` with a strict `^  go:` regex → returned empty → "could not read run.go" → Test job red. **Fixed** (`b6a04b8`): indent-tolerant, quoted-or-bare, flake pin accepts patch-level extension.
    - (2) Same reformat broke `scripts/check-doc-claims.sh`'s linter-count extraction (returned 0 vs README's claim). **Fixed** (`b6a04b8`, same commit): ported the branch's tolerant extraction; README count corrected 107→108 (the reformat had also changed the real count).
    - (3) Website: an automation bumped 5 `package.json` specifiers **without regenerating pnpm-lock.yaml** — including `typescript ^6.0.3 → ^7.0.2`, which AGENTS.md explicitly forbids (TS7/tsgo crashes `astro check`). CI's frozen-lockfile guard correctly failed. **Fixed** (`d092158`): TypeScript restored to ^6.0.3, other bumps kept, lockfile regenerated with Nix-provided node+pnpm (Bun-shimmed system pnpm lacks `node:sqlite`). Gates verified locally: build ✓, `astro check` 0 errors ✓, html-validate ✓. Bonus: range resolution pulled **fast-uri 3.1.7** → all four Dependabot alerts closed (API confirms 0 open).
    - (4) `live/fragments.go`'s `//nolint:goconst` had been stripped as "unused" by a local-newer linter (the exact version-skew landmine AGENTS.md predicted) → CI lint red. **Fixed** (`18603d5`) at root cause with `auditlog.ProviderType*` constants — no literals, no suppression, no landmine.
    - Lint/goreleaser reds in the same run were proxy transport flakes (documented pattern; resolved by the rerun triggered by these pushes).
14. **SSE snapshot/subscribe race backported to master** (`d092158`) — justified by reproduction first: master's `TestServer_SSE_LiveEventDelivery` **hung the full 10-minute timeout** in a local pre-commit run (event lost in the subscribe/snapshot gap; test waits forever). Reordered subscribe-before-replay/snapshot with a comment explaining the no-gap/no-duplication reasoning; verified `-race -count=5` + full live suite green.
15. **Master verified by CI**: after the three fix commits, run 33820635986 → all 8 jobs success (Test, Lint, vulncheck, mod-tidy, stale-generation, actionlint, goreleaser check, example-smoke). Website deploy green again.

### Process/infra

16. **Worktree hygiene**: master fixed via `/tmp/master-wt` worktree (branch checkout never touched); worktree removed at close. All work pushed; nothing unpushed on either line.

---

## b) PARTIALLY DONE

1. **AGENTS.md Commands table** — `scripts/verify.sh` exists and is green but is **not yet documented** in the table or the branch-contract bullet.
2. **`docs/proposal/api-diff.md`** — documents the EventStore→ReplayStore rename but NOT this session's API-surface deltas: `HealthInfo` deletion, `writeSSEEvent` field-injection hardening (internal, but behavior-relevant), 2 new fuzz targets, verify.sh. The proposal should present the full surface.
3. **Master AGENTS.md ledger** — the fragments.go goconst nolint entry in master's AGENTS.md now describes a nolint that NO LONGER EXISTS (fixed with constants at 18603d5). Branch TODO_LIST was corrected; master's own AGENTS.md was not touched.
4. **Master CHANGELOG** — master's fixes (guard regexes, website lockfile/TS, race backport, goconst) have no `[Unreleased]` entry on master; only the branch CHANGELOG documents the race fix. The sync script only compares release headings, so no red check — but the history is only in commit messages.
5. **Branch-contract bullet (AGENTS.md top)** — still says "Tests: stdlib SSE wire client…" etc. but does not mention the 10 fuzz targets, verify.sh, or the XSS-checker differential redesign in its verification-status line.
6. **Fuzz runtime in CI** — verify.sh runs 8-10s live fuzzing per target locally; CI runs seed corpus only (via `go test`). No scheduled long-budget fuzzing exists.
7. **Race-loop verification** — master's live fix verified with `-race -count=5`; the branch itself did not get a `-count=10` flake loop this session (did last session).
8. **verify.sh hook dedup** — `scripts/hooks/pre-commit` and verify.sh duplicate the same checks (incl. a copy-pasted `snapshot()` function); consolidation deferred.

---

## c) NOT STARTED (carried; unchanged this session by design)

1. **Send the merge package to samber** (`docs/proposal/`) — gated on owner question 2 (timing). Master being fully green + Dependabot-clean now satisfies the stated precondition.
2. **EventStore→ReplayStore rename decision** — owner question 1.
3. **Daemon-pause convention** — owner question 3.
4. Remaining backport candidates (classSuccess/classError constants, devShell `GOEXPERIMENT=""` hardening, `live/demo/` coverage-exclusion parity) — parked pending merge-direction decision.
5. noctx/`httptest.NewRequestWithContext` items — correctly dormant per the version-skew ledger (unfixable at Go 1.23, nolint would be unused under the CI pin).
6. Scheduled/long-budget fuzz workflow in CI; broadcaster concurrency-chaos test; live-fragment fuzz target (candidates in (f)).

---

## d) TOTALLY FUCKED UP (own failures, no self-deception)

1. **The daemon ate EVERY commit again — ~14 auto-commit messages, zero information.** I kept doing `git add …` and `git commit …` as separate steps with thinking in between; the daemon won every race, on BOTH lines. Worst case: the entire AGENTS.md truth pass, both fuzz targets, the XSS-checker redesign, verify.sh — all landed under "chore: auto-commit N changed file(s) (heuristic)". My commit messages described work that git history cannot attribute. I had one job (commit early, atomically) and executed it late, every time, despite this exact failure being documented in the previous session's report.
2. **Sloppy staging corrupted one master commit.** I chained `git add A && git commit A && git add B && git commit B`; the hook failed on commit A, the chain broke, B's website files stayed STAGED, and the next commit (`b6a04b8`, doc-claims fix) silently swept the staged `check-go-version.sh` changes in — a commit whose message didn't mention the guard fix. Had to amend the message to be truthful. Lesson: never leave staged state across a failed hook; stage and commit atomically per logical change.
3. **My own SSE fix introduced a duplicate declaration.** When moving the Subscribe block I edited in two steps and the second step re-added the block I had just moved — `eventCh` declared twice; caught by build. Then a missing blank line tripped wsl on master's lint. Two rounds to land a 10-line reorder that I had ALREADY reviewed on the branch in the previous session.
4. **My first XSS-checker fix was wrong and fuzz caught it immediately.** I replaced the false-positive-prone vector matching with a "input must not appear verbatim if it contains breakout chars" rule — a lone `"` seed killed it in seconds (a single quote appears verbatim in ANY HTML as markup). The correct design (differential markup-char counts) was the second attempt. Shipping a wrong invariant that I then had to replace is exactly the "sloppy test drafts cost extra cycles" failure from last session, repeated.
5. **The goconst fix on master first failed to build** — I wrote `ProviderTypeLazy` unqualified, forgetting `live` is a separate package from `auditlog`; fixed with sed to `auditlog.ProviderTypeLazy`. Basic package-boundary slip on a one-line change.
6. **I wrote an unverified claim into TODO_LIST** — "html_view CSS-class constants already on master (as classSuccess/classError)". Checked after the fact: master has html.templ, not html_view.go; the claim was FALSE. Corrected in TODO_LIST this session, but it should never have been written — this is the "trophy-case marking" failure the verify-external-claims skill exists to prevent, applied to my own memory notes.
7. **verify.sh lost three runs to environment gotchas I already knew**: (a) `golangci-lint config verify` needs a network fetch of the JSON schema — timed out; fixed warn-only; (b) the shared stale lint cache produced 5 phantom `nolintlint` findings that CI had already proven absent — I ran verify.sh WITHOUT the fresh `GOLANGCI_LINT_CACHE` the ledger prescribes, twice; (c) golines flagged my own long lines twice (a trailing nolint comment made the line long; then a long Fatalf). The script is green now, but the default behavior should have encoded these lessons from the start — see (e) #2.
8. **Repro-test churn during the XSS hunt**: my manual reproduction script had a broken index loop producing 29 garbage "hits" (only #1 was real), referenced a nonexistent helper causing a build failure, and was ultimately deleted rather than fixed — the fuzz corpus entry was the better artifact. Net: two wasted cycles producing a throwaway.
9. **CI dispatched only at the very end** — I pushed the branch's 12 daemon-commits and THEN ran CI. Had anything been red, the mess would have been maximal. CI-as-truth should be engaged after the first meaningful commit, not after the backlog.

---

## e) WHAT WE SHOULD IMPROVE

1. **Beat the daemon or turn it off**: single atomic `git commit -- fileA fileB -m …` IMMEDIATELY after each logical change (no separate `git add`); if the daemon still wins, `git commit --amend` afterward to restore the real message. Better: pause the daemon during focused sessions (owner question 3 — this is the third session it has destroyed commit history; the cost is now proven on master too, where the corrupted `.golangci.yml` came FROM the daemon).
2. **verify.sh should default `GOLANGCI_LINT_CACHE` to a fresh `/tmp` dir** — the stale-shared-cache gotcha is documented in the ledger but still bites anyone who runs the script naively (it bit me twice this session). Encode the lesson in the script, not the operator.
3. **Kill the incident class at root**: something (formatter? updater? the daemon's heuristic?) rewrote `.golangci.yml` and bumped website specifiers. Add CI prevention: `golangci-lint fmt --diff` (or a config-format stability check) in the lint job, and an explicit `pnpm install --frozen-lockfile` pre-check step in website.yml before build. Also identify the tool that did it (unresolved — see (f) #35/36).
4. **The two guards parse YAML with sed regexes.** They've now broken once each. Move to a sturdier check (goldens, or a tiny Go parser in `cmd/`) or at least a self-test that feeds both indent styles.
5. **Dispatch CI early and often** — after the first push of a session, not after the last.
6. **Fuzzing governance**: CI runs fuzz seeds only; live fuzzing is local-only with an 8-10s budget. A weekly scheduled workflow with 2-5 minute budgets per target would have caught the checker bug without local luck.
7. **Extend the differential XSS invariant to the live dashboard** — `FuzzPluginHTML` covers the static report; `live/fragments_html.go` renders the same hostile strings into SSE fragments and has no fuzz target.
8. **Proposal docs must track API deltas** — every API deletion/change on the branch (HealthInfo, ReplayStore, hardening) should be visible in `docs/proposal/api-diff.md` so samber reviews reality, not a stale diff.
9. **Stop writing memory claims without a source** — the classSuccess-on-master lie in TODO_LIST happened because I recorded a conclusion, not an observation. Rule: TODO_LIST entries carry commit SHAs or "unverified" tags.
10. **Hook/verify.sh dedup**: make the pre-commit hook call verify.sh in fast mode (or share a lib) so there is one definition of "the checks".

---

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

*Ordered roughly by impact; ★ = needs owner input or blocks on a decision.*

1. ★ **Send the merge package to samber** — precondition (master green + Dependabot-clean) is NOW satisfied.
2. ★ **Decide EventStore→ReplayStore** — keep rename or revert for parity before the proposal lands.
3. ★ **Daemon policy** — pause convention, amend-after-fire, or leave as-is (with history corruption accepted).
4. Add `scripts/verify.sh` to AGENTS.md Commands table + branch-contract bullet + CONTRIBUTING.md as the pre-push ritual.
5. Update `docs/proposal/api-diff.md`: HealthInfo deletion, writeSSEEvent hardening, 10 fuzz targets, verify.sh, and the 5-format table count.
6. Master AGENTS.md: retire the fragments.go goconst nolint ledger bullet (nolint no longer exists there).
7. Master CHANGELOG: `[Unreleased]` entry for the 2026-09-04 fixes (guard regexes, website lockfile/TS6, race backport, goconst constants).
8. Add `golangci-lint fmt --diff` (or equivalent format-stability check) to master+branch CI to prevent the reformat incident class.
9. Add a `pnpm install --frozen-lockfile` pre-check step to website.yml before build (fail fast with a clear message).
10. Identify the tool that rewrote `.golangci.yml` to 4-space (root cause of incident #1).
11. Identify the automation that bumped website/package.json specifiers incl. TypeScript ^7 (root cause of incident #2).
12. Re-design the guards' YAML parsing (goldens or tiny parser + self-test covering both indent styles).
13. verify.sh: default to a fresh `GOLANGCI_LINT_CACHE` under /tmp (configurable override).
14. Add a weekly scheduled fuzz workflow (2-5 min/target) to CI.
15. New fuzz target: `FuzzFragmentsHTML` — hostile service/error names through the live fragment renderer.
16. New test: broadcaster concurrency chaos (subscribe/unsubscribe/broadcast/shutdown under `-race` with goroutine leak check).
17. Verify/cover Last-Event-ID reconnection over HTTP end-to-end on the branch (ring is unit-tested; wire path should be too).
18. Heartbeat behavior test (interval elapsed → comment frame observed) if not already covered.
19. Backport candidates to master (post merge-direction decision): classSuccess/classError constants, devShell `GOEXPERIMENT=""` hardening, `live/demo/` coverage-exclusion parity.
20. Deduplicate pre-commit hook vs verify.sh (hook calls verify.sh in fast mode).
21. Replace `bytes.Count([]byte(s), …)` with `strings.Count` in `assertMarkupCountsEqual` (nit).
22. Quantify `t.Skip()` rate in FuzzPluginHTML (silent skips shrink effective fuzzing; count and report in fuzz output).
23. verify.sh: optional `--full` flag adding `nix flake check`, example-smoke, actionlint.
24. Consider `-count=N` race-loop option in verify.sh (`VERIFY_RACE_COUNT`, default 1) to institutionalize flake hunting.
25. STABILITY.md: mention 10 fuzz targets + verify.sh in the branch section.
26. Branch: prune/archive the DONE items in TODO_LIST into a collapsed section (file is getting long).
27. Consolidate the "daemon broke master CI" incident into a short post-mortem doc (three root causes + prevention), link from CHANGELOG.
28. Consider Dependabot config for the website: security-updates-only (prevents spec-bump-without-lockfile drift class).
29. Append verification artifacts to `docs/proposal/` (CI run links: branch 33822730255, master 33820635986, master race-fix verification commands).
30. Check `docs/proposal/merge-samber-do.md` for stale references (EventStore, master's live/ file names) and align with the branch truth pass.
31. Live dashboard: consider emitting a `retry:` hint to clients (wire supports it; server never sets it) — decide and document either way.
32. SSE: document/test behavior when a subscriber buffer overflows repeatedly (drop-on-overflow is implemented; is the client-visible signal (EventsOverflow) covered by a test?).
33. review `sseSplitLines`/`sseJoinLines`/`sseKeyedLines` for art-dupl overlap with the template helpers.
34. Add `scripts/verify.sh` invocation to the pre-push git hook (a second hook), so pushes get the same gate as commits.
35. Consider signing/annotating: none needed, but consider a `git tag` on the branch state that was shared with samber (once sent) for traceability.
36. Confirm in the GitHub UI that the 4 fast-uri alerts show as fixed (API says 0 open; visual confirmation pending).
37. Rerun `check-changelog-sync.sh` after any master CHANGELOG entry (heading-only compare, cheap).
38. Local hygiene: delete `/tmp/fresh-gcl-*` lint caches, `/tmp/xss_repro_test.go`, `/tmp/master-golangci.yml`.
39. Review whether `initialEventCap(64)`/ring defaults (1000) should be configurable on Server Config (ReplayBufferSize exists; initial buffer cap does not) — decide.
40. Docs: note in AGENTS.md that `config verify` needs network (warn-only locally, enforced in CI).
41. Evaluate moving the fuzz corpus `testdata/fuzz/` seeds into versioned seed tables if corpus files keep accumulating (keep-on-disk is idiomatic; only revisit if corpus bloats).
42. Cross-check FEATURES.md "Testing" section against the new fuzz/live coverage facts.
43. Consider exposing `Hub.ClientCount()`/health JSON schema in docs (live dashboard guide) if not already.
44. Sweep remaining `//nolint` directives on the branch with `make nolintlint-audit`-style check (are any needed under the CI pin? are any stale under 2.13?) — keep the ledger accurate.
45. Improve fuzz seeds for FuzzRingEventsAfter: include IDs with leading zeros / whitespace / unicode digits (partially seeded: Arabic-Indic digits present; add "007", " 7", "0x7").
46. Add the differential-counts idea to `FuzzMigrateReport`-adjacent HTML paths if any user data reaches HTML there (it shouldn't — JSON only; verify and document).
47. Master: once the merge direction is decided, decide whether master gets the stdlib ports or stays 1.26 (the two lines' divergence is growing; the proposal should state the endgame).
48. Consider a top-level "VERIFICATION.md" or CONTRIBUTING section that lists every gate (hook, verify.sh, CI jobs, coverage, fuzz budgets) in one table.
49. `git worktree add`-based master-fix workflow worked well — document it in AGENTS.md as the standard way to touch master without disturbing a branch checkout.
50. Schedule a short session to re-verify the whole matrix on a cold machine (fresh nix store eval, cold lint cache) — the "fresh cache" classes of failure only show up cold.

---

## g) THREE QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Daemon policy (third time asking, cost now proven on master)**: The auto-commit daemon corrupted master twice this incident chain (the `.golangci.yml` reformat and the website spec bumps that took master down for ~30h both came from daemon commits). May I pause/disable it during focused sessions — and if so, HOW is it running (launchd/systemd/tmux + script path)? If it must keep running, do you accept `git commit --amend` after each fire to restore real messages, and do you want me to hunt down what tool reformatted `.golangci.yml` and bumped website deps?
2. **Merge proposal, now?** Master is green (8/8), the website deploys, Dependabot is clean — your stated preconditions are met. Send `docs/proposal/` to samber now, or first squash the go1.23-compat history (12 daemon-commits of noise on top of the real work) into reviewable logical commits so samber sees a clean branch? And should the proposal repo-state be a tag for traceability?
3. **EventStore→ReplayStore rename**: keep the branch's rename (concrete type, honest name) or revert to master's `EventStore()` for minimal API delta in the merge? The api-diff currently presents it as a rename-only change; samber may have an opinion, but I need your default position.

---

*Report ends. Waiting for instructions.*

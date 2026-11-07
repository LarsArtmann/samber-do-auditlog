# Status Report — templ API Skew Fix & Concurrent Commit Collision

**Date:** 2026-11-04 02:23 CET
**Session scope:** Diagnosis and repair of the BuildFlow `go-mod-update` / `govalid-generate` failures (templ `ResolveAttributeValue` version skew), golden-fixture update, and mid-session collision with a concurrent session's commit `a6111f3`.
**Files touched this session:** `testdata/golden/report.html` (committed via `a6111f3`), `html_templ.go` + `live/fragments_templ.go` (regenerated, still uncommitted).

## Scope Line

**Verified this session (fresh commands, reproducible):**

- Root cause: `grep -rn "func ResolveAttributeValue"` in the module cache — **present** in `templ@v0.3.1070`, **absent** in `templ@v0.3.1100`. The committed `live/fragments_templ.go` (HEAD before `a6111f3`) contained **11** call sites; after my regeneration: **0** (`grep -c`).
- `GOEXPERIMENT=jsonv2 go build ./...` → exit 0 (verified twice: pre-`a6111f3` working tree, and after my final regeneration).
- `GOEXPERIMENT=jsonv2 go test ./live/` → `ok 0.115s`; full `go test ./...` after final regen → **no FAIL lines** (plain mode, no `-race`).
- `UPDATE_GOLDEN=1 go test -run TestReport_WriteHTML_GoldenFile ./` → ok; diff inspected: the only golden change is go-output **v0.38.7**'s new DAG edge-label tooltip JS — verified as an upstream feature (`edgeTitle` present in `go-output/daghtml@v0.38.7/graph.js`, absent in `daghtml@v0.38.3`; upstream ships its own `TestGolden_Render_LabelledEdges` golden).
- `go tool templ generate` (pinned v0.3.1100 via go.mod `tool` directive) → `updates=2`.
- `git show a6111f3 --stat` → `go.mod` 34±, `html_templ.go` **2±** (header only), `live/fragments_templ.go` **2±** (header only — content NOT regenerated), `testdata/golden/report.html` 12±.
- `which templ` → `/run/current-system/sw/bin/templ` **v0.3.1020** (system-wide, older than the repo pin).
- `git check-ignore -v live/fragments_templ.go` → **no ignore rule** (file is tracked); `git log -- live/fragments_templ.templ` → **never committed; the source file does not exist locally** (only the generated `.go` remains).
- Templ watcher evidence: `(✓) Post-generation event received … needsRestart=true` lines appeared during my `go generate` — an external watcher/LSP is connected and regenerated files with the old CLI at 02:15:44 (all key files mass-touched within the same second).

**NOT verified this session (and why):**

- `-race`, coverage gate (≥94%), `go vet` — not re-run after the final regen (out of "explain/fixed?" scope; plain suite only).
- CI status of `a6111f3` and whether it was **pushed to origin** — not checked (remote inspection not attempted).
- Re-running the two failed BuildFlow steps (`go-mod-update`, `govalid-generate`) — not done; only the root cause was fixed.
- The "9 tools unavailable (health check failed)" from the original BuildFlow run — never diagnosed.
- The identity/persistence of the templ watcher process — indirect evidence only.
- Nix build / vendorHash implications of the go.mod churn — not attempted.
- Whether the concurrent session is still active — unknowable from inside this session.

---

## a) FULLY DONE

1. **Root-cause diagnosis of the BuildFlow failures.** `go-mod-update` bumped templ past the release that removed `templ.ResolveAttributeValue` (removed in v0.3.1100, existed in v0.3.1070); the stale generated file still called it 11×. The step's revert-to-previous-state behavior worked as designed; `govalid-generate` failed as a compile-blocked dependent. Evidence: module-cache greps above; BuildFlow's own error text matches.
2. **Build + full test suite restored green.** `go tool templ generate` with the pinned v0.3.1100 regenerated both generated files (11 → 0 `ResolveAttributeValue` occurrences); `go build ./...` exit 0, `go test ./...` all-ok. Evidence: commands listed in scope line.
3. **Golden fixture drift verified as legitimate, then fixed.** `UPDATE_GOLDEN=1` updated `testdata/golden/report.html`; the diff is exclusively go-output v0.38.7's edge-label SVG tooltip feature (checked upstream source, not guessed). The committed golden now matches the pinned-toolchain output — golden test passes without overrides.
4. **HEAD `a6111f3` identified as broken.** The concurrent commit bumped templ to v0.3.1100 but updated the generated files' **headers only** (2-line diffs), leaving stale content → HEAD does not compile. Evidence: `git show a6111f3 --stat` + current build failure against HEAD state before my regen.
5. **Version-skew map established.** System PATH templ = v0.3.1020 (emits the removed API) vs. repo pin = v0.3.1100 (removed it) — the standing trap that re-broke the tree at 02:15:44 and will do so again on any editor-triggered regen.

## b) PARTIALLY DONE

1. **The fix itself.** Working tree is green, but `html_templ.go` + `live/fragments_templ.go` are **uncommitted** → HEAD (`a6111f3`) still does not compile until they land. Remaining: a deliberate commit (daemon pickup is possible but has a known blind spot). Effort: S.
2. **Post-fix verification.** Plain `go test` only. Remaining for CI-equivalence: `-race`, coverage gate, `go vet`, `go generate ./...` zero-diff check. Effort: S.

3. **Re-break diagnosis.** Established that _something_ regenerated files with templ v0.3.1020 at 02:15:44 and that a watcher is connected, but the owning process (editor LSP? editor extension version?) was never identified. Until it is, regeneration races can recur at any time. Effort: M.

4. **Original BuildFlow failures.** Root cause fixed; the two failed steps were never re-run to confirm green end-to-end (`buildflow -s go-mod-update -v`, `-s govalid-generate -v`). Effort: M.

## c) NOT STARTED

1. **CHANGELOG entry** for the templ v0.3.1100 bump, go-output v0.38.7 edge-label tooltips, and the golden update. (Blocked only on the generated-file commit landing.)
2. **AGENTS.md gotcha entry** documenting the templ API removal + "regenerate only via `go tool templ generate`" rule + the editor-LSP trap.
3. **FEATURES.md HTML-report inventory update** — edge labels now render as native SVG tooltips (user-visible report change).
4. **Push-state audit** — was `a6111f3` pushed? If yes, remote master is broken for consumers until the fix lands.
5. **Nix build / vendorHash check** after go.mod churn.
6. **System templ fleet decision** — bump nixpkgs templ to ≥v0.3.1100 or enforce go-tool-only generation (policy call).
7. **"9 tools unavailable" diagnosis** (`buildflow doctor`).
8. **go-output v0.38.3→0.38.7 full changelog review** (beyond edge labels) and **go-error-family v0.11→0.12 review** — bumps landed in `a6111f3` unreviewed by this session.
9. **On-demand BuildFlow tools** (`gitleaks`, `codespell`, `markdown-lint`) — skipped by config, never run this session.
10. **TODO_LIST harvest** of this report's section (f) (docs-health HARVEST).

## d) TOTALLY FUCKED UP

1. **HEAD `a6111f3` does not compile.** Severity: blocks every consumer of master and all CI jobs (stale-generation job will fail on diff; build jobs fail on `undefined: templ.ResolveAttributeValue`). Root cause: templ bumped to v0.3.1100 (API removed) while generated content stayed at v0.3.1020/1070-era output. Mitigation: my regenerated files are on disk and green — but uncommitted. Anyone cloning/checkout of `a6111f3` gets a broken build.
2. **Two writers are fighting over generated files.** My pinned-CLI regeneration vs. an editor-side templ v0.3.1020 regen (02:15:44 mass-touch) — the same flip-flop can recur arbitrarily. Root cause: no single source of truth for generation; the editor's binary version differs from the repo pin. Mitigation: none currently; needs the system-templ bump or LSP config fix.
3. **A commit landed mid-session with an unverified tree.** `a6111f3` was authored with a detailed message but its generated-file changes were header-only — the authoring session did not (or could not) run the build gate. The existing `stale-generation` CI job _would_ catch this post-push; nothing caught it pre-commit.
4. **My own premature "fixed?" claim.** After the first repair I answered "fixed" on the strength of `go build` + `go test ./live/` only; the very next full-suite run exposed the golden FAIL. Caught and corrected within one exchange, but the claim was wrong when made. Lesson recorded in (e).

## e) WHAT WE SHOULD IMPROVE

1. **Declare "fixed" only after the full CI-equivalent gate.** Plain build + one package is not green. Impact: avoids whiplash claims like this session's. Fix: standard checklist (build, full test, `-race` where CI uses it, vet) before any "done".
2. **Treat concurrent-session signals as first-class.** Daemon commits, watcher log lines, and mass mtime changes all appeared before the collision; I read them only after the fact. Fix: `git log --oneline -1` + `stat` on generated files before/after any write in this repo.
3. **Single source of truth for templ generation.** CI and `html.go`'s directive use `go tool templ generate`, but the editor LSP uses system templ v0.3.1020. Fix: bump system templ fleet-wide (nixpkgs) and/or configure the editor to match the repo pin; document the rule in AGENTS.md.
4. **The stale `go.mod` retraction comment.** It says `live/fragments_templ.go` is "gitignored" — `git check-ignore` proves it is tracked. Stale claims in canonical files mislead the next debugger. Fix: one-line comment update (pair with the CHANGELOG entry).
5. **Gate before commit, not after.** `a6111f3` shipped an unbuildable tree with a polished message. Fix: the pre-commit hook runs BuildFlow pre-commit mode — verify `core.hooksPath` is installed and that generated-file commits include a build check (`git config core.hooksPath` re-install is already a documented rot risk).

## f) 50 Things To Get Done Next

Ranked by impact. Effort: S <30min, M 30min–2h, L >2h. Category: Bug / Quality / Documentation / Cleanup / Feature.

| #  | Task                                                                                                                                                                                              | Impact   | Effort | Category      |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| 1  | Commit the regenerated `html_templ.go` + `live/fragments_templ.go` so HEAD compiles again                                                                                                         | Critical | S      | Bug           |
| 2  | Check whether `a6111f3` was pushed to origin; if yes, watch the CI `stale-generation`/build jobs and follow up immediately                                                                        | Critical | S      | Bug           |
| 3  | Run CI-equivalent verification on the working tree: `GOEXPERIMENT=jsonv2 go test -race ./...`                                                                                                     | Critical | S      | Quality       |
| 4  | `GOEXPERIMENT=jsonv2 go vet ./...`                                                                                                                                                                | High     | S      | Quality       |
| 5  | `sh scripts/coverage-gate.sh` (≥94% gate after regen)                                                                                                                                             | High     | S      | Quality       |
| 6  | `GOEXPERIMENT=jsonv2 go generate ./...` → assert zero diff (stale-generation CI equivalence, would have caught `a6111f3`)                                                                         | High     | S      | Quality       |
| 7  | Re-run `buildflow -s go-mod-update -v` to confirm the originally failing step is now green                                                                                                        | High     | M      | Quality       |
| 8  | Re-run `buildflow -s govalid-generate -v`                                                                                                                                                         | High     | M      | Quality       |
| 9  | Diagnose "9 tools unavailable (health check failed)" via `buildflow doctor`                                                                                                                       | High     | M      | Quality       |
| 10 | Identify the templ watcher/LSP regenerating with v0.3.1020 (editor? which binary?) and neutralize or upgrade it                                                                                   | High     | M      | Bug           |
| 11 | Decide and execute system templ bump (nixpkgs → ≥v0.3.1100) or fleet policy "go tool templ only"                                                                                                  | High     | M      | Cleanup       |
| 12 | Add CHANGELOG entry: templ v0.3.1100, go-output v0.38.7 edge-label tooltips, golden regeneration                                                                                                  | High     | S      | Documentation |
| 13 | Add AGENTS.md gotcha: `ResolveAttributeValue` removal + regen-via-`go tool`-only rule + editor-LSP trap                                                                                           | High     | S      | Documentation |
| 14 | Update FEATURES.md HTML-report inventory: edge labels render as native SVG tooltips                                                                                                               | Medium   | S      | Documentation |
| 15 | Fix stale go.mod retraction comment ("gitignored generated file" — the file is tracked now)                                                                                                       | Medium   | S      | Documentation |
| 16 | Review go-output v0.38.3→v0.38.7 upstream changelog for behavior changes beyond edge labels                                                                                                       | Medium   | M      | Quality       |
| 17 | Review go-error-family v0.11→v0.12 bump notes (landed unreviewed in `a6111f3`)                                                                                                                    | Medium   | S      | Quality       |
| 18 | Verify all four diagram renderers (Mermaid/DOT/D2/PlantUML) unaffected by the go-output bump                                                                                                      | Medium   | M      | Quality       |
| 19 | `nix build` / vendorHash check after go.mod churn                                                                                                                                                 | Medium   | M      | Quality       |
| 20 | Run on-demand tools: `buildflow -s gitleaks`, `-s codespell`, `-s markdown-lint`                                                                                                                  | Medium   | S      | Quality       |
| 21 | `sh scripts/check-doc-claims.sh` after the doc updates (12/12 checks)                                                                                                                             | Medium   | S      | Quality       |
| 22 | `sh scripts/check-changelog-sync.sh` after the CHANGELOG entry                                                                                                                                    | Medium   | S      | Quality       |
| 23 | `sh scripts/check-go-version.sh` drift guard after go.mod churn                                                                                                                                   | Medium   | S      | Quality       |
| 24 | Example smoke: `DO_AUDITLOG_ENABLED=true go run ./example` end-to-end after dep bumps                                                                                                             | Medium   | S      | Quality       |
| 25 | Audit why `a6111f3` updated generated-file headers but not content (which CLI did that session run?)                                                                                              | Medium   | S      | Bug           |
| 26 | `go mod tidy` zero-drift check (CI mod-tidy job equivalence)                                                                                                                                      | Medium   | S      | Quality       |
| 27 | Race-test the `live/` SSE/dashboard package specifically (transport-heavy)                                                                                                                        | Medium   | S      | Quality       |
| 28 | Verify all 8 fuzz targets still pass after dep bumps                                                                                                                                              | Medium   | M      | Quality       |
| 29 | Decide whether an extra CI guard is needed beyond `stale-generation` (build right after generate) or whether #6 proves the existing job suffices                                                  | Medium   | S      | Quality       |
| 30 | Document the UPDATE_GOLDEN workflow for templ/go-output bumps (when golden drift is legitimate vs. a regression)                                                                                  | Medium   | S      | Documentation |
| 31 | Verify next-release implications: tag must include regenerated files (v0.9.0-retraction class)                                                                                                    | Medium   | S      | Cleanup       |
| 32 | Check `git config core.hooksPath scripts/hooks` is still installed (documented rot risk)                                                                                                          | Low      | S      | Cleanup       |
| 33 | Harvest this report's section (f) into TODO_LIST.md / ROADMAP.md (docs-health HARVEST)                                                                                                            | High     | S      | Documentation |
| 34 | Annotate the two re-formatted 2026-10-26 status reports (docs-health ANNOTATE, non-destructive)                                                                                                   | Low      | M      | Documentation |
| 35 | Verify `.prettierignore` still covers `testdata/` (golden reformat protection after the golden change)                                                                                            | Low      | S      | Cleanup       |
| 36 | Investigate the 02:15:44 mass-mtime event (why source `.templ` files were touched too)                                                                                                            | Low      | M      | Bug           |
| 37 | Sweep templ v0.3.1100 release notes for other API removals that could bite future regens                                                                                                          | Low      | M      | Quality       |
| 38 | Decide whether an "Unreleased" CHANGELOG section should open for the dep bumps                                                                                                                    | Medium   | S      | Documentation |
| 39 | Confirm the `tool` directive + version comment in go.mod are current                                                                                                                              | Low      | S      | Cleanup       |
| 40 | Evaluate a pre-commit guard that blocks committing generated files that fail `go build`                                                                                                           | Medium   | M      | Quality       |
| 41 | Verify `schema/report.schema.json` committed content matches `cmd/genschema` output (regen-identical check)                                                                                       | Low      | S      | Quality       |
| 42 | Review remaining `a6111f3` items for follow-ups (package.json reorder, pnpm/action-setup v6.1.1 bump verification)                                                                                | Low      | S      | Cleanup       |
| 43 | Confirm website changelog.mdx needs no entry yet (website deploys only on `website/**` pushes)                                                                                                    | Low      | S      | Documentation |
| 44 | Decide on a version-skew ledger entry in AGENTS.md if the templ-version flap recurs                                                                                                               | Low      | S      | Documentation |
| 45 | Add editor guidance (docs note): "regenerate only via `go tool templ generate`; system templ may lag"                                                                                             | Low      | S      | Documentation |
| 46 | Verify golden test passes in a clean clone (catches local-artifact-only greens)                                                                                                                   | Medium   | M      | Quality       |
| 47 | Check whether `forward/` go-auto-upgrade advisory warnings changed after the go-error-family bump                                                                                                 | Low      | M      | Quality       |
| 48 | Reconcile `BENCHMARKS.md` (12 benchmarks) against claims-linter counts after its reformatting                                                                                                     | Low      | S      | Quality       |
| 49 | ~~Record the concurrent-session contention protocol as a candidate lesson (crush-config `references/lessons.md`)~~ done 2026-11-07: recorded as a committed lesson in the crush-config repository | Low      | S      | Documentation |
| 50 | Consider retracting/re-tagging if a release was cut from the broken `a6111f3` tree (release-state audit)                                                                                          | Medium   | S      | Cleanup       |

**HARVEST note:** items 33/12/13/14/15/30 are immediate docs work; items 1–11 are the critical path. Everything else is harvestable into TODO_LIST (specific) vs ROADMAP (policy/audit items: 11, 29, 40, 46, 50).

## g) Questions I Cannot Answer Myself

1. **Is the concurrent session that authored `a6111f3` (formatter convergence / release gate work) still active in this repo?** I tried: process scan (`ps` — no buildflow/templ/pma processes visible), mtime forensics, commit timing — but I cannot see other agents' sessions or their intent. The answer decides whether I commit the regenerated files now or risk colliding again.
2. **What is the fleet policy for templ: bump the system templ (nixpkgs v0.3.1020) to ≥v0.3.1100, or freeze system templ and enforce `go tool templ generate` exclusively (including editor LSP config)?** I tried: read AGENTS.md, checked the devShell tooling notes — no documented policy for editor-side generator binaries exists. This decides whether the 02:15:44-class re-break can ever be fully closed.
3. **How should the regenerated files land: an explicit commit by me (with CHANGELOG entry in the same commit) now, or leave them for the auto-commit daemon / the other session?** I tried: the harness forbids commits without your explicit instruction, and the daemon has a documented blind spot — so this is genuinely your call.

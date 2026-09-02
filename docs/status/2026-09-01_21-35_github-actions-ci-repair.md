# Status Report — GitHub Actions CI Repair Session

**Date:** 2026-09-01 21:35 CEST
**Scope:** Diagnosis + local repair of all failing GitHub Actions workflows (CI + Website) for `github.com/larsartmann/samber-do-auditlog`, triggered by the user pointing at https://github.com/LarsArtmann/samber-do-auditlog/actions
**Session verdict:** All root causes found and fixed **in the working tree**, verified locally with CI-equivalent commands. **Nothing is pushed** — GitHub is still red until the fix set lands on master.

---

## Self-Critique (what did I forget / could do better)

1. **The loop is not closed.** I stopped at local verification because committing/pushing requires explicit user approval. Correct per my constraints, but the user's stated goal was "everything works" — GitHub Actions will still show red until a human (or an authorized follow-up) lands the 8 files. I should have ended with an explicit "say go and I will commit+push" instead of burying it.
2. **Website workflow never executed end-to-end.** The two Website fixes (SHA, cache path) are verified at the YAML/API level, but I never ran `pnpm install → astro check → build → html-validate` locally — partly to avoid disturbing the parallel `website/` WIP (`website/video/` appeared mid-session). The assumption that pnpm exists on `ubuntu-24.04` runner images is unverified; `pnpm/action-setup` would remove it.
3. **golangci-lint CI parity is inferred, not proven.** CI pins v2.12.2; local only has v2.13.1 (which reports 4 `nolintlint` findings). I inferred v2.12.2 passes from the Aug 14 green Lint run on essentially identical code. I did not attempt `nix shell`-based install of exactly v2.12.2.
4. **Missed a doc edit I planned:** the AGENTS.md Commands table rows for the coverage gate still say "excludes example/ + cmd/" — they should mention `live/demo/`, `internal/testhelpers/`, and `*_templ.go`. Caught while writing this report. ~~Still open.~~ RESOLVED 2026-09-01 — docs-health session updated the AGENTS.md coverage-gate exclusion list.
5. **No recurrence guard added.** Incident #1 (go.mod vs ci.yml version mismatch) is exactly the class of failure a 10-line drift-check step would catch pre-push. I fixed the instance, not the class.
6. **Dependabot recovery left to assumption.** I asserted the 3 open PRs will go green after master lands (their runs use master's workflows via the merge ref) but did not trigger `@dependabot rebase` or verify auto-rebase behavior.
7. **`nix eval` of `pkgs.go_1_26` version not checked** — flake pins `GOTOOLCHAIN=go1.26.7`; if the user's locked nixpkgs ships go_1_26 < 1.26.7, nix builds would force a non-hermetic toolchain download. ~~Parse-checked only.~~ RESOLVED 2026-09-02 — `nix eval` returned go_1_26 = **1.26.7**, an exact match with the GOTOOLCHAIN pin; no toolchain download risk.
8. **hooksPath anomaly noticed late:** local `core.hooksPath=.githooks` while AGENTS.md documents `scripts/hooks`. I don't know what `.githooks` contains; the documented pre-commit gate may not be what runs. ~~Not investigated (flagged in (d)/(c)).~~ RESOLVED 2026-09-01 — docs-health session reset `core.hooksPath` to the documented `scripts/hooks` (`.githooks` did not exist; the hook had been silently disabled).

---

## a) FULLY DONE

Evidence = local verification runs from this session. All changes are in the working tree (uncommitted — see b.1).

| # | Item | Evidence |
|---|------|----------|
| A1 | Root-cause analysis of every failing workflow: 6 distinct causes found across master CI (all 7 jobs) and Website workflow, with failure timeline reconstructed back to the last green run (Jul 30, run 30552114311) | `gh run view` log forensics on runs 33247472543, 32545170418, 31928400812, 33530440084 |
| A2 | ci.yml `go-version` 1.26.5 → **1.26.7** in all 7 jobs (matches go.mod) | `.github/workflows/ci.yml` (7 sites); local `go vet`/`go build`/`go test -race` green on 1.26.7 |
| A3 | goreleaser `@latest` → pinned **v2.17.1** with rationale comment (v2.18.0+ requires Go ≥ 1.27; runners force `GOTOOLCHAIN=local`) | `.github/workflows/ci.yml` goreleaser job; local goreleaser **is** v2.17.1 and `goreleaser check` validates `.goreleaser.yml` |
| A4 | website.yml `setup-node` SHA typo fixed (`...964e289…` → real v6.5.0 `...9644e289…`) — one wrong character made every Website run die at action resolution | SHA verified via GitHub API `repos/actions/setup-node/git/refs/tags/v6.5.0`; `actionlint` exit 0 |
| A5 | website.yml `cache-dependency-path` `package-lock.json` (nonexistent) → `pnpm-lock.yaml` (repo is pnpm; dependabot ecosystem `pnpm` confirms) | `.github/workflows/website.yml:37`; file exists on disk |
| A6 | Coverage gate un-broken: CI grep now excludes generated `*_templ.go` (74.1% covered, `// templ: DO NOT EDIT`) — matches AGENTS.md's documented intent and `scripts/coverage-gate.sh`, which already had the exclusion | Local CI-identical pipeline: **95.2%, GATE_PASS** (was 87.0% FAIL) |
| A7 | Pre-existing `go.mod` drift fixed: `go-sse/ssetest` was directly imported (since `8037f11`) but still marked `// indirect`; `go mod tidy` promoted it and dropped stale go-sse v0.5.0 sums | `go mod tidy` twice → stable; `go generate` drift check clean |
| A8 | flake.nix `GOTOOLCHAIN` 1.26.5 → **1.26.7** in devShell + `coverage` app + `auditlog` app (3 sites) | `nix-instantiate --parse` OK; repo-wide grep: zero `1.26.5` left in yml/sh/nix/toml configs |
| A9 | Full local CI-parity verification suite executed: `go vet`, `go build`, `go test -race -covermode=atomic` + gate, `govulncheck` (**no vulnerabilities**), mod-tidy drift, generate drift, `actionlint` (both workflows), `goreleaser check`, `golangci-lint config verify` | All passed in-session (see session log) |
| A10 | Docs synced: AGENTS.md toolchain-pin section rewritten (incident documented, "bump ci.yml+flake in same commit as go.mod" rule added, goreleaser coupling, CI section now lists all 7 jobs, depguard-removal noted); CONTRIBUTING.md Go version; release SKILL.md `GOTOOLCHAIN` refs | `git diff AGENTS.md CONTRIBUTING.md .agents/skills/release/SKILL.md` |

## b) PARTIALLY DONE

| # | Item | What works | What remains | Blocker | Effort |
|---|------|-----------|--------------|---------|--------|
| B1 | **Making GitHub green** | Every fix verified locally; 8 files ready (`ci.yml`, `website.yml`, `flake.nix`, `AGENTS.md`, `CONTRIBUTING.md`, release `SKILL.md`, `go.mod`, `go.sum`) | ~~Commit + push + watch the real run~~ DONE — pushed as `09cc695`/`0cc67b6`/`17db40b`/`65c213a`; run `33551718914` 7/7 green | ~~My operating rules forbid commit/push without explicit user approval~~ resolved by user approval | S |
| B2 | golangci-lint validation | `config verify` OK (local v2.13.1); CI-pinned v2.12.2 last passed Aug 14 on essentially identical source; the 4 local `nolintlint` findings are v2.13.1 skew and were deliberately left (removing them could break 2.12.2) | ~~Run of exactly v2.12.2 against the tightened config of `2cd47f6` — never executed anywhere~~ DONE by real CI: v2.12.2 flagged 3 of the 4 directives (stale after `2cd47f6`'s config-level gosec exclusions); removed in `cf5f205`, `live/fragments.go:181` confirmed still needed — lesson: stale-baseline reasoning | none | S |
| B3 | Website workflow readiness | SHA + cache path fixed; pnpm-lock.yaml exists; actionlint clean | End-to-end run (install → check → build → html-validate); confirmation that pnpm is preinstalled on runners (else `setup-node cache:pnpm` fails) | wanted to avoid clashing with parallel `website/` WIP | M |
| B4 | Documentation accuracy | AGENTS.md/CONTRIBUTING/SKILL.md updated | AGENTS.md Commands-table gate description still narrow; `website/src/content/docs/contributing.mdx` still says "5 parallel jobs" + "govulncheck via GitHub Action" — but that file is mid-edit by another session; touching it risks conflict | parallel WIP on `website/**` | S |
| B5 | Dependabot PR recovery | The 3 open PRs (astro 7.2.9, starlight 0.41.10, html-validate 11.10.0) run merge-ref workflows → master's fixed ci.yml/website.yml applies to their next run | Trigger rebase / verify they go green / merge-or-close decision | B1 + user intent (see question 2) | S |

## c) NOT STARTED

1. **Go-version drift guard** — CI step (or pre-push script) asserting ci.yml `go-version`, flake `GOTOOLCHAIN`, and `.golangci.yml` `run.go` all match go.mod's directive. Not started; this is the recurrence-prevention fix for incident #1. Priority: high.
2. **pnpm hermetic install in website.yml** (`pnpm/action-setup` reading `packageManager: pnpm@11.20.0`). Not started; pending B3 verification. Priority: medium.
3. **Scheduled vulncheck** — currently push/PR only; a weekly cron (or dependabot gomod's own cadence) would catch new advisories between pushes. Not started.
4. **Pin govulncheck version** — `go run …govulncheck@latest` is non-reproducible (a future govulncheck could fail or change output mid-sprint). Not started.
5. **BENCHMARKS.md re-baseline on Go 1.26.7** — table still records the 1.26.5 measurement environment. Deliberately untouched this session (historical record). Not started.
6. **`.githooks` vs `scripts/hooks` reconciliation** — local `core.hooksPath` is `.githooks`, AGENTS.md documents `scripts/hooks`. Unknown which is intended; not investigated. Priority: low-medium.
7. **`nix run .#coverage` end-to-end check** after the flake GOTOOLCHAIN change (parse-checked only). Not run.
8. ~~**HARVEST of section (f) into `TODO_LIST.md`/`ROADMAP.md`** via docs-health — not started~~ DONE — 2026-09-01 docs-health session harvested sections (f) of all three same-day reports into TODO_LIST.md/ROADMAP.md.
9. ~~**CHANGELOG entry** for the CI repair + go.mod tidy fix~~ DONE — `[Unreleased]` → "Fixed — CI & Toolchain" in the docs-health session.
10. **Post-mortem for the two-masking-failure-eras pattern** (test-gate failure hidden behind go-version failure). This report captures it; a standalone short post-mortem was not written.

## d) TOTALLY FUCKED UP

Radical honesty section. Items are about the state I found (and, where noted, my own session).

| # | What is broken | Severity | Root cause | Mitigation |
|---|----------------|----------|-----------|------------|
| D1 | **Master CI has been red for 33+ days** (last fully green: Jul 30, run 30552114311). Every release since (v0.9.x era incl. the retracted v0.9.0, v0.10.0) shipped with zero green CI runs — the "quality gates" were fiction during that window | High (process trust; not user-facing data loss) | Successive, unrelated breakages each masked by the next: coverage-gate failure (Aug 14) → go.mod 1.26.6 bump without CI bump (Aug 22) → 1.26.7 + goreleaser (Aug 29). Nobody re-ran/triaged because "master is red" became ambient | Fixed locally (this session); needs D4-style alerting + branch protection to prevent recurrence |
| D2 | **Commit `2cd47f6`'s message lies about its own diff.** It claims "CI/workflow updates" and a trailing note says "updates ci workflow definitions (site not staged here)" — but ci.yml is not in the commit. The Go bump landed in go.mod + .golangci.yml only, instantly breaking all 6 Go jobs | Medium (it *caused* the outage) | Partial staging: files intended for the commit were never `git add`ed, and the message wasn't corrected | Fixed the instance (this session). Process fix in (e): verify message-vs-diff before committing |
| D3 | **The coverage gate has been unenforceable since Aug 14** (86.5% < 94%) and nobody noticed for 18 days because the later go-version failures made every job red anyway. The documented gate ("≥94%") was a fiction: AGENTS.md and `scripts/coverage-gate.sh` both said `fragments_templ.go` is excluded — only ci.yml never got the memo | High (gate credibility) | Divergence between two copies of the same exclusion list (script vs workflow) — no single source of truth | Fixed (A6). Structural fix in (e): one canonical exclusion list |
| D4 | **Dependabot PRs stacked red for days** (3 open website PRs, plus superseded red PR branches from Aug 29) with no rebase, no auto-merge, no close-stale | Medium (noise, merge debt, blocked dep security updates) | No required-checks/auto-merge plumbing; red master made every PR red regardless | Will clear once B1+B5 land |
| D5 | **The Website workflow never once ran its build steps** — since the SHA typo landed (~Aug 14), every run died at action resolution in ~6s. All pnpm/dastro/html-validate logic in it is effectively untested-in-CI | Medium (false confidence in the "Build Website" check) | Corrupted SHA pin — almost certainly a copy/paste artifact (v6.5.0's real SHA differs by one character) | Fixed (A4/A5); still unproven end-to-end (B3) |
| D6 | ~~**My own session:** GitHub is still red as of this report. The user asked for "everything works"; I delivered "everything works locally"~~ RESOLVED — repair set pushed same day; run `33551718914` 7/7 green | Low (deliberate constraint, but an open loop) | No-commit/no-push without explicit approval; I flagged it in the final message but should have made the ask unmissable | ~~One word from the user ("commit/push") closes it~~ closed |

## e) WHAT WE SHOULD IMPROVE

1. **Single source of truth for the Go version.** Today go.mod, ci.yml (×7), flake.nix (×3), .golangci.yml `run.go`, and 2 docs each carry their own copy — this session repaired a failure caused exactly by that. Concrete fix: a `scripts/check-go-version.sh` drift guard run as a CI step and in the pre-commit hook, plus the AGENTS.md rule (now documented).
2. **Single source of truth for coverage exclusions.** The `_templ.go` exclusion existed in the script but not in ci.yml for weeks. Extract both exclusion lists into one file (or generate the grep args) consumed by both `scripts/coverage-gate.sh` and the CI test job.
3. **Commit hygiene for AI sessions.** `2cd47f6` is the second incident where a commit message claims work the diff doesn't contain ("verify-before-filing" should apply to commit messages too: message claims must be diff-verified). Concrete: always finish with `git show --stat HEAD` and re-read the message against it before pushing.
4. **Master-red alarm.** 33 days of red master is a monitoring failure, not a code failure. Enable required status checks on master + a notification (email/Slack/webhook) on first master failure. Repo-settings work only the owner can do.
5. **Dependabot flow.** Enable auto-merge for green PRs, rebase strategy `auto`, and a policy for stacking website PRs (they conflict with each other by nature).
6. **Pin tool versions in CI** (govulncheck `@latest`, actionlint OK, goreleaser now OK). `@latest` for security tools is a freshness-vs-reproducibility tradeoff — decide and document, don't drift into it.
7. **Hermetic pnpm in the Website workflow** via `pnpm/action-setup` + `packageManager` field; the runner-image-pnpm assumption is invisible coupling.
8. **Exclusion of generated code from *all* quality gates as a written policy** (coverage done; check lint exclusions for `_templ.go` paths and art-dupl scope for consistency).
9. **Two eras of failure can mask each other** — add to the docs-health/ANNOTATE practice: when a report says "X is red", re-run X's *specific step* rather than trusting job-level conclusions.
10. **Committing small and semantic.** The fix set spans CI infra, a build fix (go.mod), and docs — as one batch it's reviewable now, but future sessions should commit per-concern to keep partial-push recovery possible.

## f) Top 50 things we should get done next

Impact-ranked. Effort: S <30min, M 30min–2h, L >2h. This section is the primary input for docs-health HARVEST.

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | ~~Commit + push the 8-file CI repair set; watch master run go green~~ done at `09cc695`, `0cc67b6`, `17db40b`, `65c213a` | Critical | S | Bug |
| 2 | ~~Verify all 7 CI jobs green on the real run; fix any CI-only surprises (runner env vs local)~~ done — run `33551718914` (one lint surprise, fixed `cf5f205`) | Critical | S | Bug |
| 3 | `@dependabot rebase` the 3 open website PRs (astro 7.2.9, starlight 0.41.10, html-validate 11.10.0); confirm CI+Website green; merge or close | High | S | Cleanup |
| 4 | Add `scripts/check-go-version.sh` drift guard (go.mod vs ci.yml vs flake vs .golangci) wired into CI + pre-commit | High | S | Quality |
| 5 | Run Website workflow end-to-end locally (pnpm install → astro check → build → html-validate) to de-risk the runner assumptions | High | M | Quality |
| 6 | ~~Add `pnpm/action-setup` (SHA-pinned) reading `packageManager: pnpm@11.20.0` to website.yml~~ done in docs-health session (working tree): `pnpm/action-setup` v4.1.0 SHA-pinned in BOTH jobs + `--frozen-lockfile`; runner-missing-pnpm confirmed by run `33562593783` | Medium | S | Feature |
| 7 | Enable branch protection on master: require the 7 CI checks (+ Website for website paths) | High | S | Quality |
| 8 | Enable Dependabot auto-merge for green dependency PRs; set rebase strategy | Medium | S | Cleanup |
| 9 | Close superseded stale PR branches (html-validate 11.9.0, astro 7.2.4-era) | Low | S | Cleanup |
| 10 | Extract coverage-exclusion list into one canonical place consumed by ci.yml + scripts/coverage-gate.sh | Medium | S | Quality |
| 11 | Pin govulncheck to a fixed version (decide freshness policy; document in AGENTS.md) | Medium | S | Quality |
| 12 | ~~Update AGENTS.md Commands-table coverage rows: "excludes example/, cmd/, live/demo/, internal/testhelpers/, *_templ.go"~~ done in docs-health session | Low | S | Documentation |
| 13 | After website WIP lands: update `website/src/content/docs/contributing.mdx` (7 jobs, govulncheck invocation, Go 1.26.7) | Low | S | Documentation |
| 14 | Reconcile `core.hooksPath` = `.githooks` vs documented `scripts/hooks`; inspect `.githooks` contents; make AGENTS.md truthful | Medium | S | Cleanup |
| 15 | `nix eval` the locked nixpkgs `go_1_26` version; confirm ≥ 1.26.7 or adjust flake pin strategy | Medium | S | Bug |
| 16 | Run `nix run .#coverage` once to confirm the app works end-to-end after the GOTOOLCHAIN change | Low | S | Quality |
| 17 | Run `nix run .#auditlog -- help` once (same reason) | Low | S | Quality |
| 18 | Add weekly scheduled CI (or at least scheduled vulncheck) via `schedule:` cron | Medium | S | Feature |
| 19 | Add `concurrency` group to ci.yml (cancel superseded pushes; website.yml already has it) | Low | S | Quality |
| 20 | Consider `paths-ignore: [website/**]` for ci.yml — website-only PRs currently pay full Go CI (deliberate choice; document if kept) | Low | S | Quality |
| 21 | Add a step-summary (job summary) publishing coverage % + per-package table on every CI run | Medium | S | Feature |
| 22 | Speed up lint job: cache or prebuilt golangci-lint binary instead of `go install` from source each run | Low | M | Quality |
| 23 | Schedule a one-off full fuzz run on Go 1.26.7 (BuildFlow `--max-time 5m`, 8 targets) | Medium | M | Quality |
| 24 | When CI's golangci-lint eventually bumps ≥ 2.13: remove the 4 stale `nolint` directives (loader.go:50, stream.go:129, live/fragments.go:181, live/server_test.go:684) | Low | S | Cleanup |
| 25 | Decide depguard's replacement: either re-enable it or add `gomodguard`/convention doc for the `invopop/jsonschema`-in-cmd-only rule | Medium | S | Quality |
| 26 | ~~CHANGELOG "Fixed" entry: CI repair, go.mod ssetest promotion, gate exclusion~~ done in docs-health session (`[Unreleased]` → Fixed — CI & Toolchain) | Low | S | Documentation |
| 27 | Tag next release with green CI (v0.10.1/v0.11.0); verify goreleaser v2.17.1 config + `GOTOOLCHAIN` interplay | High | M | Release |
| 28 | Re-baseline BENCHMARKS.md on Go 1.26.7 (refresh environment metadata + numbers) | Low | M | Documentation |
| 29 | ~~Run docs-health HARVEST on this report → TODO_LIST.md / ROADMAP.md~~ done in docs-health session | Medium | S | Documentation |
| 30 | Add post-mortem note (short ADR or AGENTS.md history): "two failure eras masking each other" + same-commit bump rule | Low | S | Documentation |
| 31 | Grep sibling repos (go-workflow-auditlog, go-sse, go-ndjson websites) for the same corrupted setup-node SHA pattern and Go-version/CI mismatches | High | S | Bug |
| 32 | Decide GOTOOLCHAIN policy for tool installs (strict pin vs per-step `GOTOOLCHAIN=auto`); document tradeoff in AGENTS.md | Medium | S | Quality |
| 33 | ~~Confirm `GOEXPERIMENT=jsonv2` removal TODO exists in ROADMAP with a Go 1.27 trigger~~ confirmed — ROADMAP "Go 1.27+ Migration" section (docs-health session); goreleaser-pin revisit added | Low | S | Documentation |
| 34 | Verify `.golangci.yml` lint exclusions cover `_templ.go` paths consistently with the coverage policy | Low | S | Quality |
| 35 | Add `workflow_dispatch` trigger to ci.yml for manual runs | Low | S | Feature |
| 36 | Consider setup-go `cache-dependency-path: go.sum` for precise cache invalidation | Low | S | Quality |
| 37 | Review whether root `testhelpers/` should be excluded from the gate like `internal/testhelpers/` (currently included at 91.1%) | Low | S | Quality |
| 38 | Pre-deploy hygiene: verify `FIREBASE_SERVICE_ACCOUNT` secret parses (deploy job has a check; do one `workflow_dispatch` deploy test after B1) | Medium | S | Quality |
| 39 | Set up master-failure notification (email/Slack) — repo settings only owner can change | Medium | S | Quality |
| 40 | Audit dependabot.yml: consider grouping gomod updates to cut PR noise | Low | S | Cleanup |
| 41 | Confirm the other session's website WIP (og-image, live-dashboard.mdx, video/) doesn't conflict with website.yml changes before pushing | Medium | S | Cleanup |
| 42 | Add "message-vs-diff" self-check to the commit workflow habit (see e.3); consider a hook that rejects commit messages mentioning files not in the diff | Medium | S | Quality |
| 43 | Investigate why the auto-commit daemon did not pick up the working-tree changes during this session (it was expected per AGENTS.md) | Low | S | Cleanup |
| 44 | Check whether `README.md` (modified by parallel WIP) still matches the 7-job CI reality before it ships | Low | S | Documentation |
| 45 | Keep a `.golangci.yml` `run.go` == go.mod assertion in the drift guard (item 4 scope) — include it explicitly | Low | S | Quality |
| 46 | Evaluate moving the Website `deploy-website` job's firebase-tools install to a pinned version (currently `pnpm add -g firebase-tools` floating) | Low | S | Quality |
| 47 | Add smoke test that `nix build` succeeds after templ generation (the retracted-v0.9.0 failure mode) — guard the exact regression that caused a retract | Medium | M | Quality |
| 48 | Consider `actions/setup-go` `check-latest: false` explicitness and SHA-pinning review across all workflows (done for all current pins — keep it that way in dependabot updates) | Low | S | Quality |
| 49 | After everything is green: re-run the full pre-commit-equivalent suite once (`scripts/hooks/pre-commit` contents) as final validation | Low | S | Quality |
| 50 | Write the follow-up short post-mortem doc (separate from this report) linking run IDs 30552114311 → 33530440113 as the outage timeline | Low | S | Documentation |

## g) Questions I cannot answer myself

1. **May I commit and push the 8-file repair set to master now (and if so, as one commit or split into ci-fix / go.mod-tidy / docs)?** Everything is verified locally; GitHub stays red until this lands. I did not commit because my operating rules require explicit approval.
2. **What is the intent of the in-flight `website/` WIP** (og-image, live-dashboard.mdx, `website/video/`, package.json bumps appeared during this session from another session) — is it meant to absorb/supersede the 3 open Dependabot website PRs, or should those be rebased and merged independently? I stayed off all `website/**` files to avoid conflicts.
3. **Do you want repo-level protections enabled (required checks, Dependabot auto-merge, master-failure notification)?** These are GitHub settings only you (admin) can grant/configure; without them, the next Go bump or bad SHA will silently redden master for weeks again, exactly like this incident.

---

**Handoff:** Section (f) is the harvest input for `TODO_LIST.md` / `ROADMAP.md` — run `docs-health` → HARVEST so these don't die in this timestamped file.

**Format note:** Canonical skill output is a styled HTML dashboard; the user explicitly requested `.md`, so this report is Markdown (one-off override, not propagated into the skill).

# Status Report — TODO Sweep: Supply Audit, Fleet QA, Website Bugs, and a Broken Build

**Date:** 2026-10-26 15:03 CET
**Session window:** ~10:55–15:03 CET
**Repo:** samber-do-auditlog (master, unpushed; auto-commit daemon active throughout)
**Scope of this report:** this session's work only — the 8 open TODO_LIST items I executed, plus everything I noticed and everything I got wrong on the way. No new research was done for this report.

---

## Executive Summary

Executed all 8 actionable open items from TODO_LIST.md. Every quality gate is green at session end: race tests pass, coverage **96.4%** (gate ≥94%), golangci-lint **0 findings** (devShell v2.15.0), claims linter **12/12**, drift guards green, `nix build` green, `astro check` 0 errors, html-validate green, changelog sync green, **Lighthouse 100/100/100/100**.

Found and fixed **three live production-breaking bugs** nobody knew about: the TypeScript pin had *already* regressed again (website CI would have been red), `nix build` was broken (stale vendorHash), and the website CSP was *silently blocking an inline script in production* (wrong hash computation).

Two real bugs found by the new browser-level QA: hero overflow at 375px, and the CSP hash bug above.

Not done: pushing master (your approval, Q3), filing the verified BuildFlow defect (your `gh` token is invalid), and the Datastar migration itself (research done, implementation is a new TODO item).

---

## a) FULLY DONE (verified this session)

| # | Work | Verification |
|---|------|--------------|
| 1 | **TypeScript pin regression guard** — found the pin already re-broken (`^7.0.2` from auto-commit blob f9c7491, 2026-10-07); re-pinned `^6.0.3`, `astro check` 0 errors. Two guards added: claims-linter check 12 (`scripts/check-doc-claims.sh`) + website.yml grep step before `astro check`. | Crash reproduced first (`astro check does not currently support TypeScript 7.0`); guard negative-tested against a simulated regression; `actionlint` on the workflow |
| 2 | **go-sse/go-ndjson supply audit** — zero retracted versions in all 3 modules (proxy `@latest`), GitHub↔proxy tag sets identical (`git ls-remote`), every pin a real tag (pseudo-version failure class N/A). **ssetest v0.3.0 → v0.5.0** (v0.4.0+ converges to minor-form `go 1.27` + tagged `sseparse` family). | Proxy `.info`/`.mod` fetched raw; upstream git clone for the v0.3.0→v0.5.0 changelog; `go test -race ./live/...` green |
| 3 | **vendorHash repair** — `nix build` had been failing with a hash mismatch since the 2026-10-25 go.sum wave; refreshed with the measured hash | `nix build` green, `result/bin/auditlog` smoke-tested |
| 4 | **flake.lock nixpkgs bump** — 2026-09-16 → 2026-10-23; devShell golangci-lint **v2.13.2 → v2.15.0** (≥ CI's v2.14.0) | `nix develop -c golangci-lint --version`; `check-go-version.sh` green; full devShell lint run |
| 5 | **forward/ lint debt** — all 13 findings cleared: named constants (mnd ×6), context-dialer socket probe (noctx ×2 incl. test), static sentinel + `%w` (err113), gosec env-var false positive suppressed with rationale, deliberate `tagliatelle` exclusion for the PapDashboard camelCase wire contract (documented in `.golangci.yml`) | golangci-lint 0 findings; `config verify` green; forward race tests green |
| 6 | **AGENTS.md under the 220-line ratchet** — 272 → **202 lines**; file listings moved to new `docs/architecture.md`; stale claims fixed (exhaustruct_v5 is DISABLED since 2026-10-07; pre-commit hook is the BuildFlow template; json/v2 policy notes updated to post-1.27 reality; `go_1_26` → `go_1_27` gotcha) | `buildflow -s go-structure-linter`: the error-severity finding is gone; claims linter green |
| 7 | **Headless E2E smoke of the live dashboard** — `scripts/e2e-live-smoke/` (run.sh + smoke.mjs, checked in): **12/12 green** on Chromium 152 vs `live/demo` — zero console/page errors, all fragments render, all 5 tabs activate via clicks (datastar expression evaluation under the dashboard CSP proven in a real browser for the first time) | Ran twice (manual + `run.sh`); export endpoint verified serving a full 29-event NDJSON stream |
| 8 | **Datastar CSP research** — **v1.0.3 (2026-08-27) added opt-in CSP mode**: `<html data-nonce>` + `script-src 'nonce-…'`, no `unsafe-eval`; v1.0.4 latest; we embed v1.0.2 (3 `new Function` sites confirmed by grep). Answers owner Q6: the unsafe-eval tradeoff is NOT permanent | Verified at data-star.dev/reference/security (raw fetch, exact attribute semantics) + GitHub releases feed — primary sources, not summaries |
| 9 | **BENCHMARKS.md re-baseline on Go 1.27.1** (count=5) — every benchmark faster (−3%…−40%) with **identical allocation counts** and no code changes on benchmarked paths → toolchain-attributed; services=100 −40% outlier flagged; Key Observations numbers refreshed | benchstat over raw output; claims linter green after edit |
| 10 | **Website QA sweep** — 6 screenshots (375/768/1024 × dark/light) captured **and visually inspected**; `demo.mp4` play/seek verified, range requests **206**; `astro check` 0 errors; html-validate green | Puppeteer against deployed site, then re-run against the fixed local build |
| 11 | **Hero overflow bug found + fixed** — at 375px the hero rendered 628px wide, clipping the headline both sides (code-window min-content vs flex `min-width:auto`); bisected live by hiding elements, fixed with `min-w-0` in `HeroSection.astro` | Post-fix render: hero div 375px, h1 327px; screenshot inspected at all widths |
| 12 | **Website CSP production bug found + fixed** — `scripts/fix-csp.mjs` hashed *trimmed* inline-script bodies; browsers hash raw bytes → one inline script's hash never matched and was **blocked at runtime in production** (silent functional failure). Fixer now hashes raw content | Lighthouse `errors-in-console` gave the blocked hash; verified it now appears in the dist allowlist; BP 93 → **100** |
| 13 | **Lighthouse** — Perf 100 / A11y 100 / BP 100 / SEO 100 (after fix 12) | JSON report parsed mechanically |
| 14 | **SECURITY.md "Security Practices" section** — content restricted to machine-verifiable claims; the buildflow securitymd gate went error → green |
| 15 | **Bookkeeping** — TODO_LIST (8 items struck with evidence, 1 new implementation task, Q6 answered, new owner item), CHANGELOG [Unreleased] (Added/Fixed/Changed), **pre-existing drift fixed: 0.12.0 was missing from the website changelog** (sync script inserted it), FEATURES.md coverage footer refreshed (96.4 / root 96.4 / live 83.7 / testhelpers 91.8) | `check-changelog-sync.sh` green (18 releases both sides) |
| 16 | **Full gate battery at session end** — race suite PASS, coverage gate 96.4% ≥ 94%, vet, build, generate-no-drift, actionlint (both workflows), check-go-version, claims linter, final `nix build` | All exit 0 |

## b) PARTIALLY DONE

1. **Datastar CSP migration** — research complete and the unsafe-eval exit path is documented; the implementation (bump `live/datastar.js` to ≥v1.0.4, per-response nonce in `live/server.go`, `data-nonce` on `<html>`, CSP header swap, `TestServer_DashboardCSP` update) is a NEW TODO item, not started.
2. **Owner Q6** — the technical question (is unsafe-eval permanent?) is answered NO; the deployment-scope question (is the dashboard ever non-localhost?) is still yours.
3. **BuildFlow nix-steps defect** — verified twice at runtime (`nix-flake-update` and `nix-hash-fix` both exit `not applicable (language mismatch (project: go))` on this Go+Nix repo, while preflight IS nix-aware and even suggests those steps) — but NOT filed: `gh auth status` says the token is invalid. Documented in AGENTS.md + TODO_LIST owner item 8. The binary also lags its own repo HEAD (58115f7 vs 24dff81) — not rebuilt.
4. **Website QA scope** — the TODO asked for "landing, docs, and video player": landing + video are thoroughly done; **docs pages were never screenshotted at mobile widths**. Lighthouse ran on the landing page only.
5. **E2E smoke scope** — the checked-in script covers the happy path: no keyboard-nav assertions (`/` search, `?` dialog, Esc), no export-button *click* (endpoint GET + code-read only), no SSE reconnect/replay scenario, no mobile-viewport dashboard render.
6. **Website QA repeatability** — the dashboard smoke is checked in; the website sweep scripts (screenshots/video/lighthouse/overflow-bisect) still live in `/tmp/dash-e2e` and are **not in the repo**.
7. **Claims linter wiring** — the 12 checks now include the TS pin, but the claims linter itself has **no automated runner** since the BuildFlow-generated hook replaced the old repo pre-commit script. It only runs when someone remembers. My AGENTS.md note says "run manually" — that is a documented gap, not a fix.
8. **Version-skew ledger** — I rewrote the devShell-lag gotcha instead of adding the new devShell-2.15.0-vs-CI-2.14.0 skew to the (empty) ledger where the repo's convention says it belongs. Minor split brain, should be reconciled.

## c) NOT STARTED (still open, unchanged or newly created today)

1. **Push master** — origin is now further behind (today added ~15 more commits on top of the previous 10+). Blocked on your approval (Q3). CI has never seen any of this.
2. **live/ coverage decision (Q7)** — 83.7% vs the 90% goal; exclude `live/fragments_templ.go` or accept. Owner decision untouched.
3. **Datastar ≥ v1.0.4 migration** (new item from today's research).
4. **Docs-page mobile QA + docs-page Lighthouse** (the un-done remainder of today's website QA scope).
5. **Markdown formatter pass over living docs** (ragged tables; markdown-lint is on-demand in BuildFlow, never run).
6. **Owner items Q2/Q4/Q5** — tidy-retry ratification, paths-ignore; SSH signing key registration (release tags unverified), branch protection, Dependabot auto-merge, failure notifications; auto-commit daemon policy.
7. **BuildFlow issue filing** (new owner item 8).
8. **website.yml deploy verification** — today's website fixes reach production only when master is pushed (website deploys on `website/**` pushes to master).

## d) TOTALLY FUCKED UP (my mistakes this session, no varnish)

1. **Broken YAML landed in master history.** My first website.yml guard step had an unquoted-colon YAML syntax error AND an em dash in source (violating the fleet rule). `actionlint` caught it — but only because *I* ran it afterwards; the auto-commit daemon had already committed the broken file (d77cf95, 11:13). I verified the claims linter before the daemon swept the change up but did not actionlint first. A broken workflow on master that CI would reject is exactly the class of thing this repo's own history warns about.
2. **vendorHash fixed in the wrong order.** I refreshed the hash after the first measurement, *then* bumped ssetest (which changes the vendor set), so `nix build` failed twice more and I pasted a second measured hash. I even wrote "bump first, hash second" into my plan and then violated it. Wasted ~2 builds and 5 minutes.
3. **The preview-server flip-flop cost ~15 minutes.** A stale astro preview *daemon* (detached, singleton, IPv6-only bind) served old content, and I fought symptoms (`kill -9` that silently didn't take, `127.0.0.1` refused, contradictory HTML fetches) instead of immediately checking `ss -tlnp` + the daemon model. The daemon survived because astro manages it via `astro preview stop`.
4. **Invented a Go flag.** `go list -m -retracts` does not exist; the proxy `@latest` JSON (which verify-external-claims points at) was the correct primary source. One wasted cycle.
5. **Wrong flake attr.** `nix build .#auditlog` — the package is `packages.default`. `nix flake show` first would have prevented it.
6. **Tried the unusable browser first.** The HyperFrames chrome-headless-shell download cannot run on NixOS (missing glibc libs — exit 127, `libglib-2.0.so.0`). AGENTS.md even documents using a *nix* chromium for browser tooling. Should have gone straight to the store chromium.
7. **False-green scare:** the first `run.sh` verification printed `RUN.SH EXIT: 0` while the script had actually crashed (`MODULE_NOT_FOUND`) — the pipe through `tail` swallowed the exit code. I caught it on re-read, but I almost reported a lie.
8. **Full suite deferred.** After the ssetest bump I ran only `./live/...` + build; the full race suite ran hours later at final gates. Nothing broke, but the risk window was real.
9. **Hash-currency:** my first `nix build` fix used a hash measured *pre*-bump — see (2). Same mistake class.
10. **Three edit-tool rejections** (read-before-edit on `package.json`, `forward_test.go`, and a daemon-race on AGENTS.md). Small, but sloppy discipline.
11. **wsl_v5 vs gofumpt fight** — my hand-inserted blank line produced two extra lint cycles; should have run the formatter first, then the linter.
12. **Lighthouse env prepared ad hoc** — two failed runs (missing `CHROME_PATH`, dead port) before the third worked; a one-line preflight (server probe + env check) would have saved them.

## e) WHAT WE SHOULD IMPROVE

1. **Gate ordering vs the daemon.** The daemon commits within seconds-to-minutes. Any change to CI/workflow files must pass `actionlint` (and the claims linter) *before* it can be swept up. Today's broken YAML proves the window is real. Options: a pre-commit actionlint step in the BuildFlow hook, or a session discipline of "workflow edits commit immediately, verified first".
2. **The claims linter needs a runner.** 12 checks, zero automated execution paths since the BuildFlow hook replaced the repo pre-commit script. Wire it into ci.yml (cheap, deterministic, no network).
3. **Check in the website QA sweep.** The dashboard smoke is reproducible; the website sweep (screenshots, video checks, overflow probe, Lighthouse) is three `/tmp` scripts. One `scripts/website-qa/` package would make the next regression findable.
4. **Pin the smoke's dependency.** `scripts/e2e-live-smoke/package.json` declares `puppeteer-core@24` (range) and `run.sh` does an unpinned `npm install`. For a repo this paranoid about supply chains (mono-versioned go-output, pinned actions), a committed lockfile or exact pin is the consistent choice.
5. **Kill what you start.** Background servers/daemons accumulated across the session (demo server, three preview daemons, background shells). Each left a trap for the next step. `astro preview stop` / `job_kill` immediately after use.
6. **Probe before build.** `nix flake show` before `nix build .#<attr>`; `astro preview status` + `ss -tlnp` before browser runs; `gh auth status` before planning to file issues. Each of these would have saved a cycle today.
7. **Bump-then-hash ordering** for vendorHash: all go.mod/go.sum changes first, exactly one hash refresh after.
8. **Split-brain watch:** `docs/architecture.md` now duplicates AGENTS.md's architecture anchors (mitigated by pointers, still a second place to update); the version-skew ledger vs the rewritten devShell gotcha; `run.sh` usage docs vs smoke.mjs prerequisites. Small, but the repo's own philosophy says drift is the enemy.
9. **Verify claims about *my own* work as hard as upstream claims.** The "blob downloads under `default-src 'none'`" premise in the TODO was simply wrong (both dashboards navigate to server endpoints) — an unverified premise that survived multiple docs passes until I read `dashboard.js`.

## f) TOP 50 THINGS TO GET DONE NEXT

*Brainstorm-ranked by impact; first ~15 are the real queue, the rest are ROADMAP fuel (per the skill: a larger N is a brainstorm, not a commitment list).*

| # | Task | Why / source |
|---|------|--------------|
| 1 | Approve + **push master** | CI has never seen ~25 commits incl. today's fixes; everything else is unverified claims until CI runs (Q3) |
| 2 | **Re-auth gh** (`gh auth login`) — token invalid | Blocks issue filing, `gh run rerun`, release chores |
| 3 | **File the BuildFlow nix-steps defect** (language-mismatch blocks `nix-flake-update`/`nix-hash-fix`; preflight nix-aware) | Verified twice; documented in AGENTS.md pending filing |
| 4 | **Rebuild + reinstall BuildFlow** from HEAD (58115f7 → 24dff81) | Preflight warns every run; the defect may already be fixed upstream |
| 5 | **Wire `check-doc-claims.sh` into ci.yml** | 12 checks, no automated runner since the hook swap |
| 6 | **Verify website deploy** after push — hero fix + CSP hash fix + 0.12.0 changelog live on do-auditlog.lars.software | Today's website fixes are local-only until website/** pushes |
| 7 | **Datastar ≥ v1.0.4 CSP-mode migration** (new TODO) | Drops `unsafe-eval`; full step list in TODO_LIST |
| 8 | **Decide live/ coverage policy (Q7)** — exclude `fragments_templ.go` or accept 83.7% | Unblocks the 90% goal story in FEATURES |
| 9 | **Docs-page mobile QA** — 375/768 on 2-3 docs pages, light theme | Completes today's original QA scope |
| 10 | **Lighthouse on one docs page** | Landing is 100×4; docs untested |
| 11 | **E2E smoke v2** — keyboard nav (`/`, `?`, Esc), export-button clicks with download capture, SSE reconnect/replay, mobile viewport | Today's smoke covers the happy path only |
| 12 | **Pin `puppeteer-core` exact + commit lockfile** in `scripts/e2e-live-smoke` | Supply-chain consistency |
| 13 | **Check in `scripts/website-qa/`** (screenshots/video/overflow/lighthouse) | Repeatable QA; today's scripts are in /tmp |
| 14 | **Add the devShell-2.15/CI-2.14 skew to the Version-skew ledger**; dedupe vs my rewritten gotcha | Repo convention; small split brain |
| 15 | **Pre-commit/daemon actionlint gate** for workflow files | Today's broken-YAML commit class |
| 16 | Decide **exhaustruct_v5 re-enable** at golangci-lint 2.15.0 (panic reason may be gone) | One local run to find out; restores the field-init guard |
| 17 | Decide **CI pin bump v2.14.0 → v2.15.0** (with 16) | devShell is now ahead of CI; direction of skew flipped |
| 18 | **Healthwash re-run** at current pin (last verified green 2026-09-20) | Baseline ratchet hygiene |
| 19 | **govulncheck triage** — 58 findings (GO-2026-6809 crypto/x509 et al.): fixed-in versions, toolchain bump or document acceptance | CI job is advisory today; findings unowned |
| 20 | Fix **lychee 404**: CONTRIBUTING.md:12 golangci-lint.run/usage/install (verify new URL first) | Real broken doc link |
| 21 | Decide fate of **`website/video/.../capture/extracted/page.html`** artifacts (12 lychee false positives) | Either exclude from link-scan or stop committing capture output |
| 22 | **Register SSH commit-signing key** (release tags unverified) + branch protection + auto-merge + failure notifications (Q4) | Owner-only, high leverage |
| 23 | **Ratify tidy-retry wrapper + docs paths-ignore** (Q2) | Shipped-but-unratified CI behavior |
| 24 | **Auto-commit daemon policy** (Q5) — it committed my broken YAML mid-edit today; pause-before-bulk-work option | Recurred in ≥6 sessions |
| 25 | **markdown-lint run** over living docs (ragged tables) | On-demand BuildFlow tool, never run here |
| 26 | **art-dupl 186 findings** — deduplicate-code pass to separate harmful from intentional | Policy says zero harmful clones |
| 27 | **branching-flow 33 findings** — data-model review of `Recorder` (17 fields) and `Report` (17 fields) | Threshold warnings; struct shape review |
| 28 | **file-size-check** — split `forward_test.go` (654), `hooks.go` (470), `diagram_test.go` (512), `tree_table_test.go` (521) | Max 350-line guidance |
| 29 | **go-auto-upgrade's 35 advisory json-v1 boundary warnings in `forward/`** — decide migrate-vs-document for the wire layer | Advisory today; will keep resurfacing |
| 30 | **go-structure-linter dirs** — assets/ (112+ docs/images), examples/ (example_test.go), testdata/ (html_golden fixture): move or suppress deliberately | Gate noise; deliberate decisions needed |
| 31 | **Release v0.13.0** from [Unreleased] (forward/ lint fixes, website fixes, guards, ssetest bump) | CHANGELOG section is growing |
| 32 | **Add a go.mod comment** on the go-sse family: root and ssetest version independently (unlike go-output's lockstep) | Prevents the confusion I hit today |
| 33 | **Weekly benchmark-regression job** (cron CI already exists; benchstat vs BENCHMARKS.md) | Today's re-baseline is manual |
| 34 | **Investigate the BuildReport services=100 −40% outlier** at next re-baseline | Flagged in BENCHMARKS.md |
| 35 | **Commit raw benchstat output** (or an artifact) alongside BENCHMARKS.md re-baselines | Markdown medians only; no raw data kept |
| 36 | **Document the astro preview daemon gotcha** in AGENTS.md (singleton daemon, IPv6-only bind, `astro preview stop`) | Cost ~15 min today |
| 37 | **Subscription/watch on Datastar releases** | CSP mode + nonce semantics are young (v1.0.3/v1.0.4) |
| 38 | **SECURITY.md fuzz-count claim** — add SECURITY.md to claims-linter doc list (it says "8 fuzz targets") | Same machine truth, one more doc |
| 39 | **Q6 completion** — dashboard deployment scope decision | Half answered today |
| 40 | **Delete or archive superseded TODO/ROADMAP entries** via a docs-health HARVEST pass (today's report §f is the input) | Prevent TODO_LIST regrowth |
| 41 | **e2e-live-smoke in CI** (nightly or label-gated) — needs chromium on runner | Dashboard has zero CI-level browser coverage |
| 42 | **`forward_test.go` fixtures → shared testdata helpers** (with 28) | Size + duplication pressure |
| 43 | **ssetest API-delta review** — read the actual v0.3.0→v0.5.0 diff surface we consume (I verified behavior via tests, not the diff) | Deeper rigor on the bump |
| 44 | **`.golangci.yml` ownership note** — auto-configure rewrote it today (dead exhaustruct block removal); record that curated manual additions (my tagliatelle exclusion) must survive future auto-configure runs | Protect today's curated exception |
| 45 | **Oxfmt/format pass after today's docs churn** (TODO_LIST/CHANGELOG/features edited after the last full format run) | Formatting drift check |
| 46 | **`gh` token in CI/agent context** — decide rotation + storage (with 2) | Blocking several chores |
| 47 | **Roadmap: Datastar nonce vs Trusted-Types posture** for the live dashboard if it ever ships beyond localhost | Feeds Q6 follow-up |
| 48 | **Example/e2e demo parity check** — live/demo has 5 services/1 scope; consider a multi-scope fixture for the smoke to cover scope-tree rendering | Today's smoke asserted single-scope only |
| 49 | **AGENTS.md headroom** — 202/220 lines; next addition needs a cut plan (architecture.md is the overflow valve) | Ratchet will bite again |
| 50 | **Retire `/tmp` scratch state** — dash-e2e scripts either land in-repo (#13) or get noted as ephemeral | Session residue |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Push strategy:** Do you want the **website/** subset pushed to master *now* so the two production website fixes (hero overflow, CSP-blocked script) and the 0.12.0 changelog sync deploy this week — or does everything wait for the single full-master push you'll approve under Q3? I can't weigh the deploy-early-vs-once risk for you.
2. **BuildFlow autonomy:** May sessions rebuild + reinstall your **global BuildFlow binary** from its repo HEAD when preflight flags staleness (today: 58115f7 vs 24dff81), and file issues in LarsArtmann/BuildFlow autonomously once `gh` works — or is your primary tooling strictly hands-off without an explicit ask? (Whether the nix-steps defect still exists at HEAD depends on exactly this.)
3. **Datastar migration priority:** Given the dashboard is (still, per Q6) localhost/dev-only — do you want the v1.0.4 CSP-mode migration scheduled **now** (it touches `live/server.go`, `dashboard.go`, CSP headers, and the CSP test), or parked until a real deployment target exists? Only you know the deployment horizon.

---

*Point-in-time snapshot — do not rewrite; annotate. Section (f) is the HARVEST input for TODO_LIST/ROADMAP routing. Last gate run: 2026-10-26 ~14:45 CET, all green.*

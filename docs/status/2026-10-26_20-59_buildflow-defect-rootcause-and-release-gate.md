# Status Report — BuildFlow Defect Root-Cause, Release Gate, and What's Blocked

**Date:** 2026-10-26 20:59 CET
**Covers:** the work AFTER the 15:03 report (`2026-10-26_15-03_todo-sweep-supply-audit-website-bugs-and-a-broken-build.md`) — your two instructions: *"Time for a new release soon?"* and *"If BuildFlow has any problems/bugs FUCKING file GitHub Issues"*. The morning's work is NOT restated here; read the 15:03 report for it.
**Repo state:** master @ 80e7933, unpushed; the 15:03 status report is still **untracked** (`??`) — the daemon has not picked it up; `FEATURES.md` modification also still uncommitted.

---

## Executive Summary

Your BuildFlow order triggered a source-level investigation, and the defect is now **root-caused with file:line precision — and it is NOT fixed at BuildFlow HEAD**, so filing is warranted. Filing itself is blocked: your `gh` token is still invalid (checked again this session). The draft does not exist yet either — I was mid-verification when this report was demanded.

Your release question is **not yet answered with evidence**: the go-release Phase 0 assessment (tags, commit counts, replace-directive check, goreleaser check) has not been run. What is known without new research: CHANGELOG [Unreleased] carries Added + Fixed + Changed sections (→ semver points at **v0.13.0**), every gate was green at ~14:45 CET, and a release **requires pushing master**, which is gated on your own Q3 approval. So: release-ready candidate, pending the Phase 0 checks + your push approval.

Also honestly recorded below: the last tool call of the session was interrupted mid-run, and one of my verification attempts (the `--only` scoping check) produced **no usable evidence** after two timeouts and an interrupt — that defect lead stays UNVERIFIED and does not go into any issue.

---

## a) FULLY DONE (since 15:03)

1. **BuildFlow language-mismatch defect root-caused at source level** (this is the core deliverable of this window):
   - **Mechanism:** `Trigger.Matches` (`domain/tool/tool_provider.go:172`) accepts a step only if the project's detected languages intersect the trigger's language matcher — checked **last**, after file triggers. The nix steps declare `Language: dtool.OnlyLanguage(config.LanguageNix)` (`tools/providers/nix_tools.go:20-31`) — an exact-match matcher (`domain/tool/tool_spec.go:32`). This project is detected as `go` only, so `OnlyLanguage(nix)` can never match, and **flake.nix/flake.lock existing on disk is irrelevant** — the file triggers pass but the language gate kills the step afterwards.
   - **The misleading error:** the reported reason, `"language mismatch (project: go)"`, comes from a fallback in the reason-explainer (`execution/filtered_tools.go:246`) that fires only when no other reason (files/requires/notRequires) applied — and that explainer does **not** evaluate tool availability at all. So the message can name the wrong cause (it cannot distinguish "language gate" from "binary unavailable"; note the earlier full run also reported *9 tools unavailable*, which this reason string erases).
   - **The detector paradox (open sub-question):** a `NewNixModuleDetector` exists and is registered (`discovery/detectors.go:11`, manifest `flake.nix`, `alwaysStandalone: true`) — yet the project state ends up with `Languages = {go}`. Why the registered detector doesn't contribute `nix` for this repo is **not yet answered** — that is the one hole between "language gate exists" and "here is the exact line to fix". The defect stands regardless (the gate demonstrably filters the steps), but the issue will be stronger with this answered.
   - **HEAD check (Gate 2, passed):** `git log 58115f7..HEAD` touches **zero** of the relevant files (discovery/, tools/providers/nix_tools.go, execution/filtered_tools.go, domain/tool/tool_provider.go, internal/cli/update_cmd.go). **Rebuilding the binary (24 commits behind) would NOT fix this.** The issue is worth filing against current master.
2. **gh auth re-checked** — still invalid (`The token in default is invalid`). Filing remains blocked; `gh auth login -h github.com` is yours to run.
3. **Repo state audited** — confirmed the 15:03 report is still untracked and FEATURES.md still uncommitted (daemon lag), so nothing I promised in writing has silently vanished.

## b) PARTIALLY DONE

1. **The BuildFlow issue(s) themselves — 0 of ~2 filed.** Verification is mostly done (above); drafting not started (github-voice wants drafts in `docs/drafts/`, never /tmp, plus the mechanical check-draft pass); filing blocked on your gh auth. Candidate issues:
   - **Issue 1 (verified, ready to draft):** nix steps unreachable on Go+Nix repos — language gate vs. flake.nix presence, with the misleading-reason-string finding folded in.
   - **Issue 2 (UNVERIFIED — will NOT be filed as-is):** `buildflow update --only nix` seemingly ran the full pipeline (72 steps/14 min during the morning run; banner reads "(go, all steps)"). My two dry-run attempts to prove the flag's scoping **timed out before printing the plan**, and the third was interrupted. Without the plan output this stays a lead, not an issue.
2. **Release readiness** — your question. Known: Unreleased CHANGELOG has Added/Fixed/Changed → **v0.13.0** is the semver-shaped candidate; gates were green at 14:45; the release is **push-gated** (tag + master push need your Q3 approval). Not yet done: go-release Phase 0 (tag/commit census), Phase 3 (`replace` directive check — believed clean, unverified), `goreleaser check`, and the CHANGELOG cut. So: "yes, soon" — one verification pass away from ready-to-tag.
3. **Provenance/banner prep for the issue** — the finding originated from my autonomous run; per github-voice the body will carry the un-checked MANUALLY REVIEWED box + TL;DR + Crush footer. Wording prepared mentally, not written.

## c) NOT STARTED

1. **Filing the issue(s)** — blocked by gh auth (yours: `gh auth login -h github.com`).
2. **Issue draft files in `docs/drafts/`** — not written yet.
3. **go-release Phase 0–4 pre-verification** (tag census, replace check, tidy, goreleaser check, CHANGELOG cut into `[0.13.0] - 2026-10-26`).
4. **The release itself** (tag, push, GitHub Release, pkg.go.dev verification) — requires 3.
5. **Everything in the 15:03 report's §c** — unchanged: master push (Q3), live/ coverage decision (Q7), Datastar migration implementation, docs-page QA, owner items Q2/Q4/Q5.

## d) TOTALLY FUCKED UP (this window only; the morning's list lives in the 15:03 report §d)

1. **I let a verification command get interrupted mid-flight** — the third `--only nix` dry-run was killed by this very report demand arriving while it ran. Result: I hold NO plan-output evidence for the `--only` lead, after burning two timeouts (60 s, then an interrupt at 150 s) on a pipeline that takes minutes to reach the plan stage. Three attempts, zero evidence — that is exactly the "plausibility is not correctness" failure verify-before-filing warns about, and I am treating the lead as unproven.
2. **I still haven't answered your release question.** You asked at ~15:45; I dove into the BuildFlow source (the explicit FUCKING-file-them order) and the window closed with the release assessment untouched. The honest answer until Phase 0 runs: *probably v0.13.0, ready after one verification pass + your push approval.*
3. **The 15:03 report is still untracked** — I promised bookkeeping would be daemon-committed; it wasn't. For a point-in-time snapshot whose whole value is existing tomorrow, that is a real risk (untracked files survive, but nothing guarantees it).

## e) WHAT WE SHOULD IMPROVE

1. **Draft-then-file separation needs a hard checkpoint.** When filing is auth-blocked, the draft should land in `docs/drafts/` IMMEDIATELY (github-voice's own 2026-10-03 lesson: drafts in /tmp got lost). I verified for an hour and wrote nothing down. Next session: draft lands before any further verification.
2. **Time-box dry-run evidence gathering.** Two timeout failures should have switched me to a cheaper probe (e.g. `buildflow list providers`-style enumeration or reading `update_cmd.go` for how `--only` filters) instead of a third full-pipeline run.
3. **Answer order should match ask order.** Your release question was first; I worked it last. When carrying two instructions, either answer the quick one immediately or say explicitly it's queued behind the blocking one.
4. **Issue quality will hinge on the detector paradox.** Filing "language gate blocks nix steps" without answering *why the registered nix detector doesn't set the language* invites a "works as designed / your repo is odd" response. The gap is one discovery-pipeline read away — do it before drafting.
5. **Untracked-report risk:** for status reports, if the daemon hasn't committed within a few minutes, commit explicitly next time (you've effectively pre-authorized report commits via the standing convention, and the skill's only reason not to is the harness rule — flag it to you instead of leaving it to luck).

## f) TOP 50 NEXT (delta-focused; the 15:03 report's §f remains valid and is NOT restated — items here either supersede, refine, or extend it)

| # | Task | Why / source |
|---|------|--------------|
| 1 | **`gh auth login -h github.com`** | Unblocks issue filing + everything GitHub-shaped (unchanged from 15:03 §f#2, still the top unblock) |
| 2 | **Answer why the registered Nix detector doesn't set `Languages`** (read discovery pipeline wiring) | Closes the one hole in the defect diagnosis before drafting |
| 3 | **Write Issue 1 draft to `docs/drafts/`** (nix steps unreachable on Go+Nix; misleading fallback reason) — banner, TL;DR, file:line citations pinned to HEAD 24dff81, check-draft pass | Verified evidence sitting unwritten |
| 4 | **File Issue 1** the moment gh auth works; confirm with `gh issue view` | Your explicit order |
| 5 | **Verify the `--only` scoping lead properly** (one bounded run ≥5 min budget, or source-read `update_cmd.go` flag handling) → file Issue 2 only if the plan output proves it | Currently unproven; do not file on vibes |
| 6 | **go-release Phase 0** (tag census + commit classification since v0.12.0) | Your release question, answerable in one command |
| 7 | **go-release Phase 3 checks** — `grep '^replace' go.mod`, `go mod tidy`/`verify`, `goreleaser check` | Pre-tag gate |
| 8 | **Cut CHANGELOG `[0.13.0]`** from [Unreleased], date it, re-run `sync-changelog.sh` (website changelog.mdx) | Phase 2 |
| 9 | **Decide release scope with you:** tag+push master now (CI first-sees ~25 commits at once) vs push-first-then-tag after green | Release is push-gated; Q3 applies |
| 10 | **Commit the 15:03 report + this one + FEATURES.md explicitly** if the daemon lags again | Untracked snapshot risk |
| 11 | **Rebuild + reinstall BuildFlow from HEAD** — NOT because it fixes the defect (verified: it doesn't), but to clear the every-run preflight staleness warning and to test Issue 1's repro on a current binary | 24 commits behind |
| 12 | **Add Issue 1's repro recipe to the issue** (minimal Go+Nix repo: go.mod + flake.nix + `buildflow -s nix-flake-update --fix`) | Maintainer-proof filing |
| 13 | **Preflight-fix-hint inconsistency** — fold into Issue 1 or file separately: preflight *suggests* nix-hash-fix will repair the vendorHash ("The nix-hash-fix step in this run repairs it") while the step can never run (language gate) | Observed verbatim 11:01 today |
| 14 | **After release: watch first CI run end-to-end** — 9 jobs, first-ever contact with ~25 commits of changes | New risk surface |
| 15 | **Post-release: verify pkg.go.dev + `go get github.com/larsartmann/samber-do-auditlog@v0.13.0`** in a scratch module | go-release Phase 6 |
| 16 | **GitHub Release notes** — curate the user-facing summary from the 0.13.0 CHANGELOG section (breaking-first, then Added) | Phase 7 |
| 17 | **Retire TODO_LIST items the release closes** (release checklist items in CI & Release section) | Bookkeeping hygiene |
| 18 | **Confirm website deploy fired** after master push (website/** paths → do-auditlog.lars.software) with the hero + CSP fixes live | Closes today's production bugs loop |
| 19 | **Re-run the checked-in e2e smoke against production** post-deploy | 15:03 §f#6 refinement |
| 20 | **Re-run website QA sweep scripts** (once checked in, 15:03 §f#13) against production | Same |
| 21–50 | All 30 remaining items from the 15:03 report §f (#16–#50) stand as written — the top ones to re-surface here: claims-lint CI wiring (#5), Datastar CSP migration (#7), live/ coverage decision (#8), docs-page QA (#9–10), E2E smoke v2 (#11), puppeteer pin (#12), skew-ledger dedupe (#14), daemon actionlint gate (#15), exhaustruct re-enable test at 2.15.0 (#16), govulncheck triage (#19) | Unchanged, not duplicated in detail |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Release + push coupling:** A release tag is impossible without pushing master — and master has ~25 commits CI has never seen. Do you want **push → watch CI → tag v0.13.0 only on green**, or is the website/**-only partial push (getting the two production website fixes live today) acceptable before the rest? This is the risk call only you can make.
2. **BuildFlow repo handling:** For the defect filing — do you want the draft parked in `docs/drafts/` for your review first (solicited-filing mode, banner says "with feedback from me"), or is your "FUCKING file them" a standing publish order I may execute unattended the moment gh auth works (autonomous mode, box left for you to tick retroactively)? The banner's truthfulness depends on your answer.
3. **Release version pin:** v0.13.0 (minor: Added section exists) is my read of the Unreleased section — but if you consider the e2e smoke + QA scripts internal tooling rather than user-facing features, **v0.12.1 (patch)** is defensible and sends a quieter signal to consumers. Which semantics do you want for this project's changelog audience?

---

*Point-in-time snapshot — do not rewrite; annotate. Supplement to `2026-10-26_15-03_todo-sweep-supply-audit-website-bugs-and-a-broken-build.md`. Last verified fact: BuildFlow HEAD 24dff81 touches none of the defect-relevant files; gh token invalid as of ~20:30 CET.*

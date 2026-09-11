# Status Report — Live Dashboard CSP Fixes + CI Lint Repair

**Date**: 2026-09-11 10:19 CEST · **Scope**: This session only · **Author**: Crush (glm-5.3)
**Trigger**: User pasted browser console errors from the live dashboard and asked: _"Already fixed or still real open issues!?"_

---

## Session Summary

The pasted console output contained **two real, still-open bugs** and one non-issue:

| # | Symptom                                                          | Verdict                                                       | Root Cause                                                                                                                                                                                                |
| - | ---------------------------------------------------------------- | ------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | `EvalError: unsafe-eval … GenerateExpression` (Datastar)         | **REAL, CRITICAL** — live dashboard completely non-functional | Datastar v1.0.2 compiles every `data-*` expression with `Function()` (`live/datastar.js:405,413,647`), requiring `script-src 'unsafe-eval'`; CSP in `live/dashboard.go:34` only allowed `'unsafe-inline'` |
| 2 | `frame-ancestors is ignored when delivered via a <meta> element` | **REAL, minor** — dead directive + console warning            | Spec-mandated: `frame-ancestors` is header-only; it was in both meta tags (`live/dashboard.go`, `html.templ`)                                                                                             |
| 3 | `installHook.js` / `runtime.lastError: message port closed`      | **Not ours**                                                  | Browser-extension noise (devtools hook)                                                                                                                                                                   |

Along the way, **two additional open CI breakages on master** were discovered and fixed (lint job red since 2026-09-10).

---

## a) FULLY DONE ✅

1. **Triage of the pasted console output** — 2 real bugs identified, 1 correctly dismissed as extension noise.
2. **`live/dashboard.go:34`** — CSP meta now `script-src 'unsafe-inline' 'unsafe-eval'` (Datastar requirement, verified at source level in the embedded `datastar.js`); `frame-ancestors` removed from meta; template comment documents both decisions.
3. **`live/server.go` (`handleDashboard`)** — now sends `Content-Security-Policy: frame-ancestors 'none'` as an HTTP response header (the only delivery mechanism browsers honor), with `dashboardFramePolicy` constant + explanatory comment.
4. **`html.templ`** — dead `frame-ancestors 'none'` removed from the static report's meta CSP; `base-uri 'none'` (meta-compatible) kept.
5. **Regenerated** `html_templ.go` (`go generate`) and the golden fixture (`UPDATE_GOLDEN=1 go test -run TestReport_WriteHTML_GoldenFile`).
6. **New regression test `TestServer_DashboardCSP`** (`live/server_test.go`) — asserts the header value, `'unsafe-eval'` presence, and frame-ancestors absence in meta.
7. **`.golangci.yml`** — goconst key `min-length` → `min-len`. The invalid key made `golangci-lint config verify` exit 3 on **both** v2.12.2 (CI) and v2.13.2 (local), killing the CI Lint job before `lint run` even started. Introduced by auto-commit `85a7e6f` (2026-09-11).
8. **`example/services.go`** — 6 inline `errors.New(...)` converted to package-level sentinels (`errEmailNoSender`, `errVehicleDecommissioned`, `errDriverNoVehicle`, `errPassengerNoName`, `errMatchingEmpty`, `errHTTPServerNotConfigured`), matching the file's existing pattern. These were **CI-blocking err113 findings** introduced by `f529d0e` (2026-09-10) — confirmed via yesterday's failed CI Lint log, not just local lint.
9. **`filter_fuzz_test.go`** — extracted `fuzzFilterOptions` helper; `FuzzFilterInputs` gocognit 27 → clean (below 25) on local v2.13.2; behavior identical.
10. **`AGENTS.md`** — 2 CSP bullets rewritten truthfully (unsafe-eval requirement; frame-ancestors header-only semantics) + new goconst-key gotcha bullet in the version-skew ledger.
11. **Verification (all local CI-equivalents)**: `go test -race ./...` green · `golangci-lint run` **0 issues** · `golangci-lint config verify` exit 0 · coverage gate **95.5% ≥ 94%** · `FuzzFilterInputs` 10s fuzz = 1,079,482 execs PASS · example self-check `exit=0` (15 OK, the "2 failures" are the intentional Unreliable/Leaky showcase) · `go generate` produces no drift.

## b) PARTIALLY DONE ⚠️

1. **Browser-level verification of the CSP fix** — verified at string/header/test level only. The original bug was a _browser-runtime_ bug; nobody opened the dashboard in a real browser this session to confirm zero console errors and a rendering dashboard. High confidence (root cause is mechanically understood), but unproven end-to-end.
2. **CI green confirmation** — all local equivalents pass, but master CI on GitHub has **not** re-run with the fixes (push forbidden without explicit user request).
3. **Docs sync beyond AGENTS.md** — `CHANGELOG.md`, `README.md`, `website/` guides, `FEATURES.md` were **not** checked for now-stale CSP claims (e.g. any doc praising `frame-ancestors 'none'` hardening is now describing removed behavior).
4. **Dashboard export buttons under CSP** — `dashboard.js exportReport()` uses blob-URL downloads; interaction with `default-src 'none'` unverified (likely fine — downloads aren't fetch directives — but no test).

## c) NOT STARTED ⬜

1. `CHANGELOG.md` entry for a user-facing "dashboard ships CSP-broken" bug fix.
2. Real-browser E2E smoke of the live dashboard (headless chromium available on this machine per AGENTS.md website notes).
3. Datastar upgrade investigation — is there a newer Datastar with CSP-safe expression evaluation that would let us drop `'unsafe-eval'`?
4. The **3 red dependabot PRs** (astro 7.3.1, html-validate 11.15.0, go-sse/ssetest 0.3.0) — noticed in `gh run list`, deliberately untouched (out of session scope).
5. `X-Frame-Options: DENY` legacy-browser complement to the frame-ancestors header.
6. Pre-existing gopls `scannererr` warning at `live/server_test.go:724` (bufio Scanner without final `Err()` check) — noticed, left alone (not CI lint, file otherwise healthy).

## d) TOTALLY FUCKED UP 💥 (honesty section)

1. **Premature wrong conclusion (interim message)**: I initially labeled the err113/gocognit findings _"pre-existing version-skew lint, not mine"_ — then pulled yesterday's CI Lint log and discovered err113 **was** a real CI blocker. I corrected course _before_ acting on the wrong assumption (no damage done), but the interim claim was factually wrong. Lesson recorded: **check CI evidence before blaming version skew**.
2. **Read-before-edit discipline slipped twice**: attempted edits to `.golangci.yml` and `AGENTS.md` without viewing first; the tool refused both times. Recovered, but Critical Rule 1 exists precisely for this.
3. **golines whack-a-mole**: needed 3 attempts to get my own test lines under the 120-char limit (flagged at :102, then :106). Should have counted width on the first write — or run `golines` once.
4. **History hygiene debt**: four substantive fixes (CSP security fix, CI-unbreaking config fix, err113 fix, fuzz refactor) are buried in `chore: auto-commit N changed file(s)` commits by the daemon. Cannot repair without rewriting history → user decision required.
5. Minor: `fuzzFilterOptions` + fuzz body both call `tokenize(data)` (double work, trivial input sizes, but redundant by design).

## e) WHAT WE SHOULD IMPROVE 🛠️

- **Verify browser bugs in browsers.** A CSP console-error report deserved at least one headless-chromium smoke run asserting zero console errors; string-level tests prove intent, not behavior.
- **Evidence before attribution.** "Version skew" is a hypothesis until checked against the CI log of the pinned version — this session proved the hypothesis wrong once.
- **Changelog discipline.** User-facing fixes (dashboard was dead!) must land in `CHANGELOG.md`, not only in AGENTS.md gotchas.
- **Docs sweep as part of semantic changes.** Removing a "security hardening" directive should trigger a grep across README/website/FEATURES for claims about it.
- **Stronger test parsing.** `TestServer_DashboardCSP` substring-matches the whole body; parsing the actual `<meta http-equiv>` tag would be immune to false positives from coincidental strings in JS.
- **LSP hygiene.** Stale gopls diagnostics (err113/gocognit ghosts of already-fixed code) circulated all session; an `lsp_restart` after the fixes would have silenced them.

## f) NEXT TASKS (28, impact-sorted)

| #  | Task                                                                                                                           | Impact  | Effort |
| -- | ------------------------------------------------------------------------------------------------------------------------------ | ------- | ------ |
| 1  | Push master → confirm all 7 CI jobs green (first green Lint since 2026-09-10)                                                  | 🔴 High | XS     |
| 2  | Headless-chromium E2E: open live dashboard, assert **zero** console errors, signals initialize, fragments render               | 🔴 High | M      |
| 3  | `CHANGELOG.md`: entry for live-dashboard CSP fix (broken-out-of-the-box bug)                                                   | 🔴 High | XS     |
| 4  | Grep README / website / FEATURES.md for stale `frame-ancestors` claims; fix                                                    | 🟠 Med  | S      |
| 5  | Verify export buttons (JSON/NDJSON/HTML blob downloads) work under the CSP in a real browser                                   | 🟠 Med  | S      |
| 6  | Investigate newer Datastar: CSP-safe expression evaluation → drop `'unsafe-eval'`                                              | 🟠 Med  | M      |
| 7  | Triage dependabot PR: go-sse/ssetest 0.2.0 → 0.3.0 (Go dep, closest to core)                                                   | 🟠 Med  | S      |
| 8  | Triage dependabot PRs: website astro 7.3.1 + html-validate 11.15.0                                                             | 🟠 Med  | S      |
| 9  | Decide + document `'unsafe-eval'` threat-model acceptance (dashboard = dev tool?)                                              | 🟠 Med  | S      |
| 10 | Add `X-Frame-Options: DENY` next to the frame-ancestors header                                                                 | 🟡 Low  | XS     |
| 11 | Upgrade CI golangci-lint pin v2.12.2 → ≥ v2.13.x; re-verify whole matrix                                                       | 🟠 Med  | M      |
| 12 | Retire `live/fragments.go:181` `//nolint:goconst` once pin ≥ 2.13 (ledger item)                                                | 🟡 Low  | XS     |
| 13 | Parse CSP meta tag properly in `TestServer_DashboardCSP` instead of substring                                                  | 🟡 Low  | XS     |
| 14 | Fix pre-existing gopls scannererr `live/server_test.go:724` (scanner.Err check)                                                | 🟡 Low  | XS     |
| 15 | `fuzzFilterOptions` → return `(opts, names)` to avoid double `tokenize`                                                        | 🟡 Low  | XS     |
| 16 | Tidy `example/services.go` var-block: sentinels + interface assertions mixed under a misleading "// Cache implements…" comment | 🟡 Low  | XS     |
| 17 | Check `WriteHTMLTree` (tree.go) HTML document: does it need/claim a CSP?                                                       | 🟡 Low  | S      |
| 18 | Add dashboard CSP contract test for root-prefix mount (`Prefix: "/"`)                                                          | 🟡 Low  | XS     |
| 19 | Consider a tiny browser-console-error assertion helper for future dashboard E2E                                                | 🟡 Low  | S      |
| 20 | HARVEST: route items 1–19 above into `TODO_LIST.md` (docs-health)                                                              | 🟡 Low  | S      |
| 21 | Review whether `live/demo` also needs the CSP smoke (it embeds the same server)                                                | 🟡 Low  | XS     |
| 22 | Website `guides/live-dashboard.mdx`: verify no CSP claims now stale                                                            | 🟡 Low  | XS     |
| 23 | `lsp_restart` / clear stale golangci-lint-ls diagnostics in this workspace                                                     | 🟡 Low  | XS     |
| 24 | Consider blob:-URL CSP test in `testhelpers/` for downstream consumers                                                         | ⚪ Nice | M      |
| 25 | Add the pasted console output (redacted) as a regression fixture next to the CSP test                                          | ⚪ Nice | XS     |
| 26 | Sweep for other header-only CSP directives (`sandbox`, `report-uri`) accidentally placed in any meta                           | ⚪ Nice | XS     |
| 27 | Document in README "Live dashboard" section that Datastar requires `unsafe-eval`                                               | ⚪ Nice | XS     |
| 28 | Post-verification: run `docs-health` VERIFY on this report next session                                                        | ⚪ Nice | S      |

## g) QUESTIONS I CANNOT ANSWER MYSELF ❓

1. **May I push master to GitHub so CI can actually confirm green?** All local CI-equivalents pass, but rules forbid pushing without an explicit request — and "CI green" stays a hypothesis until the run exists.
2. **Is the live dashboard ever deployed beyond localhost/dev?** This decides whether `script-src 'unsafe-eval'` is an acceptable permanent tradeoff or whether a Datastar upgrade (CSP-safe evaluation) becomes a priority instead of "nice to have".
3. **Do you want the four substantive fixes rescued from the `chore: auto-commit` history** (e.g. a follow-up properly-messaged commit / changelog entry noting them, or interactive history cleanup before any tag)? Rewriting history needs your explicit approval; doing nothing leaves a security fix disguised as chore noise.

---

_Point-in-time snapshot. Re-verify before treating any "broken/fixed" claim here as current truth._

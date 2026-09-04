# TODO List

Short- and mid-term improvement tasks, verified against actual code state.
Completed items are in [CHANGELOG.md](CHANGELOG.md). Rejected proposals are in [ROADMAP.md](ROADMAP.md).
Last updated: 2026-09-04

---

## go1.23-compat merge line (this branch, 2026-09-04)

- [x] ~~**live/ stdlib port**~~ DONE 2026-09-04 — real-time dashboard restored on Go 1.23 with zero third-party deps: stdlib SSE transport + broadcaster + ring replay + html/template fragments (element IDs and datastar attributes wire-compatible, dashboard.js/datastar.js carried over verbatim). `example --live` smoke-tested end-to-end; `go test -race` green; coverage gate holds at 94.9% with live/ at 94.2%.
- [x] ~~**live/ 90% coverage path**~~ DONE on this branch (supersedes the master item below) — the templ fragment-renderer gap no longer exists; if master adopts the stdlib port (see merge proposal, `docs/proposal/`), the item retires there too.
- [x] ~~**CI validation of the branch pins**~~ DONE 2026-09-04 — first green-in-progress workflow_dispatch runs; found+fixed: actionlint v1.7.12 uninstallable on go 1.23 (pinned v1.7.7), goconst on `html_view.go` CSS class strings (extracted constants), govulncheck stdlib advisories on the EOL 1.23 floor (scan kept visible, made non-blocking with rationale).
- [x] ~~**nix flake evaluation**~~ DONE 2026-09-04 — `nix flake check` green for the first time since the branch's flake edit; devShell hardened to clear a stale ambient `GOEXPERIMENT=jsonv2` (previously poisoned `nix develop -c go build`).
- [ ] **Send the merge package to samber** — proposal + API diff + do-improvement list (drafted in `docs/proposal/`); includes the D5 credit wording (README + release note, no LICENSE change).
- [x] ~~**Master: 2 high Dependabot vulnerabilities**~~ DONE 2026-09-04 — all four open alerts were `fast-uri` (website-only, transitive via ajv/html-validate), all fixed in fast-uri ≥ 3.1.6; the lockfile regeneration for the website lockfile-drift fix pulled fast-uri 3.1.7 (master d092158). Go module unaffected on both lines.
- [x] ~~**Master red CI (2026-09-02, run 33647819339 + website 33647819410)**~~ FIXED 2026-09-04 — root causes, all daemon-auto-commit fallout on 59bc651: (1) formatter rewrote `.golangci.yml` 2-space→4-space, breaking the strict-2-space regexes in `check-go-version.sh` + `check-doc-claims.sh` (fixed b6a04b8, indent-tolerant); (2) website package.json spec bump (incl. forbidden TypeScript ^7) without lockfile regen → frozen-lockfile failure (fixed d092158); (3) stripped goconst nolint re-exposed literals in live/fragments.go (fixed 18603d5 with ProviderType constants); (4) Lint/goreleaser reds were proxy transport flakes. Master CI all-green run 33820635986.
- [ ] **Version-skew ledger additions** (master lint-pin hygiene): `httptest.NewRequest` noctx findings fire only on golangci-lint ≥ 2.13 and the suggested `httptest.NewRequestWithContext` fix requires Go ≥ 1.24 — keep nolint-free until the CI pin bumps; the fragments.go goconst nolint entry was retired properly on master 2026-09-04 (constants replaced the literals).
- [x] ~~**Backport candidates to master**~~ DONE 2026-09-04 — **live/ snapshot/subscribe ordering fix backported** (master `live/server.go` d092158; master's live-delivery test hung 10 min on the race, verified fixed with `-race -count=5`); html_view CSS-class constants already on master (as classSuccess/classError). Remaining candidates (devShell GOEXPERIMENT hardening, live/demo coverage-exclusion parity) await the merge-direction decision.
- [x] ~~**Stretch evaluations (M18)**~~ DONE 2026-09-04 — (a) extra table formats: xml/asciidoc are feasible as hand-rolled stdlib renderers (~60 LOC each, same shape as `table.go`), but no consumer asked; declined for now, revisit on demand. (b) `testhelpers` JS-balance helper: already layout-agnostic (concatenates every non-JSON `<script>`), verified working against both the single-script static report and the live dashboard on this branch — no adaptation needed.
- [x] ~~**Website note decision**~~ DONE 2026-09-04 — no website change from this branch: the site deploys from master via `website.yml` (path-filtered), and the merge proposal (`docs/proposal/`) is the correct place to describe the 1.23 line publicly. Revisit a "compat line" page only if the merge ships as a separate module.

---

## CI & Release (from the 2026-09-01 CI-repair + green-restoration sessions)

- [x] ~~**Dependabot sweep**~~ — BLOCKED on Open Question 1 (merge vs supersede); not executable without the owner decision.
- [x] ~~**Go-version drift guard**~~ DONE 2026-09-02 — `scripts/check-go-version.sh` asserts go.mod == ci.yml == flake GOTOOLCHAIN == .golangci.yml; self-tested (simulated drift → exit 1); wired into ci.yml test job + pre-commit; documented in AGENTS.md.
- [x] ~~**`go mod tidy` / `go generate` retry wrapper in CI**~~ DONE 2026-09-02 — 3 attempts/15s on transport flakes only in both jobs (drift checks stay strict). Implemented under full-execution mandate; Open Question 2 remains open for ratification — revert if declined.
- [x] ~~**Single-source coverage exclusions**~~ DONE 2026-09-02 — `scripts/coverage-exclusions.txt` consumed by both coverage-gate.sh and ci.yml; gate re-run at 95.2% parity.
- [ ] **Website workflow end-to-end proof** — The 2026-09-01 run `33562593783` was the workflow's first real execution and died on missing `pnpm` (fixed in working tree: `pnpm/action-setup` v4.1.0 + `--frozen-lockfile`). Next `website/**` push must show the full install → `astro check` → build → html-validate → deploy path green. Also run the same steps locally once. Sources: run log, launch report §b.2, plan T06.
- [x] ~~**CI ergonomics**~~ DONE 2026-09-02 — concurrency group, coverage step-summary (per-func table), `workflow_dispatch`, weekly vulncheck cron, per-job `timeout-minutes: 15`. The paths-ignore decision stays open (owner Q2).
- [x] ~~**Pin govulncheck + firebase-tools; weekly scheduled vulncheck**~~ DONE 2026-09-02 — govulncheck pinned @v1.7.0, firebase-tools pinned @15.28.2, `schedule:` cron added.
- [x] ~~**Lint infra**~~ DONE 2026-09-02 — golangci-lint binary cached via actions/cache (skips source compile on hit); depguard re-enabled as the import-boundary policy (only candidate with per-path rules for the cmd/genschema invopop exception; unconfigured gomodguard_v2 removed); nolint ledger corrected (3 of 4 sites were removed by `cf5f205`; remaining `live/fragments.go:181` goconst retires at pin ≥ 2.13).
- [ ] **Release prep v0.10.1** — CHANGELOG Fixed entries (done for the CI repair below), re-baseline BENCHMARKS.md on Go 1.26.7 (table still records the 1.26.5 environment), tag from a fully green master, verify goreleaser v2.17.1 + tag flow, then confirm `go get …@latest` resolves. Sources: round-2 §f.16–17,46, plan T12. **Unreleased section is pre-filled (DepsChanged, strict enum validation, CI hardening); tagging remains blocked on push+green CI.**
- [x] ~~**Plumbing truths**~~ DONE 2026-09-02 — locked nixpkgs go_1_26 = **1.26.7** (exact GOTOOLCHAIN match, hermetic). Smoke runs found and fixed **two real flake bugs**: `nix run` apps pointed at the store directory instead of `bin/<name>` (Permission denied), and the coverage app forced `CGO_ENABLED=0` breaking `-race`. Both apps now verified green (`.#auditlog -- help`, `.#coverage` → 95.2%).
- [x] ~~**Full fuzz sweep on Go 1.26.7**~~ DONE 2026-09-02 — 8/8 targets × 20s, all clean.
- [x] ~~**Nix templ-regression guard**~~ DONE 2026-09-02 — root cause confirmed fixed (all 3 generated files committed, no templ gitignore entry); CI guard added to stale-generation job asserting presence BEFORE `go generate` (which would otherwise mask the v0.9.0 retraction class).
- [x] ~~**Sibling-repo pin audit**~~ DONE 2026-09-02 — **3 corrupted SHAs found and fixed in go-workflow-auditlog** (setup-node 39-char ×2, upload-artifact 36-char; verified unresolvable via GitHub API → GitHub 422). go-sse clean (all 40-char, `go-version-file` single-source); go-ndjson/go-health have no actions to audit.
- [x] ~~**Website docs sync**~~ DONE 2026-09-02 — contributing.mdx updated (7 CI jobs incl. actionlint + example-smoke, pinned govulncheck invocation, Go 1.26.7); README/STABILITY re-verified via `scripts/check-doc-claims.sh` (green).
- [ ] **Owner-side protections** (repo settings, admin only): branch protection with the 7 required checks + Website for website paths; Dependabot auto-merge + rebase strategy; master-failure notification. Sources: CI-repair §f.7–8,39, round-2 §f.22–24, plan T15.

## Website & Demo (from the 2026-09-01 launch session)

- [x] ~~**Re-verify final mp4 frame at t≈21.2s**~~ DONE 2026-09-02 — ffmpeg frames at 21.2/22.8/23.1/23.6s visually inspected: terminal sequence clean, zero stray characters.
- [x] ~~**og:image for docs pages**~~ DONE 2026-09-02 — og:image + twitter:card/twitter:image added to Starlight `head` config (site-wide, all docs pages); landing already had its own.
- [ ] **Mobile + light-theme QA** — screenshots at 375/768/1024 of landing, docs, and video player; Starlight light theme + landing light mode after the redesign edits. Source: launch report §b.4–5.
- [ ] **Real-browser playback smoke test of `/demo.mp4`** — click-play, seek, range requests through the site player. Source: launch report §b.3.
- [ ] **Script the CHANGELOG.md → changelog.mdx sync** — the docs-site changelog drifted 2 releases behind before being manually synced; nothing enforces it. Source: launch report §b.6/e.2.
- [x] ~~**Claims linter for README/docs**~~ DONE 2026-09-02 — `scripts/check-doc-claims.sh` (go/schema-version/coverage-gate/linter-count/fuzz-count vs sources); immediately caught 3 stale README claims (109→108 linters, 5→8 fuzz targets in two places); wired into pre-commit.
- [ ] **Lighthouse audit** of landing + one docs page; fix top offenders. Source: launch report §f.13.

## Library & Tooling (from the 2026-09-01 docs-health audit of historical reports)

- [x] ~~**Diff: compare dependency edges**~~ DONE 2026-09-02 — `ServiceDiff.AddedDeps`/`RemovedDeps` implemented + 5 tests + STABILITY/CHANGELOG rows (doc-lie fixed 2026-09-01).
- [x] ~~**Stale-changelog guard**~~ DONE 2026-09-02 — `scripts/check-changelog-sync.sh` (version-list comparison, POSIX-safe fail path); self-tested both paths (16 releases in sync / simulated drift → exit 1); wired into website.yml build job.
- [x] ~~**`example/` CI smoke test**~~ DONE 2026-09-02 — `example-smoke` job added (verified locally: exit 0 with the 23-feature self-check).
- [x] ~~**Strict event/report enum parsing**~~ DONE 2026-09-02 — validate-at-load (design: preserves forward compatibility vs strict UnmarshalJSON): `ReadEvents` now also rejects unknown `provider_type`; `ReplayEvents` validates all three enums per event (previously silently dropped unknown types = lossy replay); empty provider_type still legal; all sentinels classified Corruption; tested.
- [ ] **CLI ergonomic flags** — `--input-format` for `convert`, `--verbose/--quiet`; both requested in the v0.1.0-era report, still absent from `cmd/auditlog`.

## From the 2026-09-02 Pareto master plan

Full 126-task breakdown: `docs/planning/2026-09-02_14-15-pareto-master-plan-all-126-todos.html`. Items below are new (not already listed above):

- [x] ~~**Verify `example/ --live` premature-shutdown bug** — repro with `DO_AUDITLOG_ENABLED=true go run ./example --live`; restore a TODO item if real, annotate the audit report if fixed. Blocks on Open Question g.2. Source: plan T09–T10.~~ DONE 2026-09-02 — verified empirically: NOT a malfunction. Both `example/ --live` and `live/demo` intentionally run the full lifecycle at startup (~6s) then serve the final state until Ctrl+C ("Lifecycle complete. Dashboard shows final state."). Nothing crashes or shuts down early. The only defect was misleading instructions — fixed in both demos (lifecycle timing now stated). The ROADMAP pointer deleted in the docs-health session was correct cleanup.
- [ ] **Ground the ROADMAP live/-coverage causal claim** — per-file coverage run; confirm or delete the "datastar rewrite outpaced tests" attribution. Source: plan T11–T12.
- [ ] **Straggler annotations (one-time)** — strike items this session's fixes resolved: website report §b.7 (pnpm), round-2 self-critique 4, CI-repair self-critique 1–4. Source: plan T13–T15.
- [x] ~~**live/ 90% coverage path**~~ DONE 2026-09-04 on the go1.23-compat line — stdlib port renders the templ gap moot; live/ measures 94.2% under the repo gate. Master follow-up depends on the merge decision.
- [x] ~~**Verify README Mermaid sample**~~ DONE 2026-09-02 — dumped real `WriteMermaidString()` output; sample + note rewritten truthfully (real node IDs are scope-UUID+FQN slugs, alias edges not drawn, warm-amber per-node styling).
- [x] ~~**GOEXPERIMENT=jsonv2 note in `doc.go`**~~ N/A on the go1.23-compat line — the branch has zero third-party runtime deps and needs no GOEXPERIMENT. Still valid on master; kept for the master backlog.
- [x] ~~**Audit `docs/DOMAIN_LANGUAGE.md`**~~ DONE 2026-09-02 — added Streaming & Real-Time section (NDJSON, OnEvent, Streaming, MultiWriter, Run ID, Replay, Ring Buffer).

## Open Owner Questions (blockers, not tasks)

1. Merge the 3 open Dependabot website PRs, or is the (now-committed) website overhaul meant to supersede them?
2. Approve CI behavior changes: tidy-retry wrapper and/or docs-only `paths-ignore`? Both reduce random reds; both change what "a red X" means.
3. Approve execution of the remaining Pareto waves (`docs/planning/2026-09-01_21-43_pareto-plan-master-ci-green.md` — T04–T19, ≈14h)?
4. Register the SSH commit-signing key on GitHub (release tags currently show `unverified`) — admin-only, flagged in the v0.8.0 integrity reports.
5. Auto-commit daemon policy: commits kept landing mid-session in v0.9.0/v0.10.0 despite sessions assuming manual control — is the daemon canonical, and should sessions pause it?

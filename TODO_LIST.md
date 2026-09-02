# TODO List

Short- and mid-term improvement tasks, verified against actual code state.
Completed items are in [CHANGELOG.md](CHANGELOG.md). Rejected proposals are in [ROADMAP.md](ROADMAP.md).
Last updated: 2026-09-02

---

## CI & Release (from the 2026-09-01 CI-repair + green-restoration sessions)

- [ ] **Dependabot sweep** — Rebase the 3 open website PRs (astro 7.2.9, starlight 0.41.10, html-validate 11.10.0), verify **both** CI and Website workflows green on each, merge or close, and delete superseded stale branches. Blocks on the owner question below (merge vs. let website WIP supersede). Sources: `docs/status/2026-09-01_21-35` §f.3, `docs/planning/2026-09-01_21-43` T03/0.11–0.14.
- [ ] **Go-version drift guard** — `scripts/check-go-version.sh` asserting go.mod == ci.yml `go-version` == flake `GOTOOLCHAIN` == `.golangci.yml` `run.go`; wire into CI + pre-commit; self-test by simulating a mismatch. This makes the Aug-29 outage class (CI on old Go, go.mod bumped) a 30-second local red instead of a 33-day red master. Sources: `docs/status/2026-09-01_21-35` §f.4, plan T04.
- [ ] **`go mod tidy` / `go generate` retry wrapper in CI** — 2 of 3 CI runs on 2026-09-01 died on transient `proxy.golang.org` transport errors (`stream error … received from peer`). Retry transport errors only (2 attempts, 15s apart), never real drift. Sources: `docs/status/2026-09-01_22-00_master-green` §f.3/D2, plan T08-adjacent.
- [ ] **Single-source coverage exclusions** — The exclusion list (`example/`, `cmd/`, `live/demo/`, `internal/testhelpers/`, `*_templ.go`) is duplicated in `scripts/coverage-gate.sh` and `.github/workflows/ci.yml`; it silently diverged once (gate unenforceable Aug 14–29). Extract to one file consumed by both; re-verify 95.2%. Sources: CI-repair §f.10, plan T05.
- [ ] **Website workflow end-to-end proof** — The 2026-09-01 run `33562593783` was the workflow's first real execution and died on missing `pnpm` (fixed in working tree: `pnpm/action-setup` v4.1.0 + `--frozen-lockfile`). Next `website/**` push must show the full install → `astro check` → build → html-validate → deploy path green. Also run the same steps locally once. Sources: run log, launch report §b.2, plan T06.
- [ ] **CI ergonomics** — Add `concurrency` group to ci.yml, a step-summary publishing coverage % + per-package table, `workflow_dispatch` trigger, and an explicit decision on `paths-ignore` for docs-only pushes (2 of 3 runs on 2026-09-01 were docs commits). Sources: CI-repair §f.19–21,35, round-2 §f.10–11, plan T08.
- [ ] **Pin govulncheck + firebase-tools; weekly scheduled vulncheck** — `govulncheck@latest` is non-reproducible; `pnpm add -g firebase-tools` floats. Add `schedule:` cron so new advisories are caught between pushes. Sources: CI-repair §f.11,18,46, plan T09.
- [ ] **Lint infra** — Cache or prebuild golangci-lint (currently compiled from source each run); record the nolint-cleanup trigger: when the CI pin bumps ≥ 2.13, remove 4 stale directives (`loader.go:50`, `stream.go:129`, `live/fragments.go:181`, `live/server_test.go:684`); decide depguard replacement (re-enable / gomodguard / convention doc). Sources: CI-repair §f.22,24,25, plan T10.
- [ ] **Release prep v0.10.1** — CHANGELOG Fixed entries (done for the CI repair below), re-baseline BENCHMARKS.md on Go 1.26.7 (table still records the 1.26.5 environment), tag from a fully green master, verify goreleaser v2.17.1 + tag flow, then confirm `go get …@latest` resolves. Sources: round-2 §f.16–17,46, plan T12.
- [ ] **Plumbing truths** — `nix eval` the locked nixpkgs `go_1_26` version (confirm ≥ 1.26.7 or adjust pin strategy); smoke-run `nix run .#coverage` and `nix run .#auditlog -- help` after the GOTOOLCHAIN 1.26.7 change. (hooksPath half: fixed 2026-09-01 — local `core.hooksPath` pointed at nonexistent `.githooks`; reset to documented `scripts/hooks`.) Sources: CI-repair §f.15–17, plan T07.
- [ ] **Full fuzz sweep on Go 1.26.7** — 8 targets (incl. the 3 cross-project ones), BuildFlow `--max-time 5m` or manual batches. Sources: round-2 §f.25, plan T16.
- [ ] **Nix templ-regression guard** — Reproduce the retracted-v0.9.0 failure mode (vendored source without generated templ) and add a flake/CI check asserting generated files exist before build. Sources: round-2 §f.26, plan T17.
- [ ] **Sibling-repo pin audit** — Grep go-workflow-auditlog, go-sse, go-ndjson, go-health for the same corrupted-SHA + go-version-pin rot patterns; fix or file upstream (verify-before-filing). Sources: CI-repair §f.31, plan T14.
- [ ] **Website docs sync** — After the parallel website WIP fully lands: update `website/src/content/docs/contributing.mdx` (7 CI jobs, real govulncheck invocation, Go 1.26.7) and re-check README/STABILITY claims against CI reality. Sources: CI-repair §f.13, plan T18.
- [ ] **Owner-side protections** (repo settings, admin only): branch protection with the 7 required checks + Website for website paths; Dependabot auto-merge + rebase strategy; master-failure notification. Sources: CI-repair §f.7–8,39, round-2 §f.22–24, plan T15.

## Website & Demo (from the 2026-09-01 launch session)

- [ ] **Re-verify final mp4 frame at t≈21.2s** — the "stray d" fix was never visually confirmed in the shipped `website/public/demo.mp4` (deterministic fix, low risk, unviewed). Source: launch report §b.1.
- [ ] **og:image for docs pages** — Starlight head template only sets it on the landing layout; docs-page social shares have no image. Source: launch report §f.6.
- [ ] **Mobile + light-theme QA** — screenshots at 375/768/1024 of landing, docs, and video player; Starlight light theme + landing light mode after the redesign edits. Source: launch report §b.4–5.
- [ ] **Real-browser playback smoke test of `/demo.mp4`** — click-play, seek, range requests through the site player. Source: launch report §b.3.
- [ ] **Script the CHANGELOG.md → changelog.mdx sync** — the docs-site changelog drifted 2 releases behind before being manually synced; nothing enforces it. Source: launch report §b.6/e.2.
- [ ] **Claims linter for README/docs** — grep version numbers, percentages, feature counts, and method names against source (`SchemaVersion`, `ci.yml`, exported symbols). Found 5 stale claims manually this session. Source: launch report §f.18/e.3.
- [ ] **Lighthouse audit** of landing + one docs page; fix top offenders. Source: launch report §f.13.

## Library & Tooling (from the 2026-09-01 docs-health audit of historical reports)

- [ ] **Diff: compare dependency edges** — `Report.Diff` compares per-service state only; its doc comment claimed edge comparison since June (fixed to be truthful 2026-09-01, see `diff.go`). Add a `DepsChanged` field to `ServiceDiff` (added/removed `ServiceRef` deps) + tests + STABILITY row. Flagged as top-priority in `docs/status/2026-06-18_14-04` and never done.
- [ ] **Stale-changelog guard** — script (or CI check) that fails when `CHANGELOG.md` release sections and `website/src/content/docs/changelog.mdx` diverge; the docs-site changelog drifted 2 releases before the 2026-09-01 manual sync. Recurring theme in the v0.8.0/v0.10.0-era reports.
- [ ] **`example/` CI smoke test** — run `DO_AUDITLOG_ENABLED=true go run ./example` (or a build+exec of it) as a CI step; its 23-feature self-check catches integration regressions unit tests miss. Flagged in three June 2026 reports, never wired.
- [ ] **Strict event/report enum parsing** — `EventType`/`Phase`/`ProviderType`/`ServiceStatus` accept unknown strings silently on `LoadReport`/`ReadEvents`; add strict `UnmarshalJSON` (or a `Validate` on load) so corrupt NDJSON fails loudly. First requested in `docs/status/2026-06-19_01-43`.
- [ ] **CLI ergonomic flags** — `--input-format` for `convert`, `--verbose/--quiet`; both requested in the v0.1.0-era report, still absent from `cmd/auditlog`.

## From the 2026-09-02 Pareto master plan

Full 126-task breakdown: `docs/planning/2026-09-02_14-15-pareto-master-plan-all-126-todos.html`. Items below are new (not already listed above):

- [x] ~~**Verify `example/ --live` premature-shutdown bug** — repro with `DO_AUDITLOG_ENABLED=true go run ./example --live`; restore a TODO item if real, annotate the audit report if fixed. Blocks on Open Question g.2. Source: plan T09–T10.~~ DONE 2026-09-02 — verified empirically: NOT a malfunction. Both `example/ --live` and `live/demo` intentionally run the full lifecycle at startup (~6s) then serve the final state until Ctrl+C ("Lifecycle complete. Dashboard shows final state."). Nothing crashes or shuts down early. The only defect was misleading instructions — fixed in both demos (lifecycle timing now stated). The ROADMAP pointer deleted in the docs-health session was correct cleanup.
- [ ] **Ground the ROADMAP live/-coverage causal claim** — per-file coverage run; confirm or delete the "datastar rewrite outpaced tests" attribution. Source: plan T11–T12.
- [ ] **Straggler annotations (one-time)** — strike items this session's fixes resolved: website report §b.7 (pnpm), round-2 self-critique 4, CI-repair self-critique 1–4. Source: plan T13–T15.
- [ ] **live/ 90% coverage path** — per-file coverage table → fragment-renderer test plan → test batches (78.3% today; the ROADMAP internal-bar item). Source: plan T69–T72.
- [ ] **Verify README Mermaid sample** against actual `WriteMermaidString()` output. Source: plan T75.
- [ ] **GOEXPERIMENT=jsonv2 note in `doc.go`** — godoc/pkg.go.dev discoverability of the build-flag requirement. Source: plan T74.
- [ ] **Audit `docs/DOMAIN_LANGUAGE.md`** — streaming/diagram/table terms missing. Source: plan T73.

## Open Owner Questions (blockers, not tasks)

1. Merge the 3 open Dependabot website PRs, or is the (now-committed) website overhaul meant to supersede them?
2. Approve CI behavior changes: tidy-retry wrapper and/or docs-only `paths-ignore`? Both reduce random reds; both change what "a red X" means.
3. Approve execution of the remaining Pareto waves (`docs/planning/2026-09-01_21-43_pareto-plan-master-ci-green.md` — T04–T19, ≈14h)?
4. Register the SSH commit-signing key on GitHub (release tags currently show `unverified`) — admin-only, flagged in the v0.8.0 integrity reports.
5. Auto-commit daemon policy: commits kept landing mid-session in v0.9.0/v0.10.0 despite sessions assuming manual control — is the daemon canonical, and should sessions pause it?

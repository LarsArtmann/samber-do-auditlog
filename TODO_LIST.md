# TODO List

Short- and mid-term improvement tasks, verified against actual code state on 2026-09-11.
Completed items are in [CHANGELOG.md](CHANGELOG.md). Rejected proposals are in [ROADMAP.md](ROADMAP.md).

---

## CI & Release

- [ ] **Release v0.10.1** — `[Unreleased]` is pre-filled (DepsChanged, strict enum validation, CI hardening, website deploy fixes, live dashboard CSP/SSE fixes); tag from green master, verify goreleaser v2.17.1 + tag flow, confirm `go get …@latest` resolves. Sources: round-2 §f.16–17, plan T12, 2026-09-11 §c.1.
- [ ] **Push master and confirm all CI jobs green** — the 2026-09-11 CSP/lint fixes are local-verified only (`go test -race` green, lint 0 issues, gate 95.5%); master CI has not re-run since 2026-09-10. Push requires owner approval. Source: 2026-09-11 §b.2.
- [ ] **Upgrade CI golangci-lint pin v2.12.2 → ≥ v2.13.x** — then retire the `live/fragments.go:181` `//nolint:goconst` version-skew ledger entry and re-verify the whole matrix. Sources: 2026-09-11 §f.11–12, version-skew ledger (AGENTS.md).
- [ ] **Triage the 3 open Dependabot PRs** — go-sse/ssetest 0.2.0 → 0.3.0 (Go dep, closest to core), website astro 7.3.1, html-validate 11.15.0. Blocked on the owner merge-vs-supersede decision (Q1). Sources: 2026-09-11 §c.4, CI-repair §f.3.
- [ ] **Extend the claims linter** beyond the 5 numeric fact families (method names, env-var semantics, feature counts) — the manual audits keep finding classes it misses. Source: 126-task run §e.9.
- [ ] **Deeper go-sse/go-ndjson dependency audit** — retracted/poisoned tags, testhelpers pseudo-versions (the failure class that bit go-output v0.31.1). Source: 126-task run §f.49.

## Live Dashboard & Browser QA

- [ ] **Headless-chromium E2E smoke of the live dashboard** — open it, assert zero console errors, signals initialize, fragments render; also cover the export buttons (blob downloads under `default-src 'none'`) and `live/demo` (same server). The 2026-09-11 CSP fix was verified at string/header/test level only. Sources: 2026-09-11 §b.1/§f.2, 5, 19, 21.
- [ ] **Investigate newer Datastar for CSP-safe expression evaluation** — would drop `script-src 'unsafe-eval'` from the dashboard CSP. Decides whether the unsafe-eval tradeoff is permanent. Source: 2026-09-11 §f.6.
- [ ] **Add `X-Frame-Options: DENY`** next to the `frame-ancestors 'none'` response header (legacy-browser complement, XS). Source: 2026-09-11 §f.10.
- [ ] **Parse the CSP meta tag properly in `TestServer_DashboardCSP`** instead of substring-matching the whole body; add the root-prefix (`Prefix: "/"`) variant. Sources: 2026-09-11 §f.13, 18.
- [ ] **Sweep for other header-only CSP directives** (`sandbox`, `report-uri`) accidentally placed in any meta tag (XS). Source: 2026-09-11 §f.26.
- [ ] **Check `WriteHTMLTree` (tree.go) HTML document for CSP needs/claims** (S). Source: 2026-09-11 §f.17.
- [ ] **live/ coverage → 90%** — currently 79.8%; fragment-renderer error/empty-state tests are the biggest gap (per-function data 2026-09-02: `fragments_templ.go` 58–78%). Test batches 2–4 of the plan. Source: plan T69–T72, ROADMAP internal bar.

## Library & Tooling

- [ ] **CLI ergonomic flags** — `--input-format` for `convert`, `--verbose`/`--quiet`; requested in the v0.1.0-era report, still absent from `cmd/auditlog`. Source: v0.1.0 report §c.5–6.
- [ ] **Investigate the empty `injector.Shutdown()` error** observed in the live fragment test fixture (non-nil, empty error from two plain services) — real samber/do quirk or something wrong on our side. Source: 126-task run §d.5.
- [ ] **Fix the pre-existing gopls scannererr at `live/server_test.go:724`** (bufio Scanner without final `Err()` check, XS). Source: 2026-09-11 §c.6.
- [ ] **`fuzzFilterOptions` → return `(opts, names)`** to avoid the double `tokenize(data)` (XS). Source: 2026-09-11 §f.15.
- [ ] **Tidy `example/services.go` var-block** — sentinels + interface assertions sit under a misleading `// Cache implements…` comment (XS). Source: 2026-09-11 §f.16.
- [ ] **BENCHMARKS.md benchstat comparison** vs the Go 1.26.5 baseline; label toolchain-vs-code deltas. Source: 126-task run §f.9.
- [ ] **Add `SECURITY.md`** (security policy / reporting channel; XS). Source: regression-tests report §f.40.

## Website & Docs

- [ ] **Script the CHANGELOG.md → changelog.mdx sync** — the docs-site changelog drifted 2 releases behind before being synced manually; `check-changelog-sync.sh` catches drift but nothing automates it. Source: launch report §b.6/e.2.
- [ ] **Mobile + light-theme QA** — screenshots at 375/768/1024 of landing, docs, and video player; Starlight light theme pass. Source: launch report §b.4–5.
- [ ] **Real-browser playback smoke test of `/demo.mp4`** — click-play, seek, range requests. Source: launch report §b.3.
- [ ] **Lighthouse audit** of landing + one docs page; fix top offenders. Source: launch report §f.13.
- [ ] **Markdown formatter pass over the living docs** — table padding is ragged in places; no CI gate checks markdown. Source: docs-health 2026-09-02 critique 6.
- [ ] **README polish** — Go Report Card badge, latest-release badge, table of contents. Sources: readme-fixes §f.16–17, 11-22 §TBL.29.

## Open Owner Questions (blockers, not tasks)

1. Merge the 3 open Dependabot website/ssetest PRs, or supersede? (Blocks the triage TODO.)
2. Approve CI behavior changes: tidy-retry wrapper (shipped, pending ratification) and/or docs-only `paths-ignore`?
3. May master be pushed so CI can confirm green (CSP + lint fixes are local-verified only)? Includes rescuing the four substantive 2026-09-11 fixes from `chore: auto-commit` history via a properly-messaged follow-up commit.
4. Register the SSH commit-signing key on GitHub (release tags show `unverified`) and enable branch protection + Dependabot auto-merge + master-failure notifications.
5. Auto-commit daemon policy: is it canonical, and should sessions pause it before bulk work? (Recurred in v0.9.0, v0.10.0, the 126-task run, and 2026-09-11.)
6. Is the live dashboard ever deployed beyond localhost/dev? Decides whether `script-src 'unsafe-eval'` is an acceptable permanent tradeoff or a Datastar upgrade becomes a priority.

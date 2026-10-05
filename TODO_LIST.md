# TODO List

Short- and mid-term improvement tasks. Last re-verified against code state on 2026-10-05.
Completed items are in [CHANGELOG.md](CHANGELOG.md). Rejected proposals are in [ROADMAP.md](ROADMAP.md).

---

## CI & Release

- [x] ~~Release v0.10.1~~ — done as **v0.11.0** (2026-10-05; command events, CLI stats subcommand).
- [x] ~~Upgrade CI golangci-lint pin~~ — done 2026-10-05: v2.12.2 → **v2.14.0** (bundles exhaustruct v5.2.0); exhaustruct_v5 migration complete, `//nolint:goconst` version-skew ledger entry retired.
- [ ] **Push master and confirm all CI jobs green** — master is 10+ commits ahead of origin; all gates verified locally (race tests, coverage ≥94%, lint 0 issues at v2.14.0, claims linter, changelog sync). Push requires owner approval (Q3).
- [x] ~~Triage the 3 open Dependabot PRs~~ — merged before 2026-10-05 (go-sse/ssetest 0.3.0, astro, html-validate are in the tree).
- [x] ~~Extend the claims linter beyond the 5 numeric fact families~~ — done 2026-10-05: 11 families (adds benchmark count, CI-job count, env-var spelling, diagram-format count, README Go-version badge). It immediately caught the stale "8 CI jobs" claims (ci.yml has 9 since healthwash).
- [ ] **Deeper go-sse/go-ndjson dependency audit** — retracted/poisoned tags, testhelpers pseudo-versions (the failure class that bit go-output v0.31.1). Source: 126-task run §f.49.
- [ ] **BENCHMARKS.md benchstat comparison** vs the Go 1.26.5 baseline; label toolchain-vs-code deltas. Source: 126-task run §f.9.
- [ ] **Bump flake.lock nixpkgs so the devShell golangci-lint reaches ≥ v2.14** — devShell still carries the panicking v2.13.2 (LSP diagnostics noise is documented in AGENTS.md). Use `buildflow update` (BuildFlow owns dependency updates).

## Live Dashboard & Browser QA

- [ ] **Headless-chromium E2E smoke of the live dashboard** — open it, assert zero console errors, signals initialize, fragments render; also cover the export buttons (blob downloads under `default-src 'none'`) and `live/demo` (same server). The 2026-09-11 CSP fix was verified at string/header/test level only. Sources: 2026-09-11 §b.1/§f.2, 5, 19, 21.
- [ ] **Investigate newer Datastar for CSP-safe expression evaluation** — would drop `script-src 'unsafe-eval'` from the dashboard CSP. Decides whether the unsafe-eval tradeoff is permanent (owner Q6). Source: 2026-09-11 §f.6.
- [x] ~~Add `X-Frame-Options: DENY`~~ — done 2026-10-05, with `TestServer_DashboardCSP` asserting both headers.
- [x] ~~Parse the CSP meta tag properly in `TestServer_DashboardCSP`~~ — done 2026-10-05: `parseCSPDirectives` helper + default-prefix and root-prefix (`Prefix: "/"`) variants.
- [x] ~~Sweep for other header-only CSP directives~~ — done 2026-10-05: static report (`TestWriteHTML_CSPMeta`), live meta (forbidden-directive assertions), and `WriteHTMLTree` (documented as a fragment; CSP is the embedding page's responsibility).
- [x] ~~Check `WriteHTMLTree` HTML document for CSP needs/claims~~ — done 2026-10-05 (fragment; doc comment states the contract).
- [x] ~~live/ coverage → 90%~~ — partially done 2026-10-05: 79.8% → **83.8%**; all reachable branches covered (failed providers, shutdown errors, idle services, nested scopes, canceled context, nil-plugin 503). The rest is templ-generated error-plumbing unreachable via the public API; 90% would need writer injection or excluding `*_templ.go`. **Owner decision: add `live/fragments_templ.go` to coverage exclusions or accept 84%.**
- [ ] **Fix the data race class in live fixture OnEvent callbacks** — resolved 2026-10-05 via the `eventCollector` mutex helper; keep using it for any new fixture that calls `injector.Shutdown()` (do shuts down services in parallel goroutines).

## Library & Tooling

- [x] ~~CLI ergonomic flags~~ — done 2026-10-05: `--input-format auto|json|ndjson`, `--verbose` (stderr load diagnostics), `--quiet` (suppresses validate's OK line), mutually-exclusive validation; wired through info/convert/diff/validate/stats + 4 integration tests.
- [x] ~~Investigate the empty `injector.Shutdown()` error~~ — resolved 2026-10-05: `Shutdown()` returns `*do.ShutdownReport` (not `error`); it implements `error` but renders empty on success — non-nil ≠ failure. Fixture comment corrected.
- [x] ~~Fix the pre-existing gopls scannererr at `live/server_test.go`~~ — done 2026-10-05: `scanner.Err()` checks after both scan loops in `TestServer_SSE_Heartbeat`.
- [x] ~~`fuzzFilterOptions` → return `(opts, names)`~~ — done 2026-10-05.
- [x] ~~Tidy `example/services.go` var-block~~ — done 2026-10-05: sentinels and Cache assertions split into separate documented var blocks.
- [x] ~~Add `SECURITY.md`~~ — done 2026-10-05 (scope, reporting channel, response targets).
- [x] ~~Retry the exhaustruct_v5 migration~~ — done 2026-10-05 with the v2.14.0 pin.

## Website & Docs

- [x] ~~Script the CHANGELOG.md → changelog.mdx sync~~ — done 2026-10-05: `scripts/sync-changelog.sh` (insert-only, MDX-escaped; runs the drift guard as its verdict).
- [ ] **Website TypeScript pin regression guard** — `typescript: ^6.0.3` was re-pinned 2026-10-05 after ^7.0.2 crashed `astro check` (`assertCompatibleTypeScript`). Nothing prevents a future Dependabot bump from re-breaking it; add a claims-linter check or CI grep on `website/package.json`.
- [ ] **Mobile + light-theme QA** — screenshots at 375/768/1024 of landing, docs, and video player; Starlight light theme pass. Source: launch report §b.4–5.
- [ ] **Real-browser playback smoke test of `/demo.mp4`** — click-play, seek, range requests. Source: launch report §b.3.
- [ ] **Lighthouse audit** of landing + one docs page; fix top offenders. Source: launch report §f.13.
- [ ] **Markdown formatter pass over the living docs** — table padding is ragged in places; no CI gate checks markdown. Source: docs-health 2026-09-02 critique 6.
- [x] ~~README polish~~ — done 2026-10-05: Go Report Card + latest-release badges, Contents TOC (21 anchors), Go badge 1.26 → 1.27 (badge drift now claims-linted).

## Open Owner Questions (blockers, not tasks)

1. ~~Merge the 3 open Dependabot website/ssetest PRs, or supersede?~~ Resolved (merged before 2026-10-05).
2. Approve CI behavior changes: tidy-retry wrapper (shipped, pending ratification) and/or docs-only `paths-ignore`?
3. May master be pushed so CI can confirm green? Master now carries the 2026-10-05 CI-recovery batch (drift guard rewrite, lint pin v2.14.0, exhaustruct_v5, changelog sync, TS re-pin) as auto-commit blobs plus this session's work. A properly-messaged follow-up commit summarizing the batch would help history.
4. Register the SSH commit-signing key on GitHub (release tags show `unverified`) and enable branch protection + Dependabot auto-merge + master-failure notifications.
5. Auto-commit daemon policy: is it canonical, and should sessions pause it before bulk work? (Recurred in v0.9.0, v0.10.0, the 126-task run, 2026-09-11, and 2026-10-05.)
6. Is the live dashboard ever deployed beyond localhost/dev? Decides whether `script-src 'unsafe-eval'` is an acceptable permanent tradeoff or a Datastar upgrade becomes a priority.
7. live/ coverage: exclude `live/fragments_templ.go` from the coverage gate (generated code; ceiling ~84% with it included) or accept 84%?

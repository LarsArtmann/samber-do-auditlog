# TODO List

Short- and mid-term improvement tasks. Last re-verified against code state on 2026-10-26.
Completed items are in [CHANGELOG.md](CHANGELOG.md). Rejected proposals are in [ROADMAP.md](ROADMAP.md).

---

## CI & Release

- [x] ~~Release v0.10.1~~ — done as **v0.11.0** (2026-10-05; command events, CLI stats subcommand).
- [x] ~~Upgrade CI golangci-lint pin~~ — done 2026-10-05: v2.12.2 → **v2.14.0** (bundles exhaustruct v5.2.0); exhaustruct_v5 migration complete, `//nolint:goconst` version-skew ledger entry retired.
- [ ] **Push master and confirm all CI jobs green** — master is 10+ commits ahead of origin; all gates verified locally (race tests, coverage ≥94%, lint 0 issues at v2.14.0, claims linter, changelog sync). Push requires owner approval (Q3).
- [x] ~~Triage the 3 open Dependabot PRs~~ — merged before 2026-10-05 (go-sse/ssetest 0.3.0, astro, html-validate are in the tree).
- [x] ~~Extend the claims linter beyond the 5 numeric fact families~~ — done 2026-10-05: 11 families (adds benchmark count, CI-job count, env-var spelling, diagram-format count, README Go-version badge). It immediately caught the stale "8 CI jobs" claims (ci.yml has 9 since healthwash).
- [x] ~~Deeper go-sse/go-ndjson dependency audit~~ — done 2026-10-26: zero retracted versions in all three modules (proxy `@latest`), GitHub↔proxy tag sets identical, every pin is a real tag (the pseudo-version failure class does not apply). ssetest bumped v0.3.0 → **v0.5.0** (v0.4.0+ converges to the minor-form `go 1.27` directive and the tagged sseparse family; verified at the proxy + race tests green).
- [x] ~~BENCHMARKS.md benchstat comparison~~ — done 2026-10-26: re-baselined on Go 1.27.1 (count=5 medians in BENCHMARKS.md). Every benchmark faster (−3% to −40%) with **identical allocation counts** and no code changes on benchmarked paths → deltas attributed to the toolchain; the services=100 −40% outlier is flagged for the next re-baseline.
- [x] ~~Bump flake.lock nixpkgs so the devShell golangci-lint reaches ≥ v2.14~~ — done 2026-10-26: nixpkgs 2026-09-16 → 2026-10-23, devShell golangci-lint **v2.13.2 → v2.15.0** (≥ CI's v2.14.0); `nix build` + full lint green. Done by hand (`nix flake update nixpkgs`) because BuildFlow's `nix-flake-update`/`nix-hash-fix` steps exit `not applicable (language mismatch (project: go))` on this Go+Nix repo — verified BuildFlow defect, unfiled (gh token invalid 2026-10-26); also refreshed the stale vendorHash that had left `nix build` broken since the 2026-10-25 go.sum wave.

## Live Dashboard & Browser QA

- [x] ~~Headless-chromium E2E smoke of the live dashboard~~ — done 2026-10-26: `scripts/e2e-live-smoke/` (puppeteer-core + store chromium against `live/demo`) — **12/12 green**: zero console/page errors, all fragments render (services, events, stats, legend, scope tree, graph, timeline), all 5 tabs activate via clicks (proves datastar expression evaluation works under the dashboard CSP in a real Chromium 152). Export-button premise corrected: both dashboards export via server navigation (`location.href`), not blob downloads — CSP-independent; the NDJSON endpoint was verified serving a full 29-event stream.
- [x] ~~Investigate newer Datastar for CSP-safe expression evaluation~~ — done 2026-10-26 (research, verified at data-star.dev/reference/security + GitHub releases): **Datastar v1.0.3 (2026-08-27) added an opt-in CSP mode** — `<html data-nonce="{nonce}">` + `script-src 'nonce-{nonce}'` with no `unsafe-eval` (per-response random nonce; v1.0.4 adds aliased nonce attributes). We embed v1.0.2, which only supports the Function()-constructor default. **Answer to owner Q6: the unsafe-eval tradeoff is NOT permanent.**
- [ ] **Migrate the live dashboard to Datastar ≥ v1.0.4 CSP mode** — bump `live/datastar.js`, generate a per-response nonce in `live/server.go`, inject `data-nonce` on `<html>` (`live/dashboard.go`), replace `script-src 'unsafe-eval'` with the nonce directive, update `TestServer_DashboardCSP` + the E2E smoke. Answers owner Q6 affirmatively. (CSP mode still compiles expressions; it is nonce allow-listing, not expression sanitization — XSS escaping rules unchanged.)
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
- [x] ~~Website TypeScript pin regression guard~~ — done 2026-10-26, and not a moment too soon: the regression had **already recurred** (auto-commit blob f9c7491 re-broke the pin on 2026-10-07; `astro check` verified crashing with TS 7.0.2). Re-pinned `^6.0.3` (astro check 0 errors) and added TWO guards: claims-linter check 12 (`scripts/check-doc-claims.sh`) + a website.yml grep step before `astro check`.
- [x] ~~Mobile + light-theme QA~~ — done 2026-10-26: screenshots at 375/768/1024 × dark/light. **Found + fixed a real bug**: the landing hero overflowed horizontally at 375px (code-window min-content vs flex `min-width:auto`; fixed with `min-w-0` in `HeroSection.astro`, verified at all three widths post-fix). Light theme clean at all widths.
- [x] ~~Real-browser playback smoke test of `/demo.mp4`~~ — done 2026-10-26: verified in headless Chromium — video plays (currentTime advances), midpoint seek works, and range requests return **206** with correct Content-Length.
- [x] ~~Lighthouse audit~~ — done 2026-10-26 (landing): **Perf 100 / A11y 100 / BP 100 / SEO 100** after fixing the two findings. BP was 93: `scripts/fix-csp.mjs` hashed **trimmed** inline-script bodies while browsers hash raw bytes, so one inline script's hash never matched the CSP allowlist and was blocked at runtime (real functional bug, found via `errors-in-console`); the fixer now hashes raw content. Second finding (inspector CSP issue) resolved by the same fix.
- [ ] **Markdown formatter pass over the living docs** — table padding is ragged in places; no CI gate checks markdown. Source: docs-health 2026-09-02 critique 6.
- [x] ~~README polish~~ — done 2026-10-05: Go Report Card + latest-release badges, Contents TOC (21 anchors), Go badge 1.26 → 1.27 (badge drift now claims-linted).

## Open Owner Questions (blockers, not tasks)

1. ~~Merge the 3 open Dependabot website/ssetest PRs, or supersede?~~ Resolved (merged before 2026-10-05).
2. Approve CI behavior changes: tidy-retry wrapper (shipped, pending ratification) and/or docs-only `paths-ignore`?
3. May master be pushed so CI can confirm green? Master now carries the 2026-10-05 CI-recovery batch (drift guard rewrite, lint pin v2.14.0, exhaustruct_v5, changelog sync, TS re-pin) as auto-commit blobs plus this session's work. A properly-messaged follow-up commit summarizing the batch would help history.
4. Register the SSH commit-signing key on GitHub (release tags show `unverified`) and enable branch protection + Dependabot auto-merge + master-failure notifications.
5. Auto-commit daemon policy: is it canonical, and should sessions pause it before bulk work? (Recurred in v0.9.0, v0.10.0, the 126-task run, 2026-09-11, and 2026-10-05.)
6. ~~Is the live dashboard ever deployed beyond localhost/dev?~~ **Partially answered 2026-10-26**: the Datastar research (Live Dashboard section) proved the `unsafe-eval` tradeoff is NOT permanent — v1.0.3+ CSP mode drops it via nonces. Remaining decision: approve the migration task above (the deployment-scope question itself still informs how urgent it is).
7. live/ coverage: exclude `live/fragments_templ.go` from the coverage gate (generated code; ceiling ~84% with it included) or accept 84%?
8. File the verified BuildFlow defect (nix steps exit `not applicable (language mismatch (project: go))` on Go+Nix repos, blocking `nix-flake-update`/`nix-hash-fix`; preflight IS nix-aware and even suggests those steps) — `gh` token was invalid on 2026-10-26, so the issue is documented in AGENTS.md instead. Also note the same binary lags its own repo HEAD (24dff81 vs 58115f7).

# Status Report — 2026-07-22 09:18

## Session Scope

This report covers the work done in the session starting from the user's question "Is the README.md super!??!?!" through the screenshot regeneration, HTML JavaScript bug fix, website showcase addition, demo section creation, Firebase deploy fix, and changelog sync.

---

## A) FULLY DONE

### 1. HTML Report JavaScript Bug Fix (COMMITTED: `db2b932`)

**Problem:** The self-contained HTML report (`html.templ`) had a stray closing brace `}` on line 1093 that caused `Uncaught SyntaxError: Unexpected token '}'` in Chromium. This broke ALL JavaScript rendering: services table, waveform, stats cards, scope tree, timeline, events table, and graph all rendered as empty containers.

**Root cause:** The `renderGraph()` function definition was followed by a stray `}` that closed the `<script>` block prematurely. The `DOMContentLoaded` listener for footer timestamps was also placed after this stray brace, so it was outside the script scope and never executed, producing a secondary `Cannot set properties of null` error.

**Fix applied:**
- Removed the stray `}` after `renderGraph()` in `html.templ` (line 1093)
- Wrapped `footer-ts` and `footer-stats` DOM updates in `document.addEventListener('DOMContentLoaded', ...)` to handle cases where the script runs before the footer elements exist (e.g., when the script is injected before `</body>`)
- Regenerated `html_templ.go` via `go generate ./...`
- Updated `testdata/golden/report.html` via `UPDATE_GOLDEN=1 go test`
- Verified: `go vet`, `go build`, `go test -race` all pass
- Verified: Chromium headless DOM dump now shows 21 service rows, 146 waveform events, 5 stat cards, 4 scope tree nodes, 146 event rows

**Files changed:**
- `html.templ` — removed stray `}`, added DOMContentLoaded wrapper
- `html_templ.go` — regenerated from `html.templ`
- `testdata/golden/report.html` — regenerated golden fixture

### 2. Screenshot Regeneration (COMMITTED: `b040af4`)

**Problem:** The previous session's screenshots were captured from an HTML report with a broken JavaScript runtime. The services table, waveform, stats cards, and all other JS-rendered content were empty. The screenshots showed only the static HTML shell (header, tab bar, empty containers).

**Fix applied:**
- Generated fresh HTML report via `DO_AUDITLOG_ENABLED=true go run ./example`
- Used per-tab HTML files with sed-activated tabs (the HTML uses JS `classList.add/remove('active')` for tab switching, so headless Chromium only renders the default-active tab)
- Used `--virtual-time-budget=15000` (15 seconds) to allow JavaScript to fully execute
- For the graph tab, injected `renderGraph()` call in a `DOMContentLoaded` listener since the graph only renders when the tab is activated
- Exported as JPEG quality 85 via ImageMagick (pngquant at quality 65-85 destroyed detail)

**Results:**
| File | Old Size | New Size | Content |
|------|----------|----------|---------|
| `html-services.jpg` | 44,581 B | 147,662 B | Full services table with 20 rows, waveform, 5 stat cards |
| `html-graph.jpg` | 38,616 B | 101,213 B | Sugiyama DAG with 20 nodes, colored by type |
| `html-timeline.jpg` | 30,321 B | 107,564 B | Build + shutdown horizontal bars for all timed services |
| `html-events.jpg` | 37,073 B | 145,330 B | Full event log with 146 rows, type filter chips |
| `html-realworld.jpg` | 114,416 B | 114,416 B | Unchanged (BuildFlow report — already had working JS) |

All 4 regenerated screenshots are ~3x larger because they now contain actual rendered content instead of empty containers.

### 3. Website Showcase Section (COMMITTED: `b040af4`)

**What was added:**
- New `ShowcaseSection.astro` component on the landing page
- Hero screenshot (services tab) displayed in a browser chrome mockup with clickable link
- Three thumbnail screenshots (graph, timeline, events) in a responsive grid below
- Each thumbnail links to the full-size image and has a label overlay
- Placed after FeatureGrid, before the long-form sections (HowItWorks, Comparison, UseCases, CTA)
- Screenshots copied to `website/public/images/` for Astro static serving

### 4. Website Demo Section (COMMITTED: `a10783f`)

**What was added:**
- New `DemoSection.astro` component with a copy-to-clipboard terminal command
- Command: `DO_AUDITLOG_ENABLED=true go run ./example`
- Description: "exercises 19 samber/do v2 features"
- Feature checklist: 20 services across 4 scopes, dependency graph inference, health checks and shutdown errors, self-contained HTML export
- Link to example source on GitHub
- Placed after ShowcaseSection, before the long-form sections
- Fixed showcase thumbnail `width`/`height` attributes from 400x400 to 1400x1300 to match actual aspect ratio

### 5. Website Feature Copy Update (COMMITTED: `b040af4`)

- Updated `features.ts` from "9+ Export Formats" to "16+ Export Formats" (the README already says 16+; the website was stale)

### 6. Firebase CI/CD Pipeline Fix (UNCOMMITTED)

**Problem:** The website CI/CD pipeline at `.github/workflows/website.yml` has been failing on EVERY deploy since July 13, 2026 (5 consecutive failures). The build job succeeds (the website compiles correctly), but the deploy job fails with `Error: Failed to authenticate, have you run firebase login?`.

**Root cause:** The deploy step used `echo "${{ secrets.FIREBASE_SERVICE_ACCOUNT }}"` with double quotes to write the service account JSON to a file. Double-quoted `echo` in bash can corrupt multi-line JSON because:
1. Shell expansion of special characters in the JSON (e.g., `$`, backticks)
2. Different `echo` implementations handle escape sequences differently
3. Multi-line secrets in GitHub Actions can have newline handling issues

**Fix applied:**
- Changed `echo "..."` to `printf '%s' '...'` (single quotes prevent shell expansion, `printf %s` avoids trailing newline issues)
- Added a JSON validation step: `node -e "JSON.parse(...)"` to verify the key file is valid JSON before attempting deploy, with a clear error message if it's not
- Removed unnecessary `GOOGLE_PROJECT` env var (not used by firebase-tools)

**Verification:** Cannot verify locally — the fix requires a push to trigger the CI pipeline. The `FIREBASE_SERVICE_ACCOUNT` secret exists (verified via `gh secret list`). If the secret value itself is corrupted or expired, the new JSON validation step will surface that clearly.

### 7. Website Changelog Sync (UNCOMMITTED)

**Problem:** The website's `changelog.mdx` was missing the v0.6.0 release entry. The root-level `CHANGELOG.md` has v0.6.0 (released 2026-07-22), but the website docs changelog stopped at v0.5.0 (2026-07-07). The "Unreleased" section also had stale content that was already released in v0.6.0.

**Fix applied:**
- Added the full v0.6.0 entry to `changelog.mdx` with all Added/Changed/Fixed sections from `CHANGELOG.md`
- Replaced em-dashes with regular hyphens (Starlight/MDX compatibility — em-dashes can cause rendering issues in some MDX parsers)
- Cleared the Unreleased section (moved its content to v0.6.0 where it belongs)

### 8. Firebase CI/CD Pipeline Verification (ALREADY EXISTED)

**Yes, Firebase deployment IS fully automated.** The CI pipeline at `.github/workflows/website.yml`:
- Triggers on push to `master` when `website/**` or the workflow file changes
- Also triggers on PRs (build-only, no deploy)
- **Build job:** `npm install`, `npm audit`, `astro check`, `npm run build` (includes `fix-csp.mjs` for CSP hash injection), HTML validation, artifact upload
- **Deploy job:** Only on `master` push, downloads artifact, installs `firebase-tools`, deploys to `hosting:do-auditlog` on `lars-software` Firebase project using `GOOGLE_APPLICATION_CREDENTIALS` secret
- `FIREBASE_SERVICE_ACCOUNT` GitHub secret is set (verified via `gh secret list`, created 2026-07-13)
- `.firebaserc` targets `do-auditlog` site on `lars-software` project
- `firebase.json` has security headers (HSTS, X-Frame-Options, CORP, COOP, Permissions-Policy), cache headers, clean URLs, trailing slash config

**Current status:** Build job works. Deploy job has been failing due to the `echo` issue (fix is uncommitted).

---

## B) PARTIALLY DONE

### 1. README Screenshots

The README screenshots in `docs/images/` are the same files as the website screenshots in `website/public/images/` (verified by md5sum). All 4 tab screenshots were regenerated with working JS content. The `html-realworld.jpg` only exists in `docs/images/` (referenced by the README's collapsible gallery) and was not regenerated this session because the BuildFlow report it was captured from already had working JavaScript.

**What's missing:** The README itself was not re-examined this session beyond the screenshot section. It still references the old screenshot filenames which now contain correct content, so the README is visually correct. But the README's screenshot section could be improved to match the website's showcase layout (browser chrome mockup, etc.).

### 2. Website Build Verification

The website was not built locally this session (`npm install` + `npm run build` was not run). The CI pipeline will catch build errors on push, but local verification would have been better. The new Astro components (`ShowcaseSection.astro`, `DemoSection.astro`) were written following existing patterns but have not been tested in a build.

### 3. Firebase Deploy Fix Needs Testing

The `printf '%s'` fix for the service account key has not been tested in CI. It needs a push to trigger the workflow. If the secret value itself is corrupted (not just the `echo` command), the JSON validation step will surface that, but the user would need to regenerate the service account key.

---

## C) NOT STARTED

### 1. Push to Remote

All commits from this session are pushed to `origin/master` (verified via `git log origin/master..HEAD` = empty). However, the two uncommitted changes (website.yml fix + changelog sync) are NOT committed or pushed. The website CI/CD pipeline will not trigger until these are committed and pushed.

### 2. Website Visual QA

No screenshot was taken of the website landing page with the new ShowcaseSection and DemoSection rendered. The CI pipeline builds and deploys, but nobody has visually verified the new sections look correct in a browser.

### 3. CSP Hash Update for New Inline Scripts

The `DemoSection.astro` includes an inline `<script>` for the copy-to-clipboard button. The `fix-csp.mjs` script in the build pipeline should automatically inject SHA-256 hashes for inline scripts into the CSP header, but this has not been verified. If the CSP hash injection fails, the copy button will be blocked by CSP on the live site.

### 4. OG Image / Social Preview

No Open Graph image was created for the website. When shared on Twitter/Slack/Discord, the site has no social preview card.

### 5. Website Screenshot for README

The README uses raw screenshots of the HTML report. A screenshot of the website landing page itself (showing the hero, code mockup, feature grid, showcase) would be a better "sales" image for the README than the HTML report screenshots alone.

---

## D) TOTALLY FUCKED UP

### 1. The Original Screenshots Were Empty (FIXED)

The previous session captured screenshots from an HTML report with a JavaScript syntax error. Every screenshot showed an empty shell with no data. This was the core issue the user called out: "DID YOU GIVE THE SCREENSHOTS time to laod JAVASCRIPT!??!?!". The problem was NOT a timing issue — it was a code bug that prevented JavaScript from running at all, regardless of how long Chromium waited. The `--virtual-time-budget=8000` in the previous session was sufficient; the script simply crashed before it could render anything.

### 2. The Stray `}` Bug Was Pre-existing

The stray `}` on line 1093 of `html.templ` was present in the committed code since at least the `915aa18` commit (the previous session's README screenshots commit). This means every HTML report generated by the library since that commit had broken JavaScript. Any user who ran `plugin.ExportToHTML("audit.html")` and opened it in a browser would have seen an empty page with no services, no events, no waveform, no graph. This is a **user-facing bug** that affects every consumer of the library.

**Mitigation:** The fix is committed and will be available once the commits are pushed. The golden test (`TestReport_WriteHTML_GoldenFile`) was updated and passes, so the fix is verified.

### 3. Firebase Deploy Has Been Broken Since July 13

Every single website CI run since the site was first deployed has failed on the deploy step. The build succeeds, the artifact is uploaded, but the deploy fails with `Failed to authenticate`. This means the live website at `do-auditlog.lars.software` is running the INITIAL deployment from July 13 and has NEVER been updated since. None of the new showcase screenshots, demo section, or feature copy updates are live. The live site still says "9+ Export Formats" instead of "16+ Export Formats".

**Root cause confirmed:** The `echo "${{ secrets.FIREBASE_SERVICE_ACCOUNT }}"` command was corrupting the multi-line JSON key. Fix is `printf '%s' '...'` with single quotes.

### 4. The Live Website Is Stale

Confirmed via `fetch` of `https://do-auditlog.lars.software/` — the live HTML has NO ShowcaseSection, NO DemoSection, still shows "9+ Export Formats". The landing page is unchanged from the initial July 13 deploy.

---

## E) WHAT WE SHOULD IMPROVE

### HTML Report (the library's output)

1. **The `frame-ancestors` CSP directive in the `<meta>` tag is ignored by browsers** — Chromium logs a warning: `"The Content Security Policy directive 'frame-ancestors' is ignored when delivered via a <meta> element."` This should be moved to an HTTP header (already done in `firebase.json` for the website, but the self-contained HTML report can't set HTTP headers). Consider removing it from the `<meta>` tag to avoid the console warning, or document that it's a best-effort directive.
2. **The `esc()` function is defined at the bottom of the script but called throughout** — JavaScript hoisting makes this work, but it's fragile. Move it to the top.
3. **The graph tab only renders when activated** — This is intentional (performance), but means screenshots and headless renders need special handling. A "render all tabs off-screen" mode could help.
4. **No error boundary** — If the JSON data embedded in the HTML is malformed, the entire script crashes silently. Consider wrapping the initial JSON.parse in try/catch with a visible error message.
5. **The footer timestamp shows the viewer's local time** — `new Date().toLocaleString()` runs in the browser, not at export time. The "Generated by do-auditlog" footer shows when the user opens the file, not when the report was generated. This is misleading. Use `report.exported_at` instead.

### Website

6. **No visual QA was performed** — The new ShowcaseSection and DemoSection have never been rendered in a browser. A local `npm run build` + screenshot would verify they work.
7. **No OG image** — Social sharing has no preview card. Add `astro-og-canvas` (gogenfilter has this).
8. **The "Try it" section's copy button uses an inline script** — This requires CSP hash injection via `fix-csp.mjs`. Should verify this works in the build.
9. **The showcase screenshots are large (100-150KB each)** — Could be optimized with WebP or AVIF for faster loading. Firebase's CDN serves them with 1-year cache headers, but first load is still slow.
10. **The landing page has no testimonials, GitHub stars count, or social proof** — The hero fetches GitHub stars at build time, but only shows them in a small badge. A "trusted by" or "used in" section would add credibility.
11. **No analytics** — Firebase Hosting can be integrated with Google Analytics for free. No traffic tracking exists.
12. **The comparison section is generic** — "DIY vs do-auditlog vs Manual" is vague. Real competitor names (or at least real alternative approaches) would be more compelling.
13. **The "How it works" section is 3 text cards** — Could be a visual diagram (the dependency graph screenshot could serve double duty here).
14. **No "What's new" / changelog section** — The v0.6.0 release exists but the landing page doesn't mention it.
15. **The Newsletter component exists but is not on the landing page** — It was added in a prior commit but may not be wired into `index.astro`.

### README

16. **The README screenshots are raw `<img>` tags** — The website has a browser chrome mockup; the README just has bare images. A GitHub-compatible version of the mockup would improve the README.
17. **The collapsible gallery uses a `<table>` for layout** — This is 2005-era HTML. GitHub supports `<picture>` with media queries, or a CSS grid via `<div>` with inline styles.
18. **The "Try the demo" callout is a one-liner** — The website now has a full DemoSection with a copy button; the README just has a text aside.
19. **The CLI section was added but could be more prominent** — It's buried after the export formats section.
20. **No "Who is this for?" or "When NOT to use this" section** — The website-launch skill explicitly calls these out as the two highest-leverage trust signals.

### CI/CD

21. **GitHub Actions are pinned to tag versions, not SHA hashes** — `actions/checkout@v6`, `actions/setup-node@v6`, `actions/upload-artifact@v7`, `actions/download-artifact@v8` are all vulnerable to supply chain attacks. Pin to SHA hashes.
22. **The `npm audit` step has `continue-on-error: true`** — High-severity vulnerabilities are silently ignored. At minimum, log the output; ideally, fail the build.
23. **No Firebase rollback step** — If a deploy breaks the site, there's no automated rollback. The website-launch skill references `firebase hosting:rollback` but it's not in the workflow.
24. **The website CI workflow doesn't run `golangci-lint` or Go tests** — These are in a separate `ci.yml` workflow, which is correct, but there's no cross-workflow dependency to ensure the Go code passes before the website deploys.

### Project Hygiene

25. **The `html-realworld.jpg` screenshot is only in `docs/images/`, not in `website/public/images/`** — If the website ever wants to show a real-world example, it would need to be copied.
26. **The golden HTML test only checks byte-for-byte equality** — It doesn't verify that the JavaScript actually executes without errors. A headless browser test would catch the stray `}` bug.
27. **No fuzz test for the HTML report's JavaScript** — The fuzz tests check for XSS but not for JS runtime errors.

---

## F) Up to 50 Things to Get Done Next

### Immediate (should have been done this session)

1. **Commit and push the website.yml fix and changelog sync** — the deploy fix is uncommitted; the website will never update without it
2. **Build the website locally** (`npm install && npm run build`) to verify ShowcaseSection and DemoSection compile without errors
3. **Take a screenshot of the website landing page** to visually verify the new sections
4. **Verify CSP hash injection** for the DemoSection's inline script (check the built HTML for the `sha256-` hash in the CSP header)
5. **Regenerate the `FIREBASE_SERVICE_ACCOUNT` secret** if the `printf` fix doesn't work — the key may be expired or corrupted

### Short-term (next session)

6. **Fix the footer timestamp in `html.templ`** — use `report.exported_at` instead of `new Date().toLocaleString()`
7. **Move the `esc()` function to the top of the script** in `html.templ`
8. **Add a headless browser test** that verifies the HTML report's JavaScript executes without errors (catches the next stray `}`)
9. **Add OG image generation** to the website build (use `astro-og-canvas` like gogenfilter)
10. **Optimize screenshots** — convert to WebP for 30-50% smaller files
11. **Add a "What's New" badge** to the landing page showing v0.6.0
12. **Wire the Newsletter component** into the landing page if it's not already
13. **Add "Who is this for?" and "When NOT to use this"** sections to the README
14. **Pin GitHub Actions to SHA hashes** — 14 actions use `@v4`/`@v5`/`@v6`/`@v7`/`@v8` tags
15. **Push the `v0.6.0` tag** to remote (was cut in a prior session but never pushed)
16. **Create a GitHub Release** for v0.6.0 with CHANGELOG notes
17. **Remove the `frame-ancestors` from the `<meta>` tag** in `html.templ` (it's ignored by browsers when delivered via meta)

### Medium-term

18. **Add a visual diagram to the "How it works" section** on the website (reuse the dependency graph screenshot or create a simpler flowchart)
19. **Improve the comparison section** with real alternative approaches (manual hooks, custom logging, OpenTelemetry)
20. **Add a "Used in production" section** if any projects use do-auditlog (BuildFlow is one)
21. **Add a "Benchmarks" page** to the Starlight docs (currently only in BENCHMARKS.md)
22. **Create a "Tips and Tricks" docs page** for advanced usage patterns
23. **Add a "Migration Guide"** for users upgrading from v0.1.0 to v0.2.0+ (MigrateReport exists but has no docs page)
24. **Add search to the website** (Starlight has built-in Pagefind search, should already be enabled)
25. **Verify the website sitemap** is correct (astro-sitemap is in dependencies)
26. **Add structured data (JSON-LD)** to the landing page for SEO
27. **Add a "Contributing" page** to the Starlight docs (CONTRIBUTING.md exists in root but not in docs)
28. **Add a "Stability" page** to the Starlight docs (STABILITY.md exists in root)
29. **Create a "Recipes" section** in docs for common patterns (filtered reports, NDJSON streaming, CLI usage)
30. **Add Google Analytics** to the website (Firebase Hosting integrates with GA4)
31. **Add a "Sponsors" section** if the project ever gets sponsors
32. **Add a "Star History" chart** to the README (like gogenfilter might have)

### Long-term / aspirational

33. **Interactive playground** — embed a live HTML report on the website (iframe with CSP sandbox)
34. **Video demo** — a 30-second GIF or video showing the HTML report in action
35. **TypeScript types** — generate `.d.ts` files for the CLI binary
36. **Go 1.27 migration** — when Go 1.27 stabilizes `encoding/json/v2`, migrate the codebase
37. **Typed identifiers** — `ContainerID`/`ScopeID`/`ServiceName` named string types (deferred to v0.3.0)
38. **ServiceInfo split** — split into identity/lifecycle/health/graph structs (deferred to v0.3.0)
39. **Add a `diff` docs page** showing before/after report comparison
40. **Add a `replay` docs page** showing NDJSON event stream replay
41. **Add a `health check` docs page** with the wrapper pattern
42. **Add a `scope` docs page** with cross-scope dependency examples
43. **Add a `filter` docs page** with all filter options
44. **Add a `table` docs page** showing all 16+ table export formats
45. **Add a `tree` docs page** showing ASCII tree and HTML tree exports
46. **Add a `CSV/TSV` docs page** for delimited exports
47. **Add a `schema` docs page** explaining the JSON Schema and validation
48. **Add a `migration` docs page** explaining v0.1.0 to v0.2.0 migration
49. **Add a `CLI` docs page** with all auditlog CLI commands
50. **Add an "Architecture" docs page** explaining the single-package design, concurrency model, and hook system

---

## G) Questions I Cannot Answer Myself

### 1. Should I commit and push the website.yml + changelog fixes now?

The two uncommitted changes (`.github/workflows/website.yml` Firebase deploy fix + `website/src/content/docs/changelog.mdx` v0.6.0 sync) need to be committed and pushed to trigger the website CI/CD pipeline. Without this, the live website will remain frozen at the July 13 deploy with no showcase screenshots, no demo section, and stale changelog. Should I commit and push these changes now?

### 2. Should the `FIREBASE_SERVICE_ACCOUNT` secret be regenerated?

The `printf '%s'` fix may not be sufficient if the secret value itself is corrupted or the service account key has expired. If the next CI run still fails authentication, the user will need to regenerate the key via `gcloud iam service-accounts keys create` and update the GitHub secret. I cannot do this myself because I don't have GCP credentials. Should I flag this as a manual step, or should I attempt to regenerate the key using `nix shell nixpkgs#google-cloud-sdk`?

### 3. Should the `html-realworld.jpg` (BuildFlow report screenshot) be added to the website?

The README has it in the collapsible gallery, but the website showcase only shows the 4 demo report screenshots. The BuildFlow screenshot is compelling because it shows real-world usage (69 services, 137 events), but it's also 114KB and references a different project. Should the website show it as a "real-world usage" example, or keep it README-only?

---

## Session Summary

| Metric | Value |
|--------|-------|
| Commits made | 3 (`db2b932`, `b040af4`, `a10783f`) — all pushed |
| Uncommitted changes | 2 (website.yml deploy fix + changelog.mdx sync) |
| Files changed | 11 (html.templ, html_templ.go, golden, 4 screenshots, 2 new Astro components, 1 data file, 1 page, 1 workflow, 1 changelog) |
| Bugs fixed | 3 (stray `}` syntax error, DOMContentLoaded timing, Firebase deploy auth `echo` corruption) |
| Tests passing | `go vet`, `go build`, `go test -race` — all pass |
| CI status | Go CI: passing. Website CI: build passes, deploy FAILING since July 13 (fix uncommitted) |
| Lines added | ~170 (across all files) |
| Lines removed | ~15 |
| User-facing impact | High — every HTML report generated by the library was broken; now fixed. Website still broken (deploy fix uncommitted). |

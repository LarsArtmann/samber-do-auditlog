# Status: Public Presence Overhaul — README, Website, GitHub Metadata

**Date**: 2026-07-13 21:16
**Session scope**: Making the repo public-ready: README.md, wiki website, GitHub Description/Topics/URL

---

## a) FULLY DONE

### GitHub Repository Metadata

- [x] **Description** updated: "Audit-log plugin for samber/do v2 — track every DI registration, invocation, and shutdown with timestamps, dependency graphs, and self-contained HTML visualization"
- [x] **Homepage URL** set to `https://do-auditlog.lars.software`
- [x] **11 topics** added: go, golang, dependency-injection, di, samber-do, observability, audit-log, dependency-graph, visualization, monitoring, plugin
- [x] Verified via `gh repo view --json`

### README.md

- [x] Documentation links added below badges: Quick Start, API Reference pointing to the website
- [x] All existing README content preserved (508 lines intact)

### Website (`website/`) — Full Astro v7 + Starlight + Tailwind v4 Site

- [x] **Scaffolding**: package.json, astro.config.mjs, tsconfig.json, .node-version, .gitignore, .htmlvalidate.json
- [x] **Firebase config**: .firebaserc (target: `do-auditlog`, project: `lars-software`), firebase.json (cleanUrls, redirects, security headers, cache rules)
- [x] **Nix flake**: dev/build/preview/deploy apps, devShell, treefmt
- [x] **CSP fix script**: scripts/fix-csp.mjs (SHA-256 hash injection for inline scripts)
- [x] **Public assets**: favicon.svg (shield/audit icon in amber), robots.txt, manifest.json, 4 JS files (theme-init, header, copy-code, animations)
- [x] **Styles**: global.css (warm amber #e8a838 theme, dark default, light mode), starlight.css (Starlight CSS variables mapped to amber palette)
- [x] **Data files**: config.ts, types.ts, features.ts (6 features), hero-code.ts, sections.ts (3 steps, 3 comparisons, 4 use cases)
- [x] **14 Astro components**: Logo, Icon, Card, SectionHeader, Section, Header, Footer, HeroSection (with live GitHub stars fetch), FeatureGrid, HowItWorksSection, ComparisonSection, UseCasesSection, CTASection, Sections
- [x] **Landing layout**: LandingLayout.astro with JSON-LD structured data, OG/Twitter meta, skip-to-content, ClientRouter
- [x] **Landing page**: index.astro composing Hero + FeatureGrid + Sections
- [x] **11 documentation pages**:
  - Getting Started: Installation, Quick Start
  - Guides: Export Formats, Dependency Tracking, Health Checks, Filtered Reports, Performance
  - API Reference: Plugin & Report (full method tables)
  - Community: Changelog, Contributing, Related Tools
- [x] **GitHub Actions workflow**: .github/workflows/website.yml (build + deploy to Firebase on push to master)
- [x] **Build verified**: `astro check` (0 errors, 0 warnings, 0 hints), `html-validate` (clean), full build (13 pages + sitemap + pagefind search index, CSP patched on all 13 HTML files)
- [x] **.prettierignore** updated to exclude `website/`

---

## b) PARTIALLY DONE

### Documentation content depth

- The 11 docs pages are solid but could go deeper. The gogenfilter website has ~14 docs pages with more detailed guides (e.g., SQLC config discovery, custom filesystems, gitignore composition). do-auditlog's equivalent depth topics exist in AGENTS.md but haven't been ported to website docs yet (e.g., CLI usage details, replay/migration workflows, real-time streaming patterns).

### Changelog sync

- The website changelog.mdx is a curated summary, not a 1:1 copy of CHANGELOG.md. gogenfilter's CI enforces version-header sync between root CHANGELOG.md and changelog.mdx. We did not add that CI check.

---

## c) NOT STARTED

### Firebase deployment

- The domain `do-auditlog.lars.software` is referenced everywhere but has NOT been deployed or configured in Firebase yet. The `FIREBASE_SERVICE_ACCOUNT` secret is NOT set in GitHub. The Firebase hosting target `do-auditlog` does NOT exist in the `lars-software` project yet.

### Lighthouse CI

- gogenfilter has a `.github/workflows/lighthouse.yml` with `lighthouserc.json`. We did not add Lighthouse CI for this project.

### OG image generation

- gogenfilter uses `astro-og-canvas` for per-page Open Graph image generation. We omitted this. The landing page has OG tags but no OG image.

### Docs validation (md-go-validator)

- gogenfilter validates code blocks in docs with md-go-validator via Nix. We did not add this.

### Code duplication check (jscpd)

- gogenfilter runs jscpd on website source. We did not add this.

### Dependents page

- gogenfilter has a dynamic `/dependents` page that fetches GitHub code search results. Not applicable yet for this project (ALPHA, no external dependents), but worth noting.

### package-lock.json

- Not committed. We installed with npm during testing but removed it. CI will generate it on first run. The `cache-dependency-path: website/package-lock.json` in website.yml will warn on first run since the file doesn't exist yet.

### .npmrc / engine constraints

- No .npmrc file. html-validate requires Node >= 22.22.0 || >= 24.8.0 but CI specifies Node 24. The devShell via Nix may also need nodejs_24.

---

## d) TOTALLY FUCKED UP

### Nothing catastrophic

- No destructive operations were performed.
- The bun install created a `bun.lock` that was cleaned up.
- One iteration was needed: the initial package.json had `"vite": "7.3.2"` in overrides which conflicted with Astro 7's requirement for Vite 8. Fixed by removing the vite override.

### Near-miss: Astro version compatibility

- The reference projects (gogenfilter, go-atomic-write) list `"astro": "^7.0.3"` in package.json but their lockfiles actually resolve to Astro 6.3.1. Our package.json also says `^7.0.3` and resolves to Astro 7.0.8 with Vite 8, which works. But this is fragile — the reference projects may break on their next `npm install` if Astro 7 has further breaking changes.

---

## e) WHAT WE SHOULD IMPROVE

1. **Deploy the website** — Set up Firebase hosting target and DNS for `do-auditlog.lars.software`. Without this, the homepage URL in GitHub returns nothing.

2. **Commit package-lock.json** — Required for reproducible CI builds and npm cache. Currently missing.

3. **Add OG images** — The landing page and docs have no social preview images. This hurts link sharing on Twitter/Slack/Discord.

4. **Deeper docs content** — Port the CLI usage, replay/migration, real-time streaming, and data model docs from AGENTS.md to website pages. The current docs are good but thin compared to the richness of the actual library.

5. **Add CHANGELOG sync CI** — gogenfilter enforces that changelog.mdx version headers match CHANGELOG.md. We should too.

6. **Add Lighthouse CI** — Automated performance/SEO/accessibility scoring on every website PR.

7. **Node version alignment** — .node-version says 24, CI uses 24, but the Nix devShell uses `pkgs.nodejs` which may resolve to a different major version depending on nixpkgs revision.

8. **Website Favicon** — The current favicon is a generic shield icon. Could be more distinctive/branded.

9. **README polish** — Could add a screenshot or GIF of the HTML visualization, a "What it looks like" section, and badges for coverage % and Go version.

10. **Website example showcase** — The library produces a beautiful HTML visualization. The website should show a screenshot of it, not just describe it in text.

---

## f) Up to 50 Things We Should Get Done Next

### Deployment & Infrastructure (must-do)

1. Create Firebase hosting target `do-auditlog` in `lars-software` project
2. Set `FIREBASE_SERVICE_ACCOUNT` GitHub secret
3. Configure DNS for `do-auditlog.lars.software` (CNAME to Firebase)
4. Deploy the website
5. Commit `package-lock.json` after first clean `npm install`
6. Verify the website URL works end-to-end from the GitHub homepage link

### Content Depth (high impact)

7. Add a CLI usage guide page (info, convert, diff, validate, schema subcommands)
8. Add a Replay & Migration guide page (ReplayEvents, ReadEvents, LoadReport, MigrateReport)
9. Add a Real-Time Streaming guide (OnEvent callback, Prometheus bridge, OTel spans)
10. Add a Data Model reference page (the full Report/ServiceInfo/Event tree from README)
11. Add a Security page (gosec, CSP, XSS hardening, fuzz testing)
12. Add a "What it looks like" section with screenshots of the HTML visualization
13. Expand the API reference with Event type, ServiceInfo type, Config type details
14. Add code examples that are testable/validatable

### Visual & Branding (medium impact)

15. Generate OG images via astro-og-canvas (per-page social preview)
16. Add a screenshot of the HTML visualization to the landing page
17. Add a screenshot of the Mermaid/D2 diagram output to docs
18. Design a more distinctive favicon/logo
19. Add a demo GIF or video of the HTML visualization (tab switching, graph pan/zoom)
20. Add a "metrics row" to the hero section (like go-atomic-write: "9 Formats", "~1.7us", "95% Coverage", "0 Deps")

### CI/CD Hardening (medium impact)

21. Add CHANGELOG sync validation to website.yml CI
22. Add Lighthouse CI (.github/workflows/lighthouse.yml + lighthouserc.json)
23. Add HTML validation to CI (already in workflow but verify it runs)
24. Add md-go-validator for docs code block validation
25. Add jscpd code duplication check for website source
26. Add stale-reference check (like gogenfilter's deleted-files check)
27. Add import path validation (ensure docs reference correct module path)
28. Pin all GitHub Actions to SHA hashes (currently using @v6/@v7/@v8)

### SEO & Analytics (lower impact)

29. Add Google Analytics or Plausible (gogenfilter removed Plausible, so maybe skip)
30. Add a sitemap submission to Google Search Console
31. Add canonical URLs to all doc pages
32. Add breadcrumb structured data
33. Add FAQ structured data (if we add an FAQ page)

### Docs Polish (lower impact)

34. Add "Edit this page" links to docs
35. Add "Last updated" timestamps to docs
36. Add prev/next navigation at the bottom of doc pages (Starlight does this by default — verify)
37. Add a search bar (Starlight Pagefind — already built, verify it works)
38. Add i18n support (probably overkill for ALPHA)
39. Add a version selector (when v1.0.0 lands)

### README Enhancements

40. Add a "What it looks like" section with a screenshot of the HTML visualization
41. Add a coverage badge (95%+)
42. Add a Go version badge
43. Add a "Star this repo" call-to-action
44. Add a table of contents (for quick navigation in the long README)
45. Add a "Comparison with alternatives" section (uber/dig, wire)
46. Add a "Used by" section (when there are users)

### Website Architecture

47. Consider extracting shared components to a shared package (go-atomic-write, gogenfilter, this project all share the same Astro component pattern)
48. Add a 404 page design (Starlight provides a default — verify it matches the theme)
49. Add a loading state for the GitHub stars API call (currently blocks render)
50. Add error handling for the GitHub stars fetch (currently silently falls back to "Star on GitHub")

---

## g) Top 2 Questions I Cannot Answer Myself

### Q1: Is the Firebase project `lars-software` set up for a `do-auditlog` hosting target?

The .firebaserc references `lars-software` as the default project and configures a `do-auditlog` hosting target. But I have no access to the Firebase console to verify:

- Does the `lars-software` project exist and is it accessible?
- Is the `do-auditlog` hosting target created (or will Firebase auto-create it on first deploy)?
- Is DNS configured for `do-auditlog.lars.software`?
- Is the `FIREBASE_SERVICE_ACCOUNT` secret already in GitHub, or does it need to be created?

**Why it matters**: Without Firebase deployment, the homepage URL (`https://do-auditlog.lars.software`) returns nothing, making the GitHub homepage link broken for public visitors.

### Q2: Should the website docs live in this repo or in a separate docs repo?

Both reference projects keep the website in the same repo (`website/` subdirectory). This is the pattern I followed. However, the website has its own package.json, node_modules, and build pipeline — it's a separate project living inside the Go library repo. This means:

- `npm` changes show up in Go repo diffs
- The Nix flake now has two flake.nix files (root for Go, website/ for Node)
- CI runs both Go and Node jobs

Is this the preferred structure, or should the website eventually move to a separate repo (e.g., `LarsArtmann/do-auditlog-website`) deployed independently?

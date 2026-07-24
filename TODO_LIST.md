# TODO List

Short- and mid-term improvement tasks, verified against actual code state.
Completed items are in [CHANGELOG.md](CHANGELOG.md).
Last updated: 2026-07-24

---

## live/ Sub-Package

- [ ] **Create `live/demo/main.go`** — Self-contained example that registers services with delays, invokes them, and shuts down, showing the live dashboard updating in real time. This is the most important missing UX artifact.
- [ ] **Add scope tree tab** to the live dashboard JS (present in the static templ dashboard but missing from the live version).
- [ ] **Add "Show all" pagination** for services and events tables in the live dashboard.
- [ ] **Share CSS** between the static templ dashboard and the live dashboard to prevent drift. Extract common design tokens into a shared file or `go:embed`.
- [ ] **Add CORS headers** for cross-origin dashboard embedding (currently missing on all `live/` API endpoints).
- [ ] **Integrate live dashboard into `example/` app** — Add a `--live` flag or a live endpoint to show the dashboard alongside the existing ride-sharing domain demo.
- [ ] **Add export buttons** (JSON/NDJSON/HTML) to the live dashboard for downloading the current report snapshot.

## Publishing & Release

- [ ] **Publish `go-sse` and `go-ndjson`** to GitHub and remove the `replace` directives from `go.mod`. Both are currently local-only (`replace ... => ../go-sse`, `replace ... => ../go-ndjson`).
- [ ] **Create GitHub Releases** for v0.1.0 through v0.6.0. The Releases page shows v0.0.4 as "Latest" while the actual latest tag is v0.6.0. Extract release notes from CHANGELOG.md.
- [ ] **Tag and release v0.7.0** once all [Unreleased] items in CHANGELOG.md are verified.

## Quality

- [ ] **Pin GitHub Actions to SHA hashes** — Workflows in `.github/workflows/` use `@v4`/`@v5` tag versions, which are vulnerable to supply-chain attacks. Pin all actions to commit SHAs.
- [ ] **Add headless browser test** for the HTML report's JavaScript execution. The golden test only checks byte-for-byte equality, not JS runtime correctness. A stray `}` syntax error shipped undetected because of this gap.
- [ ] **Add a note to the README Mermaid example** — The node IDs are simplified for readability; real output includes UUID-based scope prefixes. Users who copy the example see different output.
- [ ] **Fix timeline screenshot aspect ratio** — 1400x1100 vs 1400x1300 mismatch creates visual inconsistency in the website showcase grid.
- [ ] **Touch-accessible "Click to enlarge"** — The website screenshot hover-zoom is not discoverable on touch devices. Add a tap handler or visible affordance.

---

## Not Planned (Explicitly Rejected)

- **Multi-module split** — Project is too small (1 package, ~2500 LOC core + live/ sub-package). Revisit at 5+ packages.
- **External storage backends** — File and `io.Writer` exports are sufficient.
- **Prometheus/OpenTelemetry integration as a dependency** — Out of scope. Use `OnEvent` callback instead.
- **`samber/lo` dependency** — Current stdlib `slices`/`cmp` usage is sufficient.
- **`encoding/json/v2` migration in this project** — Current `encoding/json` works fine. The transitive dependency through `go-output` requires `GOEXPERIMENT=jsonv2` but this project's own code does not and should not import `encoding/json/v2`. Risk of breaking JSON output format for consumers.

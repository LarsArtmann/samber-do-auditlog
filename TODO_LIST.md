# TODO List

Short- and mid-term improvement tasks, verified against actual code state.
Completed items are in [CHANGELOG.md](CHANGELOG.md).
Last updated: 2026-07-24

---

## Bugs & Regressions

- [ ] **Fix coverage gate** — Combined coverage dropped to **91.4%** (below 94% gate). Root package is 93.9%; `live/` sub-package is 76.6%. The `scripts/coverage-gate.sh` script fails until `live/` coverage improves. Add tests for `handleDashboard` prefix injection, `handleReport` nil-provider error path, `handleSSE` without `http.Flusher`, and `normalizePrefix` edge cases.
- [ ] **Fix build without `GOEXPERIMENT=jsonv2`** — `go-ndjson` imports `encoding/json/v2`, requiring the experimental flag in Go 1.26.x. Either wait for Go 1.27 to stabilize json/v2, or migrate `go-ndjson` to standard `encoding/json`.
- [ ] **Fix README "Loading & Migrating Reports" code block** — Has undefined variables (`oldJSONBytes`, `ndjsonFile`) and unused-variable compile errors. The verification was done on rewritten code, not the actual README snippet.
- [ ] **Fix README "zero exemptions" claim** — Security & Quality table says "zero exemptions" but `.golangci.yml` has extensive `*_test.go`, `cmd/`, and `example/` path exclusions. Change to "minimal exemptions for tests and tooling".
- [ ] **Fix footer timestamp in `html.templ`** — Uses `new Date().toLocaleString()` (viewer's local time) instead of `report.exported_at` (generation time). Misleading for offline reports.

## live/ Sub-Package

- [ ] **Create `live/demo/main.go`** — Self-contained example that registers services with delays, invokes them, and shuts down, showing the live dashboard updating in real time. This is the most important missing UX artifact.
- [ ] **Fix lint warnings in `live/`** — ~14 golangci-lint warnings remain: `exhaustruct` (struct literal completeness), `varnamelen` (w/r parameter names — standard net/http convention), `errchkjson`, `gci` import ordering, `modernize` (`interface{}` to `any`).
- [ ] **Add scope tree tab** to the live dashboard JS (present in the static templ dashboard but missing from the live version).
- [ ] **Add "Show all" pagination** for services and events tables in the live dashboard.
- [ ] **Share CSS** between the static templ dashboard and the live dashboard to prevent drift. Extract common design tokens into a shared file or `go:embed`.
- [ ] **Add CORS headers** for cross-origin dashboard embedding (currently missing on all `live/` API endpoints).
- [ ] **Integrate live dashboard into `example/` app** — Add a `--live` flag or a live endpoint to show the dashboard alongside the existing ride-sharing domain demo.

## Publishing & Release

- [ ] **Publish `go-sse` and `go-ndjson`** to GitHub and remove the `replace` directives from `go.mod`. Both are currently local-only (`replace ... => ../go-sse`, `replace ... => ../go-ndjson`).
- [ ] **Create GitHub Releases** for v0.1.0 through v0.6.0. The Releases page shows v0.0.4 as "Latest" while the actual latest tag is v0.6.0. Extract release notes from CHANGELOG.md.

## Quality

- [ ] **Pin GitHub Actions to SHA hashes** — Workflows in `.github/workflows/` use `@v6`/`@v7`/`@v8` tag versions, which are vulnerable to supply-chain attacks. Pin all actions to commit SHAs.
- [ ] **Add headless browser test** for the HTML report's JavaScript execution. The golden test only checks byte-for-byte equality, not JS runtime correctness. A stray `}` syntax error shipped undetected because of this gap.
- [ ] **Add a note to the README Mermaid example** — The node IDs are simplified for readability; real output includes UUID-based scope prefixes. Users who copy the example see different output.
- [ ] **Fix timeline screenshot aspect ratio** — 1400×1100 vs 1400×1300 mismatch creates visual inconsistency in the website showcase grid.
- [ ] **Touch-accessible "Click to enlarge"** — The website screenshot hover-zoom is not discoverable on touch devices. Add a tap handler or visible affordance.

---

## Not Planned (Explicitly Rejected)

- **Multi-module split** — Project is too small (1 package, ~2500 LOC core + live/ sub-package). Revisit at 5+ packages.
- **External storage backends** — File and `io.Writer` exports are sufficient.
- **Prometheus/OpenTelemetry integration as a dependency** — Out of scope. Use `OnEvent` callback instead.
- **`samber/lo` dependency** — Current stdlib `slices`/`cmp` usage is sufficient.
- **`encoding/json/v2` migration in this project** — Current `encoding/json` works fine. The `go-ndjson` dependency uses it, but this project's own code does not and should not. Risk of breaking JSON output format for consumers.

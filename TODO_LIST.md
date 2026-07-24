# TODO List

Short- and mid-term improvement tasks, verified against actual code state.
Completed items are in [CHANGELOG.md](CHANGELOG.md). Rejected proposals are in [ROADMAP.md](ROADMAP.md).
Last updated: 2026-07-24

---

## live/ Sub-Package

- [ ] **Share CSS** between the static templ dashboard and the live dashboard to prevent drift. Extract common design tokens into a shared file or `go:embed`. `live/base_css.go`, `html.templ`
- [ ] **Fix `example/ --live` premature shutdown** — The `runLive()` function immediately calls `server.Shutdown()` after the lifecycle completes, making the dashboard disappear within seconds. Should wait for Ctrl+C (like `live/demo/main.go` does). `example/main.go`
- [ ] **Add `Healthchecker` implementations to `live/demo` services** — The demo calls `plugin.RecordHealthCheck(injector)` but none of the demo service structs implement `do.Healthchecker`, so the health-check section of the dashboard shows nothing. `live/demo/main.go`

## Publishing & Release

- [ ] **Publish `go-sse` and `go-ndjson`** to GitHub and remove the `replace` directives from `go.mod`. Both are currently local-only (`replace ... => ../go-sse`, `replace ... => ../go-ndjson`). `go.mod`
- [ ] **Create GitHub Releases** for v0.1.0 through v0.6.0. The Releases page shows v0.0.4 as "Latest" while the actual latest tag is v0.6.0. Extract release notes from CHANGELOG.md.
- [ ] **Tag and release v0.7.0** once all [Unreleased] items in CHANGELOG.md are verified.

## Quality

- [ ] **Add headless browser test** for the HTML report's JavaScript execution. The golden test only checks byte-for-byte equality, not JS runtime correctness. A stray `}` syntax error shipped undetected because of this gap. `html_templ.go`, `fuzz_test.go`
- [ ] **Add a note to the README Mermaid example** — The node IDs are simplified for readability; real output includes UUID-based scope prefixes. Users who copy the example see different output. `README.md`
- [ ] **Fix timeline screenshot aspect ratio** — 1400x1100 vs 1400x1300 mismatch creates visual inconsistency in the website showcase grid. `docs/images/`
- [ ] **Touch-accessible "Click to enlarge"** — The website screenshot hover-zoom is not discoverable on touch devices. Add a tap handler or visible affordance.

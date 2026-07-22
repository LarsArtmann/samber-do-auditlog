# TODO List

Comprehensive list of improvement tasks, verified against actual code state.
Last updated: 2026-07-22

---

## Priority 1 — Release & CI

- [x] **Push v0.0.3** — tag `v0.0.3` and `master` confirmed on remote via `git ls-remote --tags origin` (2026-06-17)
- [x] **GitHub Release for v0.0.3** — created with CHANGELOG notes + `audit-report.html` artifact (2026-06-17)
- [x] **CI pipeline** — `.github/workflows/ci.yml` runs test (with -race), vet, build, lint (golangci-lint v2.12.2) on push and PR (2026-06-17)
- [x] **govulncheck in CI** — `vulncheck` job uses `golang/govulncheck-action` on every push and PR (2026-06-17)
- [x] **Stale-generation check in CI** — `stale-generation` job installs templ@v0.3.1020 (go.mod pin), runs `go generate`, fails on any diff (2026-06-17)

## Priority 2 — Robustness & Testing

- [x] **`deriveServiceStatus` property tests** — exhaustive 16-case matrix (2^4 inputs) + priority-ordering + nil-pointer-semantics tests in `status_internal_test.go` (2026-06-17)
- [x] **`MaxEvents` concurrency stress test** — 50-goroutine stress + 20x repeat variant verifying `stored+dropped==total` invariant under `-race` in `robustness_test.go` (2026-06-17)
- [x] **Atomic-write crash path test** — rename-failure (target-is-dir) and write-error tests proving temp-file cleanup in `robustness_test.go` (2026-06-17)
- [x] **Migration round-trip test** — `TestMigrateReport_FullRoundTrip` downgrades v0.2.0 JSON to v0.1.0, migrates back, asserts all recomputed fields match + `Validate()` passes (2026-06-17)

## Priority 3 — Developer Experience

- [x] **`flake.nix` devShell** — pins Go 1.26.4, golangci-lint, govulncheck, golines; documents the go.mod templ version pin (v0.3.1020) alongside nixpkgs (2026-06-17)
- [x] **"Releasing" section in CONTRIBUTING.md** — documents tag → CHANGELOG → push → GitHub Release procedure + release-vs-schema-version distinction; also fixed stale `New()` example (2026-06-17)
- [x] **pkg.go.dev badge** — already present in README; all 7 godoc examples verified passing (`go test -run Example`) (2026-06-17)
- [x] **Benchmark baselines** — `BENCHMARKS.md` created with all 13 benchmarks (3 runs, median values) post-v0.0.3 for regression detection (2026-06-17)

## Priority 4 — Future API Exploration

- [x] **Streaming NDJSON export** — `Report.WriteNDJSON(writer)` added: streams events directly from the report's Events slice without defensive copy. Also added `Report.WriteJSON(writer)` (2026-06-17)
- [x] **`Report.Diff(other Report)`** — implemented in `diff.go` returning `DiffResult` with added/removed/changed services + event count delta. Tested with 5 test cases (2026-06-17)
- [x] **OpenTelemetry reference example** — `docs/examples/otel-bridge.md` shows how to bridge `Config.OnEvent` to OTel spans without adding a dependency (2026-06-17)
- [x] **0.x stability promise** — `STABILITY.md` documents stable vs evolving vs internal API surfaces, JSON schema versioning, and what "breaking" means in 0.x (2026-06-17)
- [x] **`go-output` adoption** — Adopted `github.com/larsartmann/go-output` for all diagram rendering (Mermaid/PlantUML/DOT), replacing the hand-rolled `diagramFormatter` pipeline (~237 LOC). Triggered when DOT (the 3rd format) shipped. See `docs/research/go-output-adoption-review.md` §8–9 (2026-06-21)
- [x] **D2 diagram export** — Added `Report.WriteD2` as the 4th diagram format via `go-output/d2`. Includes CLI `convert -f d2`, 5 unit tests, fuzz coverage, and full docs sync (2026-06-21)

## Not Planned (Explicitly Rejected)

- **Multi-module split** — Project is too small (1 package, ~2500 LOC). Revisit at 5+ packages.
- **External storage backends** — File and io.Writer exports are sufficient.
- **Prometheus/OpenTelemetry integration as a dependency** — Out of scope. Use OnEvent callback instead.
- **`samber/lo` dependency** — Current stdlib `slices`/`cmp` usage is sufficient for this project size.
- **`encoding/json/v2` migration** — Current `encoding/json` works fine. Risk of breaking JSON output format for consumers.

---

## Future Priorities (from status audit Top-25)

Sorted by impact × value ÷ effort.

### Architecture

- [ ] **Typed identifiers** — `ContainerID`, `ScopeID`, `ServiceName` as distinct named string types. Compiler rejects accidental swaps; validation moves into constructors. Low effort, high safety. **DEFERRED to a future breaking release** (v0.3.0 shipped as non-breaking tree/table export — typed identifiers need the next semver-major or pre-1.0 breaking batch): blast radius measured at 65+ compile errors across production + tests + generated templ, with zero existing bugs from string usage. Do as a single breaking-change batch alongside ServiceInfo split.
- [x] **NDJSON import** — Done via `ReadEvents` + `LoadReport` (auto-detecting loader).
- [x] **`Report` constructor validation** — `NewReport(...)` added: re-derives per-service Status + aggregates, enforces `Validate()`.
- [ ] **Split `ServiceInfo` lifecycle concerns** — 19-field struct into `ServiceIdentity` / `ServiceLifecycle` / `ServiceHealth` / `ServiceGraph`. Breaking API change; **DEFERRED to a future breaking release** (v0.3.0 shipped as non-breaking — this needs the next semver-major or pre-1.0 breaking batch): embedding breaks all struct literals (~50 sites), nesting breaks all field access; YAGNI applies (no consumer complaints, JSON flattens fine). Do alongside typed identifiers.
- [x] **JSON Schema generation** — Done: `schema/report.schema.json` generated by `cmd/genschema`, embedded via `go:embed`, exposed via `JSONSchema()`.

### Features

- [x] **CSV/TSV export** — Done (`Report.WriteCSV/WriteTSV` + `Plugin.*` wrappers).
- [x] **CLI tool** — Done: `cmd/auditlog` with info/convert/diff/validate/schema subcommands.
- [x] **WebSocket live stream** bridge for `OnEvent` — Done as reference example (`docs/examples/websocket-stream.md`).
- [x] **Prometheus exporter** example — Done (`docs/examples/prometheus-bridge.md`).
- [x] **samber/ro adapter** — Done as reference example (`docs/examples/samber-ro-adapter.md`).
- [x] **DOT diagram format** — Done (`Report.WriteDOT`, native formatter).

### Testing

- [x] **Property-based `Diff` tests** — Done: identity, added/removed duality, anti-symmetry, changed-symmetry (200 iters each).
- [x] **Property-based `MigrateReport` tests** — Done: always-valid, corrupt-repair, version-normalization, idempotency, core-data-preservation.
- [x] **HTML golden-file test** — Done (`testdata/golden/report.html`, `UPDATE_GOLDEN=1`).
- [x] **Fuzz filter inputs** — Done (`FuzzFilterInputs`).

### Release & CI

- [x] **v0.3.0 release** — Shipped 2026-06-21 as a **non-breaking** feature release (tree + table export formats). The typed-identifier + ServiceInfo-split breaking changes originally planned for v0.3.0 are deferred to a future breaking release (see Architecture section above). Also shipped v0.3.1 (daghtml SDK adoption + hook refactoring), v0.4.0 (go-output v0.30.0 API), v0.5.0 (go-output hotfix bump).
- [x] **JSON Schema file** for the report format — Done (`schema/report.schema.json`).
- [x] **Prometheus exporter example** parallel to the OTel example — Done.
- [x] **`actionlint`** in CI for workflow validation — Done.
- [x] **Coverage gate** — Done as a reusable script (`scripts/coverage-gate.sh`) + Nix app (`nix run .#coverage`); excludes `example/` + `cmd/`.
- [x] **Pre-commit hook** — Done (`scripts/hooks/pre-commit`).

---

## Completed (2026-06-17 — Post-Remediation Consolidation)

- [x] **Unified `Report` construction** — `buildReportFromCore()` + `finalizeDenormalized()` single path for `BuildReport`, `Filtered`, `MigrateReport`; eliminates 3-way duplication of 8 denormalized fields
- [x] **`ServiceInfo.DeriveStatus()` public method** — canonical status derivation moved to a method on the type it operates on
- [x] **Diagram special-char fuzz** — `FuzzDiagramSpecialChars` (3rd fuzz target): Mermaid/PlantUML structural integrity under adversarial input
- [x] **Nested scope tree coverage** — `TestNestedScopeExport` (table-driven): deep scope trees through migration + export. Originally planned as a fuzz target; consolidated to 3 fuzz targets total during the buildflow.
- [x] **`.gitattributes` for generated files** — `*_templ.go linguist-generated=true` prevents recurring `html_templ.go` format drift
- [x] **Test parallelism** — `t.Parallel()` on all eligible tests (env-var and fixed-path tests excluded)
- [x] **CHANGELOG/AGENTS/TODO docs sync** — refactor visibility, 3 fuzz targets, Go 1.26.4 everywhere

## Completed (2026-06-17 — v0.0.3 Release)

- [x] **Fix 5 lint regressions** — mnd, noinlineerr, varnamelen, exhaustruct from the perf/config commits; restored 0-issue baseline
- [x] **Reconcile CHANGELOG.md** — corrected `[0.1.0]`→`[0.0.1]` header, added missing `[0.0.2]`, wrote accurate `[0.0.3]` from verified diff
- [x] **Fix stale example run command** — `go run example/main.go` → `go run ./example` in AGENTS.md after the file split
- [x] **Cut v0.0.3 tag** — SSH-signed annotated tag at `acb098f`

## Completed (2026-06-14 — Feature Finalization)

- [x] **Go enum metadata injection** — `BuildTypeMetadata()` / `TypeMetadata` injected into HTML via `@templ.JSONScript`
- [x] **Diagram theme styling** — Mermaid `%%{init}%%` and PlantUML `skinparam` directives in `diagram.go`
- [x] **Touch event support** — 1-finger pan + 2-finger pinch-zoom on the dependency graph
- [x] **Pagination for large reports** — services table (50 rows) + events table (100 rows) with "show all" button
- [x] **HTML accessibility polish** — `aria-pressed` on filter chips, `scope="col"` on table headers, empty-state messages
- [x] **Debounced service search** — 150ms `setTimeout` debounce on search input
- [x] **Replace stripScriptTags** — `stripJSONScripts` with marker-based `<script type="application/json">` removal
- [x] **HTML integration test** — `TestWriteHTML_MultiServiceIntegration` with realistic multi-service end-to-end test
- [x] **Archive cleanup** — pruned stale docs (37 → 12 files in `docs/archive/`)
- [x] **Go Report Card badge** — added to README.md
- [x] **Atomic file writes** — temp-file + `os.Rename` for crash-safe exports
- [x] **Zero-allocation struct map key** — `serviceKey` uses struct key, not string concatenation
- [x] **Buffered export I/O** — 64KB `bufio` blocks (10–100x fewer syscalls)
- [x] **MaxEvents cap + InitialEventCapacity** — memory-bounded event capture with `DroppedEventCount()`
- [x] **Wire `Config.Validate()` into `New()`** — Breaking change: `New()` returns `(*Plugin, error)`
- [x] **Update all test files** — `mustNew()` test helper replaces `auditlog.New()` direct calls
- [x] **Update `example/`** — handle error from `New()` with `log.Fatalf`
- [x] **Harden CSP** — add `base-uri 'none'; frame-ancestors 'none'`
- [x] **Fix keyboard nav** — exclude `TEXTAREA`, `SELECT`, `BUTTON` from tab-shortcut handler
- [x] **Add `Report.Validate()`** — checks denormalized count fields match actual data
- [x] **Shared diagram formatter** — `diagramFormatter` interface with Mermaid/PlantUML implementations
- [x] **PlantUML export** — `Report.WritePlantUML(writer)`
- [x] **Complete HTML redesign** — warm amber "Container Telemetry" aesthetic, lifecycle waveform, 5-tab layout
- [x] **Robust fuzz XSS** — 3 fuzz targets with `stripJSONScripts` and 6+ injection-vector checks
- [x] **Split monolithic `recorder.go`** — into `hooks.go`, `report.go`, `report_builder.go`, `report_helpers.go`, `service.go`, `event.go`, `export.go`, `healthcheck.go`, `filter.go`, `metadata.go`
- [x] **Split `auditlog_test.go`** — into 14 feature-focused test files
- [x] **Split `example/main.go`** — into `register.go`, `services.go`, `summary.go`
- [x] **Pin Go 1.26.4** — in `go.mod` and `.golangci.yml` (bumped from 1.26.3)

## Completed (2026-06-10 — Sessions 1–6)

- [x] Fix broken Events tab — build allEvents array from report.events with full rendering
- [x] Fix XSS in deps column — esc() around d.service_name in dependency/dependent rendering
- [x] Fix XSS in status badge — esc() around s.status in CSS class attribute
- [x] Add CSP meta tag to HTML — Content-Security-Policy defense-in-depth
- [x] Expand fuzz tests — 3 targets (service names, error messages, dep chains) with 6+ XSS vectors
- [x] Add version guard to MigrateReport — return early if already current schema
- [x] Preserve ExportedAt in migration — only set time.Now() if original is zero
- [x] Validate input in MigrateReport — reject empty input and missing version
- [x] Add Config.Validate() real checks — validates ContainerID for path separators
- [x] Add ReportOption functional options and Report.Filtered(opts...)
- [x] Add Plugin.ReportFiltered + ExportFilteredToFile
- [x] Add Report.WriteMermaid(writer)
- [x] Add Report.Index() for O(1) lookups
- [x] Add ServiceStatus type with computed field on ServiceInfo
- [x] Add ProviderType named type with Icon()/String() methods
- [x] Add IsHealthchecker/IsShutdowner fields via enrichCapabilities()
- [x] Add health check auditing: EventTypeHealthCheck, RecordHealthCheck/WithContext
- [x] Add Config.OnEvent callback for real-time event streaming
- [x] Add Event convenience methods: IsRegistration, IsInvocation, IsShutdown, IsBefore, IsAfter
- [x] Rename DependencyRef to ServiceRef and embed in Event/ServiceInfo
- [x] Consolidate key format: serviceKey() as single canonical function
- [x] Single-lock Recorder optimization (4 mutexes → 1 RWMutex + 2 atomics)
- [x] Schema migration (MigrateReport v0.1.0 → v0.2.0)
- [x] 7 godoc examples for pkg.go.dev
- [x] Fix all golangci-lint issues (was 28 → 0)
- [x] Coverage: 95.5%, ~300 tests, 11 benchmarks

## Completed (2026-07-13 — Website & BuildFlow Remediation)

- [x] **Public documentation website** — Full Astro v7 + Starlight site at `do-auditlog.lars.software` with 11 docs pages, landing page, Firebase hosting, and CI deploy workflow
- [x] **BuildFlow go-auto-upgrade revert** — Reverted catastrophic `encoding/json/v2` migration; documented the incident in AGENTS.md and committed the revert (commit `fb56b6a`)

## Completed (2026-06-21 – 2026-07-07 — v0.2.0 through v0.5.0)

- [x] **go-output adoption** — All diagram rendering (Mermaid/PlantUML/DOT/D2) migrated to `github.com/larsartmann/go-output` renderers; hand-rolled `diagramFormatter` pipeline removed (~237 LOC)
- [x] **D2 diagram export** — `Report.WriteD2` as 4th diagram format via `go-output/d2`; CLI `convert -f d2`
- [x] **Tree export** — `Report.WriteTree` (ASCII) and `Report.WriteHTMLTree` (HTML nested list) via `go-output/tree` and `go-output/markup`
- [x] **Table export** — `Report.WriteTable` in 16+ formats via `go-output RenderTable`; CLI `convert` extended with `tree`, `htmltree`, `table`
- [x] **daghtml SDK adoption** — Replaced 306 lines of inline Sugiyama DAG JS in `html.templ` with `go-output/daghtml` SDK via `daghtml_adapter.go` (v0.3.1)
- [x] **Hook refactoring** — Centralized per-hook preamble into `hookContext` helpers in `hooks.go` (v0.3.1)
- [x] **go-output v0.30.0 API adoption** — `GraphStyle`→`NodeStyle`, `TableData`→`Table`, `RenderTableData`→`RenderTable` (v0.4.0)
- [x] **Nix flake modernization** — Migrated from `flake-utils`/`eachDefaultSystem` to `flake-parts` with `treefmt-nix`
- [x] **CSS token extraction** — Inline CSS color values extracted to named constants across `html.templ`, `html.go`, `daghtml_adapter.go`
- [x] **Test infrastructure centralization** — Shared fixtures and helpers extracted into `helpers_test.go`

## Completed (Historical)

- [x] Initial plugin structure with Config, New, Opts
- [x] Event capture for registration, invocation, shutdown
- [x] Stack-based dependency inference
- [x] Reverse dependency computation
- [x] Scope tree building
- [x] JSON report export
- [x] NDJSON event stream export
- [x] Self-contained HTML visualization
- [x] Environment variable toggle (DO_AUDITLOG_ENABLED)
- [x] Zero-cost disabled mode
- [x] Strict golangci-lint configuration
- [x] External test package

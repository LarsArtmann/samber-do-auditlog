# TODO List

Comprehensive list of improvement tasks, verified against actual code state.
Last updated: 2026-06-17

---

## Priority 1 — Release & CI

- [ ] **Push v0.0.3** — tag is signed locally but not pushed (`git push origin master --tags`)
- [ ] **GitHub Release for v0.0.3** — create a release with the CHANGELOG body and example HTML artifact
- [ ] **CI pipeline** — GitHub Actions workflow running `go test`, `go vet`, `golangci-lint run`, `go build` on push and PR. No CI exists today — the v0.0.3 lint regressions shipped undetected because of this gap.
- [ ] **govulncheck in CI** — scan dependencies for known CVEs on every push (gosec already runs via golangci-lint, but govulncheck is not installed or integrated)
- [ ] **Stale-generation check in CI** — fail if `go generate ./...` produces a diff (catches `html_templ.go` drift)

## Priority 2 — Robustness & Testing

- [ ] **`deriveServiceStatus` property tests** — the status state machine has 5 states and priority ordering; add table-driven tests covering every transition
- [ ] **`MaxEvents` concurrency stress test** — verify `DroppedEventCount` is accurate under racing hooks
- [ ] **Atomic-write crash path test** — simulate a rename failure (e.g. cross-device) to confirm temp-file cleanup runs
- [ ] **Migration round-trip test** — export at current schema, manually downgrade to v0.1.0, migrate back, assert equality

## Priority 3 — Developer Experience

- [ ] **`flake.nix` devShell** — pin Go 1.26.3, templ CLI, golangci-lint, govulncheck so contributors have identical tooling (resolves templ CLI version drift: local v0.3.1036 vs go.mod v0.3.1020)
- [ ] **"Releasing" section in CONTRIBUTING.md** — document the tag → CHANGELOG → push → GitHub Release procedure, including the release-vs-schema-version distinction
- [ ] **pkg.go.dev badge** — add to README and verify the 7 godoc examples render
- [ ] **Benchmark baselines** — re-run the 11 benchmarks post-v0.0.3 and record in a `BENCHMARKS.md` for regression detection

## Priority 4 — Future API Exploration

- [ ] **Streaming NDJSON export** — `Report.WriteNDJSON` currently materializes all events; a streaming `io.Reader` could serve very large reports without forcing MaxEvents drops
- [ ] **`Report.Diff(other Report)`** — useful for regression-testing DI graphs across deploys
- [ ] **OpenTelemetry reference example** — a docs-only example showing how to bridge `Config.OnEvent` to OTel spans (not a dependency)
- [ ] **0.x stability promise** — document what the API stability contract is for pre-1.0 consumers, given `New()` already broke in v0.0.3

## Not Planned (Explicitly Rejected)

- **Multi-module split** — Project is too small (1 package, ~2300 LOC). Revisit at 5+ packages.
- **External storage backends** — File and io.Writer exports are sufficient.
- **Prometheus/OpenTelemetry integration as a dependency** — Out of scope. Use OnEvent callback instead.
- **`samber/lo` dependency** — Current stdlib `slices`/`cmp` usage is sufficient for this project size.
- **`encoding/json/v2` migration** — Current `encoding/json` works fine. Risk of breaking JSON output format for consumers.

---

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
- [x] **Pin Go 1.26.3** — in `go.mod` and `.golangci.yml`

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

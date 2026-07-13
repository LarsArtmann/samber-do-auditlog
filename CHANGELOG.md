# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> **Release vs. schema versions.** Release tags follow `v0.x.y`. The report
> `schema_version` (currently `0.2.0`, see `types.go`) is a **separate**, independent
> version for the JSON report format and is upgraded via `MigrateReport`. The two
> version numbers are unrelated.

## [Unreleased]

### Added

- **Public documentation website**: full Astro v7 + Starlight + Tailwind v4 site deployed at `do-auditlog.lars.software` with 11 docs pages, landing page, Firebase hosting, and CI deploy workflow.

### Changed

- **Nix flake modernization**: migrated from `flake-utils`/`eachDefaultSystem` to `flake-parts` with `treefmt-nix` for standardized formatting checks.
- **Nix build check**: added `build` output as a check in the treefmt configuration.
- **Dependency updates**: `golang.org/x/sync` v0.21.0 → v0.22.0, `golang.org/x/sys` v0.46.0 → v0.47.0, `golang.org/x/term` v0.44.0 → v0.45.0, nixpkgs bumped to latest.

### Fixed

- **BuildFlow `go-auto-upgrade` revert**: reverted a catastrophic automated migration to `encoding/json/v2` + `encoding/json/jsontext` (build-constraint-excluded in Go 1.26.4) that left the project uncompilable. Documented the incident and the `encoding/json/v2` exclusion policy in `AGENTS.md`.

## [0.5.0] - 2026-07-07

A patch release: go-output hotfix bump. Non-breaking.

### Changed

- **go-output v0.30.0 → v0.30.1**: picked up upstream patch fixes across all go-output modules (root, d2, daghtml, escape, graph, plantuml, delimited, markdown, markup, serialization, table, tree, testhelpers). Fixed stale module checksums.

## [0.4.0] - 2026-07-06

A maintenance release: go-output v0.30.0 API adoption. Non-breaking — internal API renames only, no public API changes.

### Changed

- **go-output v0.23.x → v0.30.0**: adopted the go-output v0.30.0 API across diagram and table rendering code.
  - `d2.NewD2Diagram()` → `d2.NewDiagram()`.
  - `output.GraphStyle` → `output.NodeStyle` (diagram theming).
  - `output.TableData` → `output.Table`, `output.NewTableData()` → `output.NewTable()`, `output.RenderTableData()` → `output.RenderTable()` (table export).

## [0.3.1] - 2026-07-02

A maintenance release: daghtml SDK adoption, hook refactoring, and dependency upgrades. Non-breaking — internal refactors only.

### Changed

- **daghtml SDK adoption**: replaced 306 lines of inline Sugiyama DAG JavaScript in `html.templ` with the `go-output/daghtml` SDK via a new `daghtml_adapter.go`. The graph rendering logic is now maintained upstream, reducing the HTML template from ~950 to ~640 lines.
- **Hook refactoring**: centralized per-hook preamble logic (context creation, locking, scope recording) into `hookContext` helpers in `hooks.go`, reducing duplication across all six hook handlers.
- **CSS token extraction**: extracted inline CSS color values into named constants across `html.templ`, `html.go`, and `daghtml_adapter.go` for maintainability.
- **Test infrastructure**: centralized test fixtures and extracted shared helpers into `helpers_test.go`.
- **Go 1.26.4**: bumped Go version and updated tooling dependencies.

### Fixed

- **Golden file stability**: updated `testdata/golden/report.html` to reflect the daghtml-rendered graph output.

## [0.3.0] - 2026-06-21

A feature release: tree and table export formats, plus CLI coverage for all new formats. Non-breaking — additive only.

### Added

- **Tree export** (`Report.WriteTree`, `Report.WriteHTMLTree`): dependency DAG rendered as an ASCII tree (`go-output/tree.ASCIITreeRenderer`) or an HTML nested-list tree (`go-output/markup.HTMLTreeRenderer`). Nodes are labeled with service name and provider-type icon.
- **Plugin-level tree wrappers**: `Plugin.WriteTree`, `WriteHTMLTree` (to `io.Writer`) and `Plugin.ExportToTree`, `ExportToHTMLTree` (to file path).
- **Table export** (`Report.WriteTable`): service summary table in 16+ formats via `go-output RenderTableData` — including table, json, csv, tsv, markdown, xml, d2, yaml, html, tree, mermaid, dot, jsonl, asciidoc, toml, and plantuml. Configured via `DefaultTableOpts()`.
- **Plugin-level table wrappers**: `Plugin.WriteTable(writer, format, opts)` and `Plugin.ExportToTable(path, format, opts)`.
- **CLI `convert` subcommand extensions**: `auditlog convert` now accepts `tree`, `htmltree`, and `table` as output formats, and infers them from `.tree`, `.htmltree`, and `.table` file extensions.

## [0.2.0] - 2026-06-21

A feature release: D2 diagram export, go-output adoption for all diagram rendering, and Plugin-level API parity across all export formats. Non-breaking — additive only.

### Added

- **D2 diagram export** (`Report.WriteD2`): a fourth diagram format, produced
  via `github.com/larsartmann/go-output/d2`. Each node carries the warm-amber
  per-node style; the diagram title is set to the container ID for
  self-documenting output. D2 edges are deduplicated via a local
  `dedupGraphEdges` helper because the D2 renderer does not expose `DedupEdges`.
- **CLI `convert -f d2`**: the `auditlog convert` subcommand now accepts `d2`
  as an output format and infers it from the `.d2` file extension.
- **Plugin-level diagram wrappers**: `Plugin.WriteMermaid`, `WritePlantUML`,
  `WriteDOT`, `WriteD2` (to `io.Writer`) and `Plugin.ExportToMermaid`,
  `ExportToPlantUML`, `ExportToDOT`, `ExportToD2` (to file path). Completes
  Plugin API parity — all 9 export formats now have both Write and Export
  methods, matching the existing JSON/NDJSON/HTML/CSV/TSV wrappers.

### Changed

- **Diagram rendering adopted `go-output`**: the hand-rolled
  `diagramFormatter` interface, three formatter structs, and seven escaping
  helpers (~237 LOC) were replaced by
  `github.com/larsartmann/go-output` renderers (`graph.MermaidRenderer`,
  `plantuml.PlantUMLDiagram`, `graph.DOTRenderer`). Escaping is now delegated
  to go-output's validated `escape` package. Net external dependency delta:
  `+1` (`golang.org/x/term`; `x/sys` was already present).

### Deprecated

- Nothing.

### Removed

- Nothing.

### Fixed

- Nothing.

### Security

- Nothing.

## [0.1.0] - 2026-06-19

A milestone release: first CLI release, replay engine, NDJSON import/export,
JSON Schema generation, CSV/TSV/DOT export, comprehensive self-review
remediation, and breaking API cleanup.

### Added

- **Replay engine** (`ReplayEvents`): reconstructs a `Report` from a flat event
  stream — the inverse of hook-based recording. Processes already-captured
  events to rebuild service/scope state, then assembles a `Report` via the same
  `buildReportFromCore` finalizer. Returns `Report{Reconstructed: true}` so
  consumers can detect inherent limitations (no `IsHealthchecker`/
  `IsShutdowner`, inferred scope hierarchy).
- **NDJSON reader** (`ReadEvents`): reads line-delimited JSON events with
  blank-line skipping and a 1 MB max-line guard. Enables round-trip
  export → import workflows.
- **Loader API** (`LoadReport` + variants): auto-detects JSON vs NDJSON by
  inspecting the first non-blank line. JSON routes through `MigrateReport`;
  NDJSON routes through `ReadEvents` + `ReplayEvents`. Variants:
  `LoadReportFromReader`, `LoadReportFromBytes`.
- **`Format` enum and `WithFormat` option**: explicit format selection for
  the loader (`FormatAuto`, `FormatJSON`, `FormatNDJSON`).
- **`Report.Reconstructed` field**: boolean flag distinguishing replayed
  reports from live-recorded ones.
- **CSV/TSV export** (`Report.WriteCSV`, `Report.WriteTSV`): writes all services
  as comma- or tab-delimited values with a header row. Uses stdlib
  `encoding/csv`. Dependencies/dependents are semicolon-separated refs. Nil
  pointer fields render as empty strings. Enables data-analysis workflows
  (Excel, pandas, jq on TSV).
- **Plugin CSV/TSV methods**: `Plugin.WriteReportCSV`, `Plugin.WriteReportTSV`,
  `Plugin.ExportToCSV`, `Plugin.ExportToTSV` wire through to the Report methods.
- **DOT diagram export** (`Report.WriteDOT`): Graphviz digraph of the dependency
  graph (native formatter, zero dependencies). Joins Mermaid and PlantUML as the
  third diagram format.
- **JSON Schema** (`JSONSchema()`): the canonical Draft 2020-12 schema for the
  report format, generated from Go types by `cmd/genschema` and embedded in the
  library via `go:embed`. Consumers can validate exported JSON or generate
  clients from `schema/report.schema.json`.
- **CLI tool** (`cmd/auditlog`): `auditlog info|convert|diff|validate|schema`
  subcommands for inspecting, converting (json/ndjson/csv/tsv/html/mermaid/
  plantuml/dot), diffing and validating reports offline. Install via
  `go install ./cmd/auditlog` or `nix run .#auditlog`.
- **`NewReport` constructor**: builds a validated `Report` from core data,
  re-derives per-service `Status` and all aggregate fields, and enforces
  `Validate()`. The public counterpart to the internal `buildReportFromCore`.
- **`Report.WriteHTML`**: renders the self-contained HTML visualization from a
  loaded `Report` (not just a live `Plugin`), enabling offline rendering and the
  CLI `convert -f html` path.
- **Property-based tests**: algebraic `Diff` properties (identity, symmetry,
  anti-symmetry) and `MigrateReport` round-trip/repair properties (200
  iterations each, deterministic seeds).
- **HTML golden-file test**: deterministic report → byte-stable HTML comparison
  (`testdata/golden/report.html`, `UPDATE_GOLDEN=1` to regenerate).
- **Filter fuzz target** (`FuzzFilterInputs`): arbitrary `ReportOption`
  combinations never panic and always yield valid subset reports.
- **Reference examples**: Prometheus bridge, WebSocket live stream, and
  samber/ro reactive adapter docs under `docs/examples/`.
- **Developer tooling**: `actionlint` in CI + devShell; pre-commit hook
  (`scripts/hooks/pre-commit`); coverage-gate script (`scripts/coverage-gate.sh`);
  Nix flake apps `.#auditlog` and `.#coverage`.

### Changed

- **Deduplication refactoring**: unified scope-tree builders via a generic
  `buildScopeTreeFromMeta[T]()` (replaced two ~50-line recursive builders),
  extracted shared helpers (`getOrCreateServiceRecord`,
  `recordDependencyFromStack`, `buildServiceDeps`, `depRecToRef`,
  `sortServiceInfos`, `scopeServicesForServices`), and converted 5 switch-
  statement enum methods to map-based lookups. Net ~60 LOC removed from
  production code with zero remaining production clones at `-t 15`.
- **HTML scope tree**: collapsible/expandable scope nodes and waveform visual
  improvements.
- **`MergeReports`**: combines multiple reports from different containers into
  one, concatenating events with sequence offsets and merging services/scopes.
- **`stats` CLI subcommand**: prints aggregate statistics — invocation counts,
  error rates, build time averages, provider type and status breakdowns.
- **`Format.String()`**: returns "auto"/"json"/"ndjson" for human-readable
  error messages.
- **Enum validation on ingest**: `ReadEvents` now rejects events with unknown
  `event_type` or `phase` values. All four enums (`EventType`, `Phase`,
  `ProviderType`, `ServiceStatus`) have `IsKnown()` methods.
- **Schema drift detection test**: `TestSchemaNoDrift` compares the committed
  schema file against the embedded `JSONSchema()` to catch type/schema drift.
- **`Report.Validate()` version check**: rejects reports with empty Version.
- **CLI `version` command**: prints CLI version (ldflags-injectable) and schema
  version. **CLI `--help` to stdout**: Unix convention for explicit help.
- **CLI `--` end-of-options marker**: `reorderFlags` now handles `--`.
- **`PopStackFrame`**: shared LIFO stack-pop logic between hooks.go and replay.go.

### Breaking

- **Removed `LoadReportFromJSON`**: one-line alias for `MigrateReport`. Use
  `MigrateReport` directly.
- **Removed `LoadReportFromNDJSON`**: duplicate of `LoadReportFromReader(r,
  FormatNDJSON)`. Use `LoadReportFromReader` with `FormatNDJSON`.

### Changed

- **Unified scope metadata**: `scopeMeta` and `replayScopeMeta` merged into a
  single type, eliminating 6 duplicated accessor functions.
- **Unified service assembly**: `buildServicesLocked` and `buildReplayServices`
  collapsed into `buildServicesFromMap` — the most dangerous split brain.
- **Unified ServiceRef sorting**: single `CompareServiceRefs` (ServiceName
  primary, ScopeID secondary) replaces `compareByName` + `sortServiceRefs`.
- **`trimWhitespace` → `bytes.TrimSpace`**: stdlib handles Unicode whitespace.
- **Removed pointless wrappers**: `serviceRefLabel`, `buildDepsLocked`,
  `computeServiceStatus` inlined.
- **`CLIVersion`**: changed from const to ldflags-overridable var.

### Fixed

- **`invocationOrder` always 0 in replay** (critical): the replay engine
  computed `invocationOrder = invocationCount - 1`, always 0 since the branch
  fires on first invocation. Every replayed report silently lost cross-service
  build ordering. Fixed with a global `invocationSeq` counter.
- **CLI error handling**: `loadFile` no longer calls `os.Exit`; `flag.ExitOnError`
  → `flag.ContinueOnError`; `validate` error includes filename.
- **CLI stdin support**: path `"-"` reads from stdin via `LoadReportFromReader`.
- **Doc drift**: README "5 export formats" → 8; "~1μs" → "~1.7μs"; missing API
  methods; `doc.go` graph type corrected.
- **NDJSON enum validation**: rejects unknown `event_type`/`phase` on ingest.
- **`%w: %w` double-error-wrap** in replay.go:98 → `%w: %v`.

### Changed (prior to self-review)

- **Split-brain audit fixes** (5 findings resolved):
  - **Status consistency drift** (SB-01): `Report.Validate()` now checks that
    every `ServiceInfo.Status` matches `DeriveStatus()`. `MigrateReport` always
    re-derives status instead of preserving stale non-empty values. A report
    with `Status="active"` + `InvocationError="boom"` now fails validation.
  - **Duplicate error detection** (SB-02): `diff.go` `hasError()` checked raw
    pointers while `FailedServices()` checked the status enum — they could
    disagree on stale reports. Consolidated to `Status.IsError()` as the single
    path; `hasError()` deleted.
  - **Manual struct field copy** (SB-03): The 16-field `serviceRecord` →
    `ServiceInfo` mapping is centralized in `serviceRecordToInfo()` so any new
    field only needs wiring in one place.
  - **Enum metadata asymmetry** (SB-04): `ServiceStatus.Icon()`, `EventType.Label()`,
    `EventType.Color()`, and `ProviderType.Label()` methods are now the single
    source of truth — `BuildTypeMetadata()` calls them instead of duplicating
    hardcoded values.
  - **Duplicate JSON encoding paths** (SB-05): `Plugin.WriteReportJSON()` and
    `Plugin.ExportFilteredToFile()` now delegate to `Report.WriteJSON()`,
    eliminating three independent encoder setups.

- **Diagram output escaping**: Service names containing brackets (`]` `[` `{` `}`),
  quotes (`"`), or newlines previously produced malformed Mermaid and PlantUML
  output — the `id[label]` syntax would break, rendering the diagram invalid.
  Unified ID sanitization (`sanitizeDiagramID`) now strips non-identifier runes
  after replacing separators. Mermaid labels are escaped via `mermaidLabel`
  (brackets/braces → parentheses, quotes → apostrophes). PlantUML labels are
  escaped via `plantumlLabel` (quotes → apostrophes). The two divergent ID
  replacers (Mermaid had 4 chars, PlantUML had 7) are replaced by a single
  shared function.

- **Stack overflow in recursive scope-tree walkers**: deeply nested scope trees
  (500+ levels) caused stack overflow. Consolidated to a table-driven
  implementation that avoids unbounded recursion.
- **HTML sort indicator CSS**: unicode escape sequences were mangled by templ
  code generation, causing incorrect sort-direction glyphs. Corrected.

## [0.0.4] - 2026-06-17

### Fixed

- **`MigrateReport` scope counting**: removed the divergent `countUniqueScopes`
  helper; migration now reuses `countScopeNodes`, so an empty scope tree reports
  `scope_count: 0` instead of `1` and matches `Report.Validate()`.
- **`MigrateReport` normalizes current-schema reports too**: previously it only
  upgraded v0.1.0 input. It now re-derives every denormalized field for _any_
  input version, so stale or hand-edited current-schema reports also pass
  `Validate()`. The implied contract shifts from "upgrade old → current" to
  "repair/normalize → current".
- **Fragile scope assertion in health checks**: `injector.(*do.Scope)` was
  replaced with a `scopeAncestorWalker` interface assertion that works with
  `*do.RootScope`, `*do.Scope`, and future wrappers.
- **Dead JS timestamps in HTML**: the services-table tooltip referenced
  non-existent `registered_offset_ns`/`first_invoked_offset_ns` fields and an
  unused `formatNs()` helper. Both removed; the tooltip now uses the real
  `registered_at`/`first_invoked_at` ISO timestamps.
- **Stale-generation CI gate was perpetually red**: `html_templ.go` was committed
  with multi-line imports, but templ v0.3.1020 (pinned via the go.mod `tool`
  directive) deterministically emits single-line import statements. Re-committed
  the actual `go tool templ generate` output so the working tree matches CI.

### Changed

- **Templ CLI now managed via Go `tool` directive**: `go get -tool` pins the exact
  version in `go.mod`. `go generate` calls `go tool templ generate` — no external
  binary needed. Eliminates the Nix-vs-go.mod version drift that plagued v0.0.3.
  Removed `templ` from `flake.nix` and the `go install` step from CI.
- **Go 1.26.3 → 1.26.4** across `go.mod`, `.golangci.yml`, and the flake devShell.
- **Removed experimental `goexperiment.*` build tags** from `.golangci.yml`
  (notably `goexperiment.jsonv2`) that contradicted the stdlib-only policy.
- **Capability map refactor**: `report_builder.go` replaced an opaque
  `map[string][2]bool` with a named `capabilityFlags{isHealthchecker,
isShutdowner}` struct.
- **Unified `Report` construction**: `BuildReport`, `Filtered`, and
  `MigrateReport` now route through a single `buildReportFromCore()` +
  `finalizeDenormalized()` path. The eight denormalized aggregate fields
  (counts, durations, health/shutdown success) are derived in exactly one
  place, making count/summary drift structurally impossible. Any future
  Report construction path (e.g. NDJSON import) inherits correct aggregates
  for free.

### Added

- **`ServiceInfo.DeriveStatus() ServiceStatus`**: public method that computes
  the canonical status from the service's error pointers and lifecycle
  timestamps. Single source of truth for status derivation, reusable beyond
  report building and migration.

- **CI coverage gate** (`.github/workflows/ci.yml`): the test job now produces a
  coverage profile, excludes the `example/` demo package, and fails if production
  statement coverage drops below 95%.
- **`go mod tidy` drift check**: a new `mod-tidy` CI job fails if `go.sum` is out
  of sync with `go.mod`.
- **`golangci-lint config verify`** step in the lint job, run before
  `golangci-lint run`, so a malformed config fails fast.
- **CI pipeline** (`.github/workflows/ci.yml`): four parallel jobs — test (with
  `-race`), lint (golangci-lint v2.12.2), vulnerability scan (govulncheck), and
  stale-generation check (catches `html_templ.go` drift).
- **`Report.WriteNDJSON(writer)`**: streams events as NDJSON directly from the
  report without a defensive copy.
- **`Report.WriteJSON(writer)`**: writes the full report as indented JSON.
- **`Report.Diff(other Report) DiffResult`**: structural comparison of two
  reports (added/removed/changed services, event count delta).
- **`flake.nix` devShell**: pins Go 1.26.4, golangci-lint, govulncheck, golines
  for contributor reproducibility.
- **`BENCHMARKS.md`**: post-v0.0.3 baseline of all 13 benchmarks for regression
  detection.
- **`STABILITY.md`**: 0.x API stability promise (stable vs evolving vs internal).
- **OTel reference example** (`docs/examples/otel-bridge.md`): shows how to
  bridge `Config.OnEvent` to OpenTelemetry spans without adding a dependency.
- **"Releasing" section** in `CONTRIBUTING.md` documenting the release procedure
  and the release-vs-schema-version distinction.

### Tests

- `FuzzDiagramSpecialChars`: third fuzz target — seeds Mermaid and PlantUML
  exporters with special characters (`]`, `"`, `-->`, `@enduml`, `%%`, pipes,
  newlines, 500-char strings) and verifies structural integrity.
- `TestNestedScopeExport` (table-driven): generates scope trees up to 500
  levels deep, normalizes via `MigrateReport`, exports to JSON + Mermaid +
  PlantUML. Guards against stack overflow in recursive tree walkers. (Planned
  as a fuzz target; consolidated to the 3 fuzz targets above during the buildflow.)
- `FuzzMigrateReport`: second fuzz target — arbitrary JSON → migrate → validate,
  with a re-migration round-trip property and seven seed corpora.
- `BuildTypeMetadata` unit tests covering every provider/status/event emoji,
  label, and color.
- `NewRecorder` internal test verifying constructor initialization and that an
  empty recorder yields a valid report.
- `t.Parallel()` added to ~120 independent top-level tests and ~33 subtests
  across 15 test files; env-var and fixed-path tests excluded.
- `deriveServiceStatus` exhaustive 16-case property test (all 2^4 input
  combinations + priority ordering).
- `MaxEvents` concurrency stress test (50 goroutines, 20x repeat, `-race`).
- Atomic-write crash path tests (rename failure + write error cleanup).
- Migration full round-trip test (v0.2.0 → downgrade → migrate → assert
  equality).
- Diff tests (identical, added/removed, changed, new-error).

## [0.0.3] - 2026-06-17

First release cut from a clean lint baseline. A complete HTML redesign, new
observability and memory controls, export performance work, and a codebase-wide
split of the monolithic source files into focused modules.

### Breaking

- **`New(Config)` now returns `(*Plugin, error)`**: `Config.Validate()` is enforced
  at construction time. All callers must handle the returned error. Tests use the
  new `mustNew()` helper.

### Added

- **Memory-bounded event capture**: `Config.MaxEvents` caps the number of stored
  events; `Config.InitialEventCapacity` pre-sizes the events slice to avoid
  reallocation; `Plugin.DroppedEventCount()` reports how many events were discarded
  once the cap is reached. Prevents OOM in long-running processes.
- **`Report.Validate()`**: verifies internal consistency of the denormalized count
  fields (`EventCount`, `ServiceCount`, `ScopeCount`, `HealthCheckedCount`) against
  the actual data.
- **`BuildTypeMetadata()` / `TypeMetadata`**: single source of truth for provider
  icons, status icons, event labels, and event colors, injected into the HTML via
  `@templ.JSONScript` so the client reads metadata instead of hardcoding constants.
- **HTML pagination**: the services table shows the first 50 rows and the events
  table the first 100; a "show all" button reveals the remainder. Search and filter
  bypass the limits.
- **Touch support**: one-finger pan and two-finger pinch-zoom on the dependency graph.
- **Accessibility**: ARIA roles, labels, and `sr-only` text across the HTML report.
- **Diagram themes**: Mermaid `%%{init}%%` and PlantUML `skinparam` directives using
  the warm amber palette.
- **Shared diagram formatter**: a `diagramFormatter` interface removes duplication
  between the Mermaid and PlantUML exporters.
- **Hardened CSP**: `base-uri 'none'; frame-ancestors 'none'` to block base injection
  and clickjacking.
- **Robust fuzz XSS**: `stripJSONScripts` replaces the old character-by-character
  parser; three fuzz targets now cover service names, error strings, and dependency
  chains with six-plus injection-vector checks.
- **`mustNew()` test helper** and shared providers/assertions in `helpers_test.go`.
- Security and archive documentation, an integration test, and planning/status reports.

### Performance

- **Atomic file writes**: exports write to a temp file in the target directory then
  `os.Rename` it into place, so a crash leaves the previous file intact instead of a
  partial write.
- **Zero-allocation struct map key** for `serviceKey`, removing the per-key string
  concatenation allocation.
- **Buffered export I/O**: a 64 KB `bufio` block batches writes, cutting syscall
  count 10–100x compared to writing straight to `os.File`.

### Changed

- **Complete HTML redesign**: warm amber "Container Telemetry" aesthetic with a
  lifecycle waveform signature element, a five-tab layout (Services / Scopes /
  Graph / Timeline / Events), stat cards, and keyboard navigation.
- **Pinned Go 1.26.3** in `go.mod` and the lint config.

### Fixed

- Bugs, lint failures, and XSS gaps surfaced by a full code review.

### Refactor

- **Monolith split**: `recorder.go` decomposed into focused modules — `hooks.go`,
  `report.go`, `report_builder.go`, `report_helpers.go`, `service.go`, `event.go`,
  `export.go`, `healthcheck.go`, `filter.go`, `metadata.go`.
- **Test split**: the monolithic `auditlog_test.go` split into 14 feature-focused
  test files.
- **Example split**: `example/main.go` split into `register.go`, `services.go`, and
  `summary.go`.
- Removed dead code and stale comments.

## [0.0.2] - 2026-06-11

### Fixed

- Commit the generated `html_templ.go` so the project builds without running the
  `templ` generator (Nix build compatibility).

## [0.0.1] - 2026-06-10

First alpha release. An audit-log plugin for samber/do v2 that records every DI
container lifecycle event with timestamps, dependency-graph inference, build-duration
tracking, health-check auditing, and export to JSON / NDJSON / self-contained HTML.

### Added

- Drop-in plugin: `New(Config)` + `Opts()` → one-line integration.
- Service registration, invocation, shutdown, and health-check tracking.
- Stack-based dependency-graph inference with reverse dependencies.
- Scope tree with hierarchical visualization.
- Provider-type detection (`lazy` / `eager` / `transient` / `alias`).
- Configurable `OnEvent` callback for real-time observability.
- JSON, NDJSON, Mermaid, PlantUML, and self-contained HTML export.
- Report filtering (`Report.Filtered`) with five filter options.
- Schema migration (`MigrateReport`) upgrading v0.1.0 JSON reports to the current
  schema.
- Zero-cost disabled mode via `DO_AUDITLOG_ENABLED`.
- Concurrent-safe single-lock `Recorder` design.
- ~95% test coverage, 140 tests, 11 benchmarks.
- XSS-hardened HTML with a Content-Security-Policy.
- Strict `golangci-lint` configuration.

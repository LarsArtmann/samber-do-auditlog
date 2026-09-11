# AGENTS.md — samber-do-auditlog

Go plugin for [samber/do v2](https://github.com/samber/do) that records every DI container lifecycle event (registration, invocation, shutdown) with timestamps, dependency graph inference, build duration tracking, and export to JSON / NDJSON / self-contained HTML.

**Module**: `github.com/larsartmann/samber-do-auditlog` · **Package**: `auditlog` · **Go**: 1.26.7 (go.mod + devShell) · **Status**: BETA (per [STABILITY.md](STABILITY.md); internal 1.0 bar tracked in ROADMAP.md)

---

## Commands

| Command                                                                             | Purpose                                                                                                                                   |
| ----------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| `go generate ./...`                                                                 | Regenerate templ (and any other generated code)                                                                                           |
| `go test ./...`                                                                     | Run all tests                                                                                                                             |
| `go test -race ./...`                                                               | Run all tests with race detector (CI uses this)                                                                                           |
| `GOEXPERIMENT=jsonv2 go test -race -coverprofile=cover.out -covermode=atomic ./...` | Tests with coverage (CI gate: ≥94% of non-`example/`/`cmd/` code) — `GOEXPERIMENT=jsonv2` is set automatically in the Nix devShell and CI |
| `go test -run TestPlugin_DisabledIsNoOp`                                            | Run single test                                                                                                                           |
| `go vet ./...`                                                                      | Static analysis (**`GOEXPERIMENT=jsonv2` required** — set automatically in Nix devShell)                                                  |
| `golangci-lint config verify`                                                       | Validate the lint config (CI runs this before `lint run`)                                                                                 |
| `golangci-lint run`                                                                 | Full lint (heavy config, see below)                                                                                                       |
| `go mod tidy`                                                                       | Sync `go.sum` (CI `mod-tidy` job fails on drift)                                                                                          |
| `nix develop`                                                                       | Enter devShell (Go 1.26.7, golangci-lint, govulncheck, actionlint, golines, **GOEXPERIMENT=jsonv2 enabled**)                              |
| `go run ./example`                                                                  | Run the example (set `DO_AUDITLOG_ENABLED=true`)                                                                                          |
| `go run ./example --live`                                                           | Example + real-time SSE dashboard                                                                                                         |
| `go run ./cmd/auditlog help`                                                        | CLI: info/convert/diff/validate/stats/schema subcommands                                                                                  |
| `go install ./cmd/auditlog`                                                         | Install the `auditlog` CLI to `$GOBIN`                                                                                                    |
| `nix run .#auditlog -- help`                                                        | Run the CLI via Nix (no install)                                                                                                          |
| `nix run .#coverage`                                                                | Run the CI-equivalent coverage gate via Nix                                                                                               |
| `sh scripts/coverage-gate.sh`                                                       | Coverage gate (exclusions single-sourced in `scripts/coverage-exclusions.txt`; ≥94%)                                                      |
| `sh scripts/check-go-version.sh`                                                    | Go-version drift guard: go.mod == ci.yml == flake GOTOOLCHAIN == .golangci.yml (CI + pre-commit)                                          |
| `sh scripts/check-doc-claims.sh`                                                    | Claims linter: go version, schema version, coverage gate, linter count, fuzz count vs machine truth (pre-commit)                          |
| `sh scripts/check-changelog-sync.sh`                                                | CHANGELOG.md ↔ website changelog.mdx version-list sync (website CI)                                                                       |
| `git config core.hooksPath scripts/hooks`                                           | Install the pre-commit hook                                                                                                               |

A `flake.nix` devShell is available for Nix users. No Makefile, no justfile.

---

## Architecture

Single-package library (`auditlog`) with these source files:

```
plugin.go           — Public API: New(), Opts(), Enable(), SetOnEvent(), Report(), Export*(), Write*(), Events(), RecordHealthCheck*
recorder.go         — Core state machine: event capture, invocation stack, service aggregation
hooks.go            — Hook methods + shared helpers (newEventFromRef, newServiceRecordCore, inferServiceType, getOrCreateServiceRecord, recordDependencyFromStack)
types.go            — Domain enums: EventType, Phase, ProviderType, ServiceStatus, ServiceRef
metadata.go         — TypeMetadata struct + BuildTypeMetadata() — Go enum display metadata for HTML
event.go            — Event type + convenience methods (IsRegistration, Duration, etc.)
service.go          — ServiceInfo (split into ServiceIdentity/ServiceLifecycle/ServiceHealth/ServiceGraph embedded structs), ScopeNode types + methods (Uptime, HasHealthError, DeriveStatus)
report.go           — Report type + Validate() + query methods (ServiceByName, EventsByType, Index, WriteNDJSON, WriteJSON, etc.) + buildReportFromCore/finalizeDenormalized (unified construction)
diff.go             — Report.Diff(other) + DiffResult/ServiceDiff types (incl. AddedDeps/RemovedDeps)
report_builder.go   — BuildReport assembly: services, scope tree, capability enrichment, shared helpers (serviceRecordToInfo, buildServiceDeps, sortServiceInfos)
report_helpers.go   — Report aggregate helpers (sumBuildMs, deriveServiceStatus, etc.)
replay.go           — ReplayEvents: reconstructs Report from a flat event stream (inverse of hook-based recording)
ndjson.go           — ReadEvents + StreamEvents: re-exports go-ndjson reader + callback-based NDJSON line scanner
filter.go           — Report filtering (Filtered, ReportOption, WithServicesByName, etc.)
healthcheck.go      — Health check recording (RecordHealthCheck, ResolveServiceScope)
export.go           — Shared diagram label helpers (serviceLabel, serviceRefLabel)
diagram.go          — go-output-backed graph builder: buildDiagramNodes/Edges, warmAmberNodeStyle, diagramNodeID, writeRendered
mermaid.go/plantuml.go/dot.go/d2.go — The 4 diagram exports (go-output renderers; graphID="do_auditlog", rankdir configurable)
html.go             — HTML export entry points (Plugin.WriteHTML delegates to Report.WriteHTML)
html.templ          — Templ template for self-contained HTML visualization (CSS + JS)
html_templ.go       — Generated by `go tool templ generate` (DO NOT EDIT)
daghtml_adapter.go  — Bridges go-output/daghtml Sugiyama DAG SDK into the HTML template graph renderer
loader.go           — LoadReport: auto-detecting loader (JSON via MigrateReport, NDJSON via ReadEvents + ReplayEvents)
migration.go        — MigrateReport: upgrades older JSON reports to current schema, re-derives denormalized fields + Status
csv.go              — WriteCSV/WriteTSV (stdlib encoding/csv)
tree.go             — ASCII tree + HTML nested-list tree (go-output tree/markup renderers)
table.go/table_options.go — Service summary table via go-output RenderTable (16+ formats); WithColumns selects 10 columns
diagram_options.go  — DiagramOption/WithDirection: layout direction for all 4 diagram formats
schema.go           — go:embed of schema/report.schema.json + JSONSchema() + //go:generate directive
design_tokens.go    — DesignTokensCSS: canonical CSS design tokens shared static + live (TestDesignTokensInSync enforces sync with html.templ)
shared_components.go — SharedComponentCSS: canonical keyboard-nav overlay CSS (TestSharedComponentCSSInSync enforces sync)
classify.go         — Error classification: registers all sentinel errors into go-error-family families (Corruption/Rejection) via init()
stream.go           — NDJSONStreamer: real-time NDJSON event streaming via Config.OnEvent (auto-flush, buffer size, WithFlushInterval)
multi_writer.go     — MultiWriter: event fan-out to multiple OnEvent callbacks (thread-safe, ordered)
runid.go            — RunID: 128-bit hex branded string type (crypto/rand)
doc.go              — Package doc comment (incl. the GOEXPERIMENT note)
schema/             — report.schema.json: Draft 2020-12 JSON Schema generated by cmd/genschema
cmd/genschema/      — JSON Schema generator (tooling-only; imports invopop/jsonschema — never linked by the library)
cmd/auditlog/       — CLI binary: info/convert/diff/validate/stats subcommands (stdlib flag, no deps)
testhelpers/        — Exported test helpers (JS syntax validation, etc.) for downstream integration testing
scripts/hooks/      — pre-commit hook (generate drift + vet + lint + test); install via `git config core.hooksPath scripts/hooks`
scripts/            — coverage-gate.sh, coverage-exclusions.txt, check-go-version.sh, check-doc-claims.sh, check-changelog-sync.sh
example/            — Self-checking demo with 23 samber/do v2 features
live/               — Real-time SSE dashboard sub-package (see below)
```

**Note:** The `health/` sub-package was extracted to its own project: [github.com/larsartmann/go-health](https://github.com/larsartmann/go-health). The `*Plugin` type satisfies `go-health`'s `HealthRecorder` interface implicitly via `RecordHealthCheckWithContext`.

### `live/` sub-package files

```
live/hub.go         — Hub: facade over sse.Broadcaster[sse.Event] (subscriber buffer 128, ring-buffer replay via ReplayBufferSize, SignalComplete, OnEvent, EventStore()/BufferedEventCount(), Shutdown/Health). Defines sseEventType.
live/server.go      — HTTP server with 6 endpoints (dashboard, report JSON, SSE, health, export NDJSON, export HTML), configurable prefix, CORS, graceful shutdown, CSP response header.
live/replay.go      — Ring-buffer EventStore + reconnection replay on top of sse.Replay.
live/fragments.go   — Go helpers for fragment rendering: signal structs, renderAllFragments(), pure helpers (humanizeDuration, providerIcon, computeWaveformMarks…), renderToString.
live/fragments.templ / fragments_templ.go — Templ components for all dashboard sections (stats, legend, waveform, services/events tbody, scope tree, graph, timeline, footer).
live/fragments_internal_test.go / fragments_render_internal_test.go — Internal tests for fragment helpers + renderers.
live/dashboard.go   — Dashboard HTML template (datastar attributes: data-signals, data-init, data-bind, data-show, data-class:active, data-on:click). Embeds datastar.js via go:embed.
live/dashboard.css  — Dashboard stylesheet (warm amber theme, responsive)
live/dashboard.js   — Keyboard nav (handleKeydown), export helpers, scope tree toggle, keyboard help dialog (~223 lines)
live/datastar.js    — Datastar v1.0.2 runtime (~56KB, go:embed). SSE parsing, DOM morphing by element ID, reactive signals.
live/base_css.go    — Composes shared tokens (auditlog.DesignTokensCSS + SharedComponentCSS) + live aliases + base component styles
live/doc.go         — Package doc comment
live/server_test.go / replay_test.go — External tests: server lifecycle, SSE streaming, handler edge cases, CORS, exports, replay
live/demo/          — Self-contained real-time demo (services implement do.Healthchecker)
```

### Data Flow

1. User creates `Plugin` via `New(Config)` → gets `*do.InjectorOpts` via `Opts()`
2. Hooks fire on every registration/invocation/shutdown → `Recorder` captures timestamped `Event`s
3. Invocation stack (`Recorder.stack`) infers dependencies: if service A is on-stack when B's before-hook fires, A→B is a dependency
4. `Report()` / `BuildReport()` assembles `ServiceInfo` slice (with forward + reverse deps), scope tree, and event stream
5. Export methods serialize to JSON, NDJSON, or self-contained HTML
6. Health checks: user calls `plugin.RecordHealthCheck[WithContext](injector)` instead of `injector.HealthCheck()` — wrapper records `EventTypeHealthCheck` events per service

### Concurrency Model

- **`sync.RWMutex` (`mu`)** protects core mutable state: `events`, `services`, `scopes`, `stack`, `shutdownStart`. One lock acquisition per hook.
- **`onEventMu sync.RWMutex`** (separate) guards the `onEvent` callback so `Plugin.SetOnEvent` can swap it after creation. `fireEvent` copies the callback under `RLock` and invokes the copy outside any lock.
- `sequence` and `invocationSeq` are `atomic.Int64`.
- The `onEvent` callback is always invoked outside the lock (hot path must not block).
- `BuildReport()` uses `mu.RLock()` — concurrent reads don't block each other.
- **MultiWriter** has its own `sync.Mutex`; callbacks fire sequentially per event in registration order.
- **RunID** is immutable once set; auto-generated in `New()` via `crypto/rand` (128-bit hex); `Config.RunID` (non-zero) overrides.

### Shared infrastructure: `go-sse` and `go-ndjson`

- [`go-sse`](https://github.com/larsartmann/go-sse) provides the full SSE lifecycle (`Stream`, `Broadcaster[T]`, `Replay`/`EventStore`, wire primitives). The domain Hub + Server are local; go-sse is transport-only. `go-sse/ssetest` (direct dep) parses SSE in tests.
- [`go-ndjson`](https://github.com/larsartmann/go-ndjson) owns NDJSON read/write + format detection; `loader.go`/`ndjson.go` re-export the public API.
- Both are public — no `replace` directives in go.mod. A `go.work` at the parent directory may link sibling projects for local dev.

### GOEXPERIMENT=jsonv2 requirement

**The project requires `GOEXPERIMENT=jsonv2` to build, and consumers need it too** — a downstream module that imports this library fails with `imports encoding/json/v2: build constraints exclude all Go files` without it (verified empirically with a minimal consumer). README Install + website Installation page document this; keep them in sync. Set automatically in: Nix devShell, CI workflows, `scripts/coverage-gate.sh`, direnv (`.envrc`), `.buildflow.yml`.

The requirement exists because `go-output` (diagram/table rendering), `go-branded-id` (transitive), and `go-ndjson` use `encoding/json/v2` features. This project's own code does NOT import `encoding/json/v2` (exclusion policy below). When Go 1.27 stabilizes `json/v2`, the flag requirement disappears.

### Go 1.26.7 toolchain pin

**The canonical Go version is 1.26.7**, pinned across (1) `go.mod`, (2) `.github/workflows/ci.yml`, (3) `flake.nix` (`GOTOOLCHAIN=go1.26.7`), (4) `CONTRIBUTING.md`/`BENCHMARKS.md`. `scripts/check-go-version.sh` asserts all four agree.

**Rule: bump `go-version` in ci.yml and `GOTOOLCHAIN` in flake.nix in the SAME commit as any `go.mod` bump.** GitHub runners set `GOTOOLCHAIN=local`; a CI go-version below go.mod's requirement fails every Go job instantly. (This exact mismatch broke all Go jobs on master once — see git history.)

**Gotcha:** a separately-installed `go` nix-store derivation can shadow the devShell's `go_1_26` on PATH; the `GOTOOLCHAIN` env var pins the effective toolchain anyway. Outside the devShell, prefix with `GOTOOLCHAIN=go1.26.7`.

**goreleaser coupling:** goreleaser v2.18.0+ declares `go >= 1.27.0` and fails under runner `GOTOOLCHAIN=local` on Go 1.26.x. CI pins goreleaser to `v2.17.1`; revisit when CI's Go reaches 1.27.

---

## CI

`.github/workflows/ci.yml` runs on every push and PR (plus a weekly cron) with 8 parallel jobs:

- **test**: `go vet`, `go build`, `go test -race` with coverage, and the **coverage gate** (≥94% of non-`example/`/`cmd/` statements; exclusions from `scripts/coverage-exclusions.txt`; per-function step summary) + `check-go-version.sh` drift guard.
- **lint**: `golangci-lint config verify` then the pinned `golangci-lint v2.12.2` run (binary cached).
- **vulncheck**: govulncheck pinned `@v1.7.0`.
- **mod-tidy**: `go mod tidy` fails on `go.sum` drift (transport-flake retry wrapper).
- **stale-generation**: `go generate ./...` fails on diff (uses `go tool templ` from the go.mod `tool` directive — no manual install) + committed-generated-files guard (a v0.9.0-retraction-class failure).
- **actionlint**: workflow lint.
- **goreleaser**: `goreleaser check` on `.goreleaser.yml`.
- **example-smoke**: runs the example's 23-feature self-check; fails if it exits non-zero.

`website.yml` builds/deploys the docs site on pushes touching `website/**` (see Website section below).

## Lint Configuration (.golangci.yml)

Extremely strict — nearly every golangci-lint linter enabled (~108). Key implications:

- **exhaustruct**: All struct fields must be explicitly initialized (tests exempted). `newEventFromRef()` and `newServiceRecordCore()` centralize field init to satisfy it in one place.
- **depguard**: per-path import rules — keep external non-stdlib deps out of `cmd/` (sole exception: `cmd/genschema` may use `invopop/jsonschema`).
- **noinlineerr**: `err := ...` then check, not `if err := ...; err != nil`.
- **forbidigo**: `fmt.Print*` forbidden in non-example code.
- **goconst config key is `min-len`, NOT `min-length`** — every golangci-lint version rejects `min-length` at `config verify` with exit 3, killing the CI Lint job before `lint run` even starts.
- Test relaxations (`*_test.go`): exhaustruct, testpackage, gochecknoglobals, funlen, cyclop, goconst.
- Formatters: gci, goimports, gofumpt, golines (max-len 120).

---

## Gotchas

- **Repo directory is `samber-do-metrics`** but `go.mod` says `samber-do-auditlog`. The module name is canonical.
- **JSON tags use snake_case** (`scope_name`, `service_name`, …) via `tagliatelle` — intentional for JSON API compatibility.
- **Package doc comment lives in `doc.go`** (with the GOEXPERIMENT note); `plugin.go` has no package comment.
- **`Report.Services` is sorted** by (scope_name, service_name); dependencies and dependents too — deterministic output across runs.
- **Test files use the external test package** (`auditlog_test`), importing the package under test as `auditlog`.
- **`example/` is exempt** from some lint rules (forbidigo, noinlineerr) — it's demo code; `cmd/` and `example/` are excluded from the coverage gate.
- **`buildReportFromCore()` is the single Report construction path** — `BuildReport`, `Filtered`, `MigrateReport`, and `ReplayEvents` all route through it + `finalizeDenormalized()`; `NewReport()` additionally re-derives per-service `Status` and enforces `Validate()`. **Critical invariant**: any new Report construction path MUST use `buildReportFromCore()` — never hand-compute aggregates, or they will drift and fail `Validate()`.
- **`serviceRecordToInfo()`** is the single `serviceRecord`→`ServiceInfo` conversion; Dependencies/Dependents/capability flags are left zero for the caller. Any new ServiceInfo field must be wired here.
- **`do.ExplainInjector()` MUST NOT be called from inside any hook** — it acquires internal locks that conflict with the hook execution context (deadlock). It's called by `enrichCapabilities()` in `BuildReport()` after releasing the RLock, using `*do.Scope` refs stored in `scopeMeta.ref`. Capabilities are only visible for invoked services (lazy providers must be built first).
- **Health checks use a wrapper pattern**, not hooks — samber/do v2 has no health-check hooks in `InjectorOpts`. `RecordHealthCheck[WithContext](injector)` wraps `injector.HealthCheckWithContext()`, records `EventTypeHealthCheck` events (**PhaseAfter only** — no interception point exists before the bulk check), and updates health fields. When disabled, it delegates directly. Per-service timing is unavailable from the bulk API (`DurationMs: nil`).
- **`Report.HealthCheckSucceeded` is `false` when no health checks ran** — `allHealthChecksPassed()` requires ≥1 health-checked service.
- **`serviceKey(scopeID, serviceName)`** is the canonical `scopeID + "/" + serviceName` key format.
- **Stack pop uses a LIFO fast path** (checks last element first, O(1) common case).
- **Disabled path is zero-cost**: `Opts()` returns empty hooks, so samber/do never calls recorder methods.
- **`New()` returns `(*Plugin, error)`** — `Config.Validate()` (rejects ContainerID path separators) is enforced at construction; tests use the `mustNew()` helper.
- **Typed identifiers**: `ContainerID`/`ScopeID`/`ServiceName` are named string types throughout. External library calls (go-output, csv, fmt) wrap with `string()` at the IO boundary. Test struct literals must use the embedded struct names (`ServiceIdentity: ServiceIdentity{ServiceRef: ...}`) — promoted fields can't be used in composite literals.
- **JSON Schema** (`schema/report.schema.json`) is GENERATED by `cmd/genschema` (`//go:generate`), go:embed'ded, exposed via `JSONSchema()`. Never hand-edit — change struct tags and regenerate. `invopop/jsonschema` is tooling-only.
- **Diagram rendering via `go-output`**: renderers (`graph.MermaidRenderer`, `plantuml.PlantUMLDiagram`, `graph.DOTRenderer`, `d2.D2Diagram`); `diagram.go` builds nodes/edges; escape via go-output's validated `escape` package. For formats lacking built-in edge dedup (D2), call `dedupGraphEdges()` before `SetEdges()`. Node IDs via `diagramNodeID()`, labels via `serviceLabel(svc)`.
- **Diagram theming**: `warmAmberNodeStyle` applied per-node (fills/strokes/fonts); edge line-colors and DOT `bgcolor` are NOT emitted (go-output lacks a graph-level bgcolor setter — re-introducing needs upstream support).
- **go-output sub-modules are mono-versioned in lockstep** — bump all together (see the go.mod comment). Old releases (≤ v0.31.1) had broken pseudo-version manifests for `testhelpers`; the indirect pins in go.mod track the family version.
- **CSP — static report** (`html.templ`): `default-src 'none'` + inline styles/scripts + Google Fonts + `base-uri 'none'`. **CSP — live dashboard** (`live/dashboard.go`): additionally `connect-src 'self'` (SSE) and **`script-src 'unsafe-eval'`** — REQUIRED because the embedded datastar.js compiles every `data-*` expression with the `Function()` constructor; without it the dashboard throws EvalErrors and renders nothing. **`frame-ancestors` is header-only** (browsers ignore it in `<meta>`): the live server sends `Content-Security-Policy: frame-ancestors 'none'` as a response header; the static `file://` report simply has no framing protection. `TestServer_DashboardCSP` guards the live contract.
- **`html.templ` XSS escaping**: all user-controlled strings use `esc()`.
- **Fuzz tests** (8 targets): `FuzzPluginHTML` (XSS; uses `stripJSONScripts()` to avoid false positives from JSON in `<script>` tags, checks 6+ vectors), `FuzzMigrateReport`, `FuzzDiagramSpecialChars`, `FuzzFilterInputs`, `FuzzReadEvents`, `FuzzMultiWriter`, `FuzzNDJSONStreamer`, `FuzzClassifyAdversarialChains`.
- **JS syntax validation test** (`plugin_html_syntax_test.go`): extracts `<script>` content, strips strings/comments/regexes via the `jsStripper` state machine (exported in `testhelpers/`), asserts delimiters balanced — catches syntax errors the golden byte-for-byte test misses.
- **CSS is shared, sync-tested**: `DesignTokensCSS` (design_tokens.go) and `SharedComponentCSS` (shared_components.go) are the single sources of truth; `html.templ` inlines them, `live/base_css.go` composes them. `TestDesignTokensInSync` + `TestSharedComponentCSSInSync` enforce byte-equality — change both or the tests fail.
- **Keyboard navigation**: both dashboards implement WAI-ARIA patterns (skip link, tablist roving tabindex, `?` help dialog with focus trap/restoration, `/` search, `e` errors-only [static], Esc, sortable headers with `aria-sort`). Static uses ES6, live uses ES5-compatible IIFE — shared JS extraction is intentionally NOT done.
- **Datastar-powered live dashboard**: server renders templ fragments → `datastar-patch-elements` SSE events (go-sse `SendLines`/`KeyedLines`) → datastar.js morphs DOM by element ID. Client state via datastar signals (`data-signals`/`data-bind`/`data-show`/`data-class`/`data-on:click`) — no hand-written rendering JS; `dashboard.js` is only keyboard nav/export helpers. The SSE handler coalesces bursts via `drainEvents`; reconnection sends a fresh full snapshot (the snapshot IS the replay).
- **`encoding/json/v2` exclusion policy**: no `.go` file in this project may import `encoding/json/v2`/`jsontext` (they're behind `//go:build goexperiment.jsonv2`; project targets Go 1.26.x). Transitive deps use it — hence GOEXPERIMENT. Revisit at Go 1.27.
- **BuildFlow config** (`.buildflow.yml`): sets `env: GOEXPERIMENT=jsonv2` (applied at pipeline startup; `config view` doesn't display it), `max_time: 5m` (default 2m is too short for 8 fuzz targets), and **permanently skips `go-auto-upgrade`** — its `jsonv1tov2` migrator rewrites `encoding/json`→v2+jsontext, breaking API calls and violating the exclusion policy above. Re-enable only when Go 1.27 lifts the policy.
- **Test helpers** (helpers_test.go + friends): `mkEvent`/`mkEventWithDur`/`mkInvAfterWithDur`, `mkRegEvent` (cmd), `setupWithDB`, `replayFromPlugin`, `newPluginAndInjector[WithID]`, `newPluginWithCapture`, assertion wrappers (`assertReportValid`, `assertErrIs`, …), struct factories (`rootRef`, `csvServiceRef`, `mkNewReport`). Use these instead of inline struct literals to keep art-dupl clone-free.
- **Duplication policy**: art-dupl `-t 3 --semantic`, zero harmful clones in production AND tests (policy at `-t 15` gate is exceeded). Enum metadata uses map-based lookups (`eventTypeMetaTable`, `providerTypeMeta`, `serviceStatusIcons`).
- **godoclint false positive**: "package has more than one godoc" — it counts the `// templ: version:` header in generated `html_templ.go`; suppressed via a text exclusion rule in `.golangci.yml`.
- **Pre-commit hook runs checks only** (generate drift, vet, lint, test-race + claims linter) — never auto-commits/auto-stages. Bypass: `git commit --no-verify`. Local `core.hooksPath` can silently rot — re-run `git config core.hooksPath scripts/hooks` after fresh clones.
- **`.prettierignore`** excludes `testdata/`, `schema/`, `docs/`, `CHANGELOG.md` from oxfmt — without it the golden HTML fixture, generated schema, and status reports get reformatted and break tests/CI.
- **Website workflow needs pnpm on the runner** — runners don't ship pnpm; `pnpm/action-setup` must run BEFORE `actions/setup-node` (cache: pnpm fails otherwise).
- **Proxy transport flakes**: a red job with `stream error … INTERNAL_ERROR; received from peer` on module download is proxy.golang.org instability — `gh run rerun --failed` after a cooldown; never "fix" code for it. CI's retry wrappers cover mod-tidy/generate.
- **Version-skew ledger**: `live/fragments.go:181` carries `//nolint:goconst` (provider-type literals) needed by CI's pinned golangci-lint v2.12.2 but flagged unused by local v2.13.x nolintlint — retire when the CI pin bumps ≥ 2.13.
- **Auto-commit daemon**: a daemon may commit working-tree changes mid-session as `chore: auto-commit …` heuristic blobs. Expect it; it never runs tests. Substantive fixes buried in those blobs need follow-up CHANGELOG/doc entries.
- **docs/status/ is point-in-time**: status reports are historical snapshots, never rewritten — resolved items are annotated inline (`~~…~~ done at …`) by docs-health passes, and fully-resolved files are `git mv`'d to `docs/archive/`. Recent reports are the primary TODO_LIST harvest source.

---

## Testing Patterns

- Standard `testing.T` + table-driven tests; no ginkgo/testify. External test package (`auditlog_test`).
- Each test creates its own `Plugin` + `do.Injector` — no shared state. `t.Setenv()` for the env var; `t.TempDir()` for file exports.
- Shared provider factories, lookup/assertion helpers, and struct factories live in `helpers_test.go` (grep it before writing new setup code).
- Nearly all tests use `t.Parallel()` — only `t.Setenv()` tests run sequentially.
- Coverage gate ≥94% excluding `example/`, `cmd/`, generated `*_templ.go` (exclusions single-sourced in `scripts/coverage-exclusions.txt`). Current numbers: FEATURES.md footer.
- Benchmarks (12) cover hot paths: Invocation, Disabled, Registration, ConcurrentInvocation, BuildReport (50/100/500), EventsCopy, OnEventCallback, HealthCheck, WriteD2 — see BENCHMARKS.md.
- The HTML report's feature inventory (5-tab layout, waveform, Sugiyama DAG, filter chips, pagination, etc.) is owned by FEATURES.md; example/ verifies 23 features via its self-check (exit code 0 = all pass; the Unreliable/Leaky "failures" are intentional showcase).

---

## Example

The `example/` package (`main.go`, `register.go`, `services.go`, `summary.go`) demonstrates 23 samber/do v2 features with a ride-sharing domain model and a **self-checking feature checklist** (CI runs it as the `example-smoke` job). Run with `DO_AUDITLOG_ENABLED=true go run ./example`; add `--live` for the dashboard. The checklist and APIs are enumerated in `example/summary.go`.

---

## Website

The docs site (`website/`, Astro + Starlight + Tailwind v4 + Firebase Hosting) deploys on pushes touching `website/**` via `.github/workflows/website.yml` → **do-auditlog.lars.software** (Firebase shared project `lars-software`, hosting target `do-auditlog`). Quality gates: `astro check` = 0 errors, `html-validate dist/**/*.html`, `check-changelog-sync.sh` (CHANGELOG ↔ changelog.mdx).

- **`website/package.json` must keep `typescript: ^6.0.3`** — TypeScript 7 (tsgo) crashes `astro check` (`assertCompatibleTypeScript`).
- **`website/pnpm-workspace.yaml`** sets `allowBuilds: {esbuild: true, sharp: true}` (pnpm v11 blocks native postinstalls otherwise).
- **Demo video**: 25s promo composition at `website/video/videos/do-auditlog-demo/`, deployed as `website/public/demo.mp4` (`#demo` anchor). Re-render via the HyperFrames CLI directly (`node …/hyperframes/dist/cli.js render`; `npx` wrappers fail on NixOS — set `HYPERFRAMES_BROWSER_PATH` to a nix chromium). Size target <3MB (`ffmpeg -crf 25`).

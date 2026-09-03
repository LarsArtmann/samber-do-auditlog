# AGENTS.md — samber-do-auditlog

Go plugin for [samber/do v2](https://github.com/samber/do) that records every DI container lifecycle event (registration, invocation, shutdown) with timestamps, dependency graph inference, build duration tracking, and export to JSON / NDJSON / self-contained HTML.

**Module**: `github.com/larsartmann/samber-do-auditlog` · **Package**: `auditlog` · **Go**: 1.23 on this branch (`go1.23-compat`; master is 1.26.7) · **Status**: BETA (per [STABILITY.md](STABILITY.md); internal 1.0 bar tracked in ROADMAP.md)

---

## THIS BRANCH: `go1.23-compat` (divergence contract for the samber/do merge)

Created 2026-09-03 because **samber is considering merging the plugin into `github.com/samber/do`**, whose own go.mod floor is `go 1.18`; samber approved **Go 1.23** as the target (exactly what master's code needs — `slices.Backward` is 1.23). Everything below describes how this branch differs from master. Master remains the flagship (Go 1.26.7, full dep family, live dashboard).

**Decisions locked by the samber chat (2026-09-03, LinkedIn DM, full addendum in `docs/status/2026-09-03_22-51_go1-23-compat_branch_status.md`):** (1) merge vehicle = **subpackage inside `github.com/samber/do`**, replacing the existing `http` debug sub-package (samber: "I think it could replace the current http sub-package. Your API is much better"); (2) **Go floor 1.23 confirmed** ("I don't think we need to support go 1.18. 1.23 means 2 years back. Seems good for a UI") — no 1.21/1.22 shim; (3) **live updates are REQUIRED, not optional** ("I would prefer keeping live update vs backward compatibility") — the `live/` stdlib port is the branch's main open work item; (4) samber floated splitting auditlog into multiple go modules (go workspace) to cut deps — **moot on this branch**: the stdlib ports already reduced runtime deps to `samber/do/v2` only; (5) credit agreed = **GitHub release mention + README mention** (samber declined LICENSE attribution); (6) samber asked "did you miss hooks / make hacks?" → concrete answers for the merge proposal: the `RecordHealthCheck` wrapper (do has no health-check hooks) and the `ExplainInjector` outside-hooks deadlock constraint.

- **go.mod: `go 1.23`.** Runtime deps: `github.com/samber/do/v2` ONLY (+ `samber/go-type-to-string` transitively). Tooling dep: `invopop/jsonschema v0.13.0` (v0.14 needs go 1.24; v0.13 needs 1.18 and regenerates `schema/report.schema.json` byte-identically — verified).
- **GOEXPERIMENT=jsonv2 is GONE** — it existed only for the master dep family. Consumers no longer set any env var. README/CONTRIBUTING/BENCHMARKS/.buildflow.yml/ci.yml all scrubbed.
- **All `larsartmann/*` deps replaced with stdlib ports** (they require go 1.26.x): go-output → `diagram.go` (Mermaid/DOT/PlantUML/D2 renderers + escapes ported byte-for-byte from go-output v0.37.0 for our node/edge shapes) and `table.go` (ASCII/JSON/CSV/TSV/Markdown via `TableFormat`; the other 11 go-output table formats return `errUnsupportedTableFormat`); go-ndjson → `ndjson.go`/`loader.go` (same sentinels/messages, `MaxLineBytes` 1 MB); go-atomic-write → `writeToFile` in `plugin.go` (temp+fsync+rename); go-error-family → `classify.go` DELETED (feature is master-only; `fuzz_crossproject_test.go` now fuzzes `errors.Is` through adversarial chains).
- **templ (needs go 1.25) replaced by `html/template`**: `html.go` (API + template) + `html_view.go` (view models). Same five-tab warm-amber identity, CSP hardened (`default-src 'none'`), zero external resources, sortable columns with `aria-sort`, `?` help dialog with focus trap, `e` errors-toggle, `/` search — the master a11y feature set is preserved. `html.templ`/`html_templ.go`/`daghtml_adapter.go` deleted; golden-file test replaced by `TestReport_WriteHTML_Structure` + `TestDesignTokensInSync`/`TestSharedComponentCSSInSync` now assert the constants are embedded verbatim in rendered output.
- **`live/` RESTORED on this branch as a pure-stdlib port (2026-09-04)** — samber's chat made live updates a merge requirement ("I would prefer keeping live update vs backward compatibility"). go-sse → `live/sse.go` (wire format, per-connection stream writer, heartbeats, Last-Event-ID) + `live/broadcaster.go` (drop-on-overflow fan-out, graceful drain) + `live/replay.go` (FIFO ring buffer); `Hub.EventStore()` renamed to `Hub.ReplayStore()` (concrete type, no interface). templ fragments → `live/fragments_html.go` (`html/template` with PRECOMPUTED view rows in `live/fragments.go` — same pattern as the static report) with IDENTICAL element IDs + datastar attributes so `dashboard.js`/`datastar.js` (56KB) carried over verbatim from master. `example --live` restored; smoke-tested end-to-end (SSE client observes patch-elements + patch-signals). Tests: stdlib SSE wire client replaces `ssetest`; suites cover transport, ring, broadcaster drain, hub replay, view builders, and the full external HTTP surface. Coverage: live/ 94.2%, gate 94.9% (demo/ excluded like master).
- **API deltas vs master**: `Report.WriteTable(w, format TableFormat, opts RenderOptions, ...)` (string literals still work; old `output.FormatCSV` → `auditlog.TableFormatCSV`); `Direction`/`WithDirection` are local (was `output.Direction`); file-format `Format`/`FormatAuto`/`FormatJSON`/`FormatNDJSON` unchanged. All other public API identical.
- **CI (ci.yml)**: all jobs `go-version: "1.23"`; golangci-lint pinned **v2.1.6** (newest v2 whose go directive ≤ 1.23 — v2.12.2 needs go 1.25 to `go install` on GOTOOLCHAIN=local runners); govulncheck pinned **v1.1.4** (same reason); goreleaser job DROPPED (releases happen from master); stale-generation checks only `schema/report.schema.json`; `.golangci.yml` trimmed of the 9 post-2.1.6 linters + `run.go: "1.23"` (quoted — 1.23 parses as YAML number otherwise) + templ/live paths. Local lint verified: `golangci-lint v2.1.6` → 0 issues.
- **flake.nix**: `go_1_23` no longer exists in nixpkgs (EOL-removed) — devShell uses bootstrap `pkgs.go` + `GOTOOLCHAIN = "go1.23.12"` (bare `go1.23` is NOT a valid toolchain name; must be a full patch version). `scripts/check-go-version.sh` accepts patch-extended flake pins and quoted `.golangci.yml` values.
- **Tests**: `b.Loop` → `for range b.N`, `wg.Go` → `wg.Add`+`go`+`Done` (sync.Go is 1.25), fuzz XSS vectors re-anchored on RAW quotes (`html/template` escapes quotes as entities in attributes, so `&#34; onload=&#34;` is inert — strip entities before matching, so only genuine breakouts with raw quotes trip the check).
- **Verification status 2026-09-04**: build ✓ vet ✓ full test ✓ `go test -race` ✓ coverage gate 94.9% (incl. live/ 94.2%) ✓ lint (v2.1.6, fresh cache) ✓ `go generate` drift ✓ `sh scripts/check-go-version.sh` ✓ real go1.23.12 toolchain ✓ `nix flake check` ✓ (first green evaluation) ✓ `nix develop -c go build` ✓ (with hostile ambient GOEXPERIMENT) ✓ **CI: all 7 jobs green via workflow_dispatch** (run 33813979691: Test, Lint, actionlint, vulncheck, mod-tidy, stale-generation, example-smoke).
- **CI findings fixed on 2026-09-04**: actionlint pin v1.7.12 → **v1.7.7** (v1.7.9+ need go ≥1.24 — uninstallable on the 1.23 job with GOTOOLCHAIN=local); `html_view.go` "error"/"success" string triple → `classError`/`classSuccess` constants (goconst); vulncheck job `continue-on-error: true` with rationale (Go 1.23 is EOL; all findings are stdlib advisories fixed only in newer toolchains; zero third-party deps) — scan stays visible in logs.
- **Version-skew ledger (branch)**: `httptest.NewRequest` noctx findings fire only on golangci-lint ≥ 2.13 and their suggested fix (`httptest.NewRequestWithContext`) requires Go ≥ 1.24 — do NOT add nolint directives for them (they would be "unused" under the CI-pinned v2.1.6's nolintlint); do NOT run local golangci-lint 2.13.1 as the gate. The shared local lint cache (`/mnt/buildcache/golangci-lint`) can serve stale package verdicts — use `GOLANGCI_LINT_CACHE=/tmp/...` for authoritative local runs.

---

## Commands

| Command               | Purpose                                         |
| --------------------- | ----------------------------------------------- |
| `go generate ./...`   | Regenerate the JSON schema (templ is gone on this branch)      |
| `go test ./...`       | Run all tests                                   |
| `go test -race ./...` | Run all tests with race detector (CI uses this) |

| `GOEXPERIMENT=jsonv2 go test -race -coverprofile=cover.out \\
  -covermode=atomic ./...` | Run tests with coverage (CI gate: ≥94% of non-`example/`/`cmd/` code) — **`GOEXPERIMENT=jsonv2` is set automatically in the Nix devShell and CI** |
| `go test -run TestPlugin_DisabledIsNoOp` | Run single test |
| `go vet ./...` | Static analysis (plain `go vet`; no GOEXPERIMENT on this branch) |
| `golangci-lint config verify` | Validate the lint config (CI runs this before `lint run`) |
| `golangci-lint run` | Full lint (heavy config, see below) |
| `go mod tidy` | Sync `go.sum` (CI `mod-tidy` job fails on drift) |
| `nix develop` | Enter devShell (go1.23.12 via GOTOOLCHAIN, golangci-lint, govulncheck, actionlint, golines; **GOEXPERIMENT explicitly cleared** so ambient jsonv2 cannot leak) |
| `go run ./example` | Run the example (set `DO_AUDITLOG_ENABLED=true`) |
| `go run ./cmd/auditlog help` | CLI: inspect/convert/diff/validate reports |
| `go install ./cmd/auditlog` | Install the `auditlog` CLI to `$GOBIN` |
| `nix run .#auditlog -- help` | Run the CLI via Nix (no install) |
| `nix run .#coverage` | Run the CI-equivalent coverage gate via Nix |
| `sh scripts/coverage-gate.sh` | Coverage gate (exclusions single-sourced in `scripts/coverage-exclusions.txt`, shared with ci.yml; ≥94%) |
| `sh scripts/check-go-version.sh` | Go-version drift guard: go.mod == ci.yml == flake GOTOOLCHAIN == .golangci.yml (runs in CI + pre-commit) |
| `git config core.hooksPath scripts/hooks` | Install the pre-commit hook |

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
diff.go             — Report.Diff(other) + DiffResult/ServiceDiff types
report_builder.go   — BuildReport assembly: services, scope tree (generic buildScopeTreeFromMeta), capability enrichment, shared helpers (serviceRecordToInfo, buildServiceDeps, depRecToRef, sortServiceInfos)
report_helpers.go   — Report aggregate helpers (sumBuildMs, deriveServiceStatus, etc.)
replay.go           — ReplayEvents: reconstructs Report from a flat event stream (inverse of hook-based recording)
ndjson.go           — ReadEvents: re-exports go-ndjson reader with domain-specific event validation
filter.go           — Report filtering (Filtered, ReportOption, WithServicesByName, etc.)
healthcheck.go      — Health check recording (RecordHealthCheck, ResolveServiceScope)
export.go           — Shared diagram label helpers (serviceLabel, serviceRefLabel)
diagram.go          — go-output-backed graph builder: buildDiagramNodes/Edges, warmAmberNodeStyle, diagramNodeID, writeRendered
mermaid.go          — Mermaid flowchart export (go-output graph.MermaidRenderer, code-fence off)
plantuml.go         — PlantUML component diagram export (go-output plantuml.PlantUMLDiagram)
dot.go              — Graphviz DOT digraph export (go-output graph.DOTRenderer, graphID="do_auditlog", rankdir=LR)
d2.go               — D2 diagram export (go-output d2.D2Diagram, dedupGraphEdges helper)
html.go             — HTML export entry points (Plugin.WriteHTML delegates to Report.WriteHTML)
html.templ          — Templ template for self-contained HTML visualization (CSS + JS)
html_templ.go       — Generated by `go tool templ generate` from html.templ (DO NOT EDIT)
daghtml_adapter.go  — Bridges go-output/daghtml Sugiyama DAG SDK into the HTML template graph renderer
loader.go           — LoadReport: auto-detecting loader (re-exports go-ndjson/loader Format + Detect, routes JSON via MigrateReport, NDJSON via ReadEvents + ReplayEvents)
migration.go        — MigrateReport: upgrades older JSON reports to current schema, re-derives all denormalized fields and service Status
csv.go              — WriteCSV/WriteTSV: delimited-value export of all services (uses stdlib encoding/csv)
tree.go             — ASCII tree (go-output/tree.ASCIITreeRenderer) + HTML nested list tree (go-output/markup.HTMLTreeRenderer)
table.go            — Service summary table export via go-output RenderTable: 16+ formats
schema.go           — go:embed of schema/report.schema.json + JSONSchema() accessor + //go:generate directive
design_tokens.go    — DesignTokensCSS: canonical CSS design tokens shared between static HTML + live dashboard (TestDesignTokensInSync enforces sync with html.templ)
shared_components.go — SharedComponentCSS: canonical keyboard-nav overlay CSS (skip-link, kbd-help dialog) shared between static + live dashboards (TestSharedComponentCSSInSync enforces sync)
classify.go         — Error classification: registers all sentinel errors into go-error-family families (Corruption/Rejection) via init()
stream.go           — NDJSONStreamer: real-time NDJSON event streaming via Config.OnEvent (auto-flush, buffer size, thread-safe, WithFlushInterval for bounded-latency flushing)
multi_writer.go     — MultiWriter: event fan-out to multiple OnEvent callbacks simultaneously (thread-safe, ordered)
runid.go            — RunID: 128-bit hex branded string type for cross-system correlation (auto-generated via crypto/rand)
ndjson.go           — ReadEvents + StreamEvents: re-exports go-ndjson reader + callback-based NDJSON line scanner
diagram_options.go  — DiagramOption/WithDirection: layout direction for all 4 diagram formats (Mermaid/PlantUML/DOT/D2)
table_options.go    — TableColumn/WithColumns: selectable columns for WriteTable (10 columns available, default matches original 7)
schema/             — report.schema.json: Draft 2020-12 JSON Schema generated by cmd/genschema
cmd/genschema/      — JSON Schema generator (invoked by `go generate`; imports invopop/jsonschema — tooling-only, never imported by the library)
cmd/auditlog/       — CLI binary: info/convert/diff/validate/schema subcommands (stdlib flag, no deps)
testhelpers/        — Exported test helpers (JS syntax validation, etc.) for downstream integration testing
scripts/hooks/      — pre-commit hook (generate drift check + vet + lint + test); install via `git config core.hooksPath scripts/hooks`
scripts/coverage-gate.sh — CI-equivalent coverage gate (exclusions single-sourced in scripts/coverage-exclusions.txt, consumed by both the gate and ci.yml; ≥94% threshold)
scripts/check-go-version.sh — Go-version drift guard (go.mod is canonical; asserts ci.yml go-version, flake GOTOOLCHAIN, .golangci.yml run.go agree; wired into ci.yml test job + pre-commit hook; test via CHECK_GO_VERSION_ROOT pointing at a doctored tree)
doc.go              — Package doc comment
example/            — Self-checking demo with 23 samber/do v2 features
live/               — Real-time SSE dashboard sub-package (see below)

**Note:** The `health/` sub-package was extracted to its own project: [github.com/larsartmann/go-health](https://github.com/larsartmann/go-health). The `*Plugin` type satisfies `go-health`'s `HealthRecorder` interface implicitly via `RecordHealthCheckWithContext`.
```

### `live/` sub-package files

```
live/hub.go         — Hub: facade over sse.Broadcaster[sse.Event] with subscriber buffer (128 events), ring buffer for reconnection replay (ReplayBufferSize config), SignalComplete, OnEvent callback, EventStore()/BufferedEventCount() methods, Shutdown/Health pass-through. Defines sseEventType constant.
live/server.go      — HTTP server with 6 endpoints (dashboard, report JSON, SSE events, health, export NDJSON, export HTML), configurable prefix, CORS middleware, graceful shutdown. SSE handler sends datastar-patch-elements + patch-signals via go-sse.
live/fragments.go   — Go helpers for fragment rendering: constants, datastar signal structs, renderAllFragments(), pure-Go helpers (humanizeDuration, providerIcon, computeWaveformMarks, etc.), renderToString wrapper for templ components
live/fragments.templ — Templ components for all dashboard sections: statsFragment, legendFragment, waveformFragment, servicesTbody, eventsTbody, scopeTreeFragment+scopeNode, graphFragment, timelineFragment, footerStatsFragment, containerIDFragment
live/fragments_templ.go — Generated by `go tool templ generate` from fragments.templ (DO NOT EDIT). Excluded from coverage gate.
live/fragments_internal_test.go — Internal tests for unexported helper functions
live/dashboard.go   — Dashboard HTML template (datastar attributes: data-signals, data-init, data-bind, data-show, data-class:active, data-on:click). Embeds datastar.js via go:embed. Contains renderEventFilterChips().
live/dashboard.css  — Dashboard stylesheet (warm amber theme, responsive layout)
live/dashboard.js   — Keyboard nav (handleKeydown), export helpers (exportReport), scope tree toggle, keyboard help dialog (~223 lines)
live/datastar.js    — Datastar v1.0.2 runtime (~56KB), embedded via go:embed. Handles SSE parsing, DOM morphing by element ID, reactive signal evaluation.
live/base_css.go    — Live dashboard CSS: composes shared design tokens (auditlog.DesignTokensCSS) + live-specific aliases + base component styles
live/doc.go         — Package doc comment
live/server_test.go — External tests: server lifecycle, SSE streaming, handler edge cases, hub unit tests, CORS, export endpoints
live/demo/          — Self-contained real-time demo (registers services with delays, shows dashboard updating live). Demo services implement do.Healthchecker.
```

### Data Flow

1. User creates `Plugin` via `New(Config)` → gets `*do.InjectorOpts` via `Opts()`
2. Hooks fire on every registration/invocation/shutdown → `Recorder` captures timestamped `Event`s
3. Invocation stack (`Recorder.stack`) infers dependencies: if service A is on-stack when B's before-hook fires, A→B is a dependency
4. `Report()` / `BuildReport()` assembles `ServiceInfo` slice (with forward + reverse deps), scope tree, and event stream
5. Export methods serialize to JSON, NDJSON, or self-contained HTML
6. Health checks: user calls `plugin.RecordHealthCheck[WithContext](injector)` instead of `injector.HealthCheck()` — wrapper records `EventTypeHealthCheck` events per service

### Concurrency Model

- **`sync.RWMutex` (`mu`)** protects core mutable state: `events`, `services`, `scopes`, `stack`, `shutdownStart`. This reduces lock acquisition overhead from 2–4 per hook to exactly 1.
- **`onEventMu sync.RWMutex`** (separate from `mu`) guards the `onEvent` callback so `Plugin.SetOnEvent` can swap it after creation. `fireEvent` copies the callback under `RLock` and invokes the copy outside any lock; `setOnEvent` takes `Lock` only on swap.
- `sequence` and `invocationSeq` are `atomic.Int64` — no mutex needed for counters.
- Each hook acquires `mu` once, performs all mutations (scope recording, stack management, event append, service updates), then releases.
- `onEvent` callback is always invoked outside the lock to avoid blocking the hot path.
- `BuildReport()` uses `mu.RLock()` for reading — concurrent reads don't block each other.
- **`MultiWriter`** has its own internal `sync.Mutex` — safe for concurrent use from multiple hooks. Preserves callback registration order; callbacks fire sequentially per event.
- **`RunID`** is immutable once set. Auto-generated in `New()` via `crypto/rand` (128-bit hex). Stored on `Recorder` and stamped on every `Event` and the `Report`. `Config.RunID` (non-zero) overrides auto-generation.

### Shared infrastructure: `go-sse`

The `live/` sub-package depends on [`github.com/larsartmann/go-sse`](https://github.com/larsartmann/go-sse)
(v0.5.1, public) for the full SSE lifecycle — `Stream`, `Broadcaster[T]`,
`Replay`/`EventStore`, plus wire-format primitives (`Event`, `WriteEvent`,
`ContentType`). The domain-specific Hub (facade over `Broadcaster[sse.Event]`)
and Server are implemented locally in `live/` (samber/do service events, scope
tree, dashboard HTML) on top of those primitives; go-sse itself is
transport-only and owns no domain types here.

### Shared infrastructure: `go-ndjson`

NDJSON read/write and format-detection logic delegate to the external
[`github.com/larsartmann/go-ndjson`](https://github.com/larsartmann/go-ndjson) module
(public, v0.0.1). The local `loader.go` and `ndjson.go` re-export the
public API so existing callers are unaffected.

Both `go-sse` and `go-ndjson` are now public — no `replace` directives remain in `go.mod`.
A `go.work` workspace at the parent directory may still link the projects for local development.

### GOEXPERIMENT=jsonv2 requirement (MASTER ONLY — not this branch)

> **go1.23-compat note**: this branch has zero third-party runtime deps and needs NO GOEXPERIMENT anywhere. The section below describes master. Do not set the flag when working here — a stale ambient value breaks the go1.23 toolchain ("unknown GOEXPERIMENT jsonv2"); the devShell clears it.

**The project requires `GOEXPERIMENT=jsonv2` to build.** **Consumers need it too**: a downstream module that imports this library fails with `imports encoding/json/v2: build constraints exclude all Go files` unless `GOEXPERIMENT=jsonv2` is set (verified empirically 2026-09-01 with a minimal consumer). The README Install section and the website Installation page document this; keep them in sync.

This is set automatically in:

- The Nix devShell (`flake.nix` sets `GOEXPERIMENT = "jsonv2"`)
- CI workflows (`.github/workflows/ci.yml` sets `env: GOEXPERIMENT: jsonv2` at the workflow level)
- The coverage-gate script (`scripts/coverage-gate.sh` exports it)
- direnv (`.envrc` with `use flake` auto-loads the devShell on `cd`; see `.envrc.example`)
- BuildFlow config (`.buildflow.yml` sets `env: GOEXPERIMENT: jsonv2` via `ApplyConfigEnv` at pipeline startup)

The requirement exists because `go-output` (used for diagram/table rendering), `go-branded-id`
(transitive dependency of `go-output`), and `go-ndjson` (v0.0.1, public dependency)
intentionally use `encoding/json/v2` features
(`jsontext.Encoder`, `json.Deterministic`, `json.MarshalEncode`). This project's own code does
NOT import `encoding/json/v2` — the `encoding/json/v2` exclusion policy in AGENTS.md still holds
for this project's `.go` files. The dependency is purely transitive through `go-output`.

When Go 1.27 stabilizes `json/v2`, the `GOEXPERIMENT` flag requirement will be removed
automatically.

### Go 1.26.7 toolchain pin

**The canonical Go version is 1.26.7** (since commit `2cd47f6`; go-sse v0.5.1 itself declares `go 1.26.6`), pinned across (1) `go.mod` `go` directive, (2) `.github/workflows/ci.yml` (`go-version: "1.26.7"` in all 7 jobs), (3) `flake.nix` (`GOTOOLCHAIN=go1.26.7` in devShell + `coverage` + `auditlog` apps; `pkgs.go_1_26` tracks latest 1.26.x), (4) `CONTRIBUTING.md` / `BENCHMARKS.md`.

**Why it's mandatory, not discretionary:** Go computes the main module's language version as the maximum `go` directive across all dependencies, and `go mod tidy` rewrites the main `go` directive to match. GitHub runners set `GOTOOLCHAIN=local` globally, so a CI `go-version` below the `go.mod` requirement fails EVERY Go job instantly with `go: go.mod requires go >= 1.26.7 (running go 1.26.5; GOTOOLCHAIN=local)` — this exact mismatch (CI on 1.26.5, go.mod on 1.26.7) broke all 6 Go jobs on master on 2026-08-29. **Rule: bump `go-version` in ci.yml and `GOTOOLCHAIN` in flake.nix in the SAME commit as any `go.mod` bump.**

**Gotcha:** A separately-installed newer `go` nix-store derivation (e.g. from `nix profile install nixpkgs#go`) can shadow the devShell's `go_1_26` on PATH. The `GOTOOLCHAIN` env var in `flake.nix` (devShell + `coverage` app + `auditlog` app) pins the effective toolchain even when a different `go` is on PATH. If you must run `go mod tidy` outside the devShell, prefix it with `GOTOOLCHAIN=go1.26.7 go mod tidy`.

**goreleaser coupling:** goreleaser v2.18.0+ declares `go >= 1.27.0`; with runner `GOTOOLCHAIN=local`, `go install .../goreleaser/v2@latest` fails under Go 1.26.x. CI pins goreleaser to `v2.17.1` (latest release with a `go 1.26.5` directive). Revisit the pin when CI's Go reaches 1.27.

History: The Go directive bounced between 1.26.4 and 1.26.5 across three releases (v0.7.0–v0.8.0; see git blame for details). 7b361e8 bumped to 1.26.6 (forced by go-sse v0.5.1), 2cd47f6 to 1.26.7 (canonical). The `GOTOOLCHAIN` pin from v0.7.1 stays.

---

## CI

GitHub Actions workflow at `.github/workflows/ci.yml` runs on every push and PR with 7 parallel jobs:

- **test**: `go vet`, `go build`, `go test -race` with a coverage profile, and a **coverage gate** that fails if non-`example/`/`cmd/` statement coverage drops below 94%.
- **lint**: `golangci-lint config verify` then `golangci-lint v2.12.2` run (pinned to match local dev).
- **vulncheck**: `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`.
- **mod-tidy**: runs `go mod tidy` and fails if `go.sum` drifts from the committed version.
- **stale-generation**: runs `go generate ./...` (uses `go tool templ` from go.mod `tool` directive), fails on diff. No manual templ install needed — Go toolchain auto-builds the exact pinned version.
- **actionlint**: workflow lint via `go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12`.
- **goreleaser**: `goreleaser check` on `.goreleaser.yml`; goreleaser pinned to `v2.17.1` (v2.18.0+ needs Go 1.27, see toolchain pin section).

## Lint Configuration (.golangci.yml)

Extremely strict — nearly every golangci-lint linter enabled. Key implications:

- **exhaustruct**: All struct fields must be explicitly initialized. Tests are exempted. This is why `newEventFromRef()` and `newServiceRecordCore()` exist as constructor helpers — they centralize field init to satisfy exhaustruct in one place.
- **depguard**: REMOVED from the enabled linters in `2cd47f6` — import restriction is now by convention (keep external non-stdlib deps out of `cmd/`).
- **noinlineerr**: Use `err := ...` then check, not `if err := ...; err != nil`.
- **forbidigo**: `fmt.Print*` forbidden in non-example code.
- **exclusions for tests**: exhaustruct, testpackage, gochecknoglobals, funlen, cyclop, goconst are relaxed in `*_test.go`.
- **Formatters**: gci, goimports, gofumpt, golines (max-len 120).

---

## Gotchas

- **Repo directory is `samber-do-metrics`** but `go.mod` says `samber-do-auditlog`. The module name is canonical.
- **JSON tags use snake_case** (`scope_name`, `service_name`, etc.) via `tagliatelle` config set to `json: snake_case`. This is intentional for JSON API compatibility.
- **`doc.go`** has the package-level doc comment. `plugin.go` has no package comment (the dual-comment issue was fixed).
- **`Plugin.containerID` was removed** — containerID is stored only in `Recorder` (passed at construction via `NewRecorder`). `Plugin.Report()` calls parameterless `BuildReport()`.
- **`Report.Services` is sorted** by (scope_name, service_name) for deterministic output across runs. Dependencies and dependents within each service are also sorted.
- **`writeToFile()` helper** in `plugin.go` properly returns Close errors after write errors (write error takes priority).
- **Test file uses external test package** (`auditlog_test`) — imports the package under test as `auditlog`.
- **`example/` directory** is exempt from some lint rules (forbidigo, noinlineerr) since it's demo code.
- **`html.templ` CSP meta tag** restricts to `default-src 'none'` + `inline styles/scripts + Google Fonts` (blocks external resource loads; `default-src 'none'` IS present — earlier docs claiming otherwise were wrong). The live dashboard's CSP (`live/dashboard.go`) additionally allows `connect-src 'self'` for SSE.
- **`html.templ` XSS escaping**: all user-controlled strings use `esc()` function. Dependencies use `esc(d.service_name)`. Status classes use `esc(s.status)`. Error messages in `data-error` attributes are escaped.
- **`html.templ` Events tab**: `allEvents` array built from `report.events.map(...)` with type badges, provider badges, phase icons (▲/▾), duration, error tooltips. Filter chips use `data-type` attribute on rows.
- **`RootScopeName` constant** (`"[root]"`) in `types.go` replaces the magic string. Used in `IsRoot()` and test struct literals (not in JSON strings).
- **`MigrateReport` validation**: rejects empty input (`errMigrationEmptyInput`), missing version (`errMigrationMissingVersion`). Returns early if already at current schema. Preserves original `ExportedAt`.
- **Fuzz tests**: 5 targets — `FuzzPluginHTML` (HTML XSS across service names, error messages, and dependency chains), `FuzzMigrateReport` (schema-migration integrity), `FuzzDiagramSpecialChars` (Mermaid/PlantUML structural integrity), `FuzzFilterInputs` (filter option robustness), `FuzzReadEvents` (NDJSON parsing resilience). HTML target uses `stripJSONScripts()` to avoid false positives from JSON inside `<script>` tags and checks 6+ XSS vectors via `assertNoRawXSS`.
- **`Config.Validate()`** validates ContainerID for path separators (`/` and `\`). Returns `errContainerIDPathSep` sentinel error wrapped with the offending value.
- **Do NOT modularize** — Project is 1 package, ~2,500 LOC. Too small for multi-module split. Revisit at 5+ packages.
- **`ServiceStatus`** is computed in `buildServicesLocked` via `computeServiceStatus()`. Priority: invocation_error > shutdown_error > shutdown > active > registered. The HTML template uses `s.status` instead of deriving from individual fields. The canonical public derivation entry point is `ServiceInfo.DeriveStatus()` — a method on the type it operates on, reusable beyond report building.
- **`buildReportFromCore()` is the single Report construction path** — `BuildReport`, `Filtered`, `MigrateReport`, and `ReplayEvents` all route through `buildReportFromCore()` + `finalizeDenormalized()`. The public `NewReport()` wraps the same path but additionally re-derives per-service `Status` and enforces `Validate()`. **Critical invariant**: any new Report construction path MUST use `buildReportFromCore()` — never hand-compute aggregates, or they will drift from the underlying data and fail `Validate()`.
- **`buildScopeTreeLocked`** uses `sortedScopesLocked()` to iterate scopes deterministically (sorted by scope ID), since map iteration order is non-deterministic in Go.
- **`newServiceRecordCore`** uses lazy deps map (`nil` until first dependency recorded). `buildDepsLocked` returns `nil` for services with no deps (no empty slice allocation).
- **`inferServiceType`** is called only during `OnAfterRegistration` (once per service), not per event. Events look up the type from the existing `serviceRecord`.
- **Stack pop** uses LIFO fast path: checks last element first (O(1) common case), falls back to backward search only for unusual orderings.
- **`serviceKey(scopeID, serviceName)`** is the single canonical function for the `scopeID + "/" + serviceName` key format. The `scopeKey()` helper was removed — all callers use `serviceKey` directly.
- **Disabled path** is zero-cost: `Opts()` returns empty hooks, so samber/do never calls recorder methods. Disabled overhead is entirely samber/do's own (4 allocs, ~115ns).
- **Benchmark suite** covers: Invocation (hot path), Disabled, Registration, ConcurrentInvocation, BuildReport (50/100/500 services), EventsCopy, OnEventCallback, HealthCheck.
- **`ServiceRef`** (renamed from `DependencyRef`) is embedded in `Event` and `ServiceInfo` — single source of truth for service identity (ScopeID, ScopeName, ServiceName). JSON output is flat because Go flattens embedded struct fields.
- **`ServiceType`** is captured via `do.ExplainNamedService(scope, serviceName)` in `OnAfterRegistration` → `inferServiceType()`. Uses the public `ExplainNamedService` API to get the provider type (lazy/eager/transient/alias). Empty string if the type cannot be determined.
- **`Config.OnEvent`** callback is called after each event is captured, outside the mutex lock. Must not block. Enables real-time observability (Prometheus, OTel, live dashboards) without polling.
- **Health checks use a wrapper pattern**, not hooks. samber/do v2 has no `HookBeforeHealthCheck`/`HookAfterHealthCheck` in `InjectorOpts`. The plugin provides `RecordHealthCheck[WithContext](injector)` which wraps `injector.HealthCheckWithContext()`, records `EventTypeHealthCheck` events (PhaseAfter only), and updates `ServiceInfo` health fields. When disabled, delegates directly to the injector without recording.
- **`ResolveServiceScope`** resolves scope metadata from our `serviceRecord` map by service name. Handles both `*do.RootScope` (from `do.NewWithOpts`) and `*do.Scope` (from `injector.Scope()`). Returns `(scopeID, scopeName, found)` — no `*do.Scope` needed since `RecordHealthCheck` on Recorder takes metadata strings directly.
- **Health check events are `PhaseAfter` only** — unlike registration/invocation/shutdown which have before+after phases. There's no interception point before the bulk health check runs.
- **`IsHealthchecker`/`IsShutdowner` are populated via `enrichCapabilities()`** in `BuildReport()`. The function calls `do.ExplainInjector(scope)` on each stored `*do.Scope` reference AFTER releasing the recorder RLock. Capabilities are only visible for invoked services — lazy providers must be built before `ExplainInjector` can detect interface implementations. **DEADLOCK RISK**: `do.ExplainInjector()` MUST NOT be called from inside any hook — it acquires internal locks that conflict with the hook execution context.
- **`Event.ServiceType`** (ProviderType) carries the provider type per event. Looked up from the existing `serviceRecord` in each hook, avoiding redundant `do.ExplainNamedService` calls. Health check events look up from the record too (empty string if not yet registered).
- **`scopeMeta.ref`** stores `*do.Scope` references in the recorder. Used by `enrichCapabilities()` in `BuildReport()` to call `do.ExplainInjector` outside the mutex. Set in `recordScope`.
- **`HealthCheckDurationMs` was removed** — Per-service timing is unavailable from the bulk `injector.HealthCheckWithContext()` API. Health check events have `DurationMs: nil`.
- **`HealthCheckSucceeded` is `false` when no health checks ran** — `allHealthChecksPassed()` requires at least one health-checked service to return `true`.
- **`newEventFromRef()`** builds events from `ServiceRef` instead of `*do.Scope`. Used by `RecordHealthCheck` which doesn't have a scope object.
- **`serviceTypeForLocked()`** centralizes the `if rec, ok := r.services[key]; ok { svcType = rec.serviceType }` lookup used by all 3 hook handlers (invocation, shutdown-before, shutdown-after). Caller must hold `r.mu`.
- **HTML redesign**: Warm amber "Container Telemetry" aesthetic — phosphor amber (#e8a838) on dark charcoal (#14110d) palette, Space Grotesk + IBM Plex Mono fonts, **lifecycle waveform** signature element (plots all events as colored vertical marks on a timeline, height-encoded by duration, colored by type, errors in coral), color-coded type badges (purple=lazy, amber=eager, warm orange=transient, jade=alias), animated tab transitions, stat cards with hover accent bar, reduced-motion support, subtle warm radial glow on body background.
- **New() returns (\*Plugin, error)**: Breaking API change. `Config.Validate()` is enforced at construction. Tests use `mustNew()` helper (panics on error).
- **TypeMetadata injection**: `BuildTypeMetadata()` in `metadata.go` calls enum methods (`ProviderType.Icon()`/`Label()`, `ServiceStatus.Icon()`, `EventType.Label()`/`Color()`) — single source of truth for display metadata. Injected into HTML via `@templ.JSONScript("type-metadata", ...)`. JS reads from injected metadata — no hardcoded constants.
- **Report.Validate()**: Checks denormalized count fields (`EventCount`, `ServiceCount`, `ScopeCount`, `HealthCheckedCount`) match actual data, AND checks every `ServiceInfo.Status == DeriveStatus()` (status consistency). Uses sentinel errors (`errReportEventCountMismatch`, `errReportStatusDrift`, etc.) with `%w` wrapping.
- **Diagram rendering via `go-output`**: Mermaid/PlantUML/DOT/D2 are produced by `github.com/larsartmann/go-output` renderers (`graph.MermaidRenderer`, `plantuml.PlantUMLDiagram`, `graph.DOTRenderer`, `d2.D2Diagram`). `diagram.go` builds `[]output.GraphNode`/`[]output.GraphEdge` from the report; each `Write*` method configures its renderer (`SetNodes`/`SetEdges`/`DedupEdges` for Mermaid/PlantUML/DOT, `dedupGraphEdges` helper for D2), then `writeRendered()` does the single `Render()+Write`. Escaping is go-output's validated `escape` package (`SlugifyID`+`MermaidID` for node IDs, `MermaidText`/`PlantUML`/`DOT`/`D2` for labels). Compiled go-output packages: root + enum + envdetect + escape + graph + plantuml + d2 (+ `go-branded-id`, `x/term`); `delimited`/`markdown`/`tree`/`testhelpers` are graph-verification-only, zero packages linked. Adoption logged in `docs/research/go-output-adoption-review.md` §9.
- **Diagram theming (warm amber, per-node)**: `warmAmberNodeStyle` (`output.GraphStyle{Fill:#e8a838, Stroke:#4a4030, FontColor:#14110d}`) is applied per-node, replacing the former global Mermaid `%%{init}%%` directive and PlantUML `skinparam` block. Renderers translate it to Mermaid `style <id> fill:...,stroke:...,color:...`, PlantUML `#e8a838;line:#4a4030;text:#14110d`, DOT `fillcolor`/`color`. As of go-output v0.31.1, D2 hex colors and labels-with-spaces are properly quoted (e.g. `style.fill: "#e8a838"`, `"db 😴"`) — go-output's `d2Quote()` wraps values that D2 would misinterpret (`#` = comment char). **Tradeoff**: node fills/strokes/fonts preserved; edge line-colors and the DOT dark `bgcolor` are no longer emitted (go-output renderers lack a graph-level bgcolor setter). Re-introducing the DOT dark background requires adding graph-attribute support upstream in go-output.
- **HTML pagination**: Services table shows first 50 rows; events table shows first 100. "Show all" button reveals remaining rows. Search and filter bypass pagination.
- **Touch events**: Graph supports 1-finger pan and 2-finger pinch-zoom via touchstart/touchmove/touchend handlers with `passive:false`.
- **Fuzz test XSS checking**: `stripJSONScripts()` replaces `stripScriptTags()` — targets `<script type="application/json">` blocks specifically using marker search + `LastIndex` backtracking. More robust than the old character-by-character parser.
- **`MigrateReport` always re-derives per-service Status** from the underlying error/timestamp fields — the old `if Status == ""` guard that preserved stale statuses was removed. Combined with the Validate() status-consistency check, stale/hand-edited reports are repaired.
- **`serviceRecordToInfo()`** is the single conversion function from internal `serviceRecord` to public `ServiceInfo`. Dependencies, Dependents, IsHealthchecker, and IsShutdowner are left as zero values for the caller to set. Any new field on ServiceInfo must be wired here.
- **`diff.go` uses `Status.IsError()`** as the single error-detection path — the old `hasError()` helper that checked raw pointers was deleted to prevent drift.
- **`Plugin.WriteReportJSON()` and `ExportFilteredToFile()`** delegate to `Report.WriteJSON()` — single JSON encoding path.
- **CSP hardened**: `base-uri 'none'; frame-ancestors 'none'` added to prevent base injection and clickjacking.
- **JSON Schema** (`schema/report.schema.json`) is GENERATED from Go types by `cmd/genschema` (invoked via `//go:generate go run ./cmd/genschema` in `schema.go`). It is `go:embed`ded and exposed via `JSONSchema()`. Never hand-edit it — change the Go struct tags and regenerate. The `invopop/jsonschema` dependency is tooling-only (`cmd/`); the library never imports it.
- **`cmd/` packages are tooling, not library code** — `cmd/genschema` (schema generator) and `cmd/auditlog` (CLI binary). The `.golangci.yml` `cmd/` path excludes pragmatic tooling linters (forbidigo, exhaustruct, gosec, err113, errcheck, wrapcheck, nlreturn, goconst). `cmd/` and `example/` are EXCLUDED from the 94% coverage gate (their logic is exercised by integration/golden tests that exec a built binary, not in-process).
- **Adding a 4th diagram format** — DONE: D2 export added via `go-output/d2` (`Report.WriteD2()`). For formats that lack built-in edge dedup (like D2), use the `dedupGraphEdges()` helper before `SetEdges()`. Node IDs via `diagramNodeID(scopeID, serviceName)` (SlugifyID+MermaidID); labels via `serviceLabel(svc)` (with type icon) or bare `dep.ServiceName` for external deps.
- **Typed identifiers + ServiceInfo split are DONE** — `ContainerID`/`ScopeID`/`ServiceName` are named string types propagated through the entire codebase (production, tests, CLI, example). `ServiceInfo` is split into four embedded structs: `ServiceIdentity` (ServiceRef + ServiceType), `ServiceLifecycle` (status, timestamps, errors, durations), `ServiceHealth` (health check fields), `ServiceGraph` (Dependencies + Dependents). Fields stay flat via Go embedding (both for code access and JSON output). **Key pattern**: external library calls (go-output, csv, fmt) wrap typed values with `string()` at the IO boundary. Test struct literals must use the embedded struct names (e.g. `ServiceIdentity: ServiceIdentity{ServiceRef: ...}`) — promoted fields cannot be used in composite literals.
- **`.prettierignore`** excludes `testdata/`, `schema/`, `docs/`, and `CHANGELOG.md` from oxfmt (which reads `.prettierignore` by default). Without this, oxfmt pretty-prints the golden HTML test fixture (breaking `TestReport_WriteHTML_GoldenFile`), reformats generated JSON schema (breaking `stale-generation` CI check), and pads markdown tables (noise in docs/status reports).
- **BuildFlow `--max-time`** (RESOLVED via `.buildflow.yml`): The default 2m hard timeout is too short for 5 fuzz targets (30s each = 2.5m). The config sets `max_time: 5m`. CLI `--max-time` overrides if needed.
- **BuildFlow `GOEXPERIMENT=jsonv2`** (RESOLVED via `.buildflow.yml`): `.buildflow.yml` sets `env: GOEXPERIMENT: jsonv2`, applied via `ApplyConfigEnv` at pipeline startup (`pipeline.go:102`). Existing process env vars take precedence. Note: `config view` does NOT display the `env:` block (display-only limitation), but it IS applied at runtime. The `go-fix` step still requires the global `--fix` flag to be executable (otherwise: `no executable nodes after compilation`).
- **BuildFlow `go-auto-upgrade`** (RESOLVED via `.buildflow.yml`): `go-auto-upgrade` is permanently skipped via `skip_steps` in `.buildflow.yml`. Its `jsonv1tov2` migrator rewrites `encoding/json` → `encoding/json/v2` + `jsontext`, which (1) breaks API calls (`enc.SetIndent` has no equivalent on `jsontext.Encoder`) and (2) violates the project's `encoding/json/v2` exclusion policy. The migrator runs because `GOEXPERIMENT=jsonv2` (required by transitive deps) makes `HasJSONv2Experiment()` return true. Incident post-mortem: `docs/status/2026-07-13_21-28_buildflow-go-auto-upgrade-breakage-remediation.md`. Re-enable when Go 1.27 stabilizes json/v2 and the project lifts its exclusion policy.
- **`encoding/json/v2` exclusion policy**: No `.go` file in **this project** may import `encoding/json/v2` or `encoding/json/jsontext`. These packages are behind `//go:build goexperiment.jsonv2`. The project targets Go 1.26.x. Revisit when Go 1.27 stabilizes json/v2. **Exception**: transitive dependencies (`go-output`, `go-branded-id`, `go-ndjson`) use `encoding/json/v2` — the project builds with `GOEXPERIMENT=jsonv2` set automatically in the Nix devShell and CI. See the **GOEXPERIMENT=jsonv2 requirement** section above.
- **Test helpers**: `mkEvent` (replay_test.go, `auditlog_test` package) and `mkRegEvent` (cli_integration_test.go, `main` package) create standard event structs. `mkEventWithDur` extends `mkEvent` with `DurationMs`. `mkInvAfterWithDur` extends with invocation-after semantics. `setupWithDB(url)` wraps `newPluginAndInjector + provideDB + invoke` (the standard 4-line plugin preamble). `replayFromPlugin(t, p)` wraps `WriteEventsNDJSON → ReadEvents → ReplayEvents` (the standard 8-line round-trip). `assertWriteFails`/`assertErrIs`/`assertLen`/`assertReportValidNoFatal` centralize the most common assertions. `csvServiceRef`/`rootRef`/`rootScopeTree`/`csvSplitLines`/`mkNewReport`/`assertMetadataLabel` centralize struct creation. Use these instead of inline struct literals to keep art-dupl clone-free.
- **godoclint false positive**: `godoclint` reports "package has more than one godoc" because it counts the `// templ: version:` header in the generated `html_templ.go` as a second package doc. A text-based exclusion rule (`text: 'package has more than one godoc'`) in `.golangci.yml` suppresses this. The path exclusion `_templ\.go$` doesn't catch it because the issue is reported on `doc.go`, not the generated file.
- **Pre-commit hook runs checks only**: The hook at `scripts/hooks/pre-commit` runs `go generate` drift check, `go vet`, `golangci-lint`, and `go test -race`. It does NOT auto-commit or auto-stage. Bypass with `git commit --no-verify`.
- **CSS design tokens are shared**: `DesignTokensCSS` in `design_tokens.go` is the single source of truth for the warm amber color palette. The static HTML report (`html.templ`) embeds these tokens inline (necessary for self-contained HTML). The live dashboard (`live/base_css.go`) composes them from `auditlog.DesignTokensCSS` + live-specific aliases (`--bg-card`, `--bg-hover`, `--border-light`, `--font`, `--font-mono`). `TestDesignTokensInSync` verifies the html.templ inline `:root` block matches `DesignTokensCSS` exactly — if you change a color in one, update both or the test fails.
- **Keyboard-nav overlay CSS is shared**: `SharedComponentCSS` in `shared_components.go` is the single source of truth for `.skip-link`, `.kbd-help`, `.kbd-help-content` styles. The static report inlines them in `html.templ`; the live dashboard composes them via `auditlog.SharedComponentCSS` in `live/base_css.go`. `TestSharedComponentCSSInSync` (in `shared_components_test.go`) verifies the html.templ inline rules match the Go constant — if you change a rule, update both or the test fails. Token names (`--bg-elevated`, `--border-active`) are canonical; the live dashboard's aliases (`--bg-card`, `--border-light`) resolve to the same values.
- **Keyboard navigation architecture**: Both dashboards implement WAI-ARIA patterns: skip link → `<main tabindex="-1">`, tablist with Arrow/Home/End + roving tabindex, `?` help dialog with focus trap + restoration (`closeKbdHelp()` saves/restores `kbdHelpPrevFocus`, Tab cycles within dialog), `/` focuses search, `e` toggles errors-only (static only), Esc closes overlays. Sortable column headers have `tabindex="0"`, Enter/Space activation, and `aria-sort` state management. **The two dashboards use different JS styles**: static uses modern ES6 (arrow functions, `const`), live uses ES5-compatible IIFE (`var`, `function`). Shared JS extraction is intentionally NOT done — the style difference makes a shared file impractical.
- **JS syntax validation test**: `TestHTMLJavaScriptSyntax` and `TestHTMLJavaScriptSyntax_MultiService` in `plugin_html_syntax_test.go` extract `<script>` content from the HTML report, strip strings/comments/regexes via a `jsStripper` state machine, and assert `{}`, `()`, `[]` delimiters are balanced. This catches syntax errors (like a stray `}`) that the golden byte-for-byte test misses.
- **Dependency family versions (2026-09-01)**: `go-output` v0.37.0, `go-sse` v0.5.1 (+ `go-sse/ssetest` v0.2.0, direct), `go-atomic-write` v0.5.0, `go-error-family` v0.10.0, `go-ndjson` v0.0.1. Historical bullets below may cite older versions — go.mod is canonical.
- **Website workflow needs pnpm on the runner** — GitHub runner images do not ship `pnpm`; `actions/setup-node` with `cache: pnpm` fails with `Unable to locate executable file: pnpm` unless `pnpm/action-setup` runs FIRST. Both website.yml jobs (build + deploy) carry it (added 2026-09-01 after run `33562593783` died exactly this way).
- **Proxy transport flakes**: a red job whose log shows `stream error … INTERNAL_ERROR; received from peer` on a module download is upstream proxy.golang.org instability — `gh run rerun --failed` after a short cooldown; never "fix" code for it. 2 of 3 runs on 2026-09-01 hit this.
- **Version-skew ledger**: `live/fragments.go:181` carries a `//nolint:goconst` (provider-type literals) needed by CI's pinned golangci-lint v2.12.2 but flagged as unused by local v2.13.1's nolintlint. Retire it when the CI pin bumps ≥ 2.13. The three other 2026-08 sites (`loader.go:50`, `stream.go:129`, `live/server_test.go:684`) were unused directives and were removed by the lint-fix commit `cf5f205` (2026-09-01).
- **Local `core.hooksPath` can silently rot**: a checkout pointing at a nonexistent dir (e.g. `.githooks`) disables the pre-commit hook with no error. The documented install command is `git config core.hooksPath scripts/hooks`; re-run it after fresh clones.
- **go-output version history**: upgraded through v0.30.1 → v0.31.1 → v0.32.0 → … → v0.37.0 (all sub-modules in lockstep). v0.31.1 added `d2Quote()` (fixes D2 hex-color/label quoting — previously `#e8a838` was treated as a comment by D2). The v0.31.1 published manifests contained broken zero pseudo-versions for `testhelpers` and `testhelpers/graphtest` (local `replace` directives were not stripped before tagging), requiring consumer-side explicit indirect pins at v0.31.1; corrected manifests exist from v0.32.0 onward. The indirect pins in `go.mod` track the go-output family version.
- **Cross-project feature ports from `go-workflow-auditlog`** (sibling project, same author, same patterns): Five patterns were ported from the sibling project: (1) **`go-error-family` classification** — `classify.go` registers all sentinel errors into Families (Corruption/Rejection) with auto-registration in `init()`; upgraded to v0.10.0 (direct dep). (2) **`go-atomic-write`** — `writeToFile()` in `plugin.go` delegates to `atomicwrite.WriteFunc` (v0.5.0) for crash-durable atomic writes. Note: v0.4.0 split the API — `WriteFunc(path, fn)` is the plain 2-arg write; `WriteFuncVerified` adds fingerprint TOCTOU protection. Audit exports use plain writes (no read-modify-write cycle). (3) **NDJSON streaming** — `stream.go` provides `NDJSONStreamer` for real-time event streaming via `Config.OnEvent` with `WithAutoFlush`/`WithStreamBufferSize`; uses standard `encoding/json` (no `jsontext` dependency). (4) **Diagram direction** — `diagram_options.go` provides `WithDirection(output.Direction)` across all 4 diagram formats. (5) **Table column selection** — `table_options.go` provides `WithColumns(TableColumn...)` with 10 selectable columns.
- **Datastar-powered live dashboard** (v1.0.2): The live dashboard uses [Datastar](https://data-star.dev/) for SSE transport, DOM morphing, and signal-based reactivity. The server renders HTML fragments (`fragments.go`) and sends them as `datastar-patch-elements` SSE events (via go-sse `SendLines` + `KeyedLines`). Datastar morphs the DOM by element ID, preserving focus/scroll/transitions. Client-side state (tab, search, filter, pagination) is managed via datastar signals (`data-signals`, `data-bind`, `data-show`, `data-class`, `data-on:click`) — no hand-written rendering JS. The old 977-line `dashboard.js` SSE client/rendering engine is replaced by `datastar.js` (~56KB runtime) + a ~180-line keyboard nav/export helper script. Per-row `data-signals` + `data-show` expressions enable instant client-side search/filter/pagination without server roundtrips. The SSE handler coalesces event bursts via `drainEvents` (non-blocking channel drain) before re-rendering. Reconnection sends a fresh full snapshot (all fragments) — the snapshot IS the replay.
- **go-sse full adoption** (since v0.4.0, now v0.5.1): `live/` uses `sse.Stream` (connection lifecycle, `Send`/`SendLines`/`SendKeyed`, `Heartbeat` goroutine), `sse.Broadcaster[sse.Event]` (fan-out with `Shutdown`/`Health`), and `sse.KeyedLines` (datastar wire format helper). The Hub is a thin facade over `Broadcaster`. `handleSSE` keeps the flusher check before `NewStream` (AD3). The SSE handler uses hub events as render triggers: on each event, it drains the channel (burst coalescing), re-renders all fragments from `plugin.Report()`, and sends them as `datastar-patch-elements` events. The old event-by-event JSON replay is replaced by snapshot-on-reconnect.

---

## Testing Patterns

- Standard `testing.T` + table-driven tests. No ginkgo/testify in this project.
- Each test creates its own `Plugin` + `do.Injector` — no shared state.
- `t.Setenv()` for testing `DO_AUDITLOG_ENABLED` env var behavior.
- `t.TempDir()` for file export tests.
- Benchmarks exist in the test file for performance measurement.
- **Shared test helpers** in `helpers_test.go` (external test package `auditlog_test`):
  - **Provider factories** — `provideDB`, `provideCacheWithSleep`, `provideCache`, `provideHealthyDB`, `provideUnhealthyCache`, `provideFailing`, `provideCrashing`, `provideString`, `provideUserServiceWithDB`, `provideUserServiceWithDeps`, `provideHTTPServerWithUsers`.
  - **Service lookup** — `findServiceByName(t, report, name)`, `findServiceBySuffix(t, report, suffix)`.
  - **Plugin construction** — `newPluginAndInjector()`, `newPluginAndInjectorWithID(id)`, `newPluginWithCapture()`.
  - **Assertion helpers** — `assertVersion`, `assertIntField`, `assertStringField`, `assertContainerID`, `assertServiceCount`, `assertEventCount`, `assertDependenciesCount`, `assertServiceIntField`, `assertServiceInvocationCount`, `assertServiceHealthCheckCount`, `assertReportServiceCount`, `assertFilteredServiceCount`, `assertUnhealthyServiceCount`, `assertHTMLContains`, `assertStringContains`, `assertAllEventsOfType`, `assertAllEventsForService`, `assertErrorExpected`, `assertReportValid`, `requireOneService`, `unmarshalJSONForTest`.
- Tests cover: disabled/enabled toggle, env var values, registration/invocation, dependency tracking, shutdown tracking (clean and error), scope tree, scope_id correctness, export formats (JSON, NDJSON, HTML to file and writer), error paths, container_id propagation, report version, event sequence numbers, empty report, concurrent invocations, ServiceStatus computation across all states, transient and value providers, health checks (healthy/unhealthy/multiple/disabled/count/report/scope/UnhealthyServices).
- **Duplication policy**: art-dupl at `-t 15` with `--semantic` is the standard gate; the codebase is also clone-free at the aggressive `-t 3` threshold (zero groups, zero occurrences in production AND test code). **Zero harmful clones.** Test helpers (`mkEvent`, `mkEventWithDur`, `mkRegEvent`, `rootRef`, `assertEqual[T comparable]`, `newPluginAndInjector`, assertion wrappers) centralize struct creation, plugin setup, and assertions to prevent drift. Shared production helpers: `getOrCreateServiceRecord(evt)` (replay path), `recordDependencyFromStack`, `buildServiceDeps`, `depRecToRef`, `sortServiceInfos`, `buildScopeTreeFromMeta` (generic), `newFlagSet` (cmd/), `fireEvent` (hooks), `publishLockedEvent` (hooks — append + unlock + fire), `beginLockedBeforeHook` (hooks — context + lock + recordScope), `renderGraphDiagram` (diagram.go — SetNodes + SetEdges + DedupEdges + writeRendered for DOT/Mermaid/PlantUML), `graphRendererWithDedup` interface (subset of go-output renderers embedding `GraphRendererState`). Enum metadata uses map-based lookups (`eventTypeMetaTable`, `providerTypeMeta`, `serviceStatusIcons`) — single source of truth for Label/Icon/Color.
- **Coverage**: see FEATURES.md for current numbers. Gate is ≥94% (excludes `example/` and `cmd/`). Nearly all tests use `t.Parallel()` — only `t.Setenv()` env-var tests run sequentially.
- **HTML visualization features**: 5-tab layout (Services/Scopes/Graph/Timeline/Events), services table with type badges + status badges + shutdown duration + reverse deps + health column + search filter, collapsible scope tree with type emoji chips, Sugiyama layered DAG graph with type-colored nodes + pan/zoom + click-to-highlight, dual build+shutdown timeline bars with type icons, event type filter chips (registration/invocation/shutdown/health_check), keyboard nav (1-5), animated tab transitions, stat cards (including health checks when checked), responsive layout, footer with schema version.
- **Tree export** (`WriteTree` / `WriteHTMLTree`): ASCII tree and HTML nested-list tree of the service dependency DAG, via go-output renderers. Available on both `Report` and `Plugin`.
- **Table export** (`WriteTable`): 16+ format table export of service summary (Service, Scope, Type, Status, Invocations, Build(ms), Error) via go-output `RenderTable`. Formats include: table, json, csv, tsv, markdown, xml, d2, yaml, html, tree, mermaid, dot, jsonl, asciidoc, toml, plantuml. Available on both `Report` and `Plugin`.
- **Service type tracking**: `ServiceInfo.ServiceType` field (JSON: `service_type`) populated from `do.ExplainNamedService`. Values: "lazy", "eager", "transient", "alias". Displayed with samber/do's canonical emojis throughout the HTML visualization.

---

## Example

The `example/` package (split across `main.go`, `register.go`, `services.go`, and `summary.go`) demonstrates every major samber/do v2 feature with a ride-sharing domain model. 23 features verified by a self-checking feature checklist:

| Feature                  | API                                                                                                                                 |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------------------------- |
| Container with hooks     | `do.NewWithOpts(plugin.Opts())`                                                                                                     |
| Eager value injection    | `do.ProvideValue`, `do.ProvideNamedValue`                                                                                           |
| Lazy singletons          | `do.Provide`                                                                                                                        |
| Named services           | `do.ProvideNamed`, `do.MustInvokeNamed`                                                                                             |
| Transient providers      | `do.ProvideTransient`                                                                                                               |
| Interface aliasing       | `do.As[*EmailNotifier, Notifier]`                                                                                                   |
| Override (hot-swap)      | `do.OverrideValue`                                                                                                                  |
| Child scopes             | `injector.Scope("drivers")`                                                                                                         |
| Cross-scope dependencies | MatchingEngine invokes from driver/passenger scopes                                                                                 |
| Dependency graph         | Auto-inferred from provider call chains                                                                                             |
| Health checks            | `do.Healthchecker`, `do.HealthcheckerWithContext`, `plugin.RecordHealthCheck*`                                                      |
| Health check audit       | `EventTypeHealthCheck`, `ServiceInfo.HealthCheckCount`, `Report.HealthCheckSucceeded`                                               |
| Graceful shutdown        | `do.ShutdownerWithError`, `injector.Shutdown()`                                                                                     |
| Invocation errors        | `UnreliableService` provider returns error                                                                                          |
| Shutdown errors          | `LeakyService.Shutdown()` returns error                                                                                             |
| Build duration           | Millisecond-precision per service                                                                                                   |
| Scope tree               | Root → 3 child scopes with service listings                                                                                         |
| OnEvent callback         | Real-time event streaming via `Config.OnEvent`                                                                                      |
| Convenience methods      | `Report.ServiceByName`, `ServiceByRef`, `ServicesByScope`, `EventsByService`, `EventsByType`, `FailedServices`, `UnhealthyServices` |
| Event helpers            | `Event.Duration()`, `ServiceInfo.Uptime()`, `Plugin.EventsCount()`                                                                  |
| Report filtering         | `Report.Filtered(opts...)`, `Plugin.ReportFiltered(opts...)` with 5 filter options                                                  |
| Export enhancements      | `ExportFilteredToFile(path, opts...)`, `Report.WriteMermaid(writer)`                                                                |
| Service type tracking    | Auto-detected via `do.ExplainNamedService`                                                                                          |
| Live dashboard           | `go run ./example --live` starts the real-time SSE dashboard alongside the demo                                                     |

- **Website demo video** (2026-09-01): a 25s silent HyperFrames promo lives at `website/video/videos/do-auditlog-demo/` (composition committed; renders in `renders/`), deployed as `website/public/demo.mp4` and embedded above the fold in the landing hero (`#demo` anchor). Re-render: `cd website/video/videos/do-auditlog-demo && HYPERFRAMES_BROWSER_PATH=$(ls -d /nix/store/*-chromium-*/bin/chromium | head -1) nix shell nixpkgs#nodejs -c node ../../node_modules/hyperframes/dist/cli.js render ...` (invoke the CLI directly; `npx` wrappers fail on NixOS). Size target <3MB — re-encode with `ffmpeg -crf 25` if a render exceeds it (25s @ 1080p ≈ 1.1MB at CRF 25).
- **Website toolchain pins**: `website/pnpm-workspace.yaml` sets `allowBuilds: {esbuild: true, sharp: true}` (pnpm v11 blocks native postinstalls otherwise) and `website/package.json` must keep `typescript: ^6.0.3` — TypeScript 7 (tsgo) crashes `astro check` (`assertCompatibleTypeScript`). `astro check` = 0 errors and `html-validate dist/**/*.html` are the website quality gates; CI (`.github/workflows/website.yml`) deploys on push to master touching `website/**`.
- **Website retrofit state** (2026-09-01): Starlight `lastUpdated` + `editLink` enabled; OG image (`public/images/og-image.jpg`, 1200x630, generated from the demo-video poster); new `guides/live-dashboard.mdx`; all docs pages carry curated "Where to go next" sections; README has "Who is this for?", "When NOT to use this", and the `GOEXPERIMENT=jsonv2` consumer requirement. Live site: `do-auditlog.lars.software` (Firebase shared project `lars-software`, hosting target `do-auditlog`).

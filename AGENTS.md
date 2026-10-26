# AGENTS.md — samber-do-auditlog

Go plugin for [samber/do v2](https://github.com/samber/do) that records every DI container lifecycle event (registration, invocation, shutdown) with timestamps, dependency graph inference, build duration tracking, and export to JSON / NDJSON / self-contained HTML.

**Module**: `github.com/larsartmann/samber-do-auditlog` · **Package**: `auditlog` · **Go**: 1.27.1 (go.mod minor-only `go 1.27` + devShell) · **Status**: BETA (per [STABILITY.md](STABILITY.md); internal 1.0 bar tracked in ROADMAP.md)

---

## Commands

| Command                                                                             | Purpose                                                                                                                                                             |
| ----------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `go generate ./...`                                                                 | Regenerate templ (and any other generated code)                                                                                                                     |
| `go test ./...`                                                                     | Run all tests                                                                                                                                                       |
| `go test -race ./...`                                                               | Run all tests with race detector (CI uses this)                                                                                                                     |
| `GOEXPERIMENT=jsonv2 go test -race -coverprofile=cover.out -covermode=atomic ./...` | Tests with coverage (CI gate: ≥94% of non-`example/`/`cmd/` code) — `GOEXPERIMENT=jsonv2` is set automatically in the Nix devShell and CI                           |
| `go test -run TestPlugin_DisabledIsNoOp`                                            | Run single test                                                                                                                                                     |
| `go vet ./...`                                                                      | Static analysis (**`GOEXPERIMENT=jsonv2` required** — set automatically in Nix devShell)                                                                            |
| `golangci-lint config verify`                                                       | Validate the lint config (CI runs this before `lint run`)                                                                                                           |
| `golangci-lint run`                                                                 | Full lint (heavy config, see below)                                                                                                                                 |
| `go mod tidy`                                                                       | Sync `go.sum` (CI `mod-tidy` job fails on drift)                                                                                                                    |
| `nix develop`                                                                       | Enter devShell (Go 1.27.1, golangci-lint, govulncheck, actionlint, golines, **GOEXPERIMENT=jsonv2 enabled**)                                                        |
| `go run ./example`                                                                  | Run the example (set `DO_AUDITLOG_ENABLED=true`)                                                                                                                    |
| `go run ./example --live`                                                           | Example + real-time SSE dashboard                                                                                                                                   |
| `go run ./cmd/auditlog help`                                                        | CLI: info/convert/diff/validate/stats/schema subcommands                                                                                                            |
| `go install ./cmd/auditlog`                                                         | Install the `auditlog` CLI to `$GOBIN`                                                                                                                              |
| `nix run .#auditlog -- help`                                                        | Run the CLI via Nix (no install)                                                                                                                                    |
| `nix run .#coverage`                                                                | Run the CI-equivalent coverage gate via Nix                                                                                                                         |
| `sh scripts/coverage-gate.sh`                                                       | Coverage gate (exclusions single-sourced in `scripts/coverage-exclusions.txt`; ≥94%)                                                                                |
| `sh scripts/check-go-version.sh`                                                    | Go-version drift guard: go.mod == ci.yml == flake GOTOOLCHAIN == .golangci.yml (CI + pre-commit)                                                                    |
| `sh scripts/check-doc-claims.sh`                                                    | Claims linter (12 checks): go/schema version, coverage gate, linter+fuzz+benchmark+CI-job+diagram counts, env-var spelling vs machine truth, website TypeScript pin |
| `sh scripts/check-changelog-sync.sh`                                                | CHANGELOG.md ↔ website changelog.mdx version-list sync (website CI)                                                                                                 |
| `git config core.hooksPath scripts/hooks`                                           | Install the pre-commit hook                                                                                                                                         |

A `flake.nix` devShell is available for Nix users. No Makefile, no justfile.

---

## Forwarding (PapDashboard collector)

`forward/` streams events to a PapDashboard audit-run collector (unix socket
by default; HTTP targets with a bearer key). `live.New` auto-attaches an
enabled Forwarder — `DO_AUDITLOG_FORWARD_TARGET=off` keeps a dashboard local.
Batching 250ms / 200 events (drop-oldest); `Shutdown` flushes. Completion is
derived from the root-scope shutdown event AND `SignalComplete` (the server
applies the same predicate). Stdlib-only (public repo; no new dependencies).

## Architecture

Single-package library (`auditlog`) + `live/` SSE dashboard sub-package +
`forward/` PapDashboard collector + `cmd/auditlog` CLI + `cmd/genschema`
tooling. The full source-file map, `live/` inventory, and data-flow walkthrough
live in [docs/architecture.md](docs/architecture.md). Non-obvious anchors:

- **Generated, never hand-edit**: `html_templ.go` / `live/fragments_templ.go`
  (`go tool templ generate`), `schema/report.schema.json` (`cmd/genschema`,
  change struct tags and regenerate).
- `health/` was extracted to [go-health](https://github.com/larsartmann/go-health);
  `*Plugin` satisfies its `HealthRecorder` via `RecordHealthCheckWithContext`.
- `testhelpers/` is exported for downstream integration testing.

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

**The project requires `GOEXPERIMENT=jsonv2` to build, and consumers need it too** (a downstream importer fails with `imports encoding/json/v2: build constraints exclude all Go files` — verified with a minimal consumer). Reason: `go-output`, `go-branded-id`, and `go-ndjson` use `encoding/json/v2`; this project's own code must NOT import it (exclusion policy in Gotchas). Go 1.27 stabilized json/v2 but the flag requirement STAYS — transitive deps built from source still sit on lower go directives and fail the go1.27 symbol gate (see CONTRIBUTING.md); re-evaluate as the graph converges on 1.27. Set automatically in: Nix devShell, CI, `scripts/coverage-gate.sh`, direnv (`.envrc`), `.buildflow.yml`. README Install + website Installation page document it; keep them in sync.

### Go 1.27 toolchain pin

**The canonical Go version is 1.27.1** (go.mod carries minor-only `go 1.27` per the fleet policy; the devShell/CI/GOTOOLCHAIN pin the patch), pinned across (1) `go.mod`, (2) `.github/workflows/ci.yml` (`1.27.1`), (3) `flake.nix` (`go_1_27` package + `buildGo127Module` builder + `GOTOOLCHAIN = "local"` — go_1_27 IS 1.27.1, and `local` stops Go from re-downloading the exact toolchain), (4) `CONTRIBUTING.md`/`BENCHMARKS.md`. `scripts/check-go-version.sh` asserts all agree: patch-level extensions of the go.mod minor are accepted everywhere; a flake `GOTOOLCHAIN=local` is validated against its `go_1_MM`/`buildGo<MM>Module` refs.

**Rule: bump `go-version` in ci.yml and `GOTOOLCHAIN` in flake.nix in the SAME commit as any `go.mod` bump.** GitHub runners set `GOTOOLCHAIN=local`; a CI go-version below go.mod's requirement fails every Go job instantly. (This exact mismatch broke all Go jobs on master once — see git history.)

**Gotcha:** a separately-installed `go` nix-store derivation can shadow the devShell's `go_1_27` on PATH; the `GOTOOLCHAIN` env var pins the effective toolchain anyway. Outside the devShell, prefix with `GOTOOLCHAIN=go1.27.1`.

**goreleaser coupling:** CI pins goreleaser `v2.18.2` because v2.18.0+ declares `go >= 1.27.0` and fails under runner `GOTOOLCHAIN=local` on older Go. Bump the pin only together with CI's Go version (`goreleaser check` locally first).

---

## CI

`.github/workflows/ci.yml` runs on every push and PR (plus a weekly cron) with 9 parallel jobs:

- **test**: `go vet`, `go build`, `go test -race` with coverage, and the **coverage gate** (≥94% of non-`example/`/`cmd/` statements; exclusions from `scripts/coverage-exclusions.txt`; per-function step summary) + `check-go-version.sh` drift guard.
- **lint**: `golangci-lint config verify` then the pinned `golangci-lint v2.14.0` run (binary cached; the devShell binary may lag — see Gotchas).
- **healthwash**: `samber-linter` baseline ratchet pinned `@v0.2.2` (see the section below).
- **vulncheck**: govulncheck pinned `@v1.7.0`.
- **mod-tidy**: `go mod tidy` fails on `go.sum` drift (transport-flake retry wrapper).
- **stale-generation**: `go generate ./...` fails on diff (uses `go tool templ` from the go.mod `tool` directive — no manual install) + committed-generated-files guard (a v0.9.0-retraction-class failure).
- **actionlint**: workflow lint.
- **goreleaser**: `goreleaser check` on `.goreleaser.yml`.
- **example-smoke**: runs the example end-to-end; fails if it exits non-zero (log.Fatalf) or panics.

`website.yml` builds/deploys the docs site on pushes touching `website/**` (see Website section below).

## Health-wash ratchet (samber-linter)

`.samber-linter-baseline.json` (schema v2) locks the health-wash floor at **12/20 = 60%** coverage: the ratchet fails on any new unchecked registration or coverage-ratio drop. Regenerate deliberately with `samber-linter --set-baseline ./...` (e.g. `go run github.com/larsartmann/samber-linter/cmd/samber-linter@<version>`, needs `GOEXPERIMENT=jsonv2`). CI: the `healthwash` job in `ci.yml` pins `@v0.2.2` — the first release with baseline schema v2 (loud validation + gate composition; v0.2.1 carries both driver defects). Bump the pin deliberately and re-run locally first. Verified green at v0.2.2 on 2026-09-20 (12/20 = 60%, baseline acknowledged).

## Lint Configuration (.golangci.yml)

Extremely strict — nearly every golangci-lint linter enabled (~107). Key implications:

- **exhaustruct_v5 is DISABLED** (dropped 2026-10-07): bundled analyzer builds panic (`makeslice: cap out of range`) on this repo's core/live packages; golangci-lint-auto-configure removed the dead settings block 2026-10-26. `newEventFromRef()` and `newServiceRecordCore()` still centralize struct-field init by convention. Re-enable only from a golangci-lint release proven panic-free here.
- **depguard**: per-path import rules — keep external non-stdlib deps out of `cmd/` (sole exception: `cmd/genschema` may use `invopop/jsonschema`).
- **noinlineerr**: `err := ...` then check, not `if err := ...; err != nil`.
- **forbidigo**: `fmt.Print*` forbidden in non-example code.
- **goconst config key is `min-len`, NOT `min-length`** — every golangci-lint version rejects `min-length` at `config verify` with exit 3, killing the CI Lint job before `lint run` even starts.
- Test relaxations (`*_test.go`): testpackage, gochecknoglobals, funlen, cyclop, goconst.
- Formatters: gci, goimports, gofumpt, golines (max-len 120).

---

## Gotchas

- **`.envrc` must NOT pin `GOTOOLCHAIN`** — the file is untracked, so pins
  rot invisibly: a leftover `GOTOOLCHAIN=go1.26.5` made every go command fail
  with "go.mod requires go >= 1.27.1" (removed 2026-10-04). The devShell's
  flake.nix `GOTOOLCHAIN = "local"` + `go_1_27` package pin is the only
  canonical location.
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
- **`encoding/json/v2` exclusion policy**: no `.go` file in this project may import `encoding/json/v2`/`jsontext` (they're behind `//go:build goexperiment.jsonv2`). Transitive deps use it — hence GOEXPERIMENT. The policy survived the Go 1.27 bump: go-auto-upgrade's gates now make the v1→v2 swap fail-safe instead (see next bullet).
- **BuildFlow config** (`.buildflow.yml`): non-default keys only — `default_step_timeout: 5m` (fuzz targets need it) and GOEXPERIMENT auto-detection (BuildFlow sets it for tool subprocesses; `config view` doesn't display it). `go-auto-upgrade` runs UNSKIPPED since 2026-09-22 (gau v0.6.3 fixed the codec-struct-field half-rewrite; its go-version/dependency/unclaimed-constructor gates keep the json/v2 swap fail-safe — remaining v1-boundary findings in `forward/` are advisory warnings, reviewed not fixed).
- **Test helpers** (helpers_test.go + friends): `mkEvent`/`mkEventWithDur`/`mkInvAfterWithDur`, `mkRegEvent` (cmd), `setupWithDB`, `replayFromPlugin`, `newPluginAndInjector[WithID]`, `newPluginWithCapture`, assertion wrappers (`assertReportValid`, `assertErrIs`, …), struct factories (`rootRef`, `csvServiceRef`, `mkNewReport`). Use these instead of inline struct literals to keep art-dupl clone-free.
- **Duplication policy**: art-dupl `-t 3 --semantic`, zero harmful clones in production AND tests (policy at `-t 15` gate is exceeded). Enum metadata uses map-based lookups (`eventTypeMetaTable`, `providerTypeMeta`, `serviceStatusIcons`).
- **godoclint false positive**: "package has more than one godoc" — it counts the `// templ: version:` header in generated `html_templ.go`; suppressed via a text exclusion rule in `.golangci.yml`.
- **Pre-commit hook** (`scripts/hooks/pre-commit`) is BuildFlow-generated (`buildflow precommit install`): compiles + vets every workspace module, then `buildflow --build-mode pre-commit --staged-only` (re-stages formatter output). It does NOT run the claims linter — run `sh scripts/check-doc-claims.sh` manually. Bypass: `git commit --no-verify`. Local `core.hooksPath` can silently rot — re-run `git config core.hooksPath scripts/hooks` after fresh clones.
- **`.prettierignore`** excludes `testdata/`, `schema/`, `docs/`, `CHANGELOG.md` from oxfmt — without it the golden HTML fixture, generated schema, and status reports get reformatted and break tests/CI.
- **Website workflow needs pnpm on the runner** — runners don't ship pnpm; `pnpm/action-setup` must run BEFORE `actions/setup-node` (cache: pnpm fails otherwise).
- **Proxy transport flakes**: a red job with `stream error … INTERNAL_ERROR; received from peer` on module download is proxy.golang.org instability — `gh run rerun --failed` after a cooldown; never "fix" code for it. CI's retry wrappers cover mod-tidy/generate.
- **Version-skew ledger**: (empty — the `live/fragments.go` goconst entry was retired with the 0.11.0 ProviderType-constants fix and the CI pin bump to v2.14.0.)
- **devShell golangci-lint**: resolved 2026-10-26 — `nix flake update nixpkgs` (2026-09-16 → 2026-10-23) brought **v2.15.0** into the devShell (CI pins v2.14.0 and stays canonical; if nixpkgs moves ahead again, verify locally with the CI version). Done by hand because BuildFlow's `nix-flake-update`/`nix-hash-fix` steps exit "not applicable (language mismatch (project: go))" on this Go+Nix repo — a verified BuildFlow defect; revisit if the fleet policy changes.
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
- The HTML report's feature inventory (5-tab layout, waveform, Sugiyama DAG, filter chips, pagination, etc.) is owned by FEATURES.md; example/ is a runnable feature tour — the `example-smoke` CI job runs it end-to-end (exit 0 = the demo executed cleanly; the Unreliable/Leaky "failures" are intentional showcase).

---

## Example

The `example/` package (`main.go`, `register.go`, `services.go`, `summary.go`) demonstrates the library across a ride-sharing domain model (20 services in 4 scopes; the exercised samber/do APIs are enumerated in `main.go`'s doc comment). CI runs it as the `example-smoke` job. Run with `DO_AUDITLOG_ENABLED=true go run ./example`; add `--live` for the dashboard. The printed summary lives in `example/summary.go`.

---

## Website

The docs site (`website/`, Astro + Starlight + Tailwind v4 + Firebase Hosting) deploys on pushes touching `website/**` via `.github/workflows/website.yml` → **do-auditlog.lars.software** (Firebase shared project `lars-software`, hosting target `do-auditlog`). Quality gates: `astro check` = 0 errors, `html-validate dist/**/*.html`, `check-changelog-sync.sh` (CHANGELOG ↔ changelog.mdx).

- **`website/package.json` must keep `typescript: ^6.0.3`** — TypeScript 7 (tsgo) crashes `astro check` ("astro check does not currently support TypeScript 7.0"; verified against astro 7.3.5 + @astrojs/check 0.9.10 on 2026-10-26). The pin has regressed twice via bulk dependency/auto-commit blobs (restored in d092158, re-broken in f9c7491); it is now guarded by claims-linter check 12 AND a website.yml grep step before `astro check`. When astro officially supports TS 7 (experimental `ts-content-mapper` path exists), bump the pin, this note, and both guards together.
- **`website/pnpm-workspace.yaml`** sets `allowBuilds: {esbuild: true, sharp: true}` (pnpm v11 blocks native postinstalls otherwise).
- **Demo video**: 25s promo composition at `website/video/videos/do-auditlog-demo/`, deployed as `website/public/demo.mp4` (`#demo` anchor). Re-render via the HyperFrames CLI directly (`node …/hyperframes/dist/cli.js render`; `npx` wrappers fail on NixOS — set `HYPERFRAMES_BROWSER_PATH` to a nix chromium). Size target <3MB (`ffmpeg -crf 25`).

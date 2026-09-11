# Roadmap

Long-term direction and raw ideas not yet refined into actionable tasks.
For short-term work, see [TODO_LIST.md](TODO_LIST.md). For shipped features, see [FEATURES.md](FEATURES.md).

---

## Stability Path

The public stability stage is **BETA** ([STABILITY.md](STABILITY.md)): the API is stabilizing but breaking changes are still possible before 1.0.

The internal quality bar on the path to 1.0:

1. ~~`go-sse` and `go-ndjson` published to GitHub with stable tags~~ ✓ (replace directives removed). The temporary `go-output/testhelpers` replacements have also been removed; explicit indirect requirements select their valid published tags over upstream's broken pseudo-versions.
2. `live/` sub-package coverage above 90% — **currently 79.8%** (2026-09-11 gate run; the datastar/templ rewrite and post-cleanup growth outpaced live tests — per-function data 2026-09-02 confirms the templ fragment renderers are the dominant gap, mostly 58–78% coverage). This is the main outstanding internal-bar item.
3. `go-sse` dependency tracking is current (v0.5.1); keep the family of sibling libraries (go-output, go-sse, go-ndjson, go-atomic-write, go-error-family) on green, non-retracted tags.

The coverage gate (94%, currently at 95.5%) and the `GOEXPERIMENT=jsonv2` flag are stable in CI. The live dashboard has reached feature parity with the static HTML export.

The path from BETA to 1.0 is:

1. API surface frozen — no more breaking changes without migration guide
2. Schema version stays at `0.3.0` unless report format changes
3. Semantic versioning discipline: breaking Go API changes bump minor version

---

## Go 1.27+ Migration

When Go 1.27 stabilizes `encoding/json/v2`:

- **Drop the `GOEXPERIMENT=jsonv2` requirement** — the `go-ndjson` module can ship without build-constraint hacks
- **Evaluate json/v2 adoption in this project's own code** — currently uses `encoding/json` (v1). Json/v2 offers safer escaping, streaming, and `jsontext` for low-level control. Migration is optional but would align with `go-ndjson`.
- **Revisit the `encoding/json/v2` exclusion policy in AGENTS.md** — the policy exists because json/v2 is behind a build constraint in Go 1.26.x. Once stable, the exclusion should be lifted.
- **Revisit the goreleaser v2.17.1 pin** — v2.18.0+ requires Go ≥ 1.27; the pin exists only because runners force `GOTOOLCHAIN=local`.
- **Consider typed generics** — `Register[T any](name ServiceName)` for type-safe service registries. Currently not possible because samber/do v2's hook interface is string-based.

---

## Live Dashboard Evolution

The `live/` sub-package has reached feature parity with the static HTML export:

- **Shipped**: scope tree tab, pagination, export buttons, CORS support, live demo (`live/demo/main.go`), `example/ --live` integration, SSE ring buffer replay, keyboard navigation, datastar-powered reactivity
- **Shared CSS** ✓ — `DesignTokensCSS` (`design_tokens.go`) is the single source of truth, enforced by `TestDesignTokensInSync` and `TestSharedComponentCSSInSync`
- **Raise live/ test coverage** — 79.8% today vs the 90% internal bar; the templ fragment renderers are the biggest untested surface (confirmed by per-function coverage 2026-09-02: `fragments_templ.go` functions at 58–78%, plus two 50% handlers in `server.go`)
- **live/ benchmarks** — Hub broadcast, SSE stream, and fragment-render throughput have no benchmarks at all (root package has 12)
- **Dark/light theme toggle** — The warm amber aesthetic currently has no light variant
- **Cross-origin CSP** — CORS headers are set but `connect-src 'self'` blocks cross-origin dashboard embedding; needs configurable CSP or documentation of the limitation

---

## Ecosystem & Distribution (from the 2026-09-01 launch session)

Ideas from shipping the website + demo video, not yet scheduled:

- **9:16 vertical demo cut** (Shorts/TikTok) and **animated GIF teaser** (≤6s, 480p) for the README
- **Launch post copy** derived from the README one-narrative; **YouTube version**
- **samber/do ecosystem PR** — add do-auditlog to samber/do's README ecosystem section (upstream, needs owner approval)
- **Privacy-friendly analytics** to learn whether the demo converts
- **Blog-style "How dependency inference works" deep-dive** — the invocation-stack story is the best content asset
- **Diff-page demo** — showcase `Report.Diff` with before/after JSONs (CI/CD use case)
- **Interactive playground** — embed a live HTML report on the website where users paste a report JSON and see the visualization

---

## CI Resilience Ideas

Born from the 2026-09-01 outage post-mortem (33 days of red master):

- **Transport-flake tolerance** — proxy.golang.org errors can redden any push; the retry wrapper (TODO_LIST) plus a written rerun playbook covers this
- **GOPROXY hardening** — evaluate alternate proxy or vendor/ for CI; investigate why setup-go's cache didn't shield mod-tidy from downloads
- **Per-job `timeout-minutes`** so transport-hang failures fail fast and rerun cheaply
- **Version-skew ledger** — directives that differ between local and pinned tool versions (e.g. `live/fragments.go:181` nolint, retired when the golangci-lint pin bumps ≥ 2.13) — tracked in AGENTS.md

---

## API Design Ideas (Raw, Not Actionable)

These are ideas that need design exploration before becoming TODO items:

- **Diff deepening** — beyond the shipped `DepsChanged`: a `ScopeDiff` type, event-type deltas, and a `--format` for CI-friendly diff output. Requested June 2026, design never explored.
- **Event schema versioning** — NDJSON event lines carry no `schema_version` (only the report does); replay of future-schema event streams is silently lossy. Needs a design before the next schema bump.
- **Schema-validation mode** — optional `auditlog validate --schema` against the embedded JSON Schema via a validator dependency (stdlib-only CLI decision currently blocks this).
- **Docker image for the CLI** — goreleaser docker section; requested June 2026.
- **`docs/releases/` practice** — pre-write `docs/releases/vX.Y.Z.md` notes and cross-link from CHANGELOG (the v0.7.1/v0.9.0 hotfix cycles showed release notes assembled ad hoc).
- **Branded type enforcement** — `ContainerID`, `ScopeID`, `ServiceName` are named string types but carry no validation. Consider constructors (`NewServiceName(string) (ServiceName, error)`) that reject empty strings or whitespace. Blocked on deciding whether validation belongs in the type or in the constructor.
- **Event type splitting** — Currently `Event` is one struct with a `Phase` field (before/after). Making before-events and after-events separate types would make impossible states unrepresentable (e.g., a "before" event with a `DurationMs`). Large blast radius; needs careful migration plan.
- **ServiceInfo sub-struct placement review** — `IsShutdowner` is in `ServiceLifecycle` but `IsHealthchecker` is in `ServiceHealth`. Both are capability flags detected by `do.ExplainInjector`. This split-brain may be wrong. Moving `IsShutdowner` to `ServiceHealth` changes JSON field order.
- **`ScopeName` as a named type** — Currently plain `string`, unlike `ScopeID` and `ServiceName`. Left as `string` because it's display-only. Consistency gap.
- **Duration as `time.Duration`** — `DurationMs` is `*float64`. Could be `time.Duration` for idiomatic Go, but this would change the JSON schema and break consumers.
- **`go-atomic-write` `WriteFuncVerified` / `WriteIfChanged` evaluation** — audit exports use plain `WriteFunc`; the verified/fingerprint and change-detecting variants were never evaluated for the export paths.
- **Diagram regression tests beyond D2/DOT** — Mermaid/PlantUML hex-color quoting has no regression test (D2 and DOT do); cross-format color-consistency test would catch them together.
- **live/ extensibility hooks** — `Server.Handle(pattern, handler)` for mounting extra routes next to the dashboard, and `Hub.OnSubscribe`/`OnUnsubscribe` callbacks. Requested July 2026, never designed.
- **Replay fidelity gaps** — `ReplayEvents` restores neither capability flags (`IsHealthchecker`/`IsShutdowner`, always false on replay — `replay.go` documents it) nor the parent/child scope tree (flattened). Both are accepted limitations today; fix if replay fidelity becomes a use case.
- **Accessibility long-tail** — ARIA grid tables, `aria-live` announcements (filter counts, SSE connect/reconnect), WCAG contrast audit, graph keyboard traversal. The WAI-ARIA tablist/dialog baseline shipped; these are the follow-ups.

---

## Documentation Depth

Ideas for deeper documentation (website + README):

- **Comparison section** — Real competitor analysis vs manual logging, vs OpenTelemetry, vs pprof
- **Migration guide** — Step-by-step for v0.1.0 to v0.2.0+ (MigrateReport exists but has no docs page)
- **Architecture deep-dive** — Single-package design, concurrency model, hook system, invocation-stack dependency inference
- **JSON-schema-driven validation example** — schema + `auditlog validate` in a CI pattern
- **Per-page feedback links** (pre-filled issue title) next to "Edit this page"
- **Auto-generated social cards for every docs page** (astro-og-canvas pattern)

---

## Open Questions (owner input needed)

Owner-blocking questions with task context live in [TODO_LIST.md](TODO_LIST.md); only product-direction questions live here.

- Hero video: stay click-to-play, or muted autoplay/loop above the fold?
- Deploy path: is "push-triggered CI deploy" the definition of deployed, or is manual `firebase deploy` a supported runbook step?

---

## Explicitly Rejected

Proposals evaluated and deliberately not pursued. Documented here so they are not re-proposed without new context.

- **Multi-module split** — Project is too small (1 package, ~2500 LOC core + live/ sub-package). Revisit at 5+ packages.
- **External storage backends** — File and `io.Writer` exports are sufficient.
- **Prometheus/OpenTelemetry integration as a dependency** — Out of scope. Use `OnEvent` callback instead.
- **`samber/lo` dependency** — Current stdlib `slices`/`cmp` usage is sufficient.
- **`encoding/json/v2` migration in this project** — Current `encoding/json` works fine. The transitive dependency through `go-output` requires `GOEXPERIMENT=jsonv2` but this project's own code does not and should not import `encoding/json/v2`. Risk of breaking JSON output format for consumers.

---

## Observability Integration (Low Priority)

- **Prometheus metrics interface** — Optional `events-sent`, `clients-connected`, `events-dropped` metrics from the live Hub
- **OpenTelemetry bridge** — Generate spans from audit events (reference example exists in `docs/examples/`)
- **Structured logging adapter** — `OnEvent` callback that writes to slog/zap/zerolog

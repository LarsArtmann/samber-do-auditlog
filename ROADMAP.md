# Roadmap

Long-term direction and raw ideas not yet refined into actionable tasks.
For short-term work, see [TODO_LIST.md](TODO_LIST.md). For shipped features, see [FEATURES.md](FEATURES.md).

---

## Stability Path

The project is in **ALPHA**. The path to BETA requires:

1. `go-sse` and `go-ndjson` published to GitHub with stable tags ✓ (replace directives removed). The temporary `go-output/testhelpers` replacements have also been removed; explicit indirect requirements select their valid published tags over upstream's broken pseudo-versions.
2. `live/` sub-package coverage above 90% (currently 89.7%)

The coverage gate (94%) and `GOEXPERIMENT=jsonv2` flag are already stable in CI. The live dashboard has reached feature parity with the static HTML export.

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
- **Consider typed generics** — `Register[T any](name ServiceName)` for type-safe service registries. Currently not possible because samber/do v2's hook interface is string-based.

---

## Live Dashboard Evolution

The `live/` sub-package has reached near-feature-parity with the static HTML export:

- **Shipped**: scope tree tab, pagination, export buttons, CORS support, live demo (`live/demo/main.go`), `example/ --live` integration, SSE ring buffer replay
- **Shared CSS** ✓ — `DesignTokensCSS` (`design_tokens.go`) is the single source of truth, enforced by `TestDesignTokensInSync`
- **Dark/light theme toggle** — The warm amber aesthetic currently has no light variant
- **Fix `example/ --live` premature shutdown** — See [TODO_LIST.md](TODO_LIST.md)
- **Cross-origin CSP** — CORS headers are set but `connect-src 'self'` blocks cross-origin dashboard embedding; needs configurable CSP or documentation of the limitation

---

## API Design Ideas (Raw, Not Actionable)

These are ideas that need design exploration before becoming TODO items:

- **Branded type enforcement** — `ContainerID`, `ScopeID`, `ServiceName` are named string types but carry no validation. Consider constructors (`NewServiceName(string) (ServiceName, error)`) that reject empty strings or whitespace. Blocked on deciding whether validation belongs in the type or in the constructor.
- **Event type splitting** — Currently `Event` is one struct with a `Phase` field (before/after). Making before-events and after-events separate types would make impossible states unrepresentable (e.g., a "before" event with a `DurationMs`). Large blast radius; needs careful migration plan.
- **ServiceInfo sub-struct placement review** — `IsShutdowner` is in `ServiceLifecycle` but `IsHealthchecker` is in `ServiceHealth`. Both are capability flags detected by `do.ExplainInjector`. This split-brain may be wrong. Moving `IsShutdowner` to `ServiceHealth` changes JSON field order.
- **`ScopeName` as a named type** — Currently plain `string`, unlike `ScopeID` and `ServiceName`. Left as `string` because it's display-only. Consistency gap.
- **Duration as `time.Duration`** — `DurationMs` is `*float64`. Could be `time.Duration` for idiomatic Go, but this would change the JSON schema and break consumers.

---

## Documentation Depth

Ideas for deeper documentation (website + README):

- **Interactive playground** — Embed a live HTML report on the website (iframe with CSP sandbox) where users paste a report JSON and see the visualization
- **Video demo** — A 30-second GIF or video showing the interactive graph (pan/zoom/click-to-highlight) and the waveform
- **Comparison section** — Real competitor analysis vs manual logging, vs OpenTelemetry, vs pprof
- **Migration guide** — Step-by-step for v0.1.0 to v0.2.0+ (MigrateReport exists but has no docs page)
- **Architecture deep-dive** — Single-package design, concurrency model, hook system, invocation-stack dependency inference

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

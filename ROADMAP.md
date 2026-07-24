# Roadmap

Long-term direction and raw ideas not yet refined into actionable tasks.
For short-term work, see [TODO_LIST.md](TODO_LIST.md). For shipped features, see [FEATURES.md](FEATURES.md).

---

## Stability Path

The project is in **ALPHA**. The path to BETA is:

1. Coverage gate back above 94% (currently 91.4% — see [TODO_LIST.md](TODO_LIST.md))
2. Build works without `GOEXPERIMENT=jsonv2` (blocked on `go-ndjson` dependency)
3. `go-sse` and `go-ndjson` published to GitHub with stable tags (remove `replace` directives)
4. `live/` sub-package lint-clean and 90%+ coverage
5. Live dashboard JS feature-parity with the static templ version

The path from BETA to 1.0 is:

1. API surface frozen — no more breaking changes without migration guide
2. Schema version stays at `0.2.0` unless report format changes
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

The `live/` sub-package is functional but less polished than the static HTML export:

- **Feature parity with static templ** — Add: collapsible scope tree, dependency detail popover, "Show all" pagination, per-event duration bars in waveform, scope-scoped service grouping, Sugiyama layered graph integration
- **Shared CSS** — Extract common design tokens into a shared file or `go:embed` so the static and live dashboards can't drift
- **Demo application** — A `live/demo/main.go` that registers services with delays and shows the dashboard updating in real time
- **Integration with `example/`** — A `--live` flag to serve the dashboard alongside the existing ride-sharing demo
- **Export buttons** — Add JSON/NDJSON/HTML export buttons to the live dashboard
- **Dark/light theme toggle** — The warm amber aesthetic currently has no light variant

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

## Observability Integration (Low Priority)

- **Prometheus metrics interface** — Optional `events-sent`, `clients-connected`, `events-dropped` metrics from the live Hub
- **OpenTelemetry bridge** — Generate spans from audit events (reference example exists in `docs/examples/`)
- **Structured logging adapter** — `OnEvent` callback that writes to slog/zap/zerolog

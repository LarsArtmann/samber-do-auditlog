# Design Notes — Batch 2026-09-02 (Pareto plan T92–T103)

Twelve time-boxed design explorations from the 2026-09-02 master plan. Each note
states the problem, the proposed shape, the blast radius, and what blocks it
from becoming a TODO_LIST item. Raw ideas — none are scheduled.

---

## T92 — ScopeDiff + diff --format for CI

**Problem:** `Report.Diff` reports per-service changes but flattens scope
structure; CI consumers must parse JSON to answer "did the graph change?".

**Proposal:** `ScopeDiff{ScopeID, Added, Removed, Changed []ServiceDiff}` +
`DiffByScope() []ScopeDiff`; CLI `auditlog diff a.json b.json --format github`
emitting `::error` annotations, or `--format text` for humans. Exit code 1 when
`!DiffResult.IsEmpty()` (opt-in `--fail-on-change`).

**Blast radius:** additive (new methods + CLI flags). Schema unchanged.

**Blocker:** none technical. Needs a use-case owner — ship after DepsChanged
(0.10.1) sees real usage.

---

## T93 — NDJSON event-line schema_version

**Problem:** event lines carry no schema marker; a future-schema stream replays
silently lossy (unknown fields dropped by json.Unmarshal).

**Proposal:** add optional `schema_version` field to Event (omitempty;
recorder stamps it). ReplayEvents/ReadEvents: if present and > current → hard
error; if absent → current behavior. Backward compatible both directions.

**Blast radius:** Event struct + schema regeneration + migration. This is the
gate for ANY future event-field addition; do it BEFORE the next schema bump.

**Blocker:** schema version decision (0.3.0 → 0.4.0).

---

## T94 — `auditlog validate --schema`

**Problem:** `JSONSchema()` exists but no command validates a report against
it; users hand-roll jsonschema tooling.

**Proposal:** `auditlog validate report.json --schema` using invopop/jsonschema
as validator (already a dependency, currently cmd-only — depguard rule already
allows it in cmd/).

**Blast radius:** cmd/ only. Library stays stdlib-only.

**Blocker:** none. Small; promote to TODO_LIST when CLI flags (T: --input-format)
ship, to batch CLI work.

---

## T95 — Docker image for the CLI

**Problem:** CLI consumers (CI steps) must install Go or download a binary.

**Proposal:** goreleaser `dockers:` section — `scratch` + static binary,
distroless labels, `ghcr.io/larsartmann/auditlog:<version>`. Requested June 2026.

**Blast radius:** .goreleaser.yml + workflow publishing step. No library impact.

**Blocker:** release cadence — only worth it once v0.10.x proves the tag flow
green (see TODO_LIST release-prep item).

---

## T96 — Branded-type constructors

**Problem:** `ContainerID`/`ScopeID`/`ServiceName` accept any string (empty,
whitespace, path separators — `Config.Validate` checks the container only).

**Proposal:** `NewServiceName(string) (ServiceName, error)` constructors
rejecting empty/whitespace; keep literals legal internally (zero-alloc), apply
at untrusted boundaries (LoadReport, ReadEvents, ReplayEvents — where strict
enum validation now lives).

**Blocker:** decide validation placement (type method vs constructor). Lean:
constructors + boundary validation; never validate inside hot hooks.

---

## T97 — Event type splitting (before/after)

**Problem:** one `Event` struct with `Phase` allows impossible states (a
"before" event with DurationMs; an "after" with nil duration).

**Proposal:** `BeforeEvent` / `AfterEvent` types sharing an embedded core;
`Event` remains the wire format (union) with a validated accessor pair
`AsBefore() (BeforeEvent, error)`.

**Blast radius:** LARGE — recorder, hooks, replay, stream, live, exports, all
tests. The wire format does not change (JSON identical), so consumers are safe.

**Blocker:** needs a dedicated session + full test-suite budget. Value is
type-safety internal only — schedule only after the 0.10.x line is stable.

---

## T98 — ServiceInfo sub-struct placement review

**Problem:** `IsShutdowner` (ServiceLifecycle) and `IsHealthchecker`
(ServiceHealth) are both capability flags detected by `do.ExplainInjector` —
the split is arbitrary. Moving `IsShutdowner` to ServiceHealth changes JSON
field order (cosmetic but diffs golden files + schema).

**Proposal:** leave JSON as-is; document the split rationale (shutdown is a
lifecycle concern; health-checker capability pairs with health data) OR move
both under a hypothetical `ServiceCapabilities` embedded struct in the next
major-ish schema bump (0.4.0).

**Blocker:** schema-bump appetite. Not worth a standalone release.

---

## T99 — ScopeName as a named type

**Problem:** `ScopeID` and `ServiceName` are named types; `ScopeName` is bare
`string` (display-only). Consistency gap.

**Proposal:** `type ScopeName string` + mechanical rename. JSON unchanged
(string). Zero behavior change; pure type hygiene. ~30-line diff.

**Blocker:** none — bundle with the next minor release touching types.go.

---

## T100 — DurationMs → time.Duration

**Problem:** `*float64` milliseconds are JSON-friendly but un-idiomatic Go.

**Proposal:** REJECTED for the wire format (schema stability; consumers in
JS/Python parse numbers). Offer `func (e Event) Duration() time.Duration` (a
typed accessor) — note: `Event.Duration()` already exists (returns
time.Duration from DurationMs). Conclusion: current design is correct;
document it and close the idea.

---

## T101 — Prometheus metrics interface for the live Hub

**Proposal:** optional `HubMetrics` interface (counters: events_sent,
clients_connected, events_dropped) wired via `live.Config.Metrics`; default
no-op. Keep `auditlog` core dependency-free (OnEvent callback remains the
integration point — see Explicitly Rejected: Prometheus-as-dependency).

**Blocker:** needs a real consumer to pin the metric names. Keep as design
note until then.

---

## T102 — OpenTelemetry bridge

**Proposal:** reference adapter (docs/examples/otel-bridge/main.go) mapping
Event → span events: registration=span start, invocation-after=span end with
duration, errors→span status. External otel SDK, example-only, never a
dependency.

**Blocker:** none — good first contribution surface.

---

## T103 — slog adapter via OnEvent

**Proposal:** ~40-line `slogWriter` example: OnEvent → `slog.Log(ctx, level,
"do event", "type", ..., "service", ...)`. Ship as docs/examples + README
snippet (the OnEvent callback section). No dependency (log/slog is stdlib).

**Blocker:** none. Cheapest of the three observability notes.

---

*Generated 2026-09-02 from docs/planning/2026-09-02_14-15-pareto-master-plan-all-126-todos.html.*

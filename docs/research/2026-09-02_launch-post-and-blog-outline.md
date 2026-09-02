# Launch Post Draft — samber-do-auditlog v0.10.x

Derived from the README one-narrative. Ready to paste into X/LinkedIn/Reddit
(r/go golang, r/devops); trim per channel. Owner review required before
posting anywhere.

---

**Short (X/Bluesky, ≤280):**

> Your DI container is a black box. samber-do-auditlog makes it a flight
> recorder: every service registration, invocation, shutdown, and health check
> — with timestamps, dependency graphs, and a self-contained HTML report.
> One line to wire in. `go get github.com/larsartmann/samber-do-auditlog`

**Medium (LinkedIn):**

> When a Go service built on samber/do v2 misbehaves in staging, the question
> is never "what broke" — it's "what did the container actually do?"
>
> samber-do-auditlog answers that. It hooks every lifecycle event in your DI
> container, infers the dependency graph from real invocation call stacks
> (not reflection guesses), and exports a self-contained HTML telemetry page:
> build durations, shutdown times, health checks, error propagation — even a
> live dashboard over SSE.
>
> No config file, no code changes beyond one option. Diff two reports and
> your CI fails when someone rewires the graph.
>
> `go get github.com/larsartmann/samber-do-auditlog` — demo:
> https://do-auditlog.lars.software

**Reddit (r/golang) body:**

**I built a flight recorder for samber/do v2 dependency injection**

What it does:

- Records every registration / invocation / shutdown / health check with timestamps
- Infers the real dependency graph from the invocation stack (A→B only when A actually resolves B)
- Exports: JSON, NDJSON, CSV, Mermaid/PlantUML/DOT/D2 diagrams, and a single-file HTML dashboard (amber phosphor aesthetic, pan/zoom graph, waveform timeline)
- Live mode: SSE dashboard that updates as the container runs (datastar-powered)
- `Report.Diff` for CI: fail the build when the dependency graph changes unexpectedly

Tradeoffs, honestly:

- ALPHA→BETA maturity; API can still change before 1.0 (STABILITY.md documents exactly what's frozen)
- Build needs `GOEXPERIMENT=jsonv2` on Go 1.26 (transitive dep requirement — documented in godoc)
- samber/do v2 only

Repo: https://github.com/LarsArtmann/samber-do-auditlog
Live demo: https://do-auditlog.lars.software

Feedback wanted on: the Diff API shape (per-service + dependency-edge deltas
since 0.10.1), and whether anyone wants a Prometheus bridge beyond the OnEvent
callback.

---

## Blog outline — "How dependency inference works" (T89)

Working title: **"Inferring a Dependency Graph From Runtime Behavior, Not
Reflection"**

1. **The problem with static DI graphs** — reflection-based graphs lie: they
   show what COULD be resolved, not what IS.
2. **The insight: the call stack is the truth** — when service B's provider
   runs while service A is mid-resolution, A depends on B. No declarations
   needed. (Explain the Recorder's invocation stack: push on before-hook, pop
   with LIFO fast path, edge recorded from stack to the resolving key.)
3. **Why before/after phases matter** — before = edge capture + timing start;
   after = duration + error capture. Health checks are after-only (samber/do
   has no before hook there) — an honest limitation.
4. **Making it observable without slowing it down** — single mutex per hook,
   atomic counters, callback outside the lock, zero-cost when disabled.
5. **From events to a graph you can see** — replay engine as the inverse of
   recording; NDJSON as the durable flight-recorder tape; diffing two tapes.
6. **What we still can't know** — cross-scope parentage flattening, missing
   capability detection on un-invoked lazy services (ExplainInjector needs a
   live scope).

CTA: repo + live dashboard demo.

_Status: outline approved for expansion; drafting blocked on owner go-ahead
(content ships to the website's blog section once it exists)._

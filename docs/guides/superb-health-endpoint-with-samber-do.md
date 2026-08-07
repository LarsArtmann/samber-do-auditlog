# Building a Superb `/health` Endpoint with samber/do v2

> **Verified API:** `github.com/samber/do/v2` (pkg.go.dev), `samber-do-auditlog` (battle-tested `RecordHealthCheckWithContext` wrapper), DO-1…DO-6 anti-pattern rules.

---

## The Core Insight

**samber/do already *is* your health engine.** Every service that implements `do.HealthcheckerWithContext` gets checked by `injector.HealthCheckWithContext(ctx)`, which runs **concurrently, in reverse-invocation order**, and returns `map[string]error` (`nil` value = healthy). Your job is *not* to write health checks — it's to **expose the right signals at the right endpoint, with the right timeouts, and without leaking internals or hanging under load.**

The #1 mistake is conflating one `/health` endpoint with three distinct probes:

| Endpoint          | Question it answers                         | samber/do role                                                        | Failure code                |
| ----------------- | ------------------------------------------- | --------------------------------------------------------------------- | --------------------------- |
| `/healthz` (liveness) | "Is the process alive & not deadlocked?"    | **None** — must stay cheap & dependency-free                          | 500 if deadlocked           |
| `/readyz` (readiness) | "Can I serve traffic right now?"            | **`injector.HealthCheckWithContext(ctx)`** — checks DB, cache, downstream | **503** if any _critical_ dep fails |
| `/startupz` (startup) | "Am I done booting?" (slow apps)            | Gate on critical eager services being invoked/healthy once            | 503 until ready             |

A superb endpoint implements **readiness** against the container, keeps **liveness** trivially fast, and degrades gracefully during **shutdown**.

---

## The Build Plan — 9 Actionable Steps

### Step 1 — Make services self-describe their health

Implement `do.HealthcheckerWithContext` (preferred over context-less `Healthchecker`) on anything that owns a connection. Add a compile-time guard.

```go
package datastore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/samber/do/v2"
)

var _ do.HealthcheckerWithContext = (*PostgresDB)(nil)

type PostgresDB struct{ pool *sql.DB }

func (d *PostgresDB) HealthCheck(ctx context.Context) error {
	if err := d.pool.PingContext(ctx); err != nil {
		return errors.Join(ErrDatabaseUnreachable, err) // sentinel + cause
	}
	return nil
}
```

**Verify:** `var _ do.HealthcheckerWithContext` compiles; the service appears in the result map of `injector.HealthCheckWithContext(ctx)`.

> **Critical gotcha:** samber/do **skips lazy services that were never invoked.** If your DB is `do.Provide` (lazy) but nothing has called it yet at boot, readiness _will not check it_. For critical infrastructure, either `do.ProvideValue` it eagerly, or explicitly `do.Invoke` it once during startup.

---

### Step 2 — Configure health-check budgets on the container

These live on `InjectorOpts`. A superb endpoint **never** lets a slow downstream hang the probe.

```go
func New() (*Container, func()) {
	plugin, _ := auditlog.New(auditlog.Config{Enabled: true, ContainerID: "my-app"})

	opts := plugin.Opts() // hooks for registration/invocation/health/shutdown auditing
	opts.HealthCheckTimeout       = 2 * time.Second // per-service deadline
	opts.HealthCheckGlobalTimeout = 5 * time.Second // whole-batch deadline
	opts.HealthCheckParallelism   = 8               // cap concurrency (0 = unlimited)

	injector := do.NewWithOpts(opts) // every health check is now recorded as an audit event
	return &Container{injector: injector, plugin: plugin}, func() { _ = injector.Shutdown() }
}
```

**Verify:** a deliberately-slow check returns `errors.New("DI: health check timeout: ...")` after 2s, not after 30s.

---

### Step 3 — Define a strongly-typed response model

Data model first. Make invalid states unrepresentable: status is a typed enum, only failures carry errors.

```go
type Status string

const (
	StatusPass Status = "pass"
	StatusFail Status = "fail"
	StatusWarn Status = "warn"
)

type Check struct {
	Status    Status `json:"status"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"` // only when Status != pass
}

type Response struct {
	Status       Status           `json:"status"`           // roll-up
	Version      string           `json:"version,omitempty"`
	Uptime       string           `json:"uptime,omitempty"`
	ShuttingDown bool             `json:"shutting_down,omitempty"`
	Checks       map[string]Check `json:"checks"`
}
```

---

### Step 4 — Separate critical from non-critical checks

This is the line between *functional* and *superb*. A failing metrics-exporter must not take your app out of rotation. Classify services, and gate readiness only on **critical** ones.

```go
type Probe struct {
	injector     do.Injector
	plugin       *auditlog.Plugin // nil if auditing disabled
	critical     map[string]struct{} // 503 if any fails
	shuttingDown atomic.Bool
	bootTime     time.Time
	version      string
}
```

---

### Step 5 — The readiness handler (aggregate + classify)

Hold the **root injector** at construction (this is *not* the service-locator smell — `HealthCheck` is a container-level operation, not business logic resolved ad-hoc per request). Never `do.Invoke` inside the handler (DO-1).

```go
func (p *Probe) Readiness(w http.ResponseWriter, r *http.Request) {
	if p.shuttingDown.Load() {
		writeHealth(w, http.StatusServiceUnavailable, p.failAll("shutting down"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Use the auditlog wrapper so every check is recorded; falls back to raw injector if nil.
	results := p.plugin.RecordHealthCheckWithContext(ctx, p.injector)
	if results == nil {
		results = p.injector.HealthCheckWithContext(ctx)
	}

	resp := Response{
		Checks:  map[string]Check{},
		Version: p.version,
		Uptime:  time.Since(p.bootTime).Round(time.Second).String(),
	}
	rollup, anyCriticalFail := StatusPass, false
	for name, err := range results {
		c := Check{Status: StatusPass}
		if err != nil {
			c.Status, c.Error = StatusFail, err.Error()
		}
		resp.Checks[name] = c
		if _, crit := p.critical[name]; crit && err != nil {
			anyCriticalFail = true
		}
	}
	resp.Status = StatusPass
	if anyCriticalFail {
		resp.Status = StatusFail
	}

	code := http.StatusOK
	if anyCriticalFail {
		code = http.StatusServiceUnavailable
	}
	writeHealth(w, code, resp)
}
```

**Verify:** unplug the DB → `/readyz` returns 503 with `database: fail`; unplug a non-critical service → still 200 with a `warn`-ish check.

---

### Step 6 — The liveness handler (trivially fast, dep-free)

Liveness must answer in <1ms with **no** dependency calls — otherwise a dependency blip causes a restart cascade.

```go
func (p *Probe) Liveness(w http.ResponseWriter, _ *http.Request) {
	// Optionally: check a deadlock watchdog / goroutine starvation detector here.
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(Response{
		Status: StatusPass,
		Uptime: time.Since(p.bootTime).Round(time.Second).String(),
	})
}
```

---

### Step 7 — Make it shutdown-aware

Flip the flag from your `Shutdown()` path so load balancers drain _before_ connections close.

```go
func (p *Probe) Shutdown(_ context.Context) error {
	p.shuttingDown.Store(true) // /readyz now returns 503; /healthz stays 200
	time.Sleep(gracePeriod)    // let LBs notice before resources drop
	return nil
}
```

---

### Step 8 — Cache to survive high-frequency polling

Kubelet + LBs can hit `/readyz` every second; `Ping()` on every call will hammer your DB. Run checks on a **bounded background loop** and serve cached results so the endpoint is always instant.

```go
type Probe struct {
	/* ... */
	latest atomic.Pointer[Response]
}

func (p *Probe) refreshLoop(ctx context.Context) {
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.latest.Store(p.evaluate(ctx))
		}
	}
}

func (p *Probe) Readiness(w http.ResponseWriter, _ *http.Request) {
	resp := p.latest.Load()
	if resp == nil {
		writeHealth(w, 503, emptyFail)
		return
	}
	writeHealth(w, codeFor(resp), *resp)
}
```

**Verify:** under 500 RPS to `/readyz`, DB `Ping()` count stays at 1/sec, not 500/sec.

---

### Step 9 — Wire routes & secure the detail

Expose **aggregate** externally; gate **detail** internally. Never mount samber/do's `dohttp` debug UI publicly (it leaks the entire DI graph — the docs warn explicitly).

```go
mux.HandleFunc("/healthz", probe.Liveness)                // public, fast
mux.HandleFunc("/readyz", internalOnly(probe.Readiness))  // 200/503 externally; detail internally
```

---

## What Separates "Superb" from "Works"

| Concern            | Naive                       | Superb                                                        |
| ------------------ | --------------------------- | ------------------------------------------------------------- |
| Probes             | One `/health`               | Split liveness / readiness / startup                         |
| Failure semantics  | Any dep down → 503          | Critical vs non-critical classification                       |
| Polling resilience | `Ping()` per request        | Background-refresh cache                                      |
| Timeouts           | None (hangs)                | Per-service + global budgets on `InjectorOpts`               |
| Shutdown           | Abrupt 503 after close      | Flips to 503 _before_ closing connections                     |
| Observability      | `fmt.Println`               | `samber-do-auditlog` records every check as a timed event    |
| Lazy gotcha        | "DB never checked at boot"  | Eager-provide or `do.Invoke` critical services at startup    |
| Security           | Leaks internals             | Aggregate externally; detail internal-only; `dohttp` never public |

---

## samber/do v2 Health-Check API Quick Reference

**Lifecycle interfaces (services implement one):**

```go
type Healthchecker           interface{ HealthCheck() error }
type HealthcheckerWithContext interface{ HealthCheck(context.Context) error }
```

**Package-level single-service checks (return `error`):**

| Function                          | Signature                                        |
| --------------------------------- | ------------------------------------------------ |
| `do.HealthCheck[T]`               | `func(i Injector) error`                         |
| `do.HealthCheckWithContext[T]`    | `func(ctx context.Context, i Injector) error`    |
| `do.HealthCheckNamed`             | `func(i Injector, name string) error`            |
| `do.HealthCheckNamedWithContext`  | `func(ctx context.Context, i Injector, name string) error` |

**Injector-level bulk checks (return `map[string]error`, nil value = healthy):**

```go
results := injector.HealthCheckWithContext(ctx)
// key = service name, value = nil (healthy) or error
```

**Key behaviors:**

- Runs in **reverse invocation order**.
- **Concurrent** by default; cap with `HealthCheckParallelism`.
- **Lazy services that were never invoked are NOT checked.**
- `*Scope.HealthCheck()` also checks **ancestor** scopes (traverses upward).
- Tunable timeouts: `HealthCheckTimeout` (per-service), `HealthCheckGlobalTimeout` (whole-batch).
- Timeout sentinel: `do.ErrHealthCheckTimeout` (`"DI: health check timeout"`).

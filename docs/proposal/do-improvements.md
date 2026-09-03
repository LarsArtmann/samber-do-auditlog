# samber/do Improvement List

You asked: "did you miss hooks / make hacks?" — here are the two concrete
gaps we hit while building the audit-log plugin, with proposed APIs. Both
come from production use of every hook do v2 exposes.

## 1. Health-check hooks (the one real gap)

**Problem.** `InjectorOpts` has before/after hooks for registration,
invocation, and shutdown — but nothing for `HealthCheck()`. To audit health
checks we had to ship a *wrapper* the user must call instead of the injector:

```go
// today: wrapper pattern (auditlog)
plugin.RecordHealthCheck(injector)   // instead of injector.HealthCheck()
```

This works but is the only lifecycle surface a plugin cannot observe
passively — every other lifecycle event reaches us through `Opts()`. It also
misses health checks triggered by anyone who calls `injector.HealthCheck()`
directly (frameworks, k8s probes wired by third-party code).

**Proposal.** Two optional hooks, mirroring the existing pairs:

```go
type InjectorOpts[T Injector] struct {
    // ... existing hooks ...
    HookBeforeHealthCheck func(s *Scope)
    HookAfterHealthCheck  func(s *Scope, err error) // err = first failure, if any
}
```

Per-service timing (which our report would love to show) would need a
`HookAfterServiceHealthCheck func(s *Scope, name string, err error, d
time.Duration)` — but the bulk pair alone already closes the observability
gap.

## 2. Do not call `ExplainInjector` from inside a hook (document the deadlock)

**Problem.** The natural way to enrich a registration hook with capability
info (`Healthchecker`? `Shutdowner`?) is:

```go
HookAfterRegistration: func(s *do.Scope, name string) {
    ex := do.ExplainInjector(s) // ⚠ DEADLOCKS for eager services
    ...
}
```

For eagerly-provided services the registration hook runs while do holds
internal locks that `ExplainInjector` also acquires — the call deadlocks
(found the hard way; the workaround is to store `*do.Scope` references in
the hook and call `ExplainInjector` later, outside the hook).

**Proposal.** Either make `ExplainInjector`/`ExplainNamedService` safe to
call from hooks (re-entrant locking), or document the constraint loudly on
both functions and in `InjectorOpts` — "do not call from hook context; stash
the scope and call after `NewWithOpts` returns."

## 3. Smaller notes (no action required)

- `HookBeforeShutdown`/`HookAfterShutdown` fire per shutdowner — a
  scope-level "shutdown started/finished" pair would let dashboards show a
  definitive *container* completion signal. (We synthesize it today.)
- `do.ProvideNamedValue` panics on duplicate registration while
  `do.Provide` returns an error — the asymmetry surprised us when wiring
  test doubles.
- The stringly-typed service name in hook signatures is fine in practice;
  a future `Register[T]` generics API would let plugins type-check
  dependencies statically (nice-to-have, zero urgency).

## What did NOT need hacks (credit where due)

- The hook set itself is complete for registration/invocation/shutdown —
  the dependency graph is fully inferable from before/after pairs plus an
  invocation stack, no instrumentation of user code required.
- `do.As`/override/scopes behave exactly as documented under hooks.
- `ExplainNamedService` outside hook context is reliable for provider-type
  detection (lazy/eager/transient/alias), which drives our UI badges.

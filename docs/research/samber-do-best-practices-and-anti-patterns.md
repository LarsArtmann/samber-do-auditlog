# samber/do v2: Best Practices & Anti-Patterns

**Scope:** Every Go project in `/home/lars/projects` that depends on `github.com/samber/do`.  
**Date:** 2026-07-23  
**Library versions observed:** v2.0.0, v2.1.0  
**Related work:** `samber-do-auditlog`, `branching-flow/pkg/doanalyzerv2`, `reports/docs/01-technical-analysis/go-libraries/samber-ro-do-linter-design.md`

---

## Executive Summary

`samber/do` v2 is the dominant dependency-injection library across this workspace: **72 modules / ~443 Go files** use it. Almost every active project uses **v2.1.0**; only the two archived projects (`experiment-100m-arr`, `website-holger-hahn`) remain on v1.6.0.

The library is powerful but has a small set of sharp edges that keep appearing in real code. This guide consolidates:

- The **canonical patterns** seen in the best containers here.
- The **six lint rules** already implemented in `branching-flow` (DO-1 → DO-6).
- The **authoritative recommendations** from the official docs (`do.samber.dev`).
- Concrete **examples taken from this workspace**.

Use this as the single reference for writing new DI code and for auditing existing code.

---

## 1. Inventory of samber/do Usage

### 1.1 Projects by version

| Version | Count | Notes |
| --- | --- | --- |
| v2.1.0 | ~55 modules | Current default across active code |
| v2.0.0 | ~15 modules | Mostly older code; fully compatible with v2.1.0 |
| v1.6.0 | 2 modules | Only in `archived/` |

### 1.2 All modules that depend on samber/do

Direct usage is concentrated in these projects:

`samber-do-auditlog`, `BuildFlow`, `Kernovia`, `KeyCountdown`, `cmdguard`, `smart-configs`, `testing`, `accountability-system`, `ast-state-analyzer`, `branching-flow`, `go-plugin-mvp`, `CreditReformBilanzampel`, `DiscordSync`, `dynamic-markdown-site`, `templates/repo-validation`, `timesheets`, `universal-workflow`, `yt-history-intel`, `licenseforge`, `Rolls-Royce-mtuGoHelpCenter-golang`, `standard-bug-tracking-schema`, `PapDashboard`, `RedditParse`, `StopTube`, `desire-secrets`, `clean-wizard`, `invoices`, `AI-von-Art-Bench`, `german-business-contract-automation`, `file-and-image-renamer`, `go-dag-app`, `go-structure-linter`, `go-wizard-sdk`, `projects-management-automation`, `storbi`, `template-CLI`, `Code-Quality-Agent`, `code-duplicate-analyzer`, `go-auto-upgrade`, `go-cqrs-lite/cmd/cqrs-lint`, `docs-organizer`, `go-plugin-mvp/marketplace/container`, `mr-sync`, `terraform-diagrams-aggregator`, `Zlota44`.

Indirect (transitive) usage also appears in: `go-composable-business-types`, `go-localsync`, `gomend`, `go-must`, `go-nix-helpers`, `go-output`, `go-plugin-mvp` sub-modules, `library-policy`, `lars.software`, `overview`, `project-dependency-graph`, `project-meta`, `prompt-crusher`, `rules`, `sales-landing-page`, `setup-github-repo`, `superb-gh-milestone-extention`, and several `cmdguard` sub-modules.

### 1.3 Key files to study

| Project | File | What it demonstrates |
| --- | --- | --- |
| `samber-do-auditlog` | `plugin.go` | Lifecycle hooks via `do.InjectorOpts` |
| `samber-do-auditlog` | `example/main.go` | Every major feature in one runnable demo |
| `BuildFlow` | `internal/di/di.go` | Wrapped container + cleanup + debug tree |
| `BuildFlow` | `internal/di/providers_singleton.go` | `ProvideValue` for eager singletons |
| `smart-configs/di` | `register.go` | `ProvideNamed` for config-as-services |
| `Kernovia` | `internal/kernel/named_services.go` | Named services + helper accessors |
| `KeyCountdown` | `internal/container/lifecycle/manager.go` | Lifecycle manager + health checks |
| `cmdguard` | `pkg/cmdguard/v3/scope.go` | Typed wrapper + child scopes |
| `go-plugin-mvp` | `marketplace/container/container.go` | Turnkey composition root |
| `branching-flow` | `pkg/doanalyzerv2/*.go` | Static detection of anti-patterns |

---

## 2. Core Concepts Cheat Sheet

### 2.1 Service lifetimes

| Registration | Lifetime | Use for |
| --- | --- | --- |
| `do.Provide` | Lazy singleton | Most services |
| `do.ProvideValue` | Eager singleton | Config, logger, DB connection |
| `do.ProvideTransient` | New instance per `Invoke` | Factories, value objects |
| `do.ProvideNamed` / `do.ProvideNamedValue` | Named singleton | Multiple impls of one type |

### 2.2 Invocation

```go
v, err := do.Invoke[*Service](injector)          // safe, preferred
v := do.MustInvoke[*Service](injector)           // panic on missing; only in main/init or Provide closures
v, err := do.InvokeNamed[*Service](injector, "name")
v := do.MustInvokeNamed[*Service](injector, "name")
```

### 2.3 Lifecycle interfaces

```go
// Health checks
type Healthchecker           interface { HealthCheck() error }
type HealthcheckerWithContext interface { HealthCheck(context.Context) error }

// Shutdown
type Shutdowner                 interface { Shutdown() }
type ShutdownerWithError         interface { Shutdown() error }
type ShutdownerWithContext       interface { Shutdown(context.Context) }
type ShutdownerWithContextAndError interface { Shutdown(context.Context) error }
```

`injector.Shutdown()` runs shutdowns in **reverse invocation order** and returns a `*do.ShutdownReport`.

### 2.4 Scopes

```go
root := do.New()
driver := root.Scope("driver")
passenger := root.Scope("passenger")
```

Child scopes can resolve parent services; parent scopes cannot see child services.

### 2.5 Aliasing

```go
// Implicit — preferred
do.Provide(injector, NewMetricsCounter)
metric := do.MustInvokeAs[Metric](injector)

// Explicit — use only when implicit is ambiguous
do.As[*MetricsCounter, Metric](injector)
metric := do.MustInvoke[Metric](injector)
```

### 2.6 Packages

```go
var Package = do.Package(
    do.Lazy(NewStore),
    do.LazyNamed("primary", NewPrimaryRepo),
    do.Eager(config),
)

injector := do.New(Package)
```

---

## 3. How to Use samber/do Well

### 3.1 Composition root: create the injector once, close it once

Every `do.New()` must have a matching `Shutdown()`. Wrap the pair in a constructor and return a cleanup function.

**Good — BuildFlow:**

```go
func New() (*Container, func()) {
    injector := do.New()

    return &Container{injector: injector}, func() {
        if IsDebugDIEnabled() {
            PrintDependencyTree(injector)
        }

        if err := injector.Shutdown(); err != nil {
            slog.Debug("DI container shutdown error", "error", err)
        }

        if httpServerStopper.IsPresent() {
            httpServerStopper.MustGet()()
        }
    }
}
```

Usage:

```go
box, cleanup := di.New()
defer cleanup()
```

### 3.2 Prefer lazy singletons

Most services should use `do.Provide`. Services are built only when first invoked, in dependency order, and only once.

```go
do.Provide(injector, func(i do.Injector) (*UserService, error) {
    db, err := do.Invoke[*Database](i)
    if err != nil {
        return nil, err
    }
    return &UserService{db: db}, nil
})
```

The official docs call `do.MustInvoke` inside a provider closure the **canonical pattern** because it runs during container build time, not per request.

### 3.3 Use `ProvideValue` for eager foundation services

If a service must exist even when nothing has invoked it yet — because it has background goroutines or because `Shutdown()` must always run — register it eagerly.

**Good — BuildFlow `providers_singleton.go`:**

```go
// Eager because its background GC goroutine and connection must be available
// even if no consumer has invoked it yet — lazy services that were never
// invoked are SKIPPED by injector.Shutdown().
func registerDBStore(injector do.Injector) {
    store, err := dbstore.Open(dbPath)
    if err != nil {
        return // graceful degradation
    }
    do.ProvideValue(injector, store)
}
```

### 3.4 Implement lifecycle interfaces

Add `var _ do.ShutdownerWithError = (*MyService)(nil)` compile-time guards.

```go
var _ do.ShutdownerWithContextAndError = (*Server)(nil)

func (s *Server) Shutdown(ctx context.Context) error {
    return s.listener.Shutdown(ctx)
}
```

### 3.5 Use named services for multiple implementations

**Good — Kernovia:**

```go
do.ProvideNamed(injector, "registry.memory", func(i do.Injector) (registry.PluginRegistry, error) {
    return registry.NewIndexedMemoryRegistry(), nil
})

do.ProvideNamed(injector, "registry.performance", func(i do.Injector) (registry.PluginRegistry, error) {
    return NewPerformanceRegistry(), nil
})

mem, err := do.InvokeNamed[registry.PluginRegistry](injector, "registry.memory")
```

### 3.6 Build scopes for request/session/tenant isolation

**Good — go-plugin-mvp:**

```go
func TenantScope(root do.Injector, tenantID string) do.Injector {
    return root.Scope(fmt.Sprintf("tenant-%s", tenantID))
}
```

**Good — KeyCountdown:** separates `session-*` and `request-*` scopes, then shuts them down on cleanup.

### 3.7 Accept interfaces, return structs

Provide concrete types, invoke by interface.

```go
type Metric interface { Inc() }

type Counter struct { n int }
func (c *Counter) Inc() { c.n++ }

do.Provide(injector, func(i do.Injector) (*Counter, error) { return &Counter{}, nil })
m := do.MustInvokeAs[Metric](injector)
```

### 3.8 Use `do.Package` for modular registration

Group related providers into a package variable.

```go
var Stores = do.Package(
    do.Lazy(NewPostgreSQLConnectionService),
    do.Lazy(NewUserRepository),
    do.EagerNamed("repository.logger", slog.Default()),
)

injector := do.New(Stores)
```

### 3.9 Use audit hooks for observability

**Good — samber-do-auditlog:**

```go
plugin, _ := auditlog.New(auditlog.Config{
    Enabled:     true,
    ContainerID: "my-app",
})

injector := do.NewWithOpts(plugin.Opts())
```

This records every registration, invocation, health check, and shutdown without polluting business code.

### 3.10 Write test containers

Create a dedicated test container that overrides production dependencies.

**Good — `testing/internal/container/test_container.go`:**

```go
type TestContainer struct {
    *Container
}

func NewTestContainer() (*TestContainer, error) {
    container, cleanup := New()
    // override DB, config, etc.
    return &TestContainer{Container: container}, nil
}
```

---

## 4. How NOT to Use samber/do

These are the six lint rules already implemented in `branching-flow/pkg/doanalyzerv2`, plus two additional structural smells.

### 4.1 DO-1: `Must*` in runtime paths

`do.MustInvoke`, `do.MustInvokeNamed`, `do.MustInvokeStruct`, `do.MustAs`, `do.MustAsNamed` **panic** if the service is missing. They are safe only in:

- `main()` / `init()`
- Provider factory closures (because they run at container build time)

**Bad:**

```go
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    db := do.MustInvoke[*Database](h.injector) // runtime panic risk
    // ...
}
```

**Good:**

```go
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    db, err := do.Invoke[*Database](h.injector)
    if err != nil {
        http.Error(w, "database unavailable", 500)
        return
    }
    // ...
}
```

Or inject the dependency into the handler struct at startup.

### 4.2 DO-2: Missing container shutdown

`do.New()` without a matching `.Shutdown()` leaks resources: DB connections, goroutines, file handles, HTTP clients.

**Bad:**

```go
func main() {
    injector := do.New()
    // ... run ...
    // never calls Shutdown()
}
```

**Good:**

```go
func main() {
    injector := do.New()
    defer injector.Shutdown()

    app, _ := do.Invoke[*App](injector)
    app.Run()
}
```

If ownership is transferred (e.g. a constructor returns the injector to the caller), the linter suppresses this case.

### 4.3 DO-3: Override outside container setup

`samber/do` explicitly warns: *"We strongly discourage using this helper in production. Please use service aliasing instead."* `do.Override*` is legitimate in:

- `_test.go` files
- setup / wire / configure / register / init functions
- options-style configuration APIs that take an injector

**Bad:**

```go
func (h *Handler) HandleRequest(ctx context.Context) {
    do.OverrideValue(h.injector, newRateLimiter()) // runtime mutation
}
```

**Good:**

```go
func WithRateLimiter(lim RateLimiter) Option {
    return func(i do.Injector) {
        do.OverrideValue(i, lim)
    }
}
```

### 4.4 DO-4: Global injector

Global mutable state prevents parallel tests, makes lifecycle management impossible, and hides dependencies.

**Bad:**

```go
var Injector = do.New()

func GetUserRepo() *UserRepository {
    return do.MustInvoke[*UserRepository](Injector)
}
```

**Good:**

```go
func NewApp() (*App, func()) {
    injector := do.New()
    // register ...
    return &App{injector: injector}, func() { injector.Shutdown() }
}
```

> Note: `do.DefaultRootScope` is the library's own global. Do not build application logic on top of it.

### 4.5 DO-5: Invoke inside loops

Repeatedly invoking inside a loop is usually accidental and wasteful. Resolve the service once and reuse it.

**Bad:**

```go
for _, id := range ids {
    svc, _ := do.Invoke[*Service](injector)
    svc.Process(id)
}
```

**Good:**

```go
svc, _ := do.Invoke[*Service](injector)
for _, id := range ids {
    svc.Process(id)
}
```

### 4.6 DO-6: Shutdown accesses other services

`samber/do` issue #219: shutdown ordering is **non-deterministic**. A `Shutdown()` method that invokes another service may run after that service is already dead.

**Bad:**

```go
func (s *Server) Shutdown() error {
    db := do.MustInvoke[*Database](s.injector)
    return db.Close() // db may already be shut down
}
```

**Good:**

```go
func (s *Server) Shutdown() error {
    // Self-contained: close only resources this service owns.
    return s.listener.Close()
}
```

If cleanup must be coordinated, use a parent lifecycle manager (e.g. `KeyCountdown.LifecycleManager`) rather than cross-service calls inside `Shutdown()`.

### 4.7 Service-locator smell

Do not pass the injector deep into business logic and resolve dependencies ad-hoc.

**Bad:**

```go
func (h *BaseHandler) Handle(w http.ResponseWriter, r *http.Request) {
    repo, _ := do.Invoke[*UserRepo](h.injector)
    repo.Save(...)
}
```

**Good:**

```go
type Handler struct {
    repo *UserRepo // injected at construction
}

func NewHandler(repo *UserRepo) *Handler { return &Handler{repo: repo} }
```

### 4.8 Storing the injector inside a service

A service should receive its dependencies at construction time, not hold a reference to the container.

**Bad:**

```go
type MyService struct {
    injector do.Injector
}

func NewMyService(i do.Injector) (*MyService, error) {
    return &MyService{injector: i}, nil
}
```

**Good:**

```go
type MyService struct {
    dep *MyDependency
}

func NewMyService(i do.Injector) (*MyService, error) {
    dep, err := do.Invoke[*MyDependency](i)
    if err != nil {
        return nil, err
    }
    return &MyService{dep: dep}, nil
}
```

This is the pattern the official docs explicitly recommend.

---

## 5. Real-World Patterns from This Workspace

### 5.1 Audit-log plugin: hook-driven observability

`samber-do-auditlog/plugin.go` attaches to all six lifecycle hooks:

```go
return &do.InjectorOpts{
    HookBeforeRegistration: []func(*do.Scope, string){p.recorder.OnBeforeRegistration},
    HookAfterRegistration:  []func(*do.Scope, string){p.recorder.OnAfterRegistration},
    HookBeforeInvocation:   []func(*do.Scope, string){p.recorder.OnBeforeInvocation},
    HookAfterInvocation:    []func(*do.Scope, string, error){p.recorder.OnAfterInvocation},
    HookBeforeShutdown:     []func(*do.Scope, string){p.recorder.OnBeforeShutdown},
    HookAfterShutdown:      []func(*do.Scope, string, error){p.recorder.OnAfterShutdown},
}
```

### 5.2 BuildFlow: debug dependency tree on demand

```go
if IsDebugDIEnabled() {
    PrintDependencyTree(injector)
}
```

Uses `do.ExplainInjector` under the hood and prints the full scope tree.

### 5.3 smart-configs: config as named services

```go
func RegisterServices(
    injector do.Injector,
    fields []metadata.FieldMetadata,
    resolve Resolver,
) error {
    for _, field := range fields {
        serviceName := field.ServiceName()
        do.ProvideNamed(injector, serviceName, func(_ do.Injector) (string, error) {
            value, found, err := resolve(field)
            // ...
            return value, nil
        })
    }
    return nil
}
```

Every config field becomes a first-class DI citizen with health checks.

### 5.4 Kernovia: named-service helpers

Provide multiple implementations, then expose typed helper accessors:

```go
func GetMemoryRegistry(injector do.Injector) (registry.PluginRegistry, error) {
    return do.InvokeNamed[registry.PluginRegistry](injector, "registry.memory")
}
```

### 5.5 cmdguard: typed scope wrapper

A thin wrapper turns panics into errors and adds child-scope navigation:

```go
func Provide[T any](scope *Scope, provider func(do.Injector) (T, error)) error {
    if scope == nil {
        return fmt.Errorf("%w: scope is nil", ErrInvalidScope)
    }
    return safeProvide(func() { do.Provide(scope.injector, provider) }, ...)
}
```

Useful, but be careful not to reimplement the entire library.

### 5.6 KeyCountdown: explicit lifecycle manager

A dedicated manager owns shutdown order, health checks, and service initialization:

```go
lm := lifecycle.NewLifecycleManager(injector)
lm.RegisterAllLifecycleServices()
status := lm.GetComprehensiveHealthStatus(ctx)
errors := lm.GracefulShutdownAll()
```

### 5.7 branching-flow: static analysis

`pkg/doanalyzerv2` detects DO-1 → DO-6 with AST-only analysis. Run it against any project to find the smells listed in section 4.

---

## 6. Decision Checklist

When adding a new service, walk through these questions:

| Question | If yes | Pattern |
| --- | --- | --- |
| Is it a long-lived singleton? | yes | `do.Provide` |
| Must it exist before first invoke (DB, logger, config)? | yes | `do.ProvideValue` |
| Is it a per-request / per-tenant resource? | yes | `do.ProvideTransient` in a child scope |
| Are there multiple implementations? | yes | `do.ProvideNamed` + accessor helper |
| Should consumers depend on an interface? | yes | Provide struct, invoke via `do.InvokeAs` |
| Does it hold resources? | yes | Implement `do.Shutdowner*` |
| Does it need a connectivity check? | yes | Implement `do.Healthchecker*` |
| Is it invoked from a request handler? | yes | Inject dependency into handler struct, do not call `do.Invoke` in handler body |
| Is shutdown order critical? | yes | Use a single lifecycle manager; never access other services from `Shutdown()` |
| Is this code in `_test.go`? | yes | `do.Override*` is acceptable |

---

## 7. Migration Notes

### 7.1 v1 → v2

- `*do.Injector` → `do.Injector` (interface)
- `do.ProvideEager` → `do.ProvideValue`
- `Shutdown()` now returns `*do.ShutdownReport`
- Hooks are slices: `[]func(...)`
- `Service[T]` no longer exported; use `Provider[T]`

Only the two archived projects still use v1. All active work should target v2.1.0.

### 7.2 Consolidating v2.0.0 and v2.1.0

v2.1.0 is backward compatible with v2.0.0. Projects on v2.0.0 can bump the dependency without code changes. Use the newer version to pick up bug fixes and the latest `dohttp` debug UI.

---

## 8. References

- Official docs: <https://do.samber.dev/docs/getting-started>
- API reference: <https://pkg.go.dev/github.com/samber/do/v2>
- Source code: <https://github.com/samber/do>
- Migration guide: <https://do.samber.dev/docs/upgrading/from-v1-x-to-v2>
- samber/do issue #219 (non-deterministic shutdown): <https://github.com/samber/do/issues/219>
- `branching-flow` linter: `/home/lars/projects/branching-flow/pkg/doanalyzerv2/`
- `samber-do-auditlog`: `/home/lars/projects/samber-do-auditlog/`
- Linter design research: `/home/lars/projects/reports/docs/01-technical-analysis/go-libraries/samber-ro-do-linter-design.md`

---

## 9. Action Items

1. Run `branching-flow` DO rules against every active project and triage findings for DO-1, DO-2, DO-4, DO-6.
2. Upgrade any remaining v2.0.0 projects to v2.1.0.
3. Add `do.Shutdowner*` / `do.Healthchecker*` interfaces to resource-holding services.
4. Replace global injectors with constructor-injected composition roots.
5. Use `samber-do-auditlog` hooks in long-running services to observe real DI behavior.

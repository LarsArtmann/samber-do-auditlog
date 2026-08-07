package health

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

const (
	// defaultTimeout is the per-request deadline for health-check batches.
	defaultTimeout = 5 * time.Second
	// defaultRefreshInterval is how often the background loop refreshes
	// the cached readiness response. Set to zero via WithRefreshInterval(0)
	// to evaluate live on every request instead.
	defaultRefreshInterval = 1 * time.Second
	// uptimeResolution is the granularity of the human-readable uptime string.
	uptimeResolution = time.Second
)

// Probe orchestrates health checks against a samber/do v2 injector and exposes
// three distinct HTTP endpoints: liveness, readiness, and startup.
//
// A Probe holds a reference to the root [do.Injector] (a container-level
// operation, not the service-locator anti-pattern) and classifies registered
// services into critical and non-critical. Only critical service failures
// cause readiness to return 503; non-critical failures are surfaced as
// individual check entries but do not affect the HTTP status code.
//
// Probe is safe for concurrent use by multiple goroutines.
type Probe struct {
	injector do.Injector
	plugin   *auditlog.Plugin

	critical map[string]struct{}

	shuttingDown  atomic.Bool
	startupPassed atomic.Bool

	bootTime time.Time
	version  string

	latest          atomic.Pointer[Response]
	refreshInterval time.Duration
	timeout         time.Duration

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Option configures a [Probe].
type Option func(*Probe)

// WithVersion sets the application version included in health responses.
func WithVersion(v string) Option {
	return func(p *Probe) { p.version = v }
}

// WithCriticalServices marks the named services as critical: if any of them
// fails its health check, readiness returns 503. Services not listed here are
// non-critical; their failures appear in the response body but do not change
// the HTTP status code.
func WithCriticalServices(names ...string) Option {
	return func(p *Probe) {
		for _, name := range names {
			p.critical[name] = struct{}{}
		}
	}
}

// WithPlugin wires an [auditlog.Plugin] so that every health-check batch is
// recorded as a timed audit event. When nil (the default), checks run against
// the raw injector without recording.
func WithPlugin(p *auditlog.Plugin) Option {
	return func(probe *Probe) { probe.plugin = p }
}

// WithRefreshInterval sets the background cache refresh cadence. When greater
// than zero, [Probe.Start] launches a goroutine that re-evaluates health checks
// on this interval and readiness handlers serve the cached result. When zero,
// readiness handlers evaluate live on every request (no background goroutine).
//
// Use caching (the default) when kubelet or load-balancer polling could
// overwhelm downstream dependencies. Use live evaluation for low-traffic or
// development scenarios.
func WithRefreshInterval(d time.Duration) Option {
	return func(p *Probe) { p.refreshInterval = d }
}

// WithTimeout sets the per-evaluation context deadline. This bounds how long
// a single health-check batch can run before individual services are timed out
// by samber/do's own per-service and global health-check timeouts.
func WithTimeout(d time.Duration) Option {
	return func(p *Probe) { p.timeout = d }
}

// WithBootTime overrides the boot timestamp used to compute uptime. Defaults
// to the time [New] was called.
func WithBootTime(t time.Time) Option {
	return func(p *Probe) { p.bootTime = t }
}

// New creates a [Probe] wired to the given injector.
//
// The injector must be the root container created via [do.NewWithOpts].
// Holding the root injector is correct here: HealthCheck is a container-level
// operation, not business logic resolved ad-hoc per request (DO-1).
func New(injector do.Injector, opts ...Option) *Probe {
	probe := &Probe{
		injector:        injector,
		critical:        make(map[string]struct{}),
		bootTime:        time.Now(),
		timeout:         defaultTimeout,
		refreshInterval: defaultRefreshInterval,
	}

	for _, opt := range opts {
		opt(probe)
	}

	return probe
}

// Start launches the background cache refresh loop (when RefreshInterval > 0)
// and performs an immediate evaluation so the cache is populated before the
// first request arrives. Calling Start more than once is a no-op.
//
// The provided ctx controls the lifetime of the background goroutine. Call
// [Probe.Shutdown] to stop the loop and mark the probe as shutting down.
func (p *Probe) Start(ctx context.Context) {
	p.mu.Lock()

	if p.cancel != nil {
		p.mu.Unlock()

		return
	}

	runCtx := ctx

	if p.refreshInterval > 0 {
		runCtx, p.cancel = context.WithCancel(ctx)
	}

	p.mu.Unlock()

	p.refreshCache(ctx)

	if p.refreshInterval > 0 {
		p.wg.Add(1)

		go p.refreshLoop(runCtx)
	}
}

// refreshLoop runs the periodic cache refresh until the start context is cancelled.
func (p *Probe) refreshLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.refreshCache(ctx)
		}
	}
}

// refreshCache evaluates health checks with a timeout-bounded context and
// stores the result atomically for cached handlers.
func (p *Probe) refreshCache(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	resp := p.Evaluate(ctx)
	p.latest.Store(&resp)
}

// Shutdown marks the probe as shutting down and stops the background refresh
// loop if one is running. After Shutdown:
//
//   - Liveness continues to return 200 (the process is still alive).
//   - Readiness returns 503 so load balancers drain traffic.
//   - Startup returns its latched value (200 if it had previously passed).
func (p *Probe) Shutdown() {
	p.shuttingDown.Store(true)

	p.mu.Lock()
	cancel := p.cancel
	p.cancel = nil
	p.mu.Unlock()

	if cancel != nil {
		cancel()
		p.wg.Wait()
	}
}

// MarkShuttingDown flips the shutdown flag without stopping the background
// loop. Use this for a two-phase graceful shutdown: mark first so load
// balancers start draining, then call [Probe.Shutdown] after a grace period
// to stop the refresh loop.
func (p *Probe) MarkShuttingDown() {
	p.shuttingDown.Store(true)
}

// Evaluate runs a full health-check batch against the injector and returns
// the aggregate response. This is the core evaluation logic used by the
// readiness and startup handlers, exposed publicly for testing and custom
// handler scenarios.
//
// The context should carry a deadline; [Probe.Start] applies [Probe.timeout]
// automatically. When a [auditlog.Plugin] is configured, checks are recorded
// as audit events via [auditlog.Plugin.RecordHealthCheckWithContext].
func (p *Probe) Evaluate(ctx context.Context) Response {
	start := time.Now()

	results := p.runHealthChecks(ctx)

	resp := Response{
		Version:        p.version,
		Uptime:         time.Since(p.bootTime).Round(uptimeResolution).String(),
		ShuttingDown:   p.shuttingDown.Load(),
		Checks:         buildChecks(results),
		TotalLatencyMs: time.Since(start).Milliseconds(),
	}

	resp.Status = p.classify(results, resp.ShuttingDown)

	return resp
}

// runHealthChecks delegates to the auditlog plugin wrapper when available,
// falling back to the raw injector otherwise.
func (p *Probe) runHealthChecks(ctx context.Context) map[string]error {
	if p.plugin != nil {
		return p.plugin.RecordHealthCheckWithContext(ctx, p.injector)
	}

	return p.injector.HealthCheckWithContext(ctx)
}

// classify computes the roll-up status: fail if shutting down or any critical
// service failed, pass otherwise.
func (p *Probe) classify(results map[string]error, shuttingDown bool) Status {
	if shuttingDown {
		return StatusFail
	}

	for name := range p.critical {
		if err, found := results[name]; found && err != nil {
			return StatusFail
		}
	}

	return StatusPass
}

// StartupComplete returns true once all critical services have passed their
// health checks at least once during a startup evaluation. After this returns
// true it always returns true (the latch is one-way).
func (p *Probe) StartupComplete() bool {
	return p.startupPassed.Load()
}

// evaluateStartup checks whether all critical services are present and healthy
// in the results map. Returns true if the startup latch should be set.
func (p *Probe) evaluateStartup(results map[string]error) bool {
	if len(p.critical) == 0 {
		return true
	}

	for name := range p.critical {
		err, found := results[name]

		if !found || err != nil {
			return false
		}
	}

	return true
}

// buildChecks converts the raw map[string]error from samber/do into typed
// Check entries. A nil error means the service passed; a non-nil error
// populates the Error field.
func buildChecks(results map[string]error) map[string]Check {
	checks := make(map[string]Check, len(results))

	for name, err := range results {
		check := Check{Status: StatusPass}

		if err != nil {
			check.Status = StatusFail
			check.Error = err.Error()
		}

		checks[name] = check
	}

	return checks
}

package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/health"
	"github.com/samber/do/v2"
)

// --- Test service types ---.

type healthyService struct{}

var _ do.HealthcheckerWithContext = (*healthyService)(nil)

func (healthyService) HealthCheck(_ context.Context) error { return nil }

type unhealthyService struct{ reason string }

var _ do.HealthcheckerWithContext = (*unhealthyService)(nil)

func (u *unhealthyService) HealthCheck(_ context.Context) error {
	return errors.New(u.reason)
}

type slowService struct {
	delay time.Duration
}

var _ do.HealthcheckerWithContext = (*slowService)(nil)

func (s *slowService) HealthCheck(ctx context.Context) error {
	select {
	case <-time.After(s.delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type countingService struct {
	calls atomic.Int64
}

var _ do.HealthcheckerWithContext = (*countingService)(nil)

func (c *countingService) HealthCheck(_ context.Context) error {
	c.calls.Add(1)

	return nil
}

// --- Test helpers ---.

func mustNewPlugin() *auditlog.Plugin {
	p, err := auditlog.New(auditlog.Config{Enabled: true, ContainerID: "test"})
	if err != nil {
		panic(err)
	}

	return p
}

func provideHealthy(i do.Injector, name string) {
	do.ProvideNamed(i, name, func(_ do.Injector) (*healthyService, error) {
		return &healthyService{}, nil
	})
}

func provideUnhealthy(i do.Injector, name, reason string) {
	do.ProvideNamed(i, name, func(_ do.Injector) (*unhealthyService, error) {
		return &unhealthyService{reason: reason}, nil
	})
}

func provideCounting(i do.Injector, name string) *countingService {
	svc := &countingService{}
	do.ProvideNamed(i, name, func(_ do.Injector) (*countingService, error) {
		return svc, nil
	})

	return svc
}

func invoke[T any](t *testing.T, i do.Injector, name string) T {
	t.Helper()

	return do.MustInvokeNamed[T](i, name)
}

func doRequest(t *testing.T, handler http.HandlerFunc, method, target string) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, target, nil)
	handler(w, r)

	return w
}

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) health.Response {
	t.Helper()

	var resp health.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	return resp
}

// --- Liveness tests ---.

func TestLiveness_AlwaysReturns200(t *testing.T) {
	t.Parallel()

	injector := do.New()
	probe := health.New(injector)

	w := doRequest(t, probe.LivenessHandler(), http.MethodGet, "/healthz")

	if w.Code != http.StatusOK {
		t.Fatalf("liveness status: want 200, got %d", w.Code)
	}

	resp := decodeResponse(t, w)
	if resp.Status != health.StatusPass {
		t.Errorf("liveness status field: want pass, got %s", resp.Status)
	}
}

func TestLiveness_PerformsNoDependencyChecks(t *testing.T) {
	t.Parallel()

	injector := do.New()
	svc := provideCounting(injector, "db")
	invoke[*countingService](t, injector, "db")

	probe := health.New(injector)
	w := doRequest(t, probe.LivenessHandler(), http.MethodGet, "/healthz")

	if w.Code != http.StatusOK {
		t.Fatalf("liveness status: want 200, got %d", w.Code)
	}

	if calls := svc.calls.Load(); calls != 0 {
		t.Errorf("liveness should not check dependencies, but HealthCheck was called %d times", calls)
	}
}

func TestLiveness_ContainsVersionAndUptime(t *testing.T) {
	t.Parallel()

	injector := do.New()
	probe := health.New(injector, health.WithVersion("v1.2.3"))

	w := doRequest(t, probe.LivenessHandler(), http.MethodGet, "/healthz")
	resp := decodeResponse(t, w)

	if resp.Version != "v1.2.3" {
		t.Errorf("version: want v1.2.3, got %s", resp.Version)
	}

	if resp.Uptime == "" {
		t.Error("uptime should not be empty")
	}
}

// --- Readiness tests ---.

func TestReadiness_AllHealthy_Returns200(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	provideHealthy(injector, "cache")
	invoke[*healthyService](t, injector, "db")
	invoke[*healthyService](t, injector, "cache")

	probe := health.New(injector,
		health.WithCriticalServices("db", "cache"),
		health.WithRefreshInterval(0),
	)

	w := doRequest(t, probe.ReadinessHandler(), http.MethodGet, "/readyz")

	if w.Code != http.StatusOK {
		t.Fatalf("readiness status: want 200, got %d", w.Code)
	}

	resp := decodeResponse(t, w)
	if resp.Status != health.StatusPass {
		t.Errorf("readiness status field: want pass, got %s", resp.Status)
	}

	if len(resp.Checks) != 2 {
		t.Errorf("checks count: want 2, got %d", len(resp.Checks))
	}
}

func TestReadiness_CriticalFailure_Returns503(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "cache")
	provideUnhealthy(injector, "db", "connection refused")
	invoke[*healthyService](t, injector, "cache")
	invoke[*unhealthyService](t, injector, "db")

	probe := health.New(injector,
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(0),
	)

	w := doRequest(t, probe.ReadinessHandler(), http.MethodGet, "/readyz")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status: want 503, got %d", w.Code)
	}

	resp := decodeResponse(t, w)
	if resp.Status != health.StatusFail {
		t.Errorf("readiness status field: want fail, got %s", resp.Status)
	}

	dbCheck, ok := resp.Checks["db"]
	if !ok {
		t.Fatal("db check missing from response")
	}

	if dbCheck.Status != health.StatusFail {
		t.Errorf("db check status: want fail, got %s", dbCheck.Status)
	}

	if dbCheck.Error != "connection refused" {
		t.Errorf("db check error: want 'connection refused', got %q", dbCheck.Error)
	}
}

func TestReadiness_NonCriticalFailure_Returns200(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	provideUnhealthy(injector, "metrics", "exporter down")
	invoke[*healthyService](t, injector, "db")
	invoke[*unhealthyService](t, injector, "metrics")

	probe := health.New(injector,
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(0),
	)

	w := doRequest(t, probe.ReadinessHandler(), http.MethodGet, "/readyz")

	if w.Code != http.StatusOK {
		t.Fatalf("readiness status: want 200 (non-critical failure), got %d", w.Code)
	}

	resp := decodeResponse(t, w)
	if resp.Status != health.StatusPass {
		t.Errorf("readiness status field: want pass (non-critical failure), got %s", resp.Status)
	}

	metricsCheck, ok := resp.Checks["metrics"]
	if !ok {
		t.Fatal("metrics check missing from response")
	}

	if metricsCheck.Status != health.StatusFail {
		t.Errorf("metrics check status: want fail, got %s", metricsCheck.Status)
	}
}

func TestReadiness_ShuttingDown_Returns503(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector,
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(0),
	)

	probe.Shutdown()

	w := doRequest(t, probe.ReadinessHandler(), http.MethodGet, "/readyz")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status during shutdown: want 503, got %d", w.Code)
	}

	resp := decodeResponse(t, w)
	if !resp.ShuttingDown {
		t.Error("response should show shutting_down=true")
	}
}

func TestReadiness_MarkShuttingDown_DoesNotStopBackgroundLoop(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector, health.WithCriticalServices("db"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	probe.Start(ctx)
	probe.MarkShuttingDown()

	// Give the loop time to prove it is still running.
	time.Sleep(50 * time.Millisecond)

	cached := probe.Evaluate(context.Background())
	if !cached.ShuttingDown {
		t.Error("Evaluate should reflect shutting down state")
	}

	probe.Shutdown()
}

func TestReadiness_NoServices_Returns200(t *testing.T) {
	t.Parallel()

	injector := do.New()
	probe := health.New(injector, health.WithRefreshInterval(0))

	w := doRequest(t, probe.ReadinessHandler(), http.MethodGet, "/readyz")

	if w.Code != http.StatusOK {
		t.Fatalf("readiness with no services: want 200, got %d", w.Code)
	}
}

// --- Readiness cache tests ---.

func TestReadiness_CachedMode_ServesFromCache(t *testing.T) {
	t.Parallel()

	injector := do.New()
	svc := provideCounting(injector, "db")
	invoke[*countingService](t, injector, "db")

	probe := health.New(injector,
		health.WithRefreshInterval(0), // live mode for Evaluate
	)

	// Manually populate cache via Start.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	probe.Start(ctx)
	defer probe.Shutdown()

	// Wait for initial evaluation.
	time.Sleep(50 * time.Millisecond)

	initialCalls := svc.calls.Load()

	// Hit readiness 10 times — should not increase call count.
	for range 10 {
		w := doRequest(t, probe.ReadinessHandler(), http.MethodGet, "/readyz")
		if w.Code != http.StatusOK {
			t.Fatalf("readiness status: want 200, got %d", w.Code)
		}
	}

	if calls := svc.calls.Load(); calls != initialCalls {
		t.Errorf("cached readiness should not call HealthCheck, initial=%d, after=%d", initialCalls, calls)
	}
}

// --- Startup tests ---.

func TestStartup_LatchesOnceAllCriticalPass(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector,
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(0),
	)

	// First call: all critical healthy → 200, latches.
	w1 := doRequest(t, probe.StartupHandler(), http.MethodGet, "/startupz")
	if w1.Code != http.StatusOK {
		t.Fatalf("first startup: want 200, got %d", w1.Code)
	}

	if !probe.StartupComplete() {
		t.Error("StartupComplete should be true after all critical pass")
	}

	// Second call: latched → still 200.
	w2 := doRequest(t, probe.StartupHandler(), http.MethodGet, "/startupz")
	if w2.Code != http.StatusOK {
		t.Fatalf("latched startup: want 200, got %d", w2.Code)
	}
}

func TestStartup_CriticalServiceMissing_Returns503(t *testing.T) {
	t.Parallel()

	injector := do.New()
	// Register but do NOT invoke — samber/do skips never-invoked lazy services.
	provideHealthy(injector, "db")

	probe := health.New(injector,
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(0),
	)

	w := doRequest(t, probe.StartupHandler(), http.MethodGet, "/startupz")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("startup with missing critical service: want 503, got %d", w.Code)
	}

	if probe.StartupComplete() {
		t.Error("StartupComplete should be false when critical service was never invoked")
	}
}

func TestStartup_CriticalServiceUnhealthy_Returns503(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideUnhealthy(injector, "db", "starting up")
	invoke[*unhealthyService](t, injector, "db")

	probe := health.New(injector,
		health.WithCriticalServices("db"),
		health.WithRefreshInterval(0),
	)

	w := doRequest(t, probe.StartupHandler(), http.MethodGet, "/startupz")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("startup with unhealthy critical: want 503, got %d", w.Code)
	}
}

func TestStartup_NoCriticalServices_ImmediatelyPasses(t *testing.T) {
	t.Parallel()

	injector := do.New()
	probe := health.New(injector, health.WithRefreshInterval(0))

	w := doRequest(t, probe.StartupHandler(), http.MethodGet, "/startupz")

	if w.Code != http.StatusOK {
		t.Fatalf("startup with no critical services: want 200, got %d", w.Code)
	}

	if !probe.StartupComplete() {
		t.Error("StartupComplete should be true when there are no critical services")
	}
}

// --- Evaluate tests ---.

func TestEvaluate_ReturnsCorrectClassification(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	provideUnhealthy(injector, "cache", "redis down")
	invoke[*healthyService](t, injector, "db")
	invoke[*unhealthyService](t, injector, "cache")

	probe := health.New(injector, health.WithCriticalServices("db"))

	resp := probe.Evaluate(context.Background())

	if resp.Status != health.StatusPass {
		t.Errorf("status: want pass (db healthy, cache non-critical), got %s", resp.Status)
	}

	if len(resp.Checks) != 2 {
		t.Fatalf("checks count: want 2, got %d", len(resp.Checks))
	}

	if resp.Checks["db"].Status != health.StatusPass {
		t.Error("db should pass")
	}

	if resp.Checks["cache"].Status != health.StatusFail {
		t.Error("cache should fail")
	}
}

func TestEvaluate_CriticalFailure_ReturnsFail(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideUnhealthy(injector, "db", "unreachable")
	invoke[*unhealthyService](t, injector, "db")

	probe := health.New(injector, health.WithCriticalServices("db"))

	resp := probe.Evaluate(context.Background())

	if resp.Status != health.StatusFail {
		t.Errorf("status: want fail (critical db down), got %s", resp.Status)
	}
}

func TestEvaluate_IncludesUptimeAndVersion(t *testing.T) {
	t.Parallel()

	injector := do.New()
	boot := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	probe := health.New(injector, health.WithVersion("v2.0.0"), health.WithBootTime(boot))

	resp := probe.Evaluate(context.Background())

	if resp.Version != "v2.0.0" {
		t.Errorf("version: want v2.0.0, got %s", resp.Version)
	}

	if resp.Uptime == "" {
		t.Error("uptime should not be empty")
	}
}

func TestEvaluate_TotalLatencyRecorded(t *testing.T) {
	t.Parallel()

	injector := do.New()
	do.ProvideNamed(injector, "slow", func(_ do.Injector) (*slowService, error) {
		return &slowService{delay: 50 * time.Millisecond}, nil
	})
	invoke[*slowService](t, injector, "slow")

	probe := health.New(injector, health.WithRefreshInterval(0))

	resp := probe.Evaluate(context.Background())

	if resp.TotalLatencyMs < 1 {
		t.Errorf("total latency should be > 0, got %dms", resp.TotalLatencyMs)
	}
}

// --- RegisterRoutes tests ---.

func TestRegisterRoutes_AllThreeHandlersRegistered(t *testing.T) {
	t.Parallel()

	injector := do.New()
	probe := health.New(injector, health.WithRefreshInterval(0))

	mux := http.NewServeMux()
	probe.RegisterRoutes(mux, health.DefaultRoutes())

	for _, path := range []string{"/healthz", "/readyz", "/startupz"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(w, r)

		if w.Code == http.StatusNotFound {
			t.Errorf("route %s not registered", path)
		}
	}
}

func TestRegisterRoutes_CustomPaths(t *testing.T) {
	t.Parallel()

	injector := do.New()
	probe := health.New(injector, health.WithRefreshInterval(0))

	mux := http.NewServeMux()
	probe.RegisterRoutes(mux, health.Routes{
		Liveness:  "/live",
		Readiness: "/ready",
		Startup:   "/started",
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/live", nil)
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("custom liveness route: want 200, got %d", w.Code)
	}
}

// --- Content-Type and format tests ---.

func TestResponse_ContentTypeIsJSON(t *testing.T) {
	t.Parallel()

	injector := do.New()
	probe := health.New(injector)

	w := doRequest(t, probe.LivenessHandler(), http.MethodGet, "/healthz")

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("content-type: want application/json, got %s", ct)
	}
}

func TestResponse_NoCacheHeader(t *testing.T) {
	t.Parallel()

	injector := do.New()
	probe := health.New(injector)

	w := doRequest(t, probe.LivenessHandler(), http.MethodGet, "/healthz")

	cc := w.Header().Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("cache-control: want no-cache, got %s", cc)
	}
}

// --- Audit integration tests ---.

func TestAuditIntegration_RecordsHealthCheckEvents(t *testing.T) {
	t.Parallel()

	plugin := mustNewPlugin()
	injector := do.NewWithOpts(plugin.Opts())
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector,
		health.WithPlugin(plugin),
		health.WithRefreshInterval(0),
	)

	resp := probe.Evaluate(context.Background())

	if resp.Status != health.StatusPass {
		t.Errorf("status: want pass, got %s", resp.Status)
	}

	report := plugin.Report()
	events := report.EventsByType(auditlog.EventTypeHealthCheck)
	if len(events) == 0 {
		t.Error("expected health check events from auditlog plugin")
	}
}

func TestAuditIntegration_DisabledPlugin_StillWorks(t *testing.T) {
	t.Parallel()

	plugin, err := auditlog.New(auditlog.Config{Enabled: false})
	if err != nil {
		t.Fatal(err)
	}

	injector := do.NewWithOpts(plugin.Opts())
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector,
		health.WithPlugin(plugin),
		health.WithRefreshInterval(0),
	)

	resp := probe.Evaluate(context.Background())
	if resp.Status != health.StatusPass {
		t.Errorf("status: want pass, got %s", resp.Status)
	}
}

// --- Lifecycle tests ---.

func TestStart_PerformsImmediateEvaluation(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	probe := health.New(injector, health.WithCriticalServices("db"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	probe.Start(ctx)

	// Cache should be populated immediately, before any tick.
	w := doRequest(t, probe.ReadinessHandler(), http.MethodGet, "/readyz")
	if w.Code != http.StatusOK {
		t.Errorf("readiness after Start: want 200, got %d", w.Code)
	}

	probe.Shutdown()
}

func TestShutdown_StopsBackgroundLoop(t *testing.T) {
	t.Parallel()

	injector := do.New()
	probe := health.New(injector, health.WithRefreshInterval(10*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	probe.Start(ctx)
	probe.Shutdown()

	// Should not panic or hang.
	w := doRequest(t, probe.ReadinessHandler(), http.MethodGet, "/readyz")
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("readiness after Shutdown: want 503, got %d", w.Code)
	}
}

func TestStart_CalledTwice_IsNoOp(t *testing.T) {
	t.Parallel()

	injector := do.New()
	probe := health.New(injector, health.WithRefreshInterval(10*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	probe.Start(ctx)
	probe.Start(ctx) // should not panic or start a second loop
	probe.Shutdown()
}

func TestReadiness_BeforeStart_EvaluatesLive(t *testing.T) {
	t.Parallel()

	injector := do.New()
	provideHealthy(injector, "db")
	invoke[*healthyService](t, injector, "db")

	// Default refresh interval > 0, but Start not called → no cache → live eval.
	probe := health.New(injector, health.WithCriticalServices("db"))

	w := doRequest(t, probe.ReadinessHandler(), http.MethodGet, "/readyz")
	if w.Code != http.StatusOK {
		t.Fatalf("readiness before Start (live fallback): want 200, got %d", w.Code)
	}
}

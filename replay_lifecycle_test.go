package auditlog_test

import (
	"bytes"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestReplayEvents_FullLifecycle(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	provideCache(injector, "cache")
	provideUserServiceWithDeps(injector, "users", "db", "cache")

	do.MustInvokeNamed[*UserService](injector, "users")

	_ = injector.Shutdown()

	replayed := replayFromPlugin(t, plugin)

	assertReportValid(t, replayed, "full lifecycle replayed")

	db := findServiceByName(t, replayed, "db")
	if db == nil {
		t.Fatal("db not found")
	}

	if db.Status != auditlog.ServiceStatusShutdown {
		t.Errorf("db status: want %q, got %q", auditlog.ServiceStatusShutdown, db.Status)
	}
}

func TestReplayEvents_ShutdownErrorPreserved(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideCrashing(injector, "crasher")
	do.MustInvokeNamed[*CrashingService](injector, "crasher")

	_ = injector.Shutdown()

	replayed := replayFromPlugin(t, plugin)

	svc := findServiceByName(t, replayed, "crasher")
	if svc == nil {
		t.Fatal("crasher not found")
	}

	assertServiceStatus(t, svc, auditlog.ServiceStatusShutdownError)
}

func TestReplayEvents_ManualShutdownWithoutBefore(t *testing.T) {
	t.Parallel()

	dur := 5.5
	errMsg := "shutdown failed"

	events := []auditlog.Event{
		mkRegEvent(1, time.Now(), "svc", "test"),
		{
			ServiceRef: rootRef("svc"),
			Sequence:   2, Timestamp: time.Now(),
			EventType: auditlog.EventTypeShutdown, Phase: auditlog.PhaseAfter,
			ContainerID: "test", ServiceType: auditlog.ProviderTypeLazy,
			DurationMs: &dur, Error: &errMsg,
		},
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "svc")
	if svc == nil {
		t.Fatal("svc not found")
	}

	assertServiceStatus(t, svc, auditlog.ServiceStatusShutdownError)
}

func TestReplayEvents_ManualHealthCheckNewService(t *testing.T) {
	t.Parallel()

	// Health check event for a service that was never registered.
	events := []auditlog.Event{
		mkEvent(1, time.Now(), auditlog.EventTypeHealthCheck, auditlog.PhaseAfter, "ghost", "test", ""),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "ghost")
	if svc == nil {
		t.Fatal("ghost not found")
	}

	assertServiceHealthCheckCount(t, svc, 1)
}

func TestReplayEvents_RegistrationOverwriteType(t *testing.T) {
	t.Parallel()

	// Two registration-after events for the same service — the second
	// should update the service type.
	events := []auditlog.Event{
		mkRegEvent(1, time.Now(), "svc", "test"),
		mkEvent(
			2,
			time.Now(),
			auditlog.EventTypeRegistration,
			auditlog.PhaseAfter,
			"svc",
			"test",
			auditlog.ProviderTypeEager,
		),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "svc")
	if svc == nil {
		t.Fatal("svc not found")
	}

	if svc.ServiceType != auditlog.ProviderTypeEager {
		t.Errorf("service type: want %q, got %q", auditlog.ProviderTypeEager, svc.ServiceType)
	}
}

func TestReplayEvents_MultiScopeDeterministic(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjectorWithID("multi-scope")

	provideDB(injector, "root-db", "postgres://root")
	provideCache(injector, "root-cache")

	childScope := injector.Scope("drivers")
	provideUserServiceWithDB(childScope, "driver-service", "root-db")

	do.MustInvokeNamed[*Database](injector, "root-db")
	do.MustInvokeNamed[*Cache](injector, "root-cache")
	do.MustInvokeNamed[*UserService](childScope, "driver-service")

	var buf bytes.Buffer

	err := plugin.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	events, err := auditlog.ReadEvents(&buf)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	report1, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents (1): %v", err)
	}

	report2, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents (2): %v", err)
	}

	if report1.ScopeCount != report2.ScopeCount {
		t.Errorf("scope count differs: %d vs %d", report1.ScopeCount, report2.ScopeCount)
	}

	if report1.ScopeTree.ID != report2.ScopeTree.ID {
		t.Errorf("root scope ID differs")
	}

	if report1.ScopeCount < 2 {
		t.Errorf("expected >=2 scopes, got %d", report1.ScopeCount)
	}

	driverSvc := findServiceByName(t, report1, "driver-service")
	if driverSvc == nil {
		t.Fatal("driver-service not found")
	}

	foundDep := false

	for _, dep := range driverSvc.Dependencies {
		if dep.ServiceName == "root-db" {
			foundDep = true
		}
	}

	if !foundDep {
		t.Error("driver-service should depend on root-db")
	}
}

func TestReplayEvents_OutOfOrderStackPop(t *testing.T) {
	t.Parallel()

	// Interleaved invocations: A starts, B starts, B ends, A ends.
	// The stack pop for A is NOT the last element (non-LIFO path).
	events := []auditlog.Event{
		mkRegEvent(1, time.Now(), "a", "c"),
		mkRegEvent(2, time.Now(), "b", "c"),
		mkInvBefore(3, time.Now(), "a"),
		mkInvBefore(4, time.Now(), "b"),
		// B finishes first (LIFO pop), then A finishes (non-LIFO: index < len-1).
		mkInvAfter(5, time.Now(), "b"),
		mkInvAfter(6, time.Now(), "a"),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svcA := findServiceByName(t, report, "a")
	if svcA == nil {
		t.Fatal("a not found")
	}

	assertServiceInvocationCount(t, svcA, 1)
}

// --- NDJSON reader tests ---

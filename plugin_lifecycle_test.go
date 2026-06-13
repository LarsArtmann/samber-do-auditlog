package auditlog_test

import (
	"sync"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestPlugin_RegistrationAndInvocation(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true, ContainerID: "test"})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")

	_, err := do.InvokeNamed[*Database](injector, "db")
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}

	report := p.Report()
	if report.ContainerID != "test" {
		t.Errorf("container_id: want test, got %s", report.ContainerID)
	}

	if report.ServiceCount != 1 {
		t.Errorf("service_count: want 1, got %d", report.ServiceCount)
	}

	if report.EventCount != 4 {
		t.Errorf("event_count: want 4, got %d", report.EventCount)
	}

	if report.ScopeCount < 1 {
		t.Errorf("scope_count: want >= 1, got %d", report.ScopeCount)
	}

	if report.TotalBuildDurationMs <= 0 {
		t.Errorf("total_build_duration_ms: want > 0, got %f", report.TotalBuildDurationMs)
	}

	if report.TotalShutdownDurationMs != 0 {
		t.Errorf("total_shutdown_duration_ms: want 0 (no shutdown), got %f", report.TotalShutdownDurationMs)
	}

	if !report.ShutdownSucceeded {
		t.Error("shutdown_succeeded: want true (no shutdown errors)")
	}

	svc := report.Services[0]
	if svc.ServiceName != "db" {
		t.Errorf("service_name: want db, got %s", svc.ServiceName)
	}

	if svc.InvocationCount != 1 {
		t.Errorf("invocation_count: want 1, got %d", svc.InvocationCount)
	}

	if svc.FirstInvokedAt == nil {
		t.Error("expected FirstInvokedAt to be set")
	}

	if svc.FirstBuildDurationMs == nil || *svc.FirstBuildDurationMs < 0 {
		t.Error("expected FirstBuildDurationMs to be set and non-negative")
	}

	if svc.InvocationOrder != 0 {
		t.Errorf("first invoked service should have order 0, got %d", svc.InvocationOrder)
	}
}

func TestPlugin_InvocationOrder(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "a", "a")
	provideCache(injector, "b")

	_ = do.MustInvokeNamed[*Database](injector, "a")
	_ = do.MustInvokeNamed[*Cache](injector, "b")

	report := p.Report()

	orderMap := map[string]int{}
	for _, svc := range report.Services {
		orderMap[svc.ServiceName] = svc.InvocationOrder
	}

	if orderMap["a"] >= orderMap["b"] {
		t.Errorf("expected a (%d) invoked before b (%d)", orderMap["a"], orderMap["b"])
	}
}

func TestPlugin_DependencyTracking(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*Database, error) {
		time.Sleep(1 * time.Millisecond)

		return &Database{URL: "postgres://localhost"}, nil
	})

	do.ProvideNamed(injector, "cache", func(i do.Injector) (*Cache, error) {
		time.Sleep(1 * time.Millisecond)

		return &Cache{Entries: make(map[string]string)}, nil
	})

	do.ProvideNamed(injector, "users", func(i do.Injector) (*UserService, error) {
		db := do.MustInvokeNamed[*Database](i, "db")
		cache := do.MustInvokeNamed[*Cache](i, "cache")

		return &UserService{DB: db, Cache: cache}, nil
	})

	_, err := do.InvokeNamed[*UserService](injector, "users")
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}

	report := p.Report()
	if report.ServiceCount != 3 {
		t.Fatalf("expected 3 services, got %d", report.ServiceCount)
	}

	var users *auditlog.ServiceInfo

	users = findServiceByName(t, report, "users")

	if users == nil {
		t.Fatal("users service not found in report")
	}

	if len(users.Dependencies) != 2 {
		t.Errorf("users dependencies: want 2, got %d (%v)", len(users.Dependencies), users.Dependencies)
	}

	var db *auditlog.ServiceInfo

	db = findServiceByName(t, report, "db")

	if db == nil {
		t.Fatal("db service not found")
	}

	if len(db.Dependents) != 1 || db.Dependents[0].ServiceName != "users" {
		t.Errorf("db dependents: want [users], got %v", db.Dependents)
	}
}

func TestPlugin_CachedInvocation(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")

	do.ProvideNamed(injector, "users", func(i do.Injector) (*UserService, error) {
		db := do.MustInvokeNamed[*Database](i, "db")

		return &UserService{DB: db}, nil
	})

	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*UserService](injector, "users")

	report := p.Report()

	var users *auditlog.ServiceInfo

	users = findServiceByName(t, report, "users")

	if users == nil {
		t.Fatal("users service not found")
	}

	if len(users.Dependencies) != 1 {
		t.Errorf("users should have exactly 1 dependency (db), got %d: %v", len(users.Dependencies), users.Dependencies)
	}

	var db *auditlog.ServiceInfo

	db = findServiceByName(t, report, "db")

	if db == nil {
		t.Fatal("db service not found")
	}

	if db.InvocationCount != 2 {
		t.Errorf("db invocation_count: want 2, got %d", db.InvocationCount)
	}
}

func TestPlugin_EventSequenceNumbers(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	events := p.Events()
	if len(events) < 2 {
		t.Fatal("expected at least 2 events")
	}

	for i := 1; i < len(events); i++ {
		if events[i].Sequence <= events[i-1].Sequence {
			t.Errorf("events not monotonically increasing: event[%d].Sequence=%d <= event[%d].Sequence=%d",
				i, events[i].Sequence, i-1, events[i-1].Sequence)
		}
	}
}

func TestPlugin_EmptyReport(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})

	report := p.Report()
	if report.EventCount != 0 {
		t.Errorf("expected 0 events, got %d", report.EventCount)
	}

	if report.ServiceCount != 0 {
		t.Errorf("expected 0 services, got %d", report.ServiceCount)
	}

	assertVersion(t, report)

	if report.ScopeTree.Name != "" && report.ScopeTree.ID != "" {
		t.Error("expected empty scope tree for empty report")
	}
}

func TestPlugin_ConcurrentInvocations(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")

	var wg sync.WaitGroup

	for range 10 {
		wg.Go(func() {
			_, _ = do.InvokeNamed[*Database](injector, "db")
		})
	}

	wg.Wait()

	report := p.Report()
	if report.ServiceCount != 1 {
		t.Errorf("expected 1 service, got %d", report.ServiceCount)
	}

	svc := findServiceByName(t, report, "db")
	if svc == nil {
		t.Fatal("db not found")
	}

	if svc.InvocationCount != 10 {
		t.Errorf("expected 10 invocations, got %d", svc.InvocationCount)
	}
}

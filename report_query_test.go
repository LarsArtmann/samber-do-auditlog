package auditlog_test

import (
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestReport_ServiceByName(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()

	svc := report.ServiceByName("db")
	if svc == nil {
		t.Fatal("expected to find db service")
	}

	assertEqual(t, "service name", svc.ServiceName, "db")

	if report.ServiceByName("nonexistent") != nil {
		t.Error("expected nil for nonexistent service")
	}
}

func TestReport_ServiceByRef(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())
	child := injector.Scope("child")

	provideDB(injector, "db", "root-db")
	provideDB(child, "db", "child-db")

	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*Database](child, "db")

	report := p.Report()

	rootSvc := report.ServiceByRef(injector.ID(), "db")
	if rootSvc == nil {
		t.Fatal("root db not found by ref")
	}

	if rootSvc.ServiceName != "db" {
		t.Errorf("root db name: want db, got %s", rootSvc.ServiceName)
	}

	childSvc := report.ServiceByRef(child.ID(), "db")
	if childSvc == nil {
		t.Fatal("child db not found by ref")
	}

	if childSvc.ServiceName != "db" {
		t.Errorf("child db name: want db, got %s", childSvc.ServiceName)
	}

	if report.ServiceByRef("nonexistent", "db") != nil {
		t.Error("expected nil for nonexistent scope")
	}
}

func TestReport_ServicesByScope(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())
	child := injector.Scope("child")

	provideDB(injector, "root-svc", "test")
	provideDB(child, "child-svc", "child")

	_ = do.MustInvokeNamed[*Database](injector, "root-svc")
	_ = do.MustInvokeNamed[*Database](child, "child-svc")

	report := p.Report()

	rootServices := report.ServicesByScope(injector.ID())

	if len(rootServices) < 1 {
		t.Fatalf("expected at least 1 root service, got %d", len(rootServices))
	}

	childServices := report.ServicesByScope(child.ID())
	requireOneService(t, "child", childServices)

	if childServices[0].ServiceName != "child-svc" {
		t.Errorf("child service: want child-svc, got %s", childServices[0].ServiceName)
	}
}

func TestReport_ServicesByScope_EmptyScope(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()

	noServices := report.ServicesByScope("nonexistent-scope")
	if len(noServices) != 0 {
		t.Errorf("expected 0 services for nonexistent scope, got %d", len(noServices))
	}
}

func TestReport_EventsByService(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	provideDB(injector, "cache", "cache")

	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*Database](injector, "cache")

	report := p.Report()

	dbEvents := report.EventsByService("db")

	if len(dbEvents) == 0 {
		t.Fatal("expected db events")
	}

	for _, e := range dbEvents {
		if e.ServiceName != "db" {
			t.Errorf("expected db event, got %s", e.ServiceName)
		}
	}
}

func TestReport_EventsByType(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()

	regEvents := report.EventsByType(auditlog.EventTypeRegistration)
	if len(regEvents) == 0 {
		t.Error("expected registration events")
	}

	invEvents := report.EventsByType(auditlog.EventTypeInvocation)
	if len(invEvents) == 0 {
		t.Error("expected invocation events")
	}

	shutdownEvents := report.EventsByType(auditlog.EventTypeShutdown)
	if len(shutdownEvents) != 0 {
		t.Error("expected no shutdown events")
	}
}

func TestReport_EventsByRef(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()

	events := report.EventsByRef(injector.ID(), "db")
	if len(events) == 0 {
		t.Error("expected events for db in root scope")
	}

	assertAllEventsForService(t, events, "db")

	noEvents := report.EventsByRef("nonexistent", "db")
	if len(noEvents) != 0 {
		t.Error("expected no events for nonexistent scope")
	}
}

func TestReport_FailedServices(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	do.ProvideNamed(injector, "flaky", func(i do.Injector) (*Database, error) {
		return nil, errConnectionRefused
	})

	_ = do.MustInvokeNamed[*Database](injector, "db")
	_, _ = do.InvokeNamed[*Database](injector, "flaky")

	report := p.Report()

	failed := report.FailedServices()
	requireOneService(t, "failed", failed)

	if failed[0].ServiceName != "flaky" {
		t.Errorf("failed service: want flaky, got %s", failed[0].ServiceName)
	}
}

func TestReport_UnhealthyServices(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideHealthyDB(injector, "healthy-svc", "ok")
	provideUnhealthyCache(injector, "sick-svc", "cache miss")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "healthy-svc")
	_ = do.MustInvokeNamed[*UnhealthyCache](injector, "sick-svc")

	_ = p.RecordHealthCheck(injector)

	report := p.Report()
	unhealthy := report.UnhealthyServices()

	assertUnhealthyServiceCount(t, unhealthy, "sick-svc")
}

func TestReport_Index(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	provideCache(injector, "cache")
	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*Cache](injector, "cache")

	report := p.Report()
	idx := report.Index()

	// ByName
	if idx.ByName["db"] == nil {
		t.Error("Index.ByName: expected 'db' service")
	}

	// ByRef
	for i := range report.Services {
		key := report.Services[i].ScopeID + "/" + report.Services[i].ServiceName
		if idx.ByRef[key] == nil {
			t.Errorf("Index.ByRef: expected %q", key)
		}
	}

	// ByScope
	rootScope := report.ScopeTree.ID
	if len(idx.ByScope[rootScope]) == 0 {
		t.Error("Index.ByScope: expected services in root scope")
	}

	// EventsByName
	if len(idx.EventsByName["db"]) == 0 {
		t.Error("Index.EventsByName: expected events for 'db'")
	}

	// EventsByRef
	if len(idx.EventsByRef[rootScope+"/db"]) == 0 {
		t.Error("Index.EventsByRef: expected events for root/db")
	}

	// EventsByType
	if len(idx.EventsByType[auditlog.EventTypeRegistration]) == 0 {
		t.Error("Index.EventsByType: expected registration events")
	}
}

func TestReport_Validate_ConsistentReport(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	provideCache(injector, "cache")
	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*Cache](injector, "cache")

	report := p.Report()

	assertReportValid(t, report, "")
}

func TestReport_Validate_WithScopesAndHealthChecks(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())
	child := injector.Scope("child")

	provideHealthyDB(injector, "healthy-svc", "ok")
	provideUnhealthyCache(child, "sick-svc", "down")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "healthy-svc")
	_ = do.MustInvokeNamed[*UnhealthyCache](child, "sick-svc")

	_ = p.RecordHealthCheck(injector)

	report := p.Report()

	assertReportValid(t, report, "with scopes+health")
}

func TestReport_Validate_DetectsCountMismatch(t *testing.T) {
	t.Parallel()

	makeDBReport := func(t *testing.T) auditlog.Report {
		t.Helper()

		p := mustNew(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(p.Opts())

		provideDB(injector, "db", "test")
		_ = do.MustInvokeNamed[*Database](injector, "db")

		return p.Report()
	}

	makeHealthCheckedReport := func(t *testing.T) auditlog.Report {
		t.Helper()

		p := mustNew(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(p.Opts())

		provideHealthyDB(injector, "healthy-svc", "ok")
		_ = do.MustInvokeNamed[*HealthyDB](injector, "healthy-svc")
		_ = p.RecordHealthCheck(injector)

		return p.Report()
	}

	tests := []struct {
		name        string
		make        func(t *testing.T) auditlog.Report
		corrupt     func(*auditlog.Report)
		errContains string
	}{
		{
			name:        "event_count",
			make:        makeDBReport,
			corrupt:     func(r *auditlog.Report) { r.EventCount = 999 },
			errContains: "event_count",
		},
		{
			name:        "service_count",
			make:        makeDBReport,
			corrupt:     func(r *auditlog.Report) { r.ServiceCount = 999 },
			errContains: "service_count",
		},
		{
			name:        "scope_count",
			make:        makeDBReport,
			corrupt:     func(r *auditlog.Report) { r.ScopeCount = 999 },
			errContains: "scope_count",
		},
		{
			name:        "health_checked_count",
			make:        makeHealthCheckedReport,
			corrupt:     func(r *auditlog.Report) { r.HealthCheckedCount = 999 },
			errContains: "health_checked_count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			report := tt.make(t)
			tt.corrupt(&report)

			if err := report.Validate(); err == nil || !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("expected error containing %q, got %v", tt.errContains, err)
			}
		})
	}
}

func TestReport_Validate_EmptyReport(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	_ = do.NewWithOpts(p.Opts())

	report := p.Report()

	assertReportValid(t, report, "empty")
}

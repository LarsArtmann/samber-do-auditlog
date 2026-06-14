package auditlog_test

import (
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestPlugin_HealthCheckHealthy(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideHealthyDB(injector, "db", "postgres://localhost")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")

	results := p.RecordHealthCheck(injector)
	if len(results) == 0 {
		t.Fatal("expected health check results")
	}

	report := p.Report()

	svc := findServiceByName(t, report, "db")
	if svc == nil {
		t.Fatal("db not found in report")
	}

	assertServiceHealthCheckCount(t, svc, 1)

	if svc.LastHealthCheckAt == nil {
		t.Error("expected LastHealthCheckAt to be set")
	}

	if svc.HealthCheckError != nil {
		t.Errorf("expected no health check error, got %s", *svc.HealthCheckError)
	}
}

func TestPlugin_HealthCheckUnhealthy(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideUnhealthyCache(injector, "cache", "connection lost")

	_ = do.MustInvokeNamed[*UnhealthyCache](injector, "cache")

	results := p.RecordHealthCheck(injector)
	if len(results) == 0 {
		t.Fatal("expected health check results")
	}

	report := p.Report()

	svc := findServiceByName(t, report, "cache")
	if svc == nil {
		t.Fatal("cache not found in report")
	}

	if svc.HealthCheckError == nil {
		t.Error("expected health check error to be set")
	}

	if *svc.HealthCheckError != "cache: unhealthy" {
		t.Errorf("health check error: want 'cache: unhealthy', got %s", *svc.HealthCheckError)
	}

	if report.HealthCheckSucceeded {
		t.Error("HealthCheckSucceeded should be false when a service is unhealthy")
	}
}

func TestPlugin_HealthCheckMultipleServices(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideHealthyDB(injector, "db", "postgres://localhost")
	provideUnhealthyCache(injector, "cache", "connection lost")
	provideDB(injector, "plain", "test")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")
	_ = do.MustInvokeNamed[*UnhealthyCache](injector, "cache")
	_ = do.MustInvokeNamed[*Database](injector, "plain")

	_ = p.RecordHealthCheck(injector)

	report := p.Report()

	db := findServiceByName(t, report, "db")
	if db == nil {
		t.Fatal("db not found")
	}

	if db.HealthCheckCount != 1 {
		t.Errorf("db health_check_count: want 1, got %d", db.HealthCheckCount)
	}

	if db.HealthCheckError != nil {
		t.Errorf("db should be healthy, got error: %s", *db.HealthCheckError)
	}

	cache := findServiceByName(t, report, "cache")
	if cache == nil {
		t.Fatal("cache not found")
	}

	if cache.HealthCheckError == nil {
		t.Fatal("cache should be unhealthy")
	}

	if report.HealthCheckSucceeded {
		t.Error("HealthCheckSucceeded should be false when a service is unhealthy")
	}
}

func TestPlugin_HealthCheckDisabled(t *testing.T) {
	t.Setenv(auditlog.EnvKeyEnabled, "")

	p := auditlog.New(auditlog.Config{})
	injector := do.NewWithOpts(p.Opts())

	provideHealthyDB(injector, "db", "test")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")

	results := p.RecordHealthCheck(injector)
	if len(results) == 0 {
		t.Fatal("disabled plugin should still delegate to injector")
	}

	report := p.Report()
	assertEventCount(t, report, 0)
}

func TestPlugin_HealthCheckCount(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideHealthyDB(injector, "db", "test")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")

	_ = p.RecordHealthCheck(injector)
	_ = p.RecordHealthCheck(injector)

	report := p.Report()

	svc := findServiceByName(t, report, "db")
	if svc == nil {
		t.Fatal("db not found")
	}

	assertServiceHealthCheckCount(t, svc, 2)

	healthEvents := report.EventsByType(auditlog.EventTypeHealthCheck)
	if len(healthEvents) != 2 {
		t.Errorf("expected 2 health check events, got %d", len(healthEvents))
	}
}

func TestPlugin_HealthCheckReport(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideHealthyDB(injector, "db", "test")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")

	_ = p.RecordHealthCheck(injector)
	report := p.Report()

	if !report.HealthCheckSucceeded {
		t.Error("HealthCheckSucceeded should be true when all services are healthy")
	}

	assertIntField(t, "HealthCheckedCount", report.HealthCheckedCount, 1)
}

func TestPlugin_HealthCheckWithScope(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	child := injector.Scope("child")

	provideHealthyDB(injector, "root-db", "root")
	provideHealthyDB(child, "child-db", "child")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "root-db")
	_ = do.MustInvokeNamed[*HealthyDB](child, "child-db")

	results := p.RecordHealthCheck(child)
	if len(results) == 0 {
		t.Fatal("expected health check results from child scope")
	}

	report := p.Report()

	rootSvc := findServiceByName(t, report, "root-db")
	if rootSvc == nil {
		t.Fatal("root-db not found")
	}

	if rootSvc.HealthCheckCount != 1 {
		t.Errorf("root-db health_check_count: want 1, got %d", rootSvc.HealthCheckCount)
	}

	childSvc := findServiceByName(t, report, "child-db")
	if childSvc == nil {
		t.Fatal("child-db not found")
	}

	if childSvc.HealthCheckCount != 1 {
		t.Errorf("child-db health_check_count: want 1, got %d", childSvc.HealthCheckCount)
	}
}

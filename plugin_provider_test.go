package auditlog_test

import (
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestPlugin_ServiceTypeCapture(t *testing.T) {
	tests := []struct {
		name     string
		want     string
		register func(do.Injector)
	}{
		{
			name: "eager",
			want: "eager",
			register: func(i do.Injector) {
				do.ProvideValue(i, &Database{URL: "test"})
			},
		},
		{
			name:     "lazy",
			want:     "lazy",
			register: newDatabaseProvider(),
		},
		{
			name:     "transient",
			want:     "transient",
			register: newTransientDatabaseProvider(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, injector := newPluginAndInjector()

			tt.register(injector)
			_ = do.MustInvoke[*Database](injector)

			report := p.Report()
			assertReportServiceCount(t, report)

			if report.Services[0].ServiceType != auditlog.ProviderType(tt.want) {
				t.Errorf("expected service_type=%s, got %q", tt.want, report.Services[0].ServiceType)
			}
		})
	}
}

func newDatabaseProvider() func(do.Injector) {
	return func(i do.Injector) {
		do.Provide(i, func(_ do.Injector) (*Database, error) {
			return &Database{URL: "test"}, nil
		})
	}
}

func newTransientDatabaseProvider() func(do.Injector) {
	return func(i do.Injector) {
		do.ProvideTransient(i, func(_ do.Injector) (*Database, error) {
			return &Database{URL: "test"}, nil
		})
	}
}

func TestPlugin_CapabilityTracking(t *testing.T) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideHealthyDB(injector, "healthy-db", "test")
	provideDB(injector, "plain", "test")
	provideCrashing(injector, "crashable")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "healthy-db")
	_ = do.MustInvokeNamed[*Database](injector, "plain")
	_ = do.MustInvokeNamed[*CrashingService](injector, "crashable")

	report := p.Report()

	hdb := findServiceByName(t, report, "healthy-db")
	if hdb == nil {
		t.Fatal("healthy-db not found")
	}

	if !hdb.IsHealthchecker {
		t.Error("healthy-db should be a healthchecker")
	}

	plain := findServiceByName(t, report, "plain")
	if plain == nil {
		t.Fatal("plain not found")
	}

	if plain.IsHealthchecker {
		t.Error("plain should not be a healthchecker")
	}

	crash := findServiceByName(t, report, "crashable")
	if crash == nil {
		t.Fatal("crashable not found")
	}

	if !crash.IsShutdowner {
		t.Error("crashable should be a shutdowner")
	}
}

func TestPlugin_CapabilityTrackingWithChildScopes(t *testing.T) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideHealthyDB(injector, "root-healthy", "ok")

	child := injector.Scope("child-scope")
	provideHealthyDB(child, "child-healthy", "ok")

	_ = do.MustInvokeNamed[*HealthyDB](injector, "root-healthy")
	_ = do.MustInvokeNamed[*HealthyDB](child, "child-healthy")

	report := p.Report()

	rootSvc := findServiceByName(t, report, "root-healthy")
	if rootSvc == nil {
		t.Fatal("root-healthy not found")
	}

	if !rootSvc.IsHealthchecker {
		t.Error("root-healthy should be a healthchecker")
	}

	childSvc := findServiceByName(t, report, "child-healthy")
	if childSvc == nil {
		t.Fatal("child-healthy not found")
	}

	if !childSvc.IsHealthchecker {
		t.Error("child-healthy should be a healthchecker")
	}
}

func TestPlugin_EventServiceType(t *testing.T) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideValue(injector, &Database{URL: "test"})
	_ = do.MustInvoke[*Database](injector)

	report := p.Report()
	if len(report.Events) == 0 {
		t.Fatal("expected events")
	}

	hasType := false

	for _, e := range report.Events {
		if e.ServiceType == auditlog.ProviderTypeEager {
			hasType = true

			break
		}
	}

	if !hasType {
		t.Error("expected at least one event with eager service type")
	}
}

func TestPlugin_ProvideEager(t *testing.T) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	type eagerDB struct{ URL string }

	do.ProvideValue(injector, &eagerDB{URL: "eager"})

	report := p.Report()

	svc := findServiceBySuffix(t, report, ".eagerDB")
	if svc == nil {
		t.Fatal("expected to find eagerDB service")
	}

	if svc.ServiceType != auditlog.ProviderTypeEager {
		t.Errorf("service_type: want eager, got %q", svc.ServiceType)
	}
}

func TestPlugin_ProvideTransient(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideTransient(injector, func(i do.Injector) (*Database, error) {
		return &Database{URL: "transient://db"}, nil
	})

	db1, err := do.Invoke[*Database](injector)
	if err != nil {
		t.Fatal(err)
	}

	db2, err := do.Invoke[*Database](injector)
	if err != nil {
		t.Fatal(err)
	}

	if db1 == db2 {
		t.Error("transient should create new instances each time")
	}

	report := p.Report()
	if report.ServiceCount == 0 {
		t.Fatal("expected at least one service in report")
	}

	svc := findServiceBySuffix(t, report, ".Database")
	if svc == nil {
		t.Fatal("expected Database service in report")
	}

	assertServiceInvocationCount(t, svc, 2)
}

func TestPlugin_ProvideTransientType(t *testing.T) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	type transientToken struct{ Val int }

	do.ProvideTransient(injector, func(_ do.Injector) (*transientToken, error) {
		return &transientToken{}, nil
	})

	_ = do.MustInvoke[*transientToken](injector)
	_ = do.MustInvoke[*transientToken](injector)

	report := p.Report()

	svc := findServiceBySuffix(t, report, ".transientToken")
	if svc == nil {
		t.Fatal("expected to find transientToken service")
	}

	if svc.ServiceType != auditlog.ProviderTypeTransient {
		t.Errorf("service_type: want transient, got %q", svc.ServiceType)
	}

	assertServiceInvocationCount(t, svc, 2)
}

func TestPlugin_ProvideValue(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideValue(injector, &Database{URL: "value://db"})

	db, err := do.Invoke[*Database](injector)
	if err != nil {
		t.Fatal(err)
	}

	if db.URL != "value://db" {
		t.Errorf("expected value://db, got %s", db.URL)
	}

	report := p.Report()
	if report.ServiceCount == 0 {
		t.Fatal("expected at least one service in report")
	}

	svc := findServiceBySuffix(t, report, ".Database")
	if svc == nil {
		t.Fatal("expected Database service in report")
	}

	if svc.InvocationCount < 1 {
		t.Errorf("expected at least 1 invocation, got %d", svc.InvocationCount)
	}
}

func TestPlugin_EnrichCapabilitiesWithNilScopeRef(t *testing.T) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()

	svc := findServiceByName(t, report, "db")
	if svc == nil {
		t.Fatal("db not found")
	}

	assertStringField(t, "service name", svc.ServiceName, "db")
}

func TestPlugin_RecordHealthCheckCreatesServiceFromMeta(t *testing.T) {
	rec := auditlog.NewRecorder("test", nil)

	rec.RecordHealthCheck("scope-1", "myscope", "discovered-svc", nil)

	report := rec.BuildReport()

	svc := report.ServiceByRef("scope-1", "discovered-svc")
	if svc == nil {
		t.Fatal("discovered-svc should exist via newServiceRecordFromMeta")
	}

	assertStringField(t, "service name", svc.ServiceName, "discovered-svc")

	assertServiceHealthCheckCount(t, svc, 1)
}

package auditlog_test

import (
	"os"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestPlugin_ShutdownTracking(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")

	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = injector.Shutdown()

	report := p.Report()
	shutdownEvents := 0

	for _, e := range report.Events {
		if e.EventType == auditlog.EventTypeShutdown {
			shutdownEvents++
		}
	}

	if shutdownEvents != 2 {
		t.Errorf("shutdown events: want 2, got %d", shutdownEvents)
	}

	var db *auditlog.ServiceInfo

	db = findServiceByName(t, report, "db")

	if db == nil {
		t.Fatal("db service not found in report")
	}

	if db.ShutdownAt == nil {
		t.Error("expected ShutdownAt to be set")
	}

	if db.ShutdownDurationMs == nil {
		t.Error("expected ShutdownDurationMs to be set")
	} else if *db.ShutdownDurationMs < 0 {
		t.Errorf("expected ShutdownDurationMs >= 0, got %f", *db.ShutdownDurationMs)
	}
}

func TestPlugin_ShutdownStatus(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "clean", "clean")

	_ = do.MustInvokeNamed[*Database](injector, "clean")
	_ = injector.Shutdown()

	report := p.Report()

	svc := findServiceByName(t, report, "clean")
	if svc == nil {
		t.Fatal("clean service not found")
	}

	if svc.Status != auditlog.ServiceStatusShutdown {
		t.Errorf("status: want shutdown, got %s", svc.Status)
	}
}

func TestPlugin_ServiceStatus(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()

	svc := findServiceByName(t, report, "db")
	if svc == nil {
		t.Fatal("db not found")
	}

	if svc.Status != auditlog.ServiceStatusActive {
		t.Errorf("active service status: want %s, got %s", auditlog.ServiceStatusActive, svc.Status)
	}

	provideCache(injector, "idle")

	report2 := p.Report()

	idle := findServiceByName(t, report2, "idle")
	if idle == nil {
		t.Fatal("idle not found")
	}

	if idle.Status != auditlog.ServiceStatusRegistered {
		t.Errorf("registered service status: want %s, got %s", auditlog.ServiceStatusRegistered, idle.Status)
	}
}

func TestPlugin_ProviderErrorStatus(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideFailing(injector, "failing")

	_, err := do.InvokeNamed[*Database](injector, "failing")
	if err == nil {
		t.Fatal("expected error from failing provider")
	}

	report := p.Report()

	svc := findServiceByName(t, report, "failing")
	if svc == nil {
		t.Fatal("failing service not found in report")
	}

	if svc.Status != auditlog.ServiceStatusInvocationError {
		t.Errorf("status: want %s, got %s", auditlog.ServiceStatusInvocationError, svc.Status)
	}
}

func TestPlugin_ProviderError(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideFailing(injector, "failing")

	_, err := do.InvokeNamed[*Database](injector, "failing")
	if err == nil {
		t.Fatal("expected error from failing provider")
	}

	report := p.Report()

	var svc *auditlog.ServiceInfo

	svc = findServiceByName(t, report, "failing")

	if svc == nil {
		t.Fatal("failing service not found in report")
	}

	if svc.InvocationError == nil {
		t.Error("expected InvocationError to be set")
	} else if !strings.Contains(*svc.InvocationError, os.ErrNotExist.Error()) {
		t.Errorf("invocation error should contain %q, got %q", os.ErrNotExist.Error(), *svc.InvocationError)
	}

	assertServiceInvocationCount(t, svc, 1)
}

func TestPlugin_ShutdownWithErrors(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideCrashing(injector, "crash")

	_ = do.MustInvokeNamed[*CrashingService](injector, "crash")

	err := injector.Shutdown()
	if err == nil {
		t.Fatal("expected shutdown error from crashing service")
	}

	report := p.Report()

	if report.ShutdownSucceeded {
		t.Error("expected ShutdownSucceeded=false when shutdown errors exist")
	}

	crash := findServiceByName(t, report, "crash")
	if crash == nil {
		t.Fatal("crash service not found in report")
	}

	if crash.ShutdownError == nil {
		t.Error("expected shutdown error to be captured")
	} else if !strings.Contains(*crash.ShutdownError, errConnectionReset.Error()) {
		t.Errorf("shutdown error should contain %q, got %q", errConnectionReset.Error(), *crash.ShutdownError)
	}
}

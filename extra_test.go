package auditlog_test

import (
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestPlugin_EventHandler(t *testing.T) {
	t.Parallel()

	p, captured, injector := newPluginWithCapture()

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	if len(*captured) == 0 {
		t.Fatal("expected events via OnEvent callback")
	}

	for _, e := range *captured {
		if e.ContainerID != "default" {
			t.Errorf("callback event container_id: want default, got %s", e.ContainerID)
		}
	}

	report := p.Report()
	if report.EventCount != len(*captured) {
		t.Errorf("event count mismatch: report=%d, callback=%d", report.EventCount, len(*captured))
	}
}

func TestPlugin_EventHandlerNil(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{
		Enabled: true,
		OnEvent: nil,
	})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()
	if report.EventCount == 0 {
		t.Error("expected events even with nil OnEvent")
	}
}

func TestPlugin_RealWorldScenario(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjectorWithID("my-app")

	do.ProvideValue(injector, &Config{Port: 8080})

	provideDB(injector, "postgres", "postgres://localhost")
	provideCacheWithSleep(injector, "redis")
	provideUserServiceWithDeps(injector, "users", "postgres", "redis")
	provideHTTPServerWithUsers(injector, "http-server", "users")

	_, err := do.InvokeNamed[*HTTPServer](injector, "http-server")
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}

	report := plugin.Report()

	assertContainerID(t, report, "my-app")

	assertServiceCount(t, report, 5)

	svr := findServiceByName(t, report, "http-server")
	if svr == nil {
		t.Fatal("http-server not found")
	}

	if svr.Status != auditlog.ServiceStatusActive {
		t.Errorf("http-server status: want active, got %s", svr.Status)
	}

	if len(svr.Dependencies) != 1 {
		t.Errorf("http-server deps: want 1 (users), got %d", len(svr.Dependencies))
	}

	users := findServiceByName(t, report, "users")
	if users == nil {
		t.Fatal("users not found")
	}

	assertDependenciesCount(t, users, 2)

	postgres := findServiceByName(t, report, "postgres")
	if postgres == nil {
		t.Fatal("postgres not found")
	}

	if len(postgres.Dependents) != 1 {
		t.Errorf("postgres dependents: want 1 (users), got %d", len(postgres.Dependents))
	}

	_ = injector.Shutdown()

	report = plugin.Report()

	svr2 := findServiceByName(t, report, "http-server")
	if svr2.Status != auditlog.ServiceStatusShutdown {
		t.Errorf("after shutdown status: want shutdown, got %s", svr2.Status)
	}

	if svr2.ShutdownDurationMs == nil {
		t.Error("expected ShutdownDurationMs to be set after shutdown")
	}

	if err := plugin.ExportToFile(t.TempDir() + "/report.json"); err != nil {
		t.Errorf("JSON export failed: %v", err)
	}

	if err := plugin.ExportEventsToNDJSON(t.TempDir() + "/events.ndjson"); err != nil {
		t.Errorf("NDJSON export failed: %v", err)
	}

	if err := plugin.ExportToHTML(t.TempDir() + "/report.html"); err != nil {
		t.Errorf("HTML export failed: %v", err)
	}
}

func TestPlugin_EventsCount(t *testing.T) {
	t.Parallel()

	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	count := p.EventsCount()
	if count == 0 {
		t.Error("expected non-zero event count")
	}

	events := p.Events()
	if count != len(events) {
		t.Errorf("EventsCount() = %d, len(Events()) = %d", count, len(events))
	}
}

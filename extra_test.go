package auditlog_test

import (
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestPlugin_EventHandler(t *testing.T) {
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
	p := auditlog.New(auditlog.Config{
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
	plugin := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "my-app",
	})
	injector := do.NewWithOpts(plugin.Opts())

	do.ProvideValue(injector, &Config{Port: 8080})

	do.ProvideNamed(injector, "postgres", func(i do.Injector) (*Database, error) {
		time.Sleep(1 * time.Millisecond)

		return &Database{URL: "postgres://localhost"}, nil
	})

	do.ProvideNamed(injector, "redis", func(i do.Injector) (*Cache, error) {
		time.Sleep(1 * time.Millisecond)

		return &Cache{Entries: make(map[string]string)}, nil
	})

	do.ProvideNamed(injector, "users", func(i do.Injector) (*UserService, error) {
		db := do.MustInvokeNamed[*Database](i, "postgres")
		cache := do.MustInvokeNamed[*Cache](i, "redis")

		return &UserService{DB: db, Cache: cache}, nil
	})

	do.ProvideNamed(injector, "http-server", func(i do.Injector) (*HTTPServer, error) {
		users := do.MustInvokeNamed[*UserService](i, "users")

		return &HTTPServer{Users: users}, nil
	})

	_, err := do.InvokeNamed[*HTTPServer](injector, "http-server")
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}

	report := plugin.Report()

	if report.ContainerID != "my-app" {
		t.Errorf("container_id: want my-app, got %s", report.ContainerID)
	}

	if report.ServiceCount != 5 {
		t.Errorf("service_count: want 5, got %d", report.ServiceCount)
	}

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

	if len(users.Dependencies) != 2 {
		t.Errorf("users deps: want 2 (postgres+redis), got %d: %v", len(users.Dependencies), users.Dependencies)
	}

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

	_ = plugin.ExportToFile(t.TempDir() + "/report.json")
	_ = plugin.ExportEventsToNDJSON(t.TempDir() + "/events.ndjson")
	_ = plugin.ExportToHTML(t.TempDir() + "/report.html")
}

func TestPlugin_EventsCount(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
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

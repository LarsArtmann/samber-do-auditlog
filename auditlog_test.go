package auditlog_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/larsartmann/do-auditlog"
	"github.com/samber/do/v2"
)

type Database struct {
	URL string
}

type Cache struct {
	Entries map[string]string
}

type UserService struct {
	DB    *Database
	Cache *Cache
}

func TestPlugin_DisabledIsNoOp(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: false})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideValue(injector, &Database{URL: "postgres://localhost"})
	_ = do.MustInvoke[*Database](injector)

	report := p.Report()
	if report.EventCount != 0 {
		t.Fatalf("expected 0 events when disabled, got %d", report.EventCount)
	}
}

func TestPlugin_RegistrationAndInvocation(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true, ContainerID: "test"})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*Database, error) {
		return &Database{URL: "postgres://localhost"}, nil
	})

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
	if report.EventCount != 4 { // before+after registration + before+after invocation
		t.Errorf("event_count: want 4, got %d", report.EventCount)
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
	if svc.BuildDurationMs == nil || *svc.BuildDurationMs < 0 {
		t.Error("expected BuildDurationMs to be set and non-negative")
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
	for i := range report.Services {
		if report.Services[i].ServiceName == "users" {
			users = &report.Services[i]
			break
		}
	}
	if users == nil {
		t.Fatal("users service not found in report")
	}
	if len(users.Dependencies) != 2 {
		t.Errorf("users dependencies: want 2, got %d (%v)", len(users.Dependencies), users.Dependencies)
	}
}

func TestPlugin_ShutdownTracking(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*Database, error) {
		return &Database{URL: "postgres://localhost"}, nil
	})

	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = injector.Shutdown()

	report := p.Report()
	shutdownEvents := 0
	for _, e := range report.Events {
		if e.EventType == auditlog.EventTypeShutdown {
			shutdownEvents++
		}
	}
	if shutdownEvents != 2 { // before + after
		t.Errorf("shutdown events: want 2, got %d", shutdownEvents)
	}

	var db *auditlog.ServiceInfo
	for i := range report.Services {
		if report.Services[i].ServiceName == "db" {
			db = &report.Services[i]
			break
		}
	}
	if db == nil {
		t.Fatal("db service not found in report")
	}
	if db.ShutdownAt == nil {
		t.Error("expected ShutdownAt to be set")
	}
}

func TestPlugin_ExportToFile(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*Database, error) {
		return &Database{URL: "postgres://localhost"}, nil
	})
	_ = do.MustInvokeNamed[*Database](injector, "db")

	path := t.TempDir() + "/report.json"
	if err := p.ExportToFile(path); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var report auditlog.Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if report.ServiceCount != 1 {
		t.Errorf("expected 1 service in exported report, got %d", report.ServiceCount)
	}
}

func TestPlugin_ExportEventsToNDJSON(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*Database, error) {
		return &Database{URL: "postgres://localhost"}, nil
	})
	_ = do.MustInvokeNamed[*Database](injector, "db")

	path := t.TempDir() + "/events.ndjson"
	if err := p.ExportEventsToNDJSON(path); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if lines != 4 { // before+after registration + before+after invocation
		t.Errorf("expected 4 ndjson lines, got %d", lines)
	}
}

func TestPlugin_ScopeTree(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	child := injector.Scope("child")

	do.ProvideNamed(injector, "root-svc", func(i do.Injector) (*Database, error) {
		return &Database{URL: "root"}, nil
	})
	do.ProvideNamed(child, "child-svc", func(i do.Injector) (*Database, error) {
		return &Database{URL: "child"}, nil
	})

	_ = do.MustInvokeNamed[*Database](injector, "root-svc")
	_ = do.MustInvokeNamed[*Database](child, "child-svc")

	report := p.Report()
	if report.ScopeTree.Name != "[root]" {
		t.Errorf("root scope name: want [root], got %s", report.ScopeTree.Name)
	}
	if len(report.ScopeTree.Children) != 1 {
		t.Fatalf("expected 1 child scope, got %d", len(report.ScopeTree.Children))
	}
	if report.ScopeTree.Children[0].Name != "child" {
		t.Errorf("child scope name: want child, got %s", report.ScopeTree.Children[0].Name)
	}
}

func TestPlugin_CachedInvocationDoesNotCreateDuplicateDependency(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	// db is invoked twice (once directly, once via users)
	do.ProvideNamed(injector, "db", func(i do.Injector) (*Database, error) {
		return &Database{URL: "postgres://localhost"}, nil
	})

	do.ProvideNamed(injector, "users", func(i do.Injector) (*UserService, error) {
		db := do.MustInvokeNamed[*Database](i, "db")
		return &UserService{DB: db}, nil
	})

	// First invoke db directly
	_ = do.MustInvokeNamed[*Database](injector, "db")
	// Then invoke users which also needs db
	_ = do.MustInvokeNamed[*UserService](injector, "users")

	report := p.Report()
	var users *auditlog.ServiceInfo
	for i := range report.Services {
		if report.Services[i].ServiceName == "users" {
			users = &report.Services[i]
			break
		}
	}
	if users == nil {
		t.Fatal("users service not found")
	}
	if len(users.Dependencies) != 1 {
		t.Errorf("users should have exactly 1 dependency (db), got %d: %v", len(users.Dependencies), users.Dependencies)
	}

	var db *auditlog.ServiceInfo
	for i := range report.Services {
		if report.Services[i].ServiceName == "db" {
			db = &report.Services[i]
			break
		}
	}
	if db == nil {
		t.Fatal("db service not found")
	}
	if db.InvocationCount != 2 {
		t.Errorf("db invocation_count: want 2, got %d", db.InvocationCount)
	}
}

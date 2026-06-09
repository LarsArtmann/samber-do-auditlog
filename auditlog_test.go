package auditlog_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
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

	if report.EventCount != 4 {
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

	if svc.InvocationOrder != 0 {
		t.Errorf("first invoked service should have order 0, got %d", svc.InvocationOrder)
	}
}

func TestPlugin_InvocationOrder(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "a", func(i do.Injector) (*Database, error) {
		return &Database{URL: "a"}, nil
	})
	do.ProvideNamed(injector, "b", func(i do.Injector) (*Cache, error) {
		return &Cache{}, nil
	})

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

	if len(db.Dependents) != 1 || db.Dependents[0].ServiceName != "users" {
		t.Errorf("db dependents: want [users], got %v", db.Dependents)
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

	if shutdownEvents != 2 {
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

	if lines != 4 {
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

	if len(report.ScopeTree.Services) != 1 || report.ScopeTree.Services[0] != "root-svc" {
		t.Errorf("root services: want [root-svc], got %v", report.ScopeTree.Services)
	}

	if len(report.ScopeTree.Children[0].Services) != 1 || report.ScopeTree.Children[0].Services[0] != "child-svc" {
		t.Errorf("child services: want [child-svc], got %v", report.ScopeTree.Children[0].Services)
	}
}

func TestPlugin_CachedInvocation(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*Database, error) {
		return &Database{URL: "postgres://localhost"}, nil
	})

	do.ProvideNamed(injector, "users", func(i do.Injector) (*UserService, error) {
		db := do.MustInvokeNamed[*Database](i, "db")

		return &UserService{DB: db}, nil
	})

	_ = do.MustInvokeNamed[*Database](injector, "db")
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

func TestPlugin_EventSequenceNumbers(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*Database, error) {
		return &Database{URL: "postgres://localhost"}, nil
	})
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

func TestPlugin_ProviderError(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "failing", func(i do.Injector) (*Database, error) {
		return nil, os.ErrNotExist
	})

	_, err := do.InvokeNamed[*Database](injector, "failing")
	if err == nil {
		t.Fatal("expected error from failing provider")
	}

	report := p.Report()

	var svc *auditlog.ServiceInfo

	for i := range report.Services {
		if report.Services[i].ServiceName == "failing" {
			svc = &report.Services[i]

			break
		}
	}

	if svc == nil {
		t.Fatal("failing service not found in report")
	}

	if svc.InvocationError == nil {
		t.Error("expected InvocationError to be set")
	}

	if svc.InvocationCount != 1 {
		t.Errorf("invocation_count: want 1, got %d", svc.InvocationCount)
	}
}

func TestPlugin_ExportToHTML(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*Database, error) {
		return &Database{URL: "postgres://localhost"}, nil
	})
	_ = do.MustInvokeNamed[*Database](injector, "db")

	path := t.TempDir() + "/report.html"
	if err := p.ExportToHTML(path); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if len(data) < 500 {
		t.Errorf("HTML file too small (%d bytes), expected a full page", len(data))
	}

	if !contains(string(data), "<!DOCTYPE html>") {
		t.Error("expected DOCTYPE in HTML output")
	}

	if !contains(string(data), "db") {
		t.Error("expected 'db' service name in HTML output")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || searchString(s, sub))
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}

	return false
}

func BenchmarkHookOverhead_Invocation(b *testing.B) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*Database, error) {
		return &Database{URL: "postgres://localhost"}, nil
	})

	b.ResetTimer()

	for range b.N {
		_, _ = do.InvokeNamed[*Database](injector, "db")
	}
}

func BenchmarkHookOverhead_Disabled(b *testing.B) {
	p := auditlog.New(auditlog.Config{Enabled: false})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*Database, error) {
		return &Database{URL: "postgres://localhost"}, nil
	})

	b.ResetTimer()

	for range b.N {
		_, _ = do.InvokeNamed[*Database](injector, "db")
	}
}

func BenchmarkHookOverhead_Registration(b *testing.B) {
	b.ResetTimer()

	for range b.N {
		p := auditlog.New(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(p.Opts())
		do.ProvideNamed(injector, "svc", func(_ do.Injector) (*Database, error) {
			return &Database{URL: "test"}, nil
		})
	}
}

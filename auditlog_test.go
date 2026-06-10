package auditlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
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
	t.Setenv(auditlog.EnvKeyEnabled, "")

	p := auditlog.New(auditlog.Config{})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideValue(injector, &Database{URL: "postgres://localhost"})
	_ = do.MustInvoke[*Database](injector)

	report := p.Report()
	if report.EventCount != 0 {
		t.Fatalf("expected 0 events when disabled, got %d", report.EventCount)
	}
}

func TestPlugin_EnvVarEnables(t *testing.T) {
	t.Setenv(auditlog.EnvKeyEnabled, "true")

	p := auditlog.New(auditlog.Config{})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()
	if report.EventCount == 0 {
		t.Fatal("expected events when env var is set")
	}

	if report.ServiceCount != 1 {
		t.Errorf("expected 1 service, got %d", report.ServiceCount)
	}
}

func TestPlugin_EnvVarValues(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"", false},
		{"random", false},
	}
	for _, tc := range tests {
		t.Run(tc.val, func(t *testing.T) {
			t.Setenv(auditlog.EnvKeyEnabled, tc.val)

			p := auditlog.New(auditlog.Config{})
			injector := do.NewWithOpts(p.Opts())

			provideDB(injector, "db", "test")
			_ = do.MustInvokeNamed[*Database](injector, "db")

			report := p.Report()
			if tc.want && report.EventCount == 0 {
				t.Errorf("env %q: expected events", tc.val)
			}

			if !tc.want && report.EventCount != 0 {
				t.Errorf("env %q: expected no events, got %d", tc.val, report.EventCount)
			}
		})
	}
}

func TestPlugin_ExplicitEnabledOverridesEnv(t *testing.T) {
	t.Setenv(auditlog.EnvKeyEnabled, "")

	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()
	if report.EventCount == 0 {
		t.Fatal("explicit Enabled:true should work even when env is unset")
	}
}

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

func TestPlugin_ShutdownTracking(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
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

func TestPlugin_ShutdownError(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "leaky", func(i do.Injector) (*Database, error) {
		return &Database{URL: "leaky"}, nil
	})

	_ = do.MustInvokeNamed[*Database](injector, "leaky")
	_ = injector.Shutdown()

	report := p.Report()

	svc := findServiceByName(t, report, "leaky")
	if svc == nil {
		t.Fatal("leaky service not found")
	}

	if svc.Status != auditlog.ServiceStatusShutdown {
		t.Errorf("status: want shutdown, got %s", svc.Status)
	}
}

func TestPlugin_ServiceStatus(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
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

	do.ProvideNamed(injector, "idle", func(i do.Injector) (*Cache, error) {
		return &Cache{}, nil
	})

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

	svc := findServiceByName(t, report, "failing")
	if svc == nil {
		t.Fatal("failing service not found in report")
	}

	if svc.Status != auditlog.ServiceStatusInvocationError {
		t.Errorf("status: want %s, got %s", auditlog.ServiceStatusInvocationError, svc.Status)
	}
}

func TestPlugin_ExportToFile(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
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

	provideDB(injector, "db", "postgres://localhost")
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

	provideDB(injector, "root-svc", "root")
	provideDB(child, "child-svc", "child")

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

	svc = findServiceByName(t, report, "failing")

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

	provideDB(injector, "db", "postgres://localhost")
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

	if !strings.Contains(strings.ToLower(string(data)), "<!doctype html>") {
		t.Error("expected DOCTYPE in HTML output")
	}

	if !strings.Contains(string(data), "db") {
		t.Error("expected 'db' service name in HTML output")
	}
}

func TestPlugin_ContainerID(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true, ContainerID: "test-container"})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	events := p.Events()
	if len(events) == 0 {
		t.Fatal("expected events")
	}

	for _, e := range events {
		if e.ContainerID != "test-container" {
			t.Errorf("event %d container_id: want test-container, got %s", e.Sequence, e.ContainerID)
		}
	}
}

func TestPlugin_ReportVersion(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()
	if report.Version != auditlog.SchemaVersion {
		t.Errorf("version: want %s, got %s", auditlog.SchemaVersion, report.Version)
	}
}

func TestPlugin_ScopeID(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	child := injector.Scope("child")

	provideDB(injector, "root-svc", "root")
	provideDB(child, "child-svc", "child")

	_ = do.MustInvokeNamed[*Database](injector, "root-svc")
	_ = do.MustInvokeNamed[*Database](child, "child-svc")

	report := p.Report()

	rootSvc := findServiceByName(t, report, "root-svc")
	if rootSvc == nil {
		t.Fatal("root-svc not found")
	}

	if rootSvc.ScopeID == "" {
		t.Error("expected ScopeID to be set on root service")
	}

	childSvc := findServiceByName(t, report, "child-svc")
	if childSvc == nil {
		t.Fatal("child-svc not found")
	}

	if childSvc.ScopeID == "" {
		t.Error("expected ScopeID to be set on child service")
	}

	if rootSvc.ScopeID == childSvc.ScopeID {
		t.Error("root and child should have different ScopeIDs")
	}
}

func TestPlugin_WriteReportJSON(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := p.WriteReportJSON(&buf)
	if err != nil {
		t.Fatalf("WriteReportJSON failed: %v", err)
	}

	var report auditlog.Report
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if report.ServiceCount != 1 {
		t.Errorf("expected 1 service, got %d", report.ServiceCount)
	}

	if report.Version != auditlog.SchemaVersion {
		t.Errorf("version: want %s, got %s", auditlog.SchemaVersion, report.Version)
	}
}

func TestPlugin_WriteEventsNDJSON(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := p.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON failed: %v", err)
	}

	lines := strings.Count(buf.String(), "\n")
	if lines != 4 {
		t.Errorf("expected 4 ndjson lines, got %d", lines)
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

	if report.Version != auditlog.SchemaVersion {
		t.Errorf("version: want %s, got %s", auditlog.SchemaVersion, report.Version)
	}

	if report.ScopeTree.Name != "" && report.ScopeTree.ID != "" {
		t.Error("expected empty scope tree for empty report")
	}
}

func TestPlugin_ConcurrentInvocations(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*Database, error) {
		return &Database{URL: "postgres://localhost"}, nil
	})

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

func provideDB(injector do.Injector, name, url string) {
	do.ProvideNamed(injector, name, func(_ do.Injector) (*Database, error) {
		return &Database{URL: url}, nil
	})
}

func findServiceByName(t *testing.T, report auditlog.Report, name string) *auditlog.ServiceInfo {
	t.Helper()

	for i := range report.Services {
		if report.Services[i].ServiceName == name {
			return &report.Services[i]
		}
	}

	return nil
}

func findServiceBySuffix(t *testing.T, report auditlog.Report, suffix string) *auditlog.ServiceInfo {
	t.Helper()

	for i := range report.Services {
		if strings.HasSuffix(report.Services[i].ServiceName, suffix) {
			return &report.Services[i]
		}
	}

	return nil
}

func TestPlugin_ProvideTransient(t *testing.T) {
	t.Parallel()

	p := auditlog.New(auditlog.Config{Enabled: true})
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

	if svc.InvocationCount != 2 {
		t.Errorf("expected 2 invocations for transient, got %d", svc.InvocationCount)
	}
}

func TestPlugin_ProvideValue(t *testing.T) {
	t.Parallel()

	p := auditlog.New(auditlog.Config{Enabled: true})
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

func BenchmarkHookOverhead_Invocation(b *testing.B) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")

	b.ResetTimer()

	for b.Loop() {
		_, _ = do.InvokeNamed[*Database](injector, "db")
	}
}

func BenchmarkHookOverhead_Disabled(b *testing.B) {
	p := auditlog.New(auditlog.Config{Enabled: false})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")

	b.ResetTimer()

	for b.Loop() {
		_, _ = do.InvokeNamed[*Database](injector, "db")
	}
}

func BenchmarkHookOverhead_Registration(b *testing.B) {
	b.ResetTimer()

	for b.Loop() {
		p := auditlog.New(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(p.Opts())
		provideDB(injector, "svc", "test")
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

func TestPlugin_EventHandler(t *testing.T) {
	var captured []auditlog.Event

	p := auditlog.New(auditlog.Config{
		Enabled: true,
		OnEvent: func(e auditlog.Event) {
			captured = append(captured, e)
		},
	})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	if len(captured) == 0 {
		t.Fatal("expected events via OnEvent callback")
	}

	for _, e := range captured {
		if e.ContainerID != "default" {
			t.Errorf("callback event container_id: want default, got %s", e.ContainerID)
		}
	}

	report := p.Report()
	if report.EventCount != len(captured) {
		t.Errorf("event count mismatch: report=%d, callback=%d", report.EventCount, len(captured))
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

type HTTPServer struct {
	Users *UserService
}

type Config struct {
	Port int
}

type CrashingService struct{}

var errConnectionReset = errors.New("connection reset")

func (c *CrashingService) Shutdown() error {
	return errConnectionReset
}

func TestEvent_ConvenienceMethods(t *testing.T) {
	events := []struct {
		event      auditlog.Event
		wantReg    bool
		wantInv    bool
		wantShut   bool
		wantHealth bool
		wantBefore bool
		wantAfter  bool
	}{
		{
			event:   auditlog.Event{EventType: auditlog.EventTypeRegistration, Phase: auditlog.PhaseBefore},
			wantReg: true, wantBefore: true,
		},
		{
			event:   auditlog.Event{EventType: auditlog.EventTypeInvocation, Phase: auditlog.PhaseAfter},
			wantInv: true, wantAfter: true,
		},
		{
			event:    auditlog.Event{EventType: auditlog.EventTypeShutdown, Phase: auditlog.PhaseBefore},
			wantShut: true, wantBefore: true,
		},
		{
			event:      auditlog.Event{EventType: auditlog.EventTypeHealthCheck, Phase: auditlog.PhaseAfter},
			wantHealth: true, wantAfter: true,
		},
	}

	for i, tc := range events {
		if got := tc.event.IsRegistration(); got != tc.wantReg {
			t.Errorf("case %d: IsRegistration() = %v, want %v", i, got, tc.wantReg)
		}

		if got := tc.event.IsInvocation(); got != tc.wantInv {
			t.Errorf("case %d: IsInvocation() = %v, want %v", i, got, tc.wantInv)
		}

		if got := tc.event.IsShutdown(); got != tc.wantShut {
			t.Errorf("case %d: IsShutdown() = %v, want %v", i, got, tc.wantShut)
		}

		if got := tc.event.IsHealthCheck(); got != tc.wantHealth {
			t.Errorf("case %d: IsHealthCheck() = %v, want %v", i, got, tc.wantHealth)
		}

		if got := tc.event.IsBefore(); got != tc.wantBefore {
			t.Errorf("case %d: IsBefore() = %v, want %v", i, got, tc.wantBefore)
		}

		if got := tc.event.IsAfter(); got != tc.wantAfter {
			t.Errorf("case %d: IsAfter() = %v, want %v", i, got, tc.wantAfter)
		}
	}
}

func TestPlugin_WriteHTMLBuffer(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := p.WriteHTML(&buf)
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := buf.String()
	if len(html) < 500 {
		t.Errorf("HTML too small (%d bytes)", len(html))
	}

	if !strings.Contains(strings.ToLower(html), "<!doctype html>") {
		t.Error("expected DOCTYPE in HTML output")
	}

	if !strings.Contains(html, "db") {
		t.Error("expected 'db' service name in HTML output")
	}
}

func TestPlugin_WriteReportJSONError(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := p.WriteReportJSON(&buf)
	if err != nil {
		t.Fatalf("WriteReportJSON failed: %v", err)
	}

	var report auditlog.Report
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if report.ServiceCount != 1 {
		t.Errorf("expected 1 service, got %d", report.ServiceCount)
	}
}

func TestPlugin_ScopeTreeWithMultipleChildren(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	child1 := injector.Scope("child-1")
	child2 := injector.Scope("child-2")

	provideDB(injector, "root-svc", "root")
	provideDB(child1, "child1-svc", "child1")
	provideDB(child2, "child2-svc", "child2")

	_ = do.MustInvokeNamed[*Database](injector, "root-svc")
	_ = do.MustInvokeNamed[*Database](child1, "child1-svc")
	_ = do.MustInvokeNamed[*Database](child2, "child2-svc")

	report := p.Report()

	if len(report.ScopeTree.Children) != 2 {
		t.Fatalf("expected 2 child scopes, got %d", len(report.ScopeTree.Children))
	}
}

func TestPlugin_ServiceTypeCapture(t *testing.T) {
	t.Run("eager", func(t *testing.T) {
		p := auditlog.New(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(p.Opts())

		do.ProvideValue(injector, &Database{URL: "test"})
		_ = do.MustInvoke[*Database](injector)

		report := p.Report()
		if len(report.Services) != 1 {
			t.Fatalf("expected 1 service, got %d", len(report.Services))
		}

		if report.Services[0].ServiceType != "eager" {
			t.Errorf("expected service_type=eager, got %q", report.Services[0].ServiceType)
		}
	})

	t.Run("lazy", func(t *testing.T) {
		p := auditlog.New(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(p.Opts())

		do.Provide(injector, func(i do.Injector) (*Database, error) {
			return &Database{URL: "test"}, nil
		})
		_ = do.MustInvoke[*Database](injector)

		report := p.Report()
		if report.Services[0].ServiceType != "lazy" {
			t.Errorf("expected service_type=lazy, got %q", report.Services[0].ServiceType)
		}
	})

	t.Run("transient", func(t *testing.T) {
		p := auditlog.New(auditlog.Config{Enabled: true})
		injector := do.NewWithOpts(p.Opts())

		do.ProvideTransient(injector, func(i do.Injector) (*Database, error) {
			return &Database{URL: "test"}, nil
		})
		_ = do.MustInvoke[*Database](injector)

		report := p.Report()
		if report.Services[0].ServiceType != "transient" {
			t.Errorf("expected service_type=transient, got %q", report.Services[0].ServiceType)
		}
	})
}

func TestPlugin_ShutdownWithErrors(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "crash", func(i do.Injector) (*CrashingService, error) {
		return &CrashingService{}, nil
	})

	_ = do.MustInvokeNamed[*CrashingService](injector, "crash")
	_ = injector.Shutdown()

	report := p.Report()

	if report.ShutdownSucceeded {
		t.Error("expected ShutdownSucceeded=false when shutdown errors exist")
	}

	if len(report.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(report.Services))
	}

	if report.Services[0].ShutdownError == nil {
		t.Error("expected shutdown error to be captured")
	}
}

// --- Health check test types ---

type HealthyDB struct {
	DSN string
}

var _ do.Healthchecker = (*HealthyDB)(nil)

func (d *HealthyDB) HealthCheck() error {
	return nil
}

type UnhealthyCache struct {
	Reason string
}

var _ do.HealthcheckerWithContext = (*UnhealthyCache)(nil)

var errCacheUnhealthy = errors.New("cache: unhealthy")

func (c *UnhealthyCache) HealthCheck(_ context.Context) error {
	return errCacheUnhealthy
}

// --- Health check tests ---

func TestPlugin_HealthCheckHealthy(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: "postgres://localhost"}, nil
	})

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

	if svc.HealthCheckCount != 1 {
		t.Errorf("health_check_count: want 1, got %d", svc.HealthCheckCount)
	}

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

	do.ProvideNamed(injector, "cache", func(i do.Injector) (*UnhealthyCache, error) {
		return &UnhealthyCache{Reason: "connection lost"}, nil
	})

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

	do.ProvideNamed(injector, "db", func(i do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: "postgres://localhost"}, nil
	})
	do.ProvideNamed(injector, "cache", func(i do.Injector) (*UnhealthyCache, error) {
		return &UnhealthyCache{Reason: "connection lost"}, nil
	})
	do.ProvideNamed(injector, "plain", func(i do.Injector) (*Database, error) {
		return &Database{URL: "test"}, nil
	})

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
		t.Error("cache should be unhealthy")
	}
}

func TestPlugin_HealthCheckDisabled(t *testing.T) {
	t.Setenv(auditlog.EnvKeyEnabled, "")

	p := auditlog.New(auditlog.Config{})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: "test"}, nil
	})

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")

	results := p.RecordHealthCheck(injector)
	if len(results) == 0 {
		t.Fatal("disabled plugin should still delegate to injector")
	}

	report := p.Report()
	if report.EventCount != 0 {
		t.Errorf("expected 0 events when disabled, got %d", report.EventCount)
	}
}

func TestPlugin_HealthCheckCount(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: "test"}, nil
	})

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")

	_ = p.RecordHealthCheck(injector)
	_ = p.RecordHealthCheck(injector)

	report := p.Report()

	svc := findServiceByName(t, report, "db")
	if svc == nil {
		t.Fatal("db not found")
	}

	if svc.HealthCheckCount != 2 {
		t.Errorf("health_check_count: want 2, got %d", svc.HealthCheckCount)
	}

	healthEvents := report.EventsByType(auditlog.EventTypeHealthCheck)
	if len(healthEvents) != 2 {
		t.Errorf("expected 2 health check events, got %d", len(healthEvents))
	}
}

func TestPlugin_HealthCheckReport(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: "test"}, nil
	})

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")

	_ = p.RecordHealthCheck(injector)
	report := p.Report()

	if !report.HealthCheckSucceeded {
		t.Error("HealthCheckSucceeded should be true when all services are healthy")
	}

	if report.HealthCheckedCount != 1 {
		t.Errorf("HealthCheckedCount: want 1, got %d", report.HealthCheckedCount)
	}
}

func TestPlugin_HealthCheckWithScope(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	child := injector.Scope("child")

	do.ProvideNamed(injector, "root-db", func(i do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: "root"}, nil
	})
	do.ProvideNamed(child, "child-db", func(i do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: "child"}, nil
	})

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

func TestPlugin_HealthCheckReportSucceeded(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "cache", func(i do.Injector) (*UnhealthyCache, error) {
		return &UnhealthyCache{Reason: "down"}, nil
	})

	_ = do.MustInvokeNamed[*UnhealthyCache](injector, "cache")

	_ = p.RecordHealthCheck(injector)
	report := p.Report()

	if report.HealthCheckSucceeded {
		t.Error("HealthCheckSucceeded should be false when a service is unhealthy")
	}

	unhealthy := report.UnhealthyServices()
	if len(unhealthy) != 1 {
		t.Fatalf("expected 1 unhealthy service, got %d", len(unhealthy))
	}

	if unhealthy[0].ServiceName != "cache" {
		t.Errorf("unhealthy service: want cache, got %s", unhealthy[0].ServiceName)
	}
}

func TestPlugin_HealthCheckSucceededFalseWhenNoChecks(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*Database, error) {
		return &Database{URL: "test"}, nil
	})

	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()
	if report.HealthCheckSucceeded {
		t.Error("HealthCheckSucceeded should be false when no health checks have been recorded")
	}

	if report.HealthCheckedCount != 0 {
		t.Errorf("HealthCheckedCount: want 0, got %d", report.HealthCheckedCount)
	}
}

func TestPlugin_HealthCheckOnEventCallback(t *testing.T) {
	var captured []auditlog.Event

	p := auditlog.New(auditlog.Config{
		Enabled: true,
		OnEvent: func(e auditlog.Event) {
			captured = append(captured, e)
		},
	})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: "test"}, nil
	})

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")

	_ = p.RecordHealthCheck(injector)

	var healthCallbacks []auditlog.Event
	for _, e := range captured {
		if e.IsHealthCheck() {
			healthCallbacks = append(healthCallbacks, e)
		}
	}

	if len(healthCallbacks) != 1 {
		t.Fatalf("expected 1 health check event via OnEvent, got %d", len(healthCallbacks))
	}

	evt := healthCallbacks[0]
	if evt.EventType != auditlog.EventTypeHealthCheck {
		t.Errorf("event type: want health_check, got %s", evt.EventType)
	}

	if evt.Phase != auditlog.PhaseAfter {
		t.Errorf("phase: want after, got %s", evt.Phase)
	}

	if evt.ServiceName != "db" {
		t.Errorf("service name: want db, got %s", evt.ServiceName)
	}

	if evt.Error != nil {
		t.Errorf("expected no error for healthy service, got %s", *evt.Error)
	}
}

func TestPlugin_HealthCheckPhaseIsAfterOnly(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: "test"}, nil
	})

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")
	_ = p.RecordHealthCheck(injector)

	report := p.Report()
	healthEvents := report.EventsByType(auditlog.EventTypeHealthCheck)

	if len(healthEvents) != 1 {
		t.Fatalf("expected 1 health check event, got %d", len(healthEvents))
	}

	if healthEvents[0].Phase != auditlog.PhaseAfter {
		t.Errorf("health check events should always be PhaseAfter, got %s", healthEvents[0].Phase)
	}

	if healthEvents[0].DurationMs != nil {
		t.Error("health check events should not have DurationMs (per-service timing unavailable)")
	}
}

func TestPlugin_HealthCheckJSONExport(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: "test"}, nil
	})
	do.ProvideNamed(injector, "cache", func(i do.Injector) (*UnhealthyCache, error) {
		return &UnhealthyCache{Reason: "down"}, nil
	})

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")
	_ = do.MustInvokeNamed[*UnhealthyCache](injector, "cache")

	_ = p.RecordHealthCheck(injector)

	var buf bytes.Buffer

	err := p.WriteReportJSON(&buf)
	if err != nil {
		t.Fatalf("WriteReportJSON: %v", err)
	}

	var report map[string]interface{}

	err = json.Unmarshal(buf.Bytes(), &report)
	if err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if report["health_check_succeeded"] != false {
		t.Error("expected health_check_succeeded to be false")
	}

	services, ok := report["services"].([]interface{})
	if !ok {
		t.Fatal("services should be an array")
	}

	for _, svc := range services {
		svcMap, ok := svc.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := svcMap["service_name"].(string)
		if name == "cache" {
			if svcMap["health_check_error"] == nil {
				t.Error("cache should have health_check_error in JSON export")
			}
		}
	}
}

func TestPlugin_HealthCheckNDJSONExport(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	do.ProvideNamed(injector, "db", func(i do.Injector) (*HealthyDB, error) {
		return &HealthyDB{DSN: "test"}, nil
	})

	_ = do.MustInvokeNamed[*HealthyDB](injector, "db")
	_ = p.RecordHealthCheck(injector)

	var buf bytes.Buffer

	err := p.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	foundHealthCheck := false

	for _, line := range lines {
		var evt map[string]interface{}

		err := json.Unmarshal([]byte(line), &evt)
		if err != nil {
			t.Fatalf("json unmarshal line: %v", err)
		}

		if evt["event_type"] == "health_check" {
			foundHealthCheck = true

			if evt["phase"] != "after" {
				t.Errorf("health check event phase: want after, got %v", evt["phase"])
			}

			if _, ok := evt["duration_ms"]; ok {
				t.Error("health check events should not have duration_ms in NDJSON export")
			}
		}
	}

	if !foundHealthCheck {
		t.Error("expected at least one health_check event in NDJSON export")
	}
}

func TestReport_ServiceByName(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()

	svc := report.ServiceByName("db")
	if svc == nil {
		t.Fatal("expected to find db service")
	}

	if svc.ServiceName != "db" {
		t.Errorf("service name: want db, got %s", svc.ServiceName)
	}

	if report.ServiceByName("nonexistent") != nil {
		t.Error("expected nil for nonexistent service")
	}
}

func TestReport_FailedServices(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	do.ProvideNamed(injector, "flaky", func(i do.Injector) (*Database, error) {
		return nil, errors.New("connection refused")
	})

	_ = do.MustInvokeNamed[*Database](injector, "db")
	_, _ = do.InvokeNamed[*Database](injector, "flaky")

	report := p.Report()
	failed := report.FailedServices()
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed service, got %d", len(failed))
	}

	if failed[0].ServiceName != "flaky" {
		t.Errorf("failed service: want flaky, got %s", failed[0].ServiceName)
	}
}

func TestServiceStatus_IsError(t *testing.T) {
	tests := []struct {
		status auditlog.ServiceStatus
		want   bool
	}{
		{auditlog.ServiceStatusInvocationError, true},
		{auditlog.ServiceStatusShutdownError, true},
		{auditlog.ServiceStatusRegistered, false},
		{auditlog.ServiceStatusActive, false},
		{auditlog.ServiceStatusShutdown, false},
	}
	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			if tc.status.IsError() != tc.want {
				t.Errorf("IsError() = %v, want %v", tc.status.IsError(), tc.want)
			}
		})
	}
}

func TestServiceRef_String(t *testing.T) {
	tests := []struct {
		ref  auditlog.ServiceRef
		want string
	}{
		{auditlog.ServiceRef{ScopeName: "api", ServiceName: "db"}, "api/db"},
		{auditlog.ServiceRef{ScopeName: "[root]", ServiceName: "db"}, "db"},
		{auditlog.ServiceRef{ScopeName: "", ServiceName: "db"}, "db"},
	}
	for _, tc := range tests {
		t.Run(tc.ref.String(), func(t *testing.T) {
			if tc.ref.String() != tc.want {
				t.Errorf("String() = %q, want %q", tc.ref.String(), tc.want)
			}
		})
	}
}

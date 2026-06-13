package auditlog_test

import (
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestReport_FilteredByName(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	provideCache(injector, "cache")
	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*Cache](injector, "cache")

	filtered := p.Report().Filtered(auditlog.WithServicesByName("db"))

	if len(filtered.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(filtered.Services))
	}

	if filtered.Services[0].ServiceName != "db" {
		t.Errorf("service name: want db, got %s", filtered.Services[0].ServiceName)
	}

	// Scope tree should be pruned to only include the matching service.
	if len(filtered.ScopeTree.Services) != 1 || filtered.ScopeTree.Services[0] != "db" {
		t.Errorf("scope_tree services: want [db], got %v", filtered.ScopeTree.Services)
	}
}

func TestReport_FilteredByType(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	do.ProvideValue(injector, &Cache{})

	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvoke[*Cache](injector)

	filtered := p.Report().Filtered(auditlog.WithServicesByType(auditlog.ProviderTypeEager))

	for _, svc := range filtered.Services {
		if svc.ServiceType != auditlog.ProviderTypeEager {
			t.Errorf("expected eager, got %s for %s", svc.ServiceType, svc.ServiceName)
		}
	}
}

func TestReport_FilteredByEventType(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	filtered := p.Report().Filtered(auditlog.WithEventsByType(auditlog.EventTypeInvocation))

	for _, evt := range filtered.Events {
		if evt.EventType != auditlog.EventTypeInvocation {
			t.Errorf("expected invocation, got %s", evt.EventType)
		}
	}
}

func TestReport_FilteredByTimeRange(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	report := p.Report()

	from := report.Events[0].Timestamp.Add(-1 * time.Hour)
	to := report.Events[0].Timestamp.Add(1 * time.Hour)

	filtered := report.Filtered(auditlog.WithTimeRange(from, to))
	if filtered.EventCount == 0 {
		t.Error("expected events in time range")
	}

	filteredEmpty := report.Filtered(auditlog.WithTimeRange(
		time.Now().Add(100*time.Hour),
		time.Now().Add(200*time.Hour),
	))
	if filteredEmpty.EventCount != 0 {
		t.Error("expected no events outside time range")
	}
}

func TestReport_FilteredByScope(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	child := injector.Scope("child")

	provideDB(injector, "root-svc", "test")
	do.ProvideNamed(child, "child-svc", func(_ do.Injector) (*Database, error) {
		return &Database{URL: "child"}, nil
	})

	_ = do.MustInvokeNamed[*Database](injector, "root-svc")
	_ = do.MustInvokeNamed[*Database](child, "child-svc")

	filtered := p.Report().Filtered(auditlog.WithScope(child.ID()))
	if len(filtered.Services) != 1 {
		t.Fatalf("expected 1 child service, got %d", len(filtered.Services))
	}

	if filtered.Services[0].ServiceName != "child-svc" {
		t.Errorf("service: want child-svc, got %s", filtered.Services[0].ServiceName)
	}

	// Scope tree preserves hierarchy: root scope remains root, child is pruned to matching scope.
	if filtered.ScopeCount != 2 {
		t.Errorf("scope_count: want 2 (root + child), got %d", filtered.ScopeCount)
	}

	if len(filtered.ScopeTree.Children) != 1 {
		t.Fatalf("scope_tree: expected 1 child, got %d", len(filtered.ScopeTree.Children))
	}

	if filtered.ScopeTree.Children[0].ID != child.ID() {
		t.Errorf("scope_tree child: expected %s, got %s", child.ID(), filtered.ScopeTree.Children[0].ID)
	}
}

func TestReport_FilteredCombined(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	provideCache(injector, "cache")
	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*Cache](injector, "cache")

	filtered := p.Report().Filtered(
		auditlog.WithServicesByName("db"),
		auditlog.WithEventsByType(auditlog.EventTypeInvocation),
	)

	if len(filtered.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(filtered.Services))
	}

	if filtered.Services[0].ServiceName != "db" {
		t.Errorf("service name: want db, got %s", filtered.Services[0].ServiceName)
	}

	for _, evt := range filtered.Events {
		if evt.EventType != auditlog.EventTypeInvocation {
			t.Errorf("expected invocation, got %s", evt.EventType)
		}
	}
}

func TestReport_FilteredTimeRangeNilChecks(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)

	filtered := p.ReportFiltered(auditlog.WithTimeRange(from, to))
	if filtered.EventCount == 0 {
		t.Error("expected events in time range")
	}

	noEvents := p.ReportFiltered(auditlog.WithTimeRange(
		time.Now().Add(-time.Hour),
		time.Now().Add(-30*time.Minute),
	))
	if noEvents.EventCount != 0 {
		t.Errorf("expected 0 events before container ran, got %d", noEvents.EventCount)
	}
}

func TestPlugin_ReportFiltered(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	provideCache(injector, "cache")
	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*Cache](injector, "cache")

	filtered := p.ReportFiltered(auditlog.WithServicesByName("db"))

	if len(filtered.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(filtered.Services))
	}
}

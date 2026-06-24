package auditlog_test

import (
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestPlugin_ScopeTree(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()

	child := injector.Scope("child")

	provideDB(injector, "root-svc", "root")
	provideDB(child, "child-svc", "child")

	_ = do.MustInvokeNamed[*Database](injector, "root-svc")
	_ = do.MustInvokeNamed[*Database](child, "child-svc")

	report := p.Report()
	if report.ScopeTree.Name != auditlog.RootScopeName {
		t.Errorf("root scope name: want [root], got %s", report.ScopeTree.Name)
	}

	assertEqual(t, "child scope count", len(report.ScopeTree.Children), 1)

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

func TestPlugin_ScopeID(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()

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

func TestPlugin_ScopeTreeWithMultipleChildren(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()

	child1 := injector.Scope("child-1")
	child2 := injector.Scope("child-2")

	provideDB(injector, "root-svc", "root")
	provideDB(child1, "child1-svc", "child1")
	provideDB(child2, "child2-svc", "child2")

	_ = do.MustInvokeNamed[*Database](injector, "root-svc")
	_ = do.MustInvokeNamed[*Database](child1, "child1-svc")
	_ = do.MustInvokeNamed[*Database](child2, "child2-svc")

	report := p.Report()

	assertEqual(t, "child scope count", len(report.ScopeTree.Children), 2)
}

func TestPlugin_ResolveServiceScopeFromChildScope(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()

	provideDB(injector, "root-db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "root-db")

	child := injector.Scope("child")
	provideDB(child, "child-db", "test")
	_ = do.MustInvokeNamed[*Database](child, "child-db")

	report := p.Report()

	rootSvc := report.ServiceByRef(injector.ID(), "root-db")
	if rootSvc == nil {
		t.Fatal("root-db not found in root scope")
	}

	childSvc := report.ServiceByRef(child.ID(), "child-db")
	if childSvc == nil {
		t.Fatal("child-db not found in child scope")
	}

	results := p.RecordHealthCheck(child)
	if len(results) == 0 {
		t.Error("expected health check results from child scope")
	}

	report2 := p.Report()

	childDbReport := report2.ServiceByRef(child.ID(), "child-db")
	if childDbReport == nil {
		t.Fatal("child-db not found after health check")
	}

	if childDbReport.HealthCheckCount == 0 {
		t.Error("expected health check count > 0 for child-db")
	}
}

func TestReport_ResolveServiceScope_NotFound(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()

	do.ProvideValue(injector, &Database{URL: "test"})
	_ = do.MustInvoke[*Database](injector)

	report := p.Report()

	noEvents := report.EventsByRef("nonexistent-scope", "nonexistent")
	if len(noEvents) != 0 {
		t.Error("expected no events for nonexistent scope/service")
	}
}

func TestResolveServiceScope_ParentScopeService(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()

	provideDB(injector, "root-db", "root-dsn")
	_ = do.MustInvokeNamed[*Database](injector, "root-db")

	child := injector.Scope("child")

	scopeID, scopeName, found := p.Events()[0].ScopeID, p.Events()[0].ScopeName, true
	_ = scopeID
	_ = scopeName
	_ = found

	results := p.RecordHealthCheck(child)
	if len(results) == 0 {
		t.Error("expected health check results when resolving root service via child scope")
	}

	report := p.Report()

	rootSvc := report.ServiceByRef(injector.ID(), "root-db")
	if rootSvc == nil {
		t.Fatal("root-db should exist in report")
	}

	if rootSvc.HealthCheckCount == 0 {
		t.Error("root-db should have health check count from child scope resolution")
	}
}

func TestResolveServiceScope_GrandparentScopeService(t *testing.T) {
	t.Parallel()

	p, injector := newPluginAndInjector()

	provideDB(injector, "grandparent-db", "gp-dsn")
	_ = do.MustInvokeNamed[*Database](injector, "grandparent-db")

	child := injector.Scope("child")
	grandchild := child.Scope("grandchild")

	results := p.RecordHealthCheck(grandchild)
	if len(results) == 0 {
		t.Error("expected health check results when resolving grandparent service via grandchild scope")
	}

	report := p.Report()

	gpSvc := report.ServiceByRef(injector.ID(), "grandparent-db")
	if gpSvc == nil {
		t.Fatal("grandparent-db should exist in report")
	}

	if gpSvc.HealthCheckCount == 0 {
		t.Error("grandparent-db should have health check from grandchild scope resolution")
	}
}

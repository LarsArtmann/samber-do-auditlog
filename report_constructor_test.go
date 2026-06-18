package auditlog_test

import (
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestNewReport_ValidAndDerivesAggregates(t *testing.T) {
	t.Parallel()

	exported := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	services := []auditlog.ServiceInfo{
		{
			ServiceRef: auditlog.ServiceRef{
				ScopeID: "root", ScopeName: auditlog.RootScopeName, ServiceName: "db",
			},
			RegisteredAt:    exported,
			InvocationCount: 3,
		},
		{
			ServiceRef: auditlog.ServiceRef{
				ScopeID: "root", ScopeName: auditlog.RootScopeName, ServiceName: "cache",
			},
			RegisteredAt: exported,
		},
	}

	scopeTree := auditlog.ScopeNode{
		ID: "root", Name: auditlog.RootScopeName,
		Services: []string{"db", "cache"},
	}

	report, err := auditlog.NewReport(
		auditlog.SchemaVersion, "test-container", exported,
		nil, services, scopeTree,
	)
	if err != nil {
		t.Fatalf("NewReport: %v", err)
	}

	if err := report.Validate(); err != nil {
		t.Fatalf("constructed report invalid: %v", err)
	}

	if report.ServiceCount != 2 {
		t.Errorf("ServiceCount: want 2, got %d", report.ServiceCount)
	}

	if report.Version != auditlog.SchemaVersion {
		t.Errorf("Version: want %s, got %s", auditlog.SchemaVersion, report.Version)
	}
}

func TestNewReport_ReDerivesStatus(t *testing.T) {
	t.Parallel()

	exported := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Pass a deliberately wrong Status; NewReport must re-derive it.
	invokedAt := exported.Add(time.Second)

	services := []auditlog.ServiceInfo{
		{
			ServiceRef: auditlog.ServiceRef{
				ScopeID: "root", ScopeName: auditlog.RootScopeName, ServiceName: "db",
			},
			RegisteredAt:    exported,
			FirstInvokedAt:  &invokedAt,
			InvocationCount: 1,
			Status:          auditlog.ServiceStatusShutdown, // wrong — should become Active
		},
	}

	scopeTree := auditlog.ScopeNode{
		ID: "root", Name: auditlog.RootScopeName, Services: []string{"db"},
	}

	report, err := auditlog.NewReport(
		auditlog.SchemaVersion, "test", exported, nil, services, scopeTree,
	)
	if err != nil {
		t.Fatalf("NewReport: %v", err)
	}

	// Invoked once, no shutdown, no error → Active. And Validate() confirms
	// Status matches DeriveStatus (which is the whole point).
	want := auditlog.ServiceStatusActive
	if report.Services[0].Status != want {
		t.Errorf("Status: want %s, got %s", want, report.Services[0].Status)
	}
}

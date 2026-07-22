package auditlog_test

import (
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestNewReport_ValidAndDerivesAggregates(t *testing.T) {
	t.Parallel()

	exported := epochTime

	services := []auditlog.ServiceInfo{
		{
			ServiceIdentity: auditlog.ServiceIdentity{
				ServiceRef: rootRef("db"),
			},
			ServiceLifecycle: auditlog.ServiceLifecycle{
				RegisteredAt:    exported,
				InvocationCount: 3,
			},
		},
		{
			ServiceIdentity: auditlog.ServiceIdentity{
				ServiceRef: rootRef("cache"),
			},
			ServiceLifecycle: auditlog.ServiceLifecycle{
				RegisteredAt: exported,
			},
		},
	}

	report := mkNewReport(t, "test-container", exported, services, rootScopeTree("db", "cache"))

	assertReportValid(t, report, "constructed")

	if report.ServiceCount != 2 {
		t.Errorf("ServiceCount: want 2, got %d", report.ServiceCount)
	}

	assertVersion(t, report)
}

func TestNewReport_ReDerivesStatus(t *testing.T) {
	t.Parallel()

	exported := epochTime

	// Pass a deliberately wrong Status; NewReport must re-derive it.
	invokedAt := exported.Add(time.Second)

	services := []auditlog.ServiceInfo{
		{
			ServiceIdentity: auditlog.ServiceIdentity{
				ServiceRef: rootRef("db"),
			},
			ServiceLifecycle: auditlog.ServiceLifecycle{
				RegisteredAt:    exported,
				FirstInvokedAt:  &invokedAt,
				InvocationCount: 1,
				Status:          auditlog.ServiceStatusShutdown, // wrong — should become Active
			},
		},
	}

	report := mkNewReport(t, "test", exported, services, rootScopeTree("db"))

	// Invoked once, no shutdown, no error → Active. And Validate() confirms
	// Status matches DeriveStatus (which is the whole point).
	want := auditlog.ServiceStatusActive
	if report.Services[0].Status != want {
		t.Errorf("Status: want %s, got %s", want, report.Services[0].Status)
	}
}

// rootRef is a 1-line ServiceRef constructor for the root scope ("root" /
// auditlog.RootScopeName). Used by every test that builds a fixture by hand.
func rootRef(serviceName auditlog.ServiceName) auditlog.ServiceRef {
	return auditlog.ServiceRef{
		ScopeID:     "root",
		ScopeName:   auditlog.RootScopeName,
		ServiceName: serviceName,
	}
}

// rootScopeTree is a 1-line ScopeNode constructor for the root scope with
// the given service names. Shared by every NewReport fixture test.
func rootScopeTree(services ...auditlog.ServiceName) auditlog.ScopeNode {
	return auditlog.ScopeNode{
		ID:       "root",
		Name:     auditlog.RootScopeName,
		Services: services,
	}
}

// mkNewReport constructs a Report via auditlog.NewReport and fails the test
// on any error. Centralizes the 7-line call used by every NewReport fixture.
func mkNewReport(
	t *testing.T,
	containerID auditlog.ContainerID,
	exported time.Time,
	services []auditlog.ServiceInfo,
	scopeTree auditlog.ScopeNode,
) auditlog.Report {
	t.Helper()

	report, err := auditlog.NewReport(
		auditlog.SchemaVersion, containerID, exported,
		nil, services, scopeTree,
	)
	if err != nil {
		t.Fatalf("NewReport: %v", err)
	}

	return report
}

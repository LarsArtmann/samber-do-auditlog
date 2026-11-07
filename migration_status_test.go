package auditlog_test

import (
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestMigrateReport_StatusComputation(t *testing.T) {
	t.Parallel()

	now := time.Now().Format(time.RFC3339)

	for _, tc := range []struct {
		name    string
		svcJSON string
		status  auditlog.ServiceStatus
	}{
		{
			"registered",
			`{"service_name":"svc","scope_id":"r","scope_name":"[root]","registered_at":"` + now + `"}`,
			auditlog.ServiceStatusRegistered,
		},
		{
			"active",
			`{"service_name":"svc","scope_id":"r","scope_name":"[root]","registered_at":"` + now + `","first_invoked_at":"` + now + `"}`,
			auditlog.ServiceStatusActive,
		},
		{
			"invocation_error",
			`{"service_name":"svc","scope_id":"r","scope_name":"[root]","registered_at":"` + now + `","invocation_error":"fail"}`,
			auditlog.ServiceStatusInvocationError,
		},
		{
			"shutdown",
			`{"service_name":"svc","scope_id":"r","scope_name":"[root]","registered_at":"` + now + `","shutdown_at":"` + now + `"}`,
			auditlog.ServiceStatusShutdown,
		},
		{
			"shutdown_error",
			`{"service_name":"svc","scope_id":"r","scope_name":"[root]","registered_at":"` + now + `","shutdown_error":"leak"}`,
			auditlog.ServiceStatusShutdownError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			input := `{"version":"0.1.0","services":[` + tc.svcJSON + `]}`

			report, err := auditlog.MigrateReport([]byte(input))
			if err != nil {
				t.Fatalf("MigrateReport: %v", err)
			}

			assertReportServiceCount(t, report)

			if report.Services[0].Status != tc.status {
				t.Errorf("status: want %s, got %s", tc.status, report.Services[0].Status)
			}
		})
	}
}

func TestMigrateReport_RecomputesStaleStatus(t *testing.T) {
	t.Parallel()

	// Input has status:"active" but no first_invoked_at — the correct derived
	// status is "registered". MigrateReport must re-derive rather than trust
	// the stale stored value.
	input := `{
		"version": "0.1.0",
		"services": [
			{"service_name":"svc","scope_id":"r","scope_name":"[root]","registered_at":"2026-01-01T00:00:00Z","status":"active"}
		]
	}`

	report, err := auditlog.MigrateReport([]byte(input))
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	if report.Services[0].Status != auditlog.ServiceStatusRegistered {
		t.Errorf("stale status should be recomputed to %s, got %s",
			auditlog.ServiceStatusRegistered, report.Services[0].Status)
	}

	assertReportValidNoFatal(t, report, "migrated")
}

func TestMigrateReport_EmptyScopeTree(t *testing.T) {
	t.Parallel()

	input := `{"version":"0.1.0","scope_tree":{"id":"","name":"","services":null}}`

	report, err := auditlog.MigrateReport([]byte(input))
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	assertEqual(t, "scope_count", report.ScopeCount, 0)
}

func TestMigrateReport_AlreadyCurrentVersion(t *testing.T) {
	t.Parallel()

	input := `{"version":"0.3.0","container_id":"test","exported_at":"2026-01-01T00:00:00Z","event_count":5,"service_count":2}`

	report, err := auditlog.MigrateReport([]byte(input))
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	if report.Version != "0.3.0" {
		t.Errorf("version should remain 0.3.0, got %s", report.Version)
	}

	assertContainerID(t, report, "test")
}

func TestMigrateReport_PreservesExportedAt(t *testing.T) {
	t.Parallel()

	originalTime := "2026-01-15T12:30:00Z"
	input := `{"version":"0.1.0","exported_at":"` + originalTime + `"}`

	report, err := auditlog.MigrateReport([]byte(input))
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	if report.ExportedAt.Format("2006-01-02") != "2026-01-15" {
		t.Errorf("ExportedAt should be preserved from original, got %v", report.ExportedAt)
	}
}

// assertReportFieldsEqual compares the recomputed denormalized fields of two
// reports, used by the migration round-trip test.

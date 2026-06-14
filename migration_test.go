package auditlog_test

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestMigrateReport_FromV01(t *testing.T) {
	v01JSON := `{
		"version": "0.1.0",
		"container_id": "test-app",
		"exported_at": "2026-01-01T00:00:00Z",
		"service_count": 2,
		"event_count": 4,
		"services": [
			{
				"service_name": "*main.DB",
				"scope_id": "root-1",
				"scope_name": "[root]",
				"registered_at": "2026-01-01T00:00:01Z",
				"invocation_count": 1,
				"invocation_order": 1,
				"first_build_duration_ms": 5.2,
				"dependencies": [],
				"dependents": [{"scope_id":"root-1","scope_name":"[root]","service_name":"*main.Server"}]
			},
			{
				"service_name": "*main.Server",
				"scope_id": "root-1",
				"scope_name": "[root]",
				"registered_at": "2026-01-01T00:00:02Z",
				"invocation_count": 1,
				"invocation_order": 2,
				"first_build_duration_ms": 12.5,
				"dependencies": [{"scope_id":"root-1","scope_name":"[root]","service_name":"*main.DB"}],
				"dependents": []
			}
		],
		"events": [
			{"sequence":1,"timestamp":"2026-01-01T00:00:01Z","event_type":"registration","phase":"before","container_id":"test-app","scope_id":"root-1","scope_name":"[root]","service_name":"*main.DB"},
			{"sequence":2,"timestamp":"2026-01-01T00:00:01Z","event_type":"registration","phase":"after","container_id":"test-app","scope_id":"root-1","scope_name":"[root]","service_name":"*main.DB"},
			{"sequence":3,"timestamp":"2026-01-01T00:00:02Z","event_type":"invocation","phase":"before","container_id":"test-app","scope_id":"root-1","scope_name":"[root]","service_name":"*main.DB"},
			{"sequence":4,"timestamp":"2026-01-01T00:00:02Z","event_type":"invocation","phase":"after","container_id":"test-app","scope_id":"root-1","scope_name":"[root]","duration_ms":5.2,"service_name":"*main.DB"}
		],
		"scope_tree": {
			"id": "root-1",
			"name": "[root]",
			"services": ["*main.DB", "*main.Server"],
			"children": []
		}
	}`

	report, err := auditlog.MigrateReport([]byte(v01JSON))
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	assertVersion(t, report)

	assertContainerID(t, report, "test-app")

	assertServiceCount(t, report, 2)

	assertEventCount(t, report, 4)

	assertIntField(t, "scope_count", report.ScopeCount, 1)

	if math.Abs(report.TotalBuildDurationMs-17.7) > 1e-9 {
		t.Errorf("total_build_duration_ms: want 17.7, got %f", report.TotalBuildDurationMs)
	}

	if !report.ShutdownSucceeded {
		t.Error("shutdown_succeeded: want true (no shutdown errors)")
	}

	if report.HealthCheckSucceeded {
		t.Error("health_check_succeeded: want false (no health checks ran)")
	}

	for _, svc := range report.Services {
		if svc.Status == "" {
			t.Errorf("service %s has empty status", svc.ServiceName)
		}
	}
}

func TestMigrateReport_InvalidJSON(t *testing.T) {
	_, err := auditlog.MigrateReport([]byte("not json"))
	assertErrorExpected(t, err, "for invalid JSON")
}

func TestMigrateReport_EmptyReport(t *testing.T) {
	report, err := auditlog.MigrateReport([]byte(`{"version":"0.1.0"}`))
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	assertVersion(t, report)

	assertServiceCount(t, report, 0)
}

func TestMigrateReport_RoundTrip(t *testing.T) {
	plugin := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(plugin.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	original := plugin.Report()

	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")

	err := enc.Encode(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	migrated, err := auditlog.MigrateReport(buf.Bytes())
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	if migrated.Version != auditlog.SchemaVersion {
		t.Errorf("version: want %s, got %s", auditlog.SchemaVersion, migrated.Version)
	}

	if migrated.ServiceCount != original.ServiceCount {
		t.Errorf("service_count: want %d, got %d", original.ServiceCount, migrated.ServiceCount)
	}

	if migrated.EventCount != original.EventCount {
		t.Errorf("event_count: want %d, got %d", original.EventCount, migrated.EventCount)
	}
}

func TestMigrateReport_NestedScopes(t *testing.T) {
	v01JSON := `{
		"version": "0.1.0",
		"container_id": "test",
		"scope_tree": {
			"id": "root",
			"name": "[root]",
			"services": [],
			"children": [
				{"id":"child1","name":"child1","services":[],"children":[]},
				{"id":"child2","name":"child2","services":[],"children":[
					{"id":"grandchild","name":"grandchild","services":[],"children":[]}
				]}
			]
		}
	}`

	report, err := auditlog.MigrateReport([]byte(v01JSON))
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	assertIntField(t, "scope_count", report.ScopeCount, 4)
}

func TestMigrateReport_StatusComputation(t *testing.T) {
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

func TestMigrateReport_PreservesExistingStatus(t *testing.T) {
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

	if report.Services[0].Status != auditlog.ServiceStatusActive {
		t.Errorf("existing status should be preserved, got %s", report.Services[0].Status)
	}
}

func TestMigrateReport_EmptyScopeTree(t *testing.T) {
	input := `{"version":"0.1.0","scope_tree":{"id":"","name":"","services":null}}`

	report, err := auditlog.MigrateReport([]byte(input))
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	assertIntField(t, "scope_count", report.ScopeCount, 1)
}

func TestMigrateReport_AlreadyCurrentVersion(t *testing.T) {
	input := `{"version":"0.2.0","container_id":"test","exported_at":"2026-01-01T00:00:00Z","event_count":5,"service_count":2}`

	report, err := auditlog.MigrateReport([]byte(input))
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	if report.Version != "0.2.0" {
		t.Errorf("version should remain 0.2.0, got %s", report.Version)
	}

	assertContainerID(t, report, "test")
}

func TestMigrateReport_PreservesExportedAt(t *testing.T) {
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

func TestMigrateReport_EmptyInput(t *testing.T) {
	_, err := auditlog.MigrateReport([]byte{})
	if err == nil {
		t.Error("expected error for empty input")
	}

	_, err = auditlog.MigrateReport(nil)
	if err == nil {
		t.Error("expected error for nil input")
	}
}

func TestMigrateReport_MissingVersion(t *testing.T) {
	_, err := auditlog.MigrateReport([]byte(`{"container_id":"test"}`))
	assertErrorExpected(t, err, "for input without version field")
}

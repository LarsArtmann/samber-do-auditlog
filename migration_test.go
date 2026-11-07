package auditlog_test

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestMigrateReport_FromV01(t *testing.T) {
	t.Parallel()

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

	assertEqual(t, "scope_count", report.ScopeCount, 1)

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

func TestMigrateReport_RejectsInvalidInput(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		input  []byte
		reason string
	}{
		{"invalid_json", []byte("not json"), "for invalid JSON"},
		{"empty_input", []byte{}, "for empty input"},
		{"nil_input", nil, "for nil input"},
		{"missing_version", []byte(`{"container_id":"test"}`), "for input without version field"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := auditlog.MigrateReport(tc.input)
			assertErrorExpected(t, err, tc.reason)
		})
	}
}

func TestMigrateReport_EmptyReport(t *testing.T) {
	t.Parallel()

	report, err := auditlog.MigrateReport([]byte(`{"version":"0.1.0"}`))
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	assertVersion(t, report)

	assertServiceCount(t, report, 0)

	assertReportValid(t, report, "empty migrated")
}

func TestMigrateReport_RoundTrip(t *testing.T) {
	t.Parallel()

	original := setupWithDBReport()

	var buf bytes.Buffer

	enc := jsontext.NewEncoder(&buf, jsontext.WithIndent("  "))

	err := json.MarshalEncode(enc, original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	migrated, err := auditlog.MigrateReport(buf.Bytes())
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	assertVersion(t, migrated)

	if migrated.ServiceCount != original.ServiceCount {
		t.Errorf("service_count: want %d, got %d", original.ServiceCount, migrated.ServiceCount)
	}

	if migrated.EventCount != original.EventCount {
		t.Errorf("event_count: want %d, got %d", original.EventCount, migrated.EventCount)
	}
}

func TestMigrateReport_NestedScopes(t *testing.T) {
	t.Parallel()

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

	assertEqual(t, "scope_count", report.ScopeCount, 4)
}

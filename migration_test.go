package auditlog_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
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
func assertReportFieldsEqual(t *testing.T, want, got auditlog.Report) {
	t.Helper()

	if got.Version != auditlog.SchemaVersion {
		t.Errorf("version: want %s, got %s", auditlog.SchemaVersion, got.Version)
	}

	if got.ContainerID != want.ContainerID {
		t.Errorf("container_id: want %s, got %s", want.ContainerID, got.ContainerID)
	}

	if !got.ExportedAt.Equal(want.ExportedAt) {
		t.Errorf("exported_at: want %v, got %v", want.ExportedAt, got.ExportedAt)
	}

	if got.EventCount != want.EventCount {
		t.Errorf("event_count: want %d, got %d", want.EventCount, got.EventCount)
	}

	if got.ServiceCount != want.ServiceCount {
		t.Errorf("service_count: want %d, got %d", want.ServiceCount, got.ServiceCount)
	}

	if got.ScopeCount != want.ScopeCount {
		t.Errorf("scope_count: want %d, got %d", want.ScopeCount, got.ScopeCount)
	}

	if got.TotalBuildDurationMs != want.TotalBuildDurationMs {
		t.Errorf("total_build_duration_ms: want %f, got %f", want.TotalBuildDurationMs, got.TotalBuildDurationMs)
	}

	if got.TotalShutdownDurationMs != want.TotalShutdownDurationMs {
		t.Errorf(
			"total_shutdown_duration_ms: want %f, got %f",
			want.TotalShutdownDurationMs,
			got.TotalShutdownDurationMs,
		)
	}

	if got.ShutdownSucceeded != want.ShutdownSucceeded {
		t.Errorf("shutdown_succeeded: want %v, got %v", want.ShutdownSucceeded, got.ShutdownSucceeded)
	}

	if got.HealthCheckSucceeded != want.HealthCheckSucceeded {
		t.Errorf("health_check_succeeded: want %v, got %v", want.HealthCheckSucceeded, got.HealthCheckSucceeded)
	}

	if got.HealthCheckedCount != want.HealthCheckedCount {
		t.Errorf("health_checked_count: want %d, got %d", want.HealthCheckedCount, got.HealthCheckedCount)
	}
}

// TestMigrateReport_FullRoundTrip builds a rich report at the current schema,
// manually downgrades the JSON to v0.1.0 (stripping v0.2.0 fields), migrates
// it back, and asserts that all recomputed denormalized fields and data match
// the original.
func TestMigrateReport_FullRoundTrip(t *testing.T) {
	t.Parallel()

	// Build a report with services (incl. invocation error + crashing shutdown),
	// events, and a scope tree.
	plugin := mustNew(auditlog.Config{Enabled: true, ContainerID: "round-trip-app"})
	injector := do.NewWithOpts(plugin.Opts())

	provideDB(injector, "db", "postgres://test")
	provideFailing(injector, "failing")
	provideCrashing(injector, "crashing")

	_ = do.MustInvokeNamed[*Database](injector, "db")

	_, invokeErr := do.InvokeNamed[*Database](injector, "failing")
	if invokeErr == nil {
		t.Fatal("expected failing service to error on invoke")
	}

	_ = do.MustInvokeNamed[*CrashingService](injector, "crashing")

	_ = injector.Shutdown()

	original := plugin.Report()

	// Marshal the original to JSON.
	origJSON, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal original: %v", err)
	}

	// --- Downgrade to v0.1.0 ---
	downgradedJSON, err := downgradeToV010(origJSON)
	if err != nil {
		t.Fatalf("downgrade: %v", err)
	}

	// --- Migrate back to current schema ---
	migrated, err := auditlog.MigrateReport(downgradedJSON)
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	// --- Assert recomputed fields match ---
	assertReportFieldsEqual(t, original, migrated)

	// Service data preserved: same names.
	if len(migrated.Services) != len(original.Services) {
		t.Fatalf("services length: want %d, got %d", len(original.Services), len(migrated.Services))
	}

	origNames := make(map[string]bool)

	for _, svc := range original.Services {
		origNames[svc.ServiceName] = true
	}

	for _, svc := range migrated.Services {
		if !origNames[svc.ServiceName] {
			t.Errorf("migrated has unexpected service %q", svc.ServiceName)
		}
	}

	// Validate the migrated report is internally consistent.
	if err := migrated.Validate(); err != nil {
		t.Errorf("migrated report failed Validate: %v", err)
	}
}

// downgradeToV010 takes a v0.2.0 report JSON and strips v0.2.0-only fields to
// simulate a v0.1.0 export.
func downgradeToV010(origJSON []byte) ([]byte, error) {
	var raw map[string]any

	if err := json.Unmarshal(origJSON, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal to map: %w", err)
	}

	raw["version"] = "0.1.0"

	for _, key := range []string{
		"scope_count", "total_build_duration_ms", "total_shutdown_duration_ms",
		"shutdown_succeeded", "health_check_succeeded", "health_checked_count",
		"dropped_event_count",
	} {
		delete(raw, key)
	}

	if services, ok := raw["services"].([]any); ok {
		for _, svc := range services {
			if svcMap, ok := svc.(map[string]any); ok {
				for _, key := range []string{
					"service_type", "status", "is_healthchecker", "is_shutdowner",
					"health_check_count", "health_check_error",
				} {
					delete(svcMap, key)
				}
			}
		}
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal downgraded: %w", err)
	}

	return data, nil
}

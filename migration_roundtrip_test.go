package auditlog_test

import (
	"encoding/json/v2"
	"fmt"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

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
		origNames[string(svc.ServiceName)] = true
	}

	for _, svc := range migrated.Services {
		if !origNames[string(svc.ServiceName)] {
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

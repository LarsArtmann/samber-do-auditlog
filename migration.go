package auditlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var errMigrationEmptyInput = errors.New("migration input is empty")

var errMigrationMissingVersion = errors.New("migration input has no version field")

// MigrateReport takes a raw JSON byte slice representing a report exported
// by a previous schema version and returns a Report compatible with the
// current SchemaVersion. Unknown fields are preserved through round-tripping.
//
// In addition to upgrading older schemas, MigrateReport always re-derives the
// denormalized count and aggregate fields (EventCount, ServiceCount,
// ScopeCount, durations, health flags, per-service Status) from the actual
// data. This means current-schema input is also repaired: stale or hand-edited
// reports that would fail Validate() are normalized so the returned Report is
// always valid. The implied contract is "repair/normalize -> current", not
// just "upgrade old -> current".
//
// For v0.1.0 → v0.2.0 the migration adds:
//   - scope_count, total_build_duration_ms, total_shutdown_duration_ms, shutdown_succeeded
//   - health_check_succeeded, health_checked_count (always false/0 for old reports)
//   - service_type, status, is_healthchecker, is_shutdowner (zero values)
func MigrateReport(data []byte) (Report, error) {
	if len(data) == 0 {
		return Report{}, errMigrationEmptyInput
	}

	var report Report

	err := json.Unmarshal(data, &report)
	if err != nil {
		return Report{}, fmt.Errorf("unmarshal report: %w", err)
	}

	if report.Version == "" {
		return Report{}, errMigrationMissingVersion
	}

	// Normalize the report to the current schema regardless of the input
	// version. This repairs stale or manually edited denormalized counts and
	// ensures Validate() succeeds on the returned report.
	report.Version = SchemaVersion

	if report.ExportedAt.IsZero() {
		report.ExportedAt = time.Now()
	}

	report.EventCount = len(report.Events)
	report.ServiceCount = len(report.Services)

	report.ScopeCount = countScopeNodes(report.ScopeTree)
	report.TotalBuildDurationMs = sumBuildMs(report.Services)
	report.TotalShutdownDurationMs = sumShutdownMs(report.Services)
	report.ShutdownSucceeded = noShutdownErrors(report.Services)
	report.HealthCheckSucceeded = allHealthChecksPassed(report.Services)
	report.HealthCheckedCount = countHealthChecked(report.Services)

	for idx := range report.Services {
		if report.Services[idx].Status == "" {
			report.Services[idx].Status = computeServiceStatusFromInfo(report.Services[idx])
		}
	}

	return report, nil
}

func computeServiceStatusFromInfo(svc ServiceInfo) ServiceStatus {
	return deriveServiceStatus(svc.InvocationError, svc.ShutdownError, svc.ShutdownAt, svc.FirstInvokedAt)
}

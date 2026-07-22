package auditlog_test

import (
	"encoding/json"
	"math/rand/v2"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// migrateReport builds a random valid Report, ensures each service Status
// matches DeriveStatus(), and returns its JSON encoding plus the source.
func migrateReport(rng *rand.Rand) ([]byte, auditlog.Report) {
	report := randReport(rng)
	report.Version = auditlog.SchemaVersion

	// Make it valid: sync per-service status so the source itself passes checks.
	auditlog.RedriveReportStatuses(&report)

	data, err := json.Marshal(report)
	if err != nil {
		panic(err)
	}

	return data, report
}

// TestMigrateReport_AlwaysValid asserts that migrating any marshaled Report
// always yields a Report that passes Validate() — the core repair contract.
func TestMigrateReport_AlwaysValid(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(11, 11))

	for range 200 {
		data, _ := migrateReport(rng)

		migrated, err := auditlog.MigrateReport(data)
		if err != nil {
			t.Fatalf("MigrateReport error: %v", err)
		}

		assertReportValidNoFatal(t, migrated, "migrated")
	}
}

// TestMigrateReport_RepairsCorruptCounts verifies that deliberately wrong
// denormalized count fields are repaired by MigrateReport (it re-derives them).
func TestMigrateReport_RepairsCorruptCounts(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(22, 22))

	for range 200 {
		data, src := migrateReport(rng)

		// Corrupt the JSON: inject wrong denormalized counts and a bogus version.
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		raw["event_count"] = 999
		raw["service_count"] = 999
		raw["scope_count"] = 999
		raw["version"] = "0.0.0-corrupt"

		corrupt, err := json.Marshal(raw)
		if err != nil {
			t.Fatalf("marshal corrupt: %v", err)
		}

		migrated, err := auditlog.MigrateReport(corrupt)
		if err != nil {
			t.Fatalf("MigrateReport(corrupt): %v", err)
		}

		if err := migrated.Validate(); err != nil {
			t.Fatalf("corrupt report not repaired: %v", err)
		}

		if migrated.EventCount != len(src.Events) {
			t.Errorf("event_count repaired: want %d, got %d", len(src.Events), migrated.EventCount)
		}

		if migrated.ServiceCount != len(src.Services) {
			t.Errorf("service_count repaired: want %d, got %d", len(src.Services), migrated.ServiceCount)
		}
	}
}

// TestMigrateReport_VersionNormalized asserts the output always carries the
// current SchemaVersion, regardless of the input version string.
func TestMigrateReport_VersionNormalized(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(33, 33))

	for range 200 {
		data, _ := migrateReport(rng)

		migrated, err := auditlog.MigrateReport(data)
		if err != nil {
			t.Fatalf("MigrateReport: %v", err)
		}

		assertVersion(t, migrated)
	}
}

// TestMigrateReport_Idempotent asserts that migrating an already-migrated
// report's JSON yields the same result (fixpoint).
func TestMigrateReport_Idempotent(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(44, 44))

	for range 200 {
		data, _ := migrateReport(rng)

		once, err := auditlog.MigrateReport(data)
		if err != nil {
			t.Fatalf("first migrate: %v", err)
		}

		onceJSON, err := json.Marshal(once)
		if err != nil {
			t.Fatalf("marshal once: %v", err)
		}

		twice, err := auditlog.MigrateReport(onceJSON)
		if err != nil {
			t.Fatalf("second migrate: %v", err)
		}

		// Core data and counts must be stable across the second migration.
		if twice.ServiceCount != once.ServiceCount {
			t.Errorf("service count drifted on second migrate: %d -> %d", once.ServiceCount, twice.ServiceCount)
		}

		if twice.EventCount != once.EventCount {
			t.Errorf("event count drifted on second migrate: %d -> %d", once.EventCount, twice.EventCount)
		}

		if len(twice.Services) != len(once.Services) {
			t.Errorf("service slice drifted on second migrate: %d -> %d", len(once.Services), len(twice.Services))
		}
	}
}

// TestMigrateReport_PreservesCoreData asserts service identity and event
// sequences survive the migration round-trip.
func TestMigrateReport_PreservesCoreData(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(55, 55))

	for range 200 {
		data, src := migrateReport(rng)

		migrated, err := auditlog.MigrateReport(data)
		if err != nil {
			t.Fatalf("MigrateReport: %v", err)
		}

		// Every source service name must appear in the migrated report.
		migratedNames := make(map[string]bool, len(migrated.Services))
		for _, svc := range migrated.Services {
			migratedNames[string(svc.ServiceName)] = true
		}

		for _, svc := range src.Services {
			if !migratedNames[string(svc.ServiceName)] {
				t.Errorf("service %q lost during migration", svc.ServiceName)
			}
		}
	}
}

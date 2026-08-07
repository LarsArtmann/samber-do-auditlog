package auditlog_test

import (
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestDiff_TimingDeltas(t *testing.T) {
	t.Parallel()

	now := time.Now()

	base := auditlog.Report{
		Version:                 auditlog.SchemaVersion,
		ContainerID:             "test",
		ExportedAt:              now,
		TotalBuildDurationMs:    100.0,
		TotalShutdownDurationMs: 50.0,
	}

	changed := auditlog.Report{
		Version:                 auditlog.SchemaVersion,
		ContainerID:             "test",
		ExportedAt:              now,
		TotalBuildDurationMs:    150.0,
		TotalShutdownDurationMs: 30.0,
	}

	diff := base.Diff(changed)

	if diff.TotalBuildDurationMsDelta != 50.0 {
		t.Errorf("TotalBuildDurationMsDelta: want 50.0, got %.2f", diff.TotalBuildDurationMsDelta)
	}

	if diff.TotalShutdownDurationMsDelta != -20.0 {
		t.Errorf("TotalShutdownDurationMsDelta: want -20.0, got %.2f", diff.TotalShutdownDurationMsDelta)
	}

	if !diff.HasChanges() {
		t.Error("expected HasChanges=true when timing deltas exist")
	}
}

func TestDiff_TimingDeltasZeroForIdenticalReports(t *testing.T) {
	t.Parallel()

	now := time.Now()

	report := auditlog.Report{
		Version:                 auditlog.SchemaVersion,
		ContainerID:             "test",
		ExportedAt:              now,
		TotalBuildDurationMs:    100.0,
		TotalShutdownDurationMs: 50.0,
		EventCount:              0,
	}

	diff := report.Diff(report)

	if diff.TotalBuildDurationMsDelta != 0 {
		t.Errorf("TotalBuildDurationMsDelta: want 0, got %.2f", diff.TotalBuildDurationMsDelta)
	}

	if diff.TotalShutdownDurationMsDelta != 0 {
		t.Errorf("TotalShutdownDurationMsDelta: want 0, got %.2f", diff.TotalShutdownDurationMsDelta)
	}

	if !diff.IsEmpty() {
		t.Error("expected IsEmpty=true for identical reports with timing fields")
	}
}

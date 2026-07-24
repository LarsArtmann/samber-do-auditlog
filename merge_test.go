package auditlog_test

import (
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestMergeReports_TwoReports(t *testing.T) {
	t.Parallel()

	base := epochTime

	makeReport := func(containerID string, seq int, svcName string) auditlog.Report {
		events := []auditlog.Event{
			mkRegEvent(seq, base, auditlog.ServiceName(svcName), auditlog.ContainerID(containerID)),
		}

		report, err := auditlog.ReplayEvents(events)
		if err != nil {
			t.Fatalf("ReplayEvents: %v", err)
		}

		report.ExportedAt = base

		return report
	}

	reports := []auditlog.Report{
		makeReport("container-a", 1, "svc-a"),
		makeReport("container-b", 1, "svc-b"),
	}

	merged, err := auditlog.MergeReports(reports)
	if err != nil {
		t.Fatalf("MergeReports: %v", err)
	}

	assertContainerID(t, merged, "merged")

	if merged.ServiceCount != 2 {
		t.Errorf("service_count: want 2, got %d", merged.ServiceCount)
	}

	if merged.EventCount != 2 {
		t.Errorf("event_count: want 2, got %d", merged.EventCount)
	}

	if err := merged.Validate(); err != nil {
		t.Errorf("merged report invalid: %v", err)
	}
}

func TestMergeReports_SingleReport(t *testing.T) {
	t.Parallel()

	base := epochTime

	events := []auditlog.Event{
		mkRegEvent(1, base, "svc", "single"),
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	report.ExportedAt = base

	merged, err := auditlog.MergeReports([]auditlog.Report{report})
	if err != nil {
		t.Fatalf("MergeReports: %v", err)
	}

	assertContainerID(t, merged, "single")
}

func TestMergeReports_Empty(t *testing.T) {
	t.Parallel()

	_, err := auditlog.MergeReports(nil)
	if err == nil {
		t.Fatal("expected error for empty reports slice")
	}
}

func TestValidate_EmptyVersion(t *testing.T) {
	t.Parallel()

	report := auditlog.Report{
		EventCount:   0,
		ServiceCount: 0,
		ScopeCount:   0,
	}

	err := report.Validate()
	if err == nil {
		t.Fatal("expected error for empty version, got nil")
	}
}

func TestMergeReports_WithChildScopes(t *testing.T) {
	t.Parallel()

	base := epochTime

	makeScopedReport := func(containerID string, svcName auditlog.ServiceName) auditlog.Report {
		events := []auditlog.Event{
			mkRegEvent(1, base, svcName, auditlog.ContainerID(containerID)),
		}

		report, err := auditlog.ReplayEvents(events)
		if err != nil {
			t.Fatalf("ReplayEvents: %v", err)
		}

		report.ExportedAt = base

		return report
	}

	reports := []auditlog.Report{
		makeScopedReport("container-a", "svc-a"),
		makeScopedReport("container-b", "svc-b"),
	}

	merged, err := auditlog.MergeReports(reports)
	if err != nil {
		t.Fatalf("MergeReports: %v", err)
	}

	if err := merged.Validate(); err != nil {
		t.Errorf("merged report invalid: %v", err)
	}

	if merged.ScopeCount < 1 {
		t.Errorf("expected at least 1 scope, got %d", merged.ScopeCount)
	}
}

func TestMergeReports_EarlierExportedAtPreserved(t *testing.T) {
	t.Parallel()

	base := epochTime
	earlier := base.Add(-time.Hour)

	r1 := mkNewReport(t, "c1", base, []auditlog.ServiceInfo{
		{
			ServiceIdentity: auditlog.ServiceIdentity{ServiceRef: rootRef("svc-a")},
			ServiceLifecycle: auditlog.ServiceLifecycle{
				RegisteredAt: base,
			},
		},
	}, rootScopeTree("svc-a"))
	r1.ExportedAt = base

	r2 := mkNewReport(t, "c2", earlier, []auditlog.ServiceInfo{
		{
			ServiceIdentity: auditlog.ServiceIdentity{ServiceRef: rootRef("svc-b")},
			ServiceLifecycle: auditlog.ServiceLifecycle{
				RegisteredAt: earlier,
			},
		},
	}, rootScopeTree("svc-b"))
	r2.ExportedAt = earlier

	merged, err := auditlog.MergeReports([]auditlog.Report{r1, r2})
	if err != nil {
		t.Fatalf("MergeReports: %v", err)
	}

	if !merged.ExportedAt.Equal(base) {
		t.Errorf("ExportedAt: want %v (latest), got %v", base, merged.ExportedAt)
	}
}

func TestFormat_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		format auditlog.Format
		want   string
	}{
		{auditlog.FormatAuto, "auto"},
		{auditlog.FormatJSON, "json"},
		{auditlog.FormatNDJSON, "ndjson"},
		{auditlog.Format(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.format.String(); got != tt.want {
			t.Errorf("Format(%d).String() = %q, want %q", tt.format, got, tt.want)
		}
	}
}

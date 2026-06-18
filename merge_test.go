package auditlog_test

import (
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestMergeReports_TwoReports(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	makeReport := func(containerID string, seq int, svcName string) auditlog.Report {
		events := []auditlog.Event{
			{
				ServiceRef: auditlog.ServiceRef{
					ScopeID: "root", ScopeName: auditlog.RootScopeName, ServiceName: svcName,
				},
				Sequence: seq, Timestamp: base,
				EventType: auditlog.EventTypeRegistration, Phase: auditlog.PhaseAfter,
				ContainerID: containerID, ServiceType: auditlog.ProviderTypeLazy,
			},
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

	if merged.ContainerID != "merged" {
		t.Errorf("container_id: want merged, got %s", merged.ContainerID)
	}

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

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	events := []auditlog.Event{
		{
			ServiceRef: auditlog.ServiceRef{
				ScopeID: "root", ScopeName: auditlog.RootScopeName, ServiceName: "svc",
			},
			Sequence: 1, Timestamp: base,
			EventType: auditlog.EventTypeRegistration, Phase: auditlog.PhaseAfter,
			ContainerID: "single", ServiceType: auditlog.ProviderTypeLazy,
		},
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

	if merged.ContainerID != "single" {
		t.Errorf("single report should pass through: got %s", merged.ContainerID)
	}
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

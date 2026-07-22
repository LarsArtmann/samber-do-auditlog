package auditlog_test

import (
	"testing"

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

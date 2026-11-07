package live

import (
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestBuildStatsEntries(t *testing.T) {
	t.Parallel()

	t.Run("without_health_checks", func(t *testing.T) {
		t.Parallel()

		report := auditlog.Report{
			ServiceCount:            3,
			EventCount:              10,
			ScopeCount:              2,
			TotalBuildDurationMs:    150,
			TotalShutdownDurationMs: 30,
		}

		stats := buildStatsEntries(report, 1)
		if len(stats) != 6 {
			t.Fatalf("expected 6 stats, got %d", len(stats))
		}

		if stats[3].label != "Errors" || stats[3].cls != "error" {
			t.Errorf("Errors stat = %+v, want cls=error", stats[3])
		}
	})

	t.Run("with_health_checks_passed", func(t *testing.T) {
		t.Parallel()

		report := auditlog.Report{
			HealthCheckedCount:   2,
			HealthCheckSucceeded: true,
		}

		stats := buildStatsEntries(report, 0)
		if len(stats) != 7 {
			t.Fatalf("expected 7 stats with health, got %d", len(stats))
		}

		if stats[6].label != "Health" || stats[6].cls != "success" {
			t.Errorf("Health stat = %+v, want cls=success", stats[6])
		}
	})

	t.Run("with_health_checks_failed", func(t *testing.T) {
		t.Parallel()

		report := auditlog.Report{
			HealthCheckedCount:   2,
			HealthCheckSucceeded: false,
		}

		stats := buildStatsEntries(report, 0)
		if len(stats) != 7 {
			t.Fatalf("expected 7 stats with health, got %d", len(stats))
		}

		if stats[6].cls != "error" {
			t.Errorf("Health stat cls = %q, want error", stats[6].cls)
		}
	})
}

func TestProviderIcon(t *testing.T) {
	t.Parallel()

	meta := auditlog.BuildTypeMetadata()

	if icon := providerIcon(meta, "lazy"); icon == "" {
		t.Errorf(`providerIcon(meta, "lazy") returned empty, expected an icon`)
	}

	if icon := providerIcon(meta, "nonexistent"); icon != "" {
		t.Errorf(`providerIcon(meta, "nonexistent") = %q, expected ""`, icon)
	}
}

func TestStatusIcon(t *testing.T) {
	t.Parallel()

	meta := auditlog.BuildTypeMetadata()

	if icon := statusIcon(meta, "active"); icon == "" {
		t.Errorf(`statusIcon(meta, "active") returned empty, expected an icon`)
	}

	if icon := statusIcon(meta, "nonexistent"); icon != "" {
		t.Errorf(`statusIcon(meta, "nonexistent") = %q, expected ""`, icon)
	}
}

func TestEventBadgeColor(t *testing.T) {
	t.Parallel()

	meta := auditlog.BuildTypeMetadata()

	if color := eventBadgeColor(auditlog.Event{EventType: "registration"}, meta); color == "" {
		t.Errorf("eventBadgeColor for registration returned empty")
	}

	if color := eventBadgeColor(auditlog.Event{EventType: "nonexistent"}, meta); color != cssVarTextMuted {
		t.Errorf("eventBadgeColor for unknown = %q, want %q", color, cssVarTextMuted)
	}
}

func TestEventBadgeLabel(t *testing.T) {
	t.Parallel()

	meta := auditlog.BuildTypeMetadata()

	if label := eventBadgeLabel(auditlog.Event{EventType: "registration"}, meta); label == "" {
		t.Errorf("eventBadgeLabel for registration returned empty")
	}

	if label := eventBadgeLabel(auditlog.Event{EventType: "nonexistent"}, meta); label != "nonexistent" {
		t.Errorf(`eventBadgeLabel for unknown = %q, want "nonexistent"`, label)
	}
}

func TestCountErrors(t *testing.T) {
	t.Parallel()

	services := []auditlog.ServiceInfo{
		{Status: "active"},
		{Status: "invocation_error"},
		{Status: "shutdown_error"},
		{Status: "active"},
	}

	if count := countErrors(services); count != 2 {
		t.Errorf("countErrors = %d, want 2", count)
	}

	if count := countErrors(nil); count != 0 {
		t.Errorf("countErrors(nil) = %d, want 0", count)
	}
}

package live

import (
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestComputeWaveformMarks(t *testing.T) {
	t.Parallel()

	marks := computeWaveformMarks(nil, auditlog.TypeMetadata{})
	if len(marks) != 0 {
		t.Errorf("computeWaveformMarks(nil) returned %d marks, want 0", len(marks))
	}
}

// --- CSS Completeness Test ---

// cssClassesFromFragments is the exhaustive list of CSS class names used in
// fragments.templ. If a new class is added to the templ file, it must be added
// here AND to dashboard.css or base_css.go. This prevents unstyled HTML from
// shipping to the live dashboard.
var cssClassesFromFragments = []string{
	// statsFragment
	"stat-card", "label", "value",
	// legendFragment
	"legend-item", "icon",
	// waveformFragment
	"waveform-placeholder", "wf-event",
	// servicesTbody / eventsTbody
	"empty-state", "event-badge",
	// scopeTreeFragment / scopeNode
	"scope-node", "scope-label", "scope-icon", "scope-body",
	"scope-services", "scope-service-chip", "scope-children",
	// graphFragment
	"graph-placeholder",
	"dep-graph", "dep-node", "dep-node-header",
	"dep-node-name", "dep-node-type", "dep-node-deps", "dep-arrow",
	// timelineFragment
	"timeline", "timeline-row", "timeline-label",
	"timeline-bars", "timeline-bar",
}

// cssCompoundSelectors are multi-class selectors that need exact compound
// matching (e.g., ".timeline-bar.build"), not just individual class presence.
var cssCompoundSelectors = []string{
	"timeline-bar.build",
	"timeline-bar.shutdown",
	"stat-card.success",
	"stat-card.error",
}

func TestTimelineMaxDurations(t *testing.T) {
	t.Parallel()

	buildMs := 100.0
	shutdownMs := 50.0

	services := []auditlog.ServiceInfo{
		{FirstBuildDurationMs: &buildMs, ShutdownDurationMs: &shutdownMs},
	}

	maxBuild, maxShutdown := timelineMaxDurations(services)
	if maxBuild != 100.0 {
		t.Errorf("maxBuild = %v, want 100", maxBuild)
	}

	if maxShutdown != 50.0 {
		t.Errorf("maxShutdown = %v, want 50", maxShutdown)
	}

	// Empty services should return (1, 1) — the floor values.
	maxBuild2, maxShutdown2 := timelineMaxDurations(nil)
	if maxBuild2 != 1 || maxShutdown2 != 1 {
		t.Errorf("timelineMaxDurations(nil) = (%v, %v), want (1, 1)", maxBuild2, maxShutdown2)
	}
}

func TestWaveformBounds(t *testing.T) {
	t.Parallel()

	dur := 200.0
	events := []auditlog.Event{
		{Timestamp: parseTime("2025-01-01T10:00:00Z")},
		{Timestamp: parseTime("2025-01-01T10:00:05Z"), DurationMs: &dur},
		{Timestamp: parseTime("2025-01-01T10:00:03Z")},
	}

	minT, maxT, maxDur := waveformBounds(events)
	if minT == maxT {
		t.Error("expected minT != maxT for events with different timestamps")
	}

	if maxDur != 200.0 {
		t.Errorf("maxDur = %v, want 200", maxDur)
	}
}

func TestWaveformTooltip(t *testing.T) {
	t.Parallel()

	evt := auditlog.Event{
		ServiceName: "db",
		EventType:   "invocation",
		Phase:       "after",
	}
	dur := 15.0
	evt.DurationMs = &dur

	tip := waveformTooltip(evt)
	if !strings.Contains(tip, "invocation") || !strings.Contains(tip, "db") || !strings.Contains(tip, "after") {
		t.Errorf("waveformTooltip missing expected parts: %q", tip)
	}
}

func TestComputeWaveformMarks_WithEvents(t *testing.T) {
	t.Parallel()

	meta := auditlog.BuildTypeMetadata()

	dur := 10.0
	dbRef := auditlog.ServiceRef{ServiceName: "db"}
	events := []auditlog.Event{
		{ServiceRef: dbRef, EventType: "registration", Timestamp: parseTime("2025-01-01T10:00:00Z")},
		{ServiceRef: dbRef, EventType: "invocation", Timestamp: parseTime("2025-01-01T10:00:01Z"), DurationMs: &dur},
	}

	marks := computeWaveformMarks(events, meta)
	if len(marks) != 2 {
		t.Fatalf("expected 2 marks, got %d", len(marks))
	}

	for _, mark := range marks {
		if mark.Style == "" {
			t.Error("expected non-empty Style for mark")
		}

		if mark.Tooltip == "" {
			t.Error("expected non-empty Tooltip for mark")
		}
	}
}

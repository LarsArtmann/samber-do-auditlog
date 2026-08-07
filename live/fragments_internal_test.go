package live

import (
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestHumanizeDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ms   float64
		want string
	}{
		{"negative", -1, "&mdash;"},
		{"sub-millisecond", 0.5, "0.500ms"},
		{"millisecond", 5, "5.0ms"},
		{"sub-second", 500, "500.0ms"},
		{"seconds", 2500, "2.5s"},
		{"minutes", 125000, "2m 5s"},
		{"hours", 3900000, "1h 5m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := humanizeDuration(tt.ms)
			if got != tt.want {
				t.Errorf("humanizeDuration(%v) = %q, want %q", tt.ms, got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	t.Parallel()

	if got := truncateString("hello", 10); got != "hello" {
		t.Errorf("truncateString short = %q", got)
	}

	if got := truncateString("hello world", 5); got != "hello" {
		t.Errorf("truncateString long = %q", got)
	}
}

func TestHealthLabel(t *testing.T) {
	t.Parallel()

	if got := healthLabel(true); got != "Pass" {
		t.Errorf("healthLabel(true) = %q, want Pass", got)
	}

	if got := healthLabel(false); got != "Fail" {
		t.Errorf("healthLabel(false) = %q, want Fail", got)
	}
}

func TestErrorCountClass(t *testing.T) {
	t.Parallel()

	if got := errorCountClass(0); got != "success" {
		t.Errorf("errorCountClass(0) = %q, want success", got)
	}

	if got := errorCountClass(5); got != "error" {
		t.Errorf("errorCountClass(5) = %q, want error", got)
	}
}

func TestDepNamesString(t *testing.T) {
	t.Parallel()

	deps := []auditlog.ServiceRef{
		{ServiceName: "db"},
		{ServiceName: "cache"},
	}

	if got := depNamesString(deps); got != "db, cache" {
		t.Errorf("depNamesString = %q, want 'db, cache'", got)
	}

	if got := depNamesString(nil); got != "&mdash;" {
		t.Errorf("depNamesString(nil) = %q, want &mdash;", got)
	}
}

func TestSafeDuration(t *testing.T) {
	t.Parallel()

	if got := safeDuration(nil); got != 0 {
		t.Errorf("safeDuration(nil) = %v, want 0", got)
	}

	val := 42.5
	if got := safeDuration(&val); got != 42.5 {
		t.Errorf("safeDuration(&42.5) = %v, want 42.5", got)
	}
}

func TestTimelineBarWidth(t *testing.T) {
	t.Parallel()

	if got := timelineBarWidth(nil, 100); got != "0%" {
		t.Errorf("timelineBarWidth(nil) = %q, want 0%%", got)
	}

	val := 50.0
	if got := timelineBarWidth(&val, 100); got != "50.0%" {
		t.Errorf("timelineBarWidth(&50, 100) = %q, want 50.0%%", got)
	}

	if got := timelineBarWidth(&val, 0); got != "0%" {
		t.Errorf("timelineBarWidth(&50, 0) = %q, want 0%%", got)
	}
}

func TestScopeNodeName(t *testing.T) {
	t.Parallel()

	if got := scopeNodeName(auditlog.ScopeNode{Name: "test"}); got != "test" {
		t.Errorf("scopeNodeName with name = %q", got)
	}

	if got := scopeNodeName(auditlog.ScopeNode{ID: "scope-1"}); got != "scope-1" {
		t.Errorf("scopeNodeName with ID = %q", got)
	}

	if got := scopeNodeName(auditlog.ScopeNode{}); got != "scope" {
		t.Errorf("scopeNodeName empty = %q", got)
	}
}

func TestFooterVersion(t *testing.T) {
	t.Parallel()

	if got := footerVersion(auditlog.Report{Version: "1.0"}); got != "1.0" {
		t.Errorf("footerVersion with version = %q", got)
	}

	if got := footerVersion(auditlog.Report{}); got != "?" {
		t.Errorf("footerVersion empty = %q", got)
	}
}

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

func TestCSSCompleteness(t *testing.T) {
	t.Parallel()

	combinedCSS := baseCSS + "\n" + liveCSS

	for _, cls := range cssClassesFromFragments {
		t.Run("class/"+cls, func(t *testing.T) {
			t.Parallel()

			if !cssHasSelector(combinedCSS, cls) {
				t.Errorf("CSS class %q used in fragments.templ has no selector in dashboard.css or base_css.go", cls)
			}
		})
	}

	for _, compound := range cssCompoundSelectors {
		t.Run("compound/"+compound, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(combinedCSS, "."+compound) {
				t.Errorf("Compound selector %q used in fragments.templ has no rule in CSS", compound)
			}
		})
	}
}

// cssHasSelector checks whether the CSS text defines a rule for the given class
// name. It looks for ".classname" followed by a non-identifier character (or end
// of string) to avoid false positives like ".dep-node" matching ".dep-node-header".
func cssHasSelector(css, className string) bool {
	target := "." + className

	idx := strings.Index(css, target)

	for idx != -1 {
		afterIdx := idx + len(target)

		if afterIdx >= len(css) {
			return true
		}

		next := css[afterIdx]

		if !isCSSIdentChar(next) {
			return true
		}

		// False positive (e.g., ".dep-node" inside ".dep-node-header").
		// Search again from the next position.
		rest := css[idx+1:]
		nextIdx := strings.Index(rest, target)

		if nextIdx == -1 {
			return false
		}

		idx = idx + 1 + nextIdx
	}

	return false
}

func isCSSIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}

// --- Coverage gap tests ---

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
		{ServiceLifecycle: auditlog.ServiceLifecycle{Status: "active"}},
		{ServiceLifecycle: auditlog.ServiceLifecycle{Status: "invocation_error"}},
		{ServiceLifecycle: auditlog.ServiceLifecycle{Status: "shutdown_error"}},
		{ServiceLifecycle: auditlog.ServiceLifecycle{Status: "active"}},
	}

	if count := countErrors(services); count != 2 {
		t.Errorf("countErrors = %d, want 2", count)
	}

	if count := countErrors(nil); count != 0 {
		t.Errorf("countErrors(nil) = %d, want 0", count)
	}
}

func TestTimelineMaxDurations(t *testing.T) {
	t.Parallel()

	buildMs := 100.0
	shutdownMs := 50.0

	services := []auditlog.ServiceInfo{
		{ServiceLifecycle: auditlog.ServiceLifecycle{FirstBuildDurationMs: &buildMs, ShutdownDurationMs: &shutdownMs}},
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
		ServiceRef: auditlog.ServiceRef{ServiceName: "db"},
		EventType:  "invocation",
		Phase:      "after",
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

func TestComputeLegendItems(t *testing.T) {
	t.Parallel()

	meta := auditlog.BuildTypeMetadata()

	report := auditlog.Report{
		Services: []auditlog.ServiceInfo{
			{ServiceIdentity: auditlog.ServiceIdentity{ServiceType: "lazy"}},
			{ServiceIdentity: auditlog.ServiceIdentity{ServiceType: "lazy"}},
			{ServiceIdentity: auditlog.ServiceIdentity{ServiceType: "eager"}},
		},
	}

	items := computeLegendItems(report, meta)
	if len(items) != 2 {
		t.Fatalf("expected 2 legend items, got %d", len(items))
	}

	if items[0].Count != 2 {
		t.Errorf("first item count = %d, want 2", items[0].Count)
	}
}

func TestRowSignalsJSON(t *testing.T) {
	t.Parallel()

	svc := auditlog.ServiceInfo{
		ServiceIdentity: auditlog.ServiceIdentity{
			ServiceRef: auditlog.ServiceRef{ServiceName: "db", ScopeName: "[root]"},
		},
	}

	json := rowSignalsJSON(svc, 3)
	if !strings.Contains(json, `"rowName":"db"`) {
		t.Errorf("rowSignalsJSON missing rowName: %s", json)
	}
}

func TestEventRowSignalsJSON(t *testing.T) {
	t.Parallel()

	evt := auditlog.Event{EventType: "invocation"}

	json := eventRowSignalsJSON(evt, 5)
	if !strings.Contains(json, `"evtType":"invocation"`) {
		t.Errorf("eventRowSignalsJSON missing evtType: %s", json)
	}
}

// --- Test helpers ---

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}

	return t
}

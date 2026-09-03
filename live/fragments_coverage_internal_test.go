package live

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// fixtureReport builds a report + events exercising every display branch:
// health checks, invocation errors, durations, deps, clean shutdown.
func fixtureReport(t *testing.T) (auditlog.Report, []auditlog.Event) {
	t.Helper()

	plugin, err := auditlog.New(auditlog.Config{Enabled: true, ContainerID: "fixture"})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	injector := do.NewWithOpts(plugin.Opts())

	do.ProvideNamed(injector, "fixture-healthy", func(do.Injector) (*strings.Reader, error) {
		return strings.NewReader("data"), nil
	})

	do.ProvideNamed(injector, "fixture-dep", func(i do.Injector) (*strings.Builder, error) {
		_ = do.MustInvokeNamed[*strings.Reader](i, "fixture-healthy")

		return &strings.Builder{}, nil
	})

	do.ProvideNamed(injector, "fixture-failing", func(do.Injector) (*bytes.Buffer, error) {
		return nil, errors.New("provider boom")
	})

	if _, err := do.InvokeNamed[*strings.Reader](injector, "fixture-healthy"); err != nil {
		t.Fatalf("invoke healthy: %v", err)
	}

	if _, err := do.InvokeNamed[*strings.Builder](injector, "fixture-dep"); err != nil {
		t.Fatalf("invoke dep: %v", err)
	}

	// The failing invocation records an invocation error event.
	_, _ = do.InvokeNamed[*bytes.Buffer](injector, "fixture-failing")

	_ = plugin.RecordHealthCheck(injector)

	// Shutdown errors are irrelevant to fragment rendering.
	_ = injector.Shutdown()

	return plugin.Report(), plugin.Events()
}

func TestViewBuildersCoverErrorAndHealthBranches(t *testing.T) {
	t.Parallel()

	report, events := fixtureReport(t)
	meta := auditlog.BuildTypeMetadata()

	svcRows := buildServiceRows(report.Services, meta)
	if len(svcRows) < 2 {
		t.Fatalf("expected >=2 service rows, got %d", len(svcRows))
	}

	sawError := false

	for _, row := range svcRows {
		if row.HasError {
			sawError = true

			if row.ErrTitle == "" || row.ErrMsg == "" {
				t.Error("error row should carry title and message")
			}
		}
	}

	if !sawError {
		t.Error("fixture should include at least one error row")
	}

	stats := buildStatsEntries(report, countErrors(report.Services))
	if len(stats) < 6 {
		t.Fatalf("expected base stats, got %d", len(stats))
	}

	if stats[len(stats)-1].Label != "Health" {
		t.Errorf("health-checked fixture should append a Health stat, got %+v", stats[len(stats)-1])
	}

	evtRows := buildEventRows(events, meta)
	if len(evtRows) == 0 {
		t.Fatal("expected event rows")
	}

	sawDur := false
	sawDown := false

	for _, row := range evtRows {
		if row.Duration != mdash {
			sawDur = true
		}

		if !row.PhaseUp {
			sawDown = true
		}
	}

	if !sawDur || !sawDown {
		t.Errorf("fixture events should include durations and after-phases (dur=%v down=%v)", sawDur, sawDown)
	}

	if got := computeLegendItems(report, meta); len(got) == 0 {
		t.Error("fixture should produce legend items")
	}

	marks := computeWaveformMarks(events, meta)
	if len(marks) != len(events) {
		t.Errorf("marks = %d, events = %d", len(marks), len(events))
	}
}

func TestDepNamesStringAndBadgeHelpers(t *testing.T) {
	t.Parallel()

	if got := depNamesString(nil); got != mdash {
		t.Errorf("depNamesString(nil) = %q, want mdash", got)
	}

	deps := []auditlog.ServiceRef{
		{ServiceName: "a"},
		{ServiceName: "b"},
	}

	if got := depNamesString(deps); got != "a, b" {
		t.Errorf("depNamesString = %q, want %q", got, "a, b")
	}

	meta := auditlog.BuildTypeMetadata()

	evt := auditlog.Event{EventType: auditlog.EventTypeInvocation}
	if got := eventBadgeColor(evt, meta); got == cssVarTextMuted {
		t.Error("known event type should have a metadata color")
	}

	if got := eventBadgeLabel(evt, meta); got == string(evt.EventType) {
		t.Error("known event type should have a metadata label")
	}

	unknown := auditlog.Event{EventType: "mystery"}
	if got := eventBadgeColor(unknown, meta); got != cssVarTextMuted {
		t.Errorf("unknown event type should fall back to muted, got %q", got)
	}

	if got := eventBadgeLabel(unknown, meta); got != "mystery" {
		t.Errorf("unknown event type should fall back to raw name, got %q", got)
	}
}

func TestIconHelpersFallBackToEmpty(t *testing.T) {
	t.Parallel()

	meta := auditlog.BuildTypeMetadata()

	if got := providerIcon(meta, "nope"); got != "" {
		t.Errorf("providerIcon(unknown) = %q, want empty", got)
	}

	if got := statusIcon(meta, "nope"); got != "" {
		t.Errorf("statusIcon(unknown) = %q, want empty", got)
	}
}

func TestScopeNodeNameFallbacks(t *testing.T) {
	t.Parallel()

	if got := scopeNodeName(auditlog.ScopeNode{ID: "s1", Name: "Named"}); got != "Named" {
		t.Errorf("scopeNodeName named = %q", got)
	}

	if got := scopeNodeName(auditlog.ScopeNode{ID: "s1"}); got != "s1" {
		t.Errorf("scopeNodeName id fallback = %q", got)
	}

	if got := scopeNodeName(auditlog.ScopeNode{}); got != "scope" {
		t.Errorf("scopeNodeName empty fallback = %q", got)
	}
}

func TestFooterVersionAndErrorClass(t *testing.T) {
	t.Parallel()

	if got := footerVersion(auditlog.Report{Version: "9.9.9"}); got != "9.9.9" {
		t.Errorf("footerVersion = %q", got)
	}

	if got := footerVersion(auditlog.Report{}); got != "?" {
		t.Errorf("footerVersion empty = %q", got)
	}

	if errorCountClass(3) != cssClassError || errorCountClass(0) != cssClassSuccess {
		t.Error("errorCountClass mapping broken")
	}
}

func TestTimelineBarWidthEdgeCases(t *testing.T) {
	t.Parallel()

	if got := timelineBarWidth(nil, 100); got != "0%" {
		t.Errorf("timelineBarWidth(nil) = %q", got)
	}

	zero := 0.0

	if got := timelineBarWidth(&zero, 100); got != "0%" {
		t.Errorf("timelineBarWidth(zero) = %q", got)
	}

	val := 50.0

	if got := timelineBarWidth(&val, 100); got != "50.0%" {
		t.Errorf("timelineBarWidth(50/100) = %q", got)
	}
}

func TestRenderFragmentUnknownTemplateReturnsEmpty(t *testing.T) {
	t.Parallel()

	if got := renderFragment("doesNotExist", nil); got != "" {
		t.Errorf("renderFragment(unknown) = %q, want empty", got)
	}
}

func TestTimelineFragmentRendersBars(t *testing.T) {
	t.Parallel()

	report, _ := fixtureReport(t)

	html := renderFragment("timelineFragment", timelineFragmentData{Rows: buildTimelineRows(report.Services)})
	if !strings.Contains(html, "timeline-bar build") || !strings.Contains(html, "timeline-bar shutdown") {
		t.Errorf("timeline fragment should contain build+shutdown bars: %.200s", html)
	}
}

func TestWaveformPlaceholderWhenNoEvents(t *testing.T) {
	t.Parallel()

	html := renderFragment("waveformFragment", waveformFragmentData{})
	if !strings.Contains(html, "waveform-placeholder") {
		t.Errorf("empty waveform should render placeholder, got %q", html)
	}
}

func TestContainerIDFragmentEmpty(t *testing.T) {
	t.Parallel()

	html := renderFragment("containerIDFragment", containerIDData{})
	if !strings.Contains(html, mdash) {
		t.Errorf("empty container id should render mdash, got %q", html)
	}
}

func TestMarshalSignalsOrEmpty(t *testing.T) {
	t.Parallel()

	if got := marshalSignalsOrEmpty(rowSignals{RowName: "x", RowScope: "[root]", RowIdx: 2}); !strings.Contains(got, `"rowIdx":2`) {
		t.Errorf("marshalSignalsOrEmpty = %q", got)
	}

	if got := marshalSignalsOrEmpty(make(chan int)); got != "{}" {
		t.Errorf("unmarshalable value should yield {}, got %q", got)
	}
}

func TestEventsShowExprConstants(t *testing.T) {
	t.Parallel()

	if !strings.Contains(servicesShowExpr, "$rowIdx < 50") {
		t.Errorf("servicesShowExpr should embed the pagination cap: %q", servicesShowExpr)
	}

	if !strings.Contains(eventsShowExpr, "$evtIdx < 100") {
		t.Errorf("eventsShowExpr should embed the pagination cap: %q", eventsShowExpr)
	}
}

func TestEventTimeFormat(t *testing.T) {
	t.Parallel()

	evt := auditlog.Event{Timestamp: time.Date(2026, 9, 3, 23, 59, 9, 0, time.UTC)}
	rows := buildEventRows([]auditlog.Event{evt}, auditlog.BuildTypeMetadata())

	if rows[0].Time != "23:59:09" {
		t.Errorf("event time = %q, want 23:59:09", rows[0].Time)
	}
}

package live

import (
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestGraphFragment_Render(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		html := renderToString(t.Context(), graphFragment(auditlog.Report{}))

		if !strings.Contains(html, "graph-placeholder") {
			t.Errorf("empty graphFragment should show placeholder, got: %s", html)
		}
	})

	t.Run("with_services", func(t *testing.T) {
		t.Parallel()

		report := auditlog.Report{
			Services: []auditlog.ServiceInfo{
				{
					ServiceName: "db",
					ServiceType: "lazy",
				},
			},
		}

		html := renderToString(t.Context(), graphFragment(report))

		if !strings.Contains(html, "dep-graph") {
			t.Errorf("graphFragment should contain dep-graph, got: %s", html)
		}

		if !strings.Contains(html, "dep-node") {
			t.Errorf("graphFragment should contain dep-node, got: %s", html)
		}

		if !strings.Contains(html, "db") {
			t.Errorf("graphFragment should contain service name 'db'")
		}
	})
}

func TestTimelineFragment_Render(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		html := renderToString(t.Context(), timelineFragment(auditlog.Report{}, 1, 1))

		if !strings.Contains(html, "graph-placeholder") {
			t.Errorf("empty timelineFragment should show placeholder")
		}
	})

	t.Run("with_services", func(t *testing.T) {
		t.Parallel()

		buildMs := 50.0
		report := auditlog.Report{
			Services: []auditlog.ServiceInfo{
				{
					ServiceName:          "cache",
					FirstBuildDurationMs: &buildMs,
				},
			},
		}

		html := renderToString(t.Context(), timelineFragment(report, 100, 1))

		if !strings.Contains(html, "timeline") {
			t.Errorf("timelineFragment should contain timeline class")
		}

		if !strings.Contains(html, "cache") {
			t.Errorf("timelineFragment should contain service name 'cache'")
		}

		if !strings.Contains(html, "timeline-bar build") {
			t.Errorf("timelineFragment should contain build bar")
		}
	})
}

func TestStatsFragment_Render(t *testing.T) {
	t.Parallel()

	stats := buildStatsEntries(auditlog.Report{
		ServiceCount: 5,
		EventCount:   20,
		ScopeCount:   2,
	}, 0)

	html := renderToString(t.Context(), statsFragment(stats))

	if !strings.Contains(html, "stat-card") {
		t.Errorf("statsFragment should contain stat-card")
	}

	if !strings.Contains(html, "Services") {
		t.Errorf("statsFragment should contain Services label")
	}

	if !strings.Contains(html, "5") {
		t.Errorf("statsFragment should contain service count")
	}
}

func TestServicesTbody_Render(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		meta := auditlog.BuildTypeMetadata()
		html := renderToString(t.Context(), servicesTbody(auditlog.Report{}, meta))

		if !strings.Contains(html, "empty-state") {
			t.Errorf("empty servicesTbody should show empty-state")
		}
	})

	t.Run("with_services", func(t *testing.T) {
		t.Parallel()

		meta := auditlog.BuildTypeMetadata()

		buildMs := 10.0
		invErr := "connection refused"
		report := auditlog.Report{
			Services: []auditlog.ServiceInfo{
				{
					ServiceName: "db", ScopeName: "[root]",
					ServiceType:          "lazy",
					Status:               "active",
					InvocationCount:      3,
					FirstBuildDurationMs: &buildMs,
				},
				{
					ServiceName: "cache", ScopeName: "[root]",
					ServiceType:     "eager",
					Status:          "invocation_error",
					InvocationCount: 1,
					InvocationError: &invErr,
				},
			},
		}

		html := renderToString(t.Context(), servicesTbody(report, meta))

		if !strings.Contains(html, "db") {
			t.Errorf("servicesTbody should contain 'db'")
		}

		if !strings.Contains(html, "cache") {
			t.Errorf("servicesTbody should contain 'cache'")
		}

		if !strings.Contains(html, "connection refused") {
			t.Errorf("servicesTbody should contain invocation error")
		}
	})
}

func TestEventsTbody_Render(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		meta := auditlog.BuildTypeMetadata()
		html := renderToString(t.Context(), eventsTbody(nil, meta))

		if !strings.Contains(html, "empty-state") {
			t.Errorf("empty eventsTbody should show empty-state")
		}
	})

	t.Run("with_events", func(t *testing.T) {
		t.Parallel()

		meta := auditlog.BuildTypeMetadata()
		dur := 5.0
		errMsg := "timeout"

		events := []auditlog.Event{
			{
				ServiceName: "db",
				EventType:   "registration",
				Phase:       "after",
				Timestamp:   parseTime("2025-01-01T10:00:00Z"),
			},
			{
				ServiceName: "cache",
				EventType:   "invocation",
				Phase:       "after",
				Timestamp:   parseTime("2025-01-01T10:00:01Z"),
				DurationMs:  &dur,
				Error:       &errMsg,
			},
		}

		html := renderToString(t.Context(), eventsTbody(events, meta))

		if !strings.Contains(html, "db") {
			t.Errorf("eventsTbody should contain 'db'")
		}

		if !strings.Contains(html, "timeout") {
			t.Errorf("eventsTbody should contain error message")
		}

		if !strings.Contains(html, "event-badge") {
			t.Errorf("eventsTbody should contain event-badge")
		}
	})
}

func TestScopeTreeFragment_Render(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		html := renderToString(t.Context(), scopeTreeFragment(auditlog.Report{}))

		if !strings.Contains(html, "graph-placeholder") {
			t.Errorf("empty scopeTreeFragment should show placeholder")
		}
	})

	t.Run("with_scopes", func(t *testing.T) {
		t.Parallel()

		report := auditlog.Report{
			ScopeTree: auditlog.ScopeNode{
				ID:       "[root]",
				Name:     "[root]",
				Services: []auditlog.ServiceName{"db", "cache"},
				Children: []auditlog.ScopeNode{
					{ID: "child", Name: "child", Services: []auditlog.ServiceName{"svc"}},
				},
			},
		}

		html := renderToString(t.Context(), scopeTreeFragment(report))

		if !strings.Contains(html, "scope-node") {
			t.Errorf("scopeTreeFragment should contain scope-node")
		}

		if !strings.Contains(html, "scope-service-chip") {
			t.Errorf("scopeTreeFragment should contain service chips")
		}
	})
}

func TestWaveformFragment_Render(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		html := renderToString(t.Context(), waveformFragment(nil))

		if !strings.Contains(html, "waveform-placeholder") {
			t.Errorf("empty waveformFragment should show placeholder")
		}
	})

	t.Run("with_marks", func(t *testing.T) {
		t.Parallel()

		marks := []waveformMark{
			{Style: "left:0%;height:10px;background:red", Tooltip: "registration db"},
			{Style: "left:50%;height:20px;background:green", Tooltip: "invocation cache"},
		}

		html := renderToString(t.Context(), waveformFragment(marks))

		if !strings.Contains(html, "wf-event") {
			t.Errorf("waveformFragment should contain wf-event")
		}

		if !strings.Contains(html, "registration db") {
			t.Errorf("waveformFragment should contain tooltip")
		}
	})
}

// --- Test helpers ---

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}

	return t
}

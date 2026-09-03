package live

import (
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func mkFragmentPlugin(t *testing.T) (*auditlog.Plugin, do.Injector) {
	t.Helper()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "frag-test",
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	injector := do.NewWithOpts(plugin.Opts())

	do.ProvideNamed(injector, "db", func(do.Injector) (*strings.Reader, error) {
		return strings.NewReader("data"), nil
	})

	if _, err := do.InvokeNamed[*strings.Reader](injector, "db"); err != nil {
		t.Fatalf("invoke db: %v", err)
	}

	return plugin, injector
}

func TestRenderAllFragmentsProducesAllSelectors(t *testing.T) {
	t.Parallel()

	plugin, _ := mkFragmentPlugin(t)

	report := plugin.Report()
	events := plugin.Events()
	meta := auditlog.BuildTypeMetadata()

	fragments := renderAllFragments(report, events, meta)

	wantSelectors := []string{
		"#stats", "#legend", "#waveform", "#services-tbody", "#events-tbody",
		"#scope-tree-container", "#graph-container", "#timeline-container",
		"#footer-stats", "#container-id",
	}

	if len(fragments) != len(wantSelectors) {
		t.Fatalf("got %d fragments, want %d", len(fragments), len(wantSelectors))
	}

	for i, sel := range wantSelectors {
		if fragments[i].selector != sel {
			t.Errorf("fragment[%d].selector = %q, want %q", i, fragments[i].selector, sel)
		}

		if fragments[i].html == "" {
			t.Errorf("fragment %s rendered empty (template error?)", sel)
		}
	}
}

func TestRenderedFragmentsContent(t *testing.T) {
	t.Parallel()

	plugin, _ := mkFragmentPlugin(t)

	report := plugin.Report()
	meta := auditlog.BuildTypeMetadata()

	html := map[string]string{}
	for _, frag := range renderAllFragments(report, plugin.Events(), meta) {
		html[frag.selector] = frag.html
	}

	checks := []struct {
		selector string
		contains []string
	}{
		{selector: "#stats", contains: []string{`class="stat-card`, "Services", "Events"}},
		{selector: "#services-tbody", contains: []string{"data-signals=", "db", "active"}},
		{selector: "#scope-tree-container", contains: []string{"scope-node", "scope-label"}},
		{selector: "#graph-container", contains: []string{"dep-node", "db"}},
		{selector: "#footer-stats", contains: []string{"Schema v" + auditlog.SchemaVersion}},
		{selector: "#container-id", contains: []string{"frag-test"}},
	}

	for _, check := range checks {
		for _, want := range check.contains {
			if !strings.Contains(html[check.selector], want) {
				t.Errorf("fragment %s: want substring %q in %q", check.selector, want, html[check.selector])
			}
		}
	}
}

func TestRenderedFragmentDatastarAttributes(t *testing.T) {
	t.Parallel()

	plugin, _ := mkFragmentPlugin(t)

	var servicesHTML string

	for _, frag := range renderAllFragments(plugin.Report(), plugin.Events(), auditlog.BuildTypeMetadata()) {
		if frag.selector == "#services-tbody" {
			servicesHTML = frag.html
		}
	}

	for _, attr := range []string{"data-signals=", `data-show="(!$serviceSearch`} {
		if !strings.Contains(servicesHTML, attr) {
			t.Errorf("services fragment missing datastar attribute %q: %q", attr, servicesHTML)
		}
	}
}

func TestFragmentHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{name: "negative", input: -1, want: mdash},
		{name: "sub-ms", input: 0.5, want: "0.500ms"},
		{name: "ms", input: 123.4, want: "123.4ms"},
		{name: "seconds", input: 2500, want: "2.5s"},
		{name: "minutes", input: 90000, want: "1m 30s"},
		{name: "hours", input: 5400000, want: "1h 30m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := humanizeDuration(tt.input); got != tt.want {
				t.Errorf("humanizeDuration(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestComputeWaveformMarksEmptyAndFilled(t *testing.T) {
	t.Parallel()

	if marks := computeWaveformMarks(nil, auditlog.BuildTypeMetadata()); marks != nil {
		t.Errorf("empty events should produce nil marks, got %+v", marks)
	}

	dur := 12.0
	events := []auditlog.Event{
		{Sequence: 1, EventType: auditlog.EventTypeRegistration, Phase: auditlog.PhaseBefore, Timestamp: time.UnixMilli(100)},
		{Sequence: 2, EventType: auditlog.EventTypeInvocation, Phase: auditlog.PhaseAfter, Timestamp: time.UnixMilli(200), DurationMs: &dur, Error: strPtr("boom")},
	}

	marks := computeWaveformMarks(events, auditlog.BuildTypeMetadata())
	if len(marks) != 2 {
		t.Fatalf("got %d marks, want 2", len(marks))
	}

	if !strings.Contains(marks[1].Style, "var(--error)") {
		t.Errorf("error event mark should use error color: %q", marks[1].Style)
	}

	if !strings.Contains(marks[1].Tooltip, "invocation") || !strings.Contains(marks[1].Tooltip, "12.0ms") {
		t.Errorf("tooltip should contain type and duration: %q", marks[1].Tooltip)
	}
}

func TestTruncateStringAndHealthLabel(t *testing.T) {
	t.Parallel()

	if got := truncateString("0123456789", 4); got != "0123" {
		t.Errorf("truncateString long = %q", got)
	}

	if got := truncateString("ab", 4); got != "ab" {
		t.Errorf("truncateString short = %q", got)
	}

	if healthLabel(true) != "Pass" || healthLabel(false) != "Fail" {
		t.Error("healthLabel should map true->Pass, false->Fail")
	}
}

func strPtr(s string) *string { return &s }

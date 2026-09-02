package live

import (
	"context"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// newFixtureReport builds a small plugin + injector with two services in a
// dependency relationship, invokes them, and returns the report plus its
// event stream. Exercises the same recording path the dashboard sees live.
func newFixtureReport(t *testing.T) (auditlog.Report, []auditlog.Event) {
	t.Helper()

	var events []auditlog.Event

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "frag-test",
		OnEvent:     func(evt auditlog.Event) { events = append(events, evt) },
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	injector := do.NewWithOpts(plugin.Opts())

	do.ProvideNamed(injector, "logger", func(do.Injector) (*strings.Builder, error) {
		return &strings.Builder{}, nil
	})
	do.ProvideNamed(injector, "db", func(i do.Injector) (*strings.Reader, error) {
		_ = do.MustInvokeNamed[*strings.Builder](i, "logger")

		return strings.NewReader("data"), nil
	})

	if _, err := do.InvokeNamed[*strings.Reader](injector, "db"); err != nil {
		t.Fatalf("invoke db: %v", err)
	}

	// Shutdown errors are irrelevant to fragment rendering; ignore err (a
	// plain *strings.Reader has no Shutdowner semantics in do v2).
	_ = injector.Shutdown()

	return plugin.Report(), events
}

func TestRenderAllFragments_AllSelectorsRender(t *testing.T) {
	t.Parallel()

	report, events := newFixtureReport(t)
	meta := auditlog.BuildTypeMetadata()

	patches := renderAllFragments(context.Background(), report, events, meta)

	wantSelectors := []string{
		"#stats",
		"#legend",
		"#waveform",
		"#services-tbody",
		"#events-tbody",
		"#scope-tree-container",
		"#graph-container",
		"#timeline-container",
		"#footer-stats",
		"#container-id",
	}

	if len(patches) != len(wantSelectors) {
		t.Fatalf("expected %d fragments, got %d", len(wantSelectors), len(patches))
	}

	for i, want := range wantSelectors {
		if patches[i].selector != want {
			t.Errorf("patch %d selector: want %q, got %q", i, want, patches[i].selector)
		}

		if strings.TrimSpace(patches[i].html) == "" {
			t.Errorf("patch %q rendered empty HTML", want)
		}
	}
}

func TestRenderAllFragments_ContentMarkers(t *testing.T) {
	t.Parallel()

	report, events := newFixtureReport(t)
	meta := auditlog.BuildTypeMetadata()

	patches := renderAllFragments(context.Background(), report, events, meta)

	bySelector := make(map[string]string, len(patches))
	for _, p := range patches {
		bySelector[p.selector] = p.html
	}

	// Services table shows both fixture services.
	svc := bySelector["#services-tbody"]
	for _, name := range []string{"logger", "db"} {
		if !strings.Contains(svc, name) {
			t.Errorf("#services-tbody missing service %q", name)
		}
	}

	// Events table reflects the recorded events (one row per event).
	evts := bySelector["#events-tbody"]
	if len(events) > 0 && !strings.Contains(evts, "evtType") && !strings.Contains(evts, "data") {
		t.Errorf("#events-tbody missing event markers for %d events", len(events))
	}

	// Stats carry the service count.
	stats := bySelector["#stats"]
	if !strings.Contains(stats, "Services") {
		t.Errorf("#stats missing Services label")
	}

	// Scope tree contains the root scope marker.
	if !strings.Contains(bySelector["#scope-tree-container"], "scope") &&
		!strings.Contains(bySelector["#scope-tree-container"], "root") {
		t.Errorf("#scope-tree-container missing scope markers")
	}

	// Graph contains both service nodes.
	graph := bySelector["#graph-container"]
	if !strings.Contains(graph, "db") || !strings.Contains(graph, "logger") {
		t.Errorf("#graph-container missing service nodes")
	}

	// Timeline renders bars for both services.
	timeline := bySelector["#timeline-container"]
	if !strings.Contains(timeline, "db") || !strings.Contains(timeline, "logger") {
		t.Errorf("#timeline-container missing service bars")
	}

	// Footer shows the schema version.
	if !strings.Contains(bySelector["#footer-stats"], string(auditlog.SchemaVersion)) {
		t.Errorf("#footer-stats missing schema version %q", string(auditlog.SchemaVersion))
	}

	// Container ID fragment echoes the configured container.
	if !strings.Contains(bySelector["#container-id"], "frag-test") {
		t.Errorf("#container-id missing container id")
	}
}

func TestRenderAllFragments_EmptyReport(t *testing.T) {
	t.Parallel()

	// An empty report (no services, no events) must still render every
	// fragment without panicking — this is what a freshly-created plugin
	// dashboard shows.
	report, err := auditlog.NewReport(
		auditlog.SchemaVersion,
		"empty",
		"",
		time.Now(),
		nil,
		nil,
		auditlog.ScopeNode{ID: "root", Name: auditlog.RootScopeName},
	)
	if err != nil {
		t.Fatalf("NewReport: %v", err)
	}

	patches := renderAllFragments(context.Background(), report, nil, auditlog.BuildTypeMetadata())

	if len(patches) != 10 {
		t.Fatalf("expected 10 fragments for empty report, got %d", len(patches))
	}
}

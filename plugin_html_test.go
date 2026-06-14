package auditlog_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestPlugin_ExportToHTML(t *testing.T) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	path := t.TempDir() + "/report.html"
	if err := p.ExportToHTML(path); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if len(data) < 500 {
		t.Errorf("HTML file too small (%d bytes), expected a full page", len(data))
	}

	if !strings.Contains(strings.ToLower(string(data)), "<!doctype html>") {
		t.Error("expected DOCTYPE in HTML output")
	}

	assertHTMLContains(t, string(data), "db")
}

func TestPlugin_WriteHTMLBuffer(t *testing.T) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := p.WriteHTML(&buf)
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := buf.String()
	if len(html) < 500 {
		t.Errorf("HTML too small (%d bytes)", len(html))
	}

	if !strings.Contains(strings.ToLower(html), "<!doctype html>") {
		t.Error("expected DOCTYPE in HTML output")
	}

	assertHTMLContains(t, html, "db")
}

func TestWriteHTML_EventsTabContent(t *testing.T) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := p.WriteHTML(&buf)
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := buf.String()
	htmlLower := strings.ToLower(html)

	assertHTMLContains(t, html, "events-tbody")
	assertHTMLContains(t, html, "allEvents")
	assertHTMLContains(t, html, "report.events.map")
	assertHTMLContains(t, html, "data-type=")
	assertHTMLContains(t, htmlLower, "event-badge")
}

func TestWriteHTML_AllFiveTabs(t *testing.T) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := p.WriteHTML(&buf)
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := buf.String()

	tabs := []string{"services", "scopes", "graph", "timeline", "events"}
	for _, tab := range tabs {
		id := "tab-" + tab
		assertHTMLContains(t, html, id)
	}

	assertHTMLContains(t, html, "services-tbody")
	assertHTMLContains(t, html, "scope-tree")
	assertHTMLContains(t, html, "graph-container")
	assertHTMLContains(t, html, "timeline-container")
	assertHTMLContains(t, html, "events-tbody")
}

func TestWriteHTML_TypeMetadataInjected(t *testing.T) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := p.WriteHTML(&buf)
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := buf.String()

	assertHTMLContains(t, html, "type-metadata")
	assertHTMLContains(t, html, "providers")
	assertHTMLContains(t, html, "statuses")
	assertHTMLContains(t, html, "events")
}

func TestWriteHTML_MultiServiceIntegration(t *testing.T) {
	p := mustNew(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())
	child := injector.Scope("child-scope")

	provideDB(injector, "db", "postgres://localhost")
	provideCache(injector, "cache")
	provideUserServiceWithDB(injector, "user-service", "db")
	provideDB(child, "child-db", "postgres://child")

	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*Cache](injector, "cache")
	_ = do.MustInvokeNamed[*UserService](injector, "user-service")
	_ = do.MustInvokeNamed[*Database](child, "child-db")

	var buf bytes.Buffer

	err := p.WriteHTML(&buf)
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := buf.String()

	for _, svc := range []string{"db", "cache", "user-service", "child-db"} {
		assertHTMLContains(t, html, svc)
	}

	assertHTMLContains(t, html, "child-scope")
	assertHTMLContains(t, html, "scope_count")

	report := p.Report()
	assertHTMLContains(t, html, report.ContainerID)

	for _, tab := range []string{"tab-services", "tab-scopes", "tab-graph", "tab-timeline", "tab-events"} {
		assertHTMLContains(t, html, tab)
	}

	assertHTMLContains(t, html, "services-tbody")
	assertHTMLContains(t, html, "events-tbody")
	assertHTMLContains(t, html, "scope-tree")
	assertHTMLContains(t, html, "graph-container")
	assertHTMLContains(t, html, "timeline-container")
	assertHTMLContains(t, html, "event-filters")
	assertHTMLContains(t, html, "service-search")
}

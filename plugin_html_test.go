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
	p := auditlog.New(auditlog.Config{Enabled: true})
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

	if !strings.Contains(string(data), "db") {
		t.Error("expected 'db' service name in HTML output")
	}
}

func TestPlugin_WriteHTMLBuffer(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
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

	if !strings.Contains(html, "db") {
		t.Error("expected 'db' service name in HTML output")
	}
}

func TestWriteHTML_EventsTabContent(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
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

	if !strings.Contains(html, "events-tbody") {
		t.Error("expected events-tbody element in HTML")
	}

	if !strings.Contains(html, "allEvents") {
		t.Error("expected allEvents JS variable in HTML")
	}

	if !strings.Contains(html, "report.events.map") {
		t.Error("expected report.events.map in allEvents construction")
	}

	if !strings.Contains(html, "data-type=") {
		t.Error("expected data-type attribute on event rows for filtering")
	}

	if !strings.Contains(htmlLower, "event-badge") {
		t.Error("expected event-badge CSS class for event type badges")
	}
}

func TestWriteHTML_AllFiveTabs(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
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
		if !strings.Contains(html, id) {
			t.Errorf("expected %s tab content div in HTML", id)
		}
	}

	if !strings.Contains(html, "services-tbody") {
		t.Error("expected services-tbody in services tab")
	}

	if !strings.Contains(html, "scope-tree") {
		t.Error("expected scope-tree element in scopes tab")
	}

	if !strings.Contains(html, "graph-container") {
		t.Error("expected graph-container in graph tab")
	}

	if !strings.Contains(html, "timeline-container") {
		t.Error("expected timeline-container in timeline tab")
	}

	if !strings.Contains(html, "events-tbody") {
		t.Error("expected events-tbody in events tab")
	}
}

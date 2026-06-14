package auditlog_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestPlugin_ExportToFile(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	path := t.TempDir() + "/report.json"
	if err := p.ExportToFile(path); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var report auditlog.Report
	unmarshalJSONForTest(t, data, &report, "unmarshal")

	assertServiceCount(t, report, 1)
}

func TestPlugin_ExportEventsToNDJSON(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	path := t.TempDir() + "/events.ndjson"
	if err := p.ExportEventsToNDJSON(path); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	lines := 0

	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}

	if lines != 4 {
		t.Errorf("expected 4 ndjson lines, got %d", lines)
	}
}

func TestPlugin_WriteReportJSON(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := p.WriteReportJSON(&buf)
	if err != nil {
		t.Fatalf("WriteReportJSON failed: %v", err)
	}

	var report auditlog.Report
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	assertServiceCount(t, report, 1)

	assertVersion(t, report)
}

func TestPlugin_WriteEventsNDJSON(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := p.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON failed: %v", err)
	}

	lines := strings.Count(buf.String(), "\n")
	if lines != 4 {
		t.Errorf("expected 4 ndjson lines, got %d", lines)
	}
}

func TestPlugin_WriteReportJSONError(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "postgres://localhost")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := p.WriteReportJSON(&buf)
	if err != nil {
		t.Fatalf("WriteReportJSON failed: %v", err)
	}

	var report auditlog.Report
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	assertServiceCount(t, report, 1)
}

func TestPlugin_WriteReportJSONErrorPath(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	err := p.WriteReportJSON(failingWriter{})
	if err == nil {
		t.Fatal("expected error from failing writer")
	}
}

func TestPlugin_WriteEventsNDJSONErrorPath(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	err := p.WriteEventsNDJSON(failingWriter{})
	if err == nil {
		t.Fatal("expected error from failing writer")
	}
}

func TestPlugin_WriteHTMLErrorPath(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	err := p.WriteHTML(failingWriter{})
	if err == nil {
		t.Fatal("expected error from failing writer")
	}
}

func TestPlugin_ExportToFileInvalidPath(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	err := p.ExportToFile("/nonexistent/dir/report.json")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestPlugin_ExportFilteredToFile(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	provideCache(injector, "cache")
	_ = do.MustInvokeNamed[*Database](injector, "db")
	_ = do.MustInvokeNamed[*Cache](injector, "cache")

	dir := t.TempDir()
	path := dir + "/filtered.json"

	err := p.ExportFilteredToFile(path, auditlog.WithServicesByName("db"))
	if err != nil {
		t.Fatalf("ExportFilteredToFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var report map[string]any
	unmarshalJSONForTest(t, data, &report, "Unmarshal")

	services, ok := report["services"].([]any)
	if !ok {
		t.Fatal("services should be an array")
	}

	if len(services) != 1 {
		t.Errorf("expected 1 service in file, got %d", len(services))
	}
}

func TestPlugin_ExportFilteredToFile_BadPath(t *testing.T) {
	p := auditlog.New(auditlog.Config{Enabled: true})
	injector := do.NewWithOpts(p.Opts())

	provideDB(injector, "db", "test")
	_ = do.MustInvokeNamed[*Database](injector, "db")

	err := p.ExportFilteredToFile(
		"/nonexistent/dir/file.json",
		auditlog.WithServicesByName("db"),
	)
	if err == nil {
		t.Error("expected error for bad path")
	}
}

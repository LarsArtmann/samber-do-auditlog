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
	t.Parallel()

	p, _ := setupWithDB("postgres://localhost")

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
	t.Parallel()

	p, _ := setupWithDB("postgres://localhost")

	path := t.TempDir() + "/events.ndjson"
	if err := p.ExportEventsToNDJSON(path); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	lines := strings.Count(string(data), "\n")

	if lines != 4 {
		t.Errorf("expected 4 ndjson lines, got %d", lines)
	}
}

func TestPlugin_WriteReportJSON(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("postgres://localhost")

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
	t.Parallel()

	p, _ := setupWithDB("postgres://localhost")

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

func TestPlugin_WriteReportJSONErrorPath(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("test")
	assertWriteFails(t, "WriteReportJSON", p.WriteReportJSON)
}

func TestPlugin_WriteEventsNDJSONErrorPath(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("test")
	assertWriteFails(t, "WriteEventsNDJSON", p.WriteEventsNDJSON)
}

func TestPlugin_WriteHTMLErrorPath(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("test")
	assertWriteFails(t, "WriteHTML", p.WriteHTML)
}

func TestPlugin_ExportToFileInvalidPath(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("test")

	err := p.ExportToFile("/nonexistent/dir/report.json")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestPlugin_ExportFilteredToFile(t *testing.T) {
	t.Parallel()

	p, injector := setupWithDB("test")
	provideCache(injector, "cache")
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
	t.Parallel()

	p, _ := setupWithDB("test")

	err := p.ExportFilteredToFile(
		"/nonexistent/dir/file.json",
		auditlog.WithServicesByName("db"),
	)
	if err == nil {
		t.Error("expected error for bad path")
	}
}

func TestPlugin_WriteReportCSV(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("postgres://localhost")

	var buf bytes.Buffer
	if err := p.WriteReportCSV(&buf); err != nil {
		t.Fatalf("WriteReportCSV failed: %v", err)
	}

	output := buf.String()
	assertStringContains(t, output, "service_name")
	assertStringContains(t, output, "db")
}

func TestPlugin_ExportToCSV(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("test")

	path := t.TempDir() + "/report.csv"
	if err := p.ExportToCSV(path); err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	assertStringContains(t, string(data), "db")
}

func TestPlugin_ExportToTSV(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("test")

	path := t.TempDir() + "/report.tsv"
	if err := p.ExportToTSV(path); err != nil {
		t.Fatalf("ExportToTSV failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	assertStringContains(t, string(data), "\t")
	assertStringContains(t, string(data), "db")
}

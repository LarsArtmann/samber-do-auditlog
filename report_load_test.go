package auditlog_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestLoadReport_JSONFile(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("postgres://localhost")

	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	err := p.ExportToFile(path)
	if err != nil {
		t.Fatalf("ExportToFile: %v", err)
	}

	report, format, err := auditlog.LoadReport(path)
	if err != nil {
		t.Fatalf("LoadReport: %v", err)
	}

	if format != auditlog.FormatJSON {
		t.Errorf("format: want JSON, got %v", format)
	}

	assertReportServiceCount(t, report)
}

func TestLoadReport_NDJSONFile(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("postgres://localhost")

	dir := t.TempDir()
	path := filepath.Join(dir, "events.ndjson")

	err := p.ExportEventsToNDJSON(path)
	if err != nil {
		t.Fatalf("ExportEventsToNDJSON: %v", err)
	}

	report, format, err := auditlog.LoadReport(path)
	if err != nil {
		t.Fatalf("LoadReport: %v", err)
	}

	if format != auditlog.FormatNDJSON {
		t.Errorf("format: want NDJSON, got %v", format)
	}

	assertReportServiceCount(t, report)

	if !report.Reconstructed {
		t.Error("expected Reconstructed=true for NDJSON-loaded report")
	}
}

func TestLoadReport_NonExistentFile(t *testing.T) {
	t.Parallel()

	_, _, err := auditlog.LoadReport("/nonexistent/path/file.json")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestLoadReport_WithFormatOption(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("postgres://localhost")

	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	err := p.ExportToFile(path)
	if err != nil {
		t.Fatalf("ExportToFile: %v", err)
	}

	report, _, err := auditlog.LoadReport(path, auditlog.WithFormat(auditlog.FormatJSON))
	if err != nil {
		t.Fatalf("LoadReport with WithFormat: %v", err)
	}

	assertReportServiceCount(t, report)
}

func TestLoadReportFromJSON_DirectBytes(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("postgres://localhost")

	var buf bytes.Buffer

	err := p.WriteReportJSON(&buf)
	if err != nil {
		t.Fatalf("WriteReportJSON: %v", err)
	}

	report, err := auditlog.MigrateReport(buf.Bytes())
	if err != nil {
		t.Fatalf("MigrateReport: %v", err)
	}

	assertReportServiceCount(t, report)
}

func TestLoadReportFromReader_AutoDetectJSON(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("postgres://localhost")

	var buf bytes.Buffer

	err := p.WriteReportJSON(&buf)
	if err != nil {
		t.Fatalf("WriteReportJSON: %v", err)
	}

	report, format, err := auditlog.LoadReportFromReader(&buf, auditlog.FormatAuto)
	if err != nil {
		t.Fatalf("LoadReportFromReader: %v", err)
	}

	if format != auditlog.FormatJSON {
		t.Errorf("format: want JSON, got %v", format)
	}

	assertReportServiceCount(t, report)
}

func TestLoadReportFromReader_AutoDetectNDJSON(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("postgres://localhost")

	var buf bytes.Buffer

	err := p.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	report, format, err := auditlog.LoadReportFromReader(&buf, auditlog.FormatAuto)
	if err != nil {
		t.Fatalf("LoadReportFromReader: %v", err)
	}

	if format != auditlog.FormatNDJSON {
		t.Errorf("format: want NDJSON, got %v", format)
	}

	assertReportServiceCount(t, report)
}

func TestLoadReportFromReader_EmptyReader(t *testing.T) {
	t.Parallel()

	_, _, err := auditlog.LoadReportFromReader(strings.NewReader(""), auditlog.FormatAuto)
	assertErrIs(t, err, auditlog.ErrEmpty, "expected ErrEmpty")
}

func TestLoadReportFromReader_ReaderError(t *testing.T) {
	t.Parallel()

	_, _, err := auditlog.LoadReportFromReader(&errorReader{}, auditlog.FormatJSON)
	if err == nil {
		t.Fatal("expected error from failing reader")
	}
}

func TestLoadReportFromBytes_AutoDetectJSON(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("postgres://localhost")

	var buf bytes.Buffer

	err := p.WriteReportJSON(&buf)
	if err != nil {
		t.Fatalf("WriteReportJSON: %v", err)
	}

	report, format := loadReportOK(t, buf.Bytes(), auditlog.FormatAuto)

	if format != auditlog.FormatJSON {
		t.Errorf("format: want JSON, got %v", format)
	}

	assertReportServiceCount(t, report)
}

func TestLoadReportFromBytes_AutoDetectNDJSON(t *testing.T) {
	t.Parallel()

	p, _ := setupWithDB("postgres://localhost")

	var buf bytes.Buffer

	err := p.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	report, format := loadReportOK(t, buf.Bytes(), auditlog.FormatAuto)

	if format != auditlog.FormatNDJSON {
		t.Errorf("format: want NDJSON, got %v", format)
	}

	assertReportServiceCount(t, report)
}

func TestLoadReportFromBytes_EmptyInput(t *testing.T) {
	t.Parallel()

	_, _, err := auditlog.LoadReportFromBytes(nil, auditlog.FormatAuto)
	assertErrIs(t, err, auditlog.ErrEmpty, "expected ErrEmpty")
}

func TestLoadReportFromBytes_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, _, err := auditlog.LoadReportFromBytes([]byte("{not json"), auditlog.FormatJSON)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadReportFromBytes_InvalidNDJSON(t *testing.T) {
	t.Parallel()

	_, _, err := auditlog.LoadReportFromBytes([]byte("{broken"), auditlog.FormatNDJSON)
	if err == nil {
		t.Fatal("expected error for invalid NDJSON")
	}
}

func TestLoadReportFromBytes_InvalidFormat(t *testing.T) {
	t.Parallel()

	_, _, err := auditlog.LoadReportFromBytes([]byte("test"), auditlog.Format(99))
	if err == nil {
		t.Fatal("expected error for invalid format value")
	}
}

func TestLoadReportFromBytes_MultiLineJSONReport(t *testing.T) {
	t.Parallel()

	// Multi-line pretty-printed JSON Report (not single-line).
	multiline := []byte("{\n  \"version\": \"0.2.0\",\n  \"services\": [],\n  \"scope_tree\": {}\n}")

	report, format, err := auditlog.LoadReportFromBytes(multiline, auditlog.FormatAuto)
	if err != nil {
		t.Fatalf("LoadReportFromBytes multiline: %v", err)
	}

	if format != auditlog.FormatJSON {
		t.Errorf("format: want JSON for multiline, got %v", format)
	}

	if report.Version != "0.3.0" {
		t.Errorf("version: want %q, got %q", "0.3.0", report.Version)
	}
}

func TestLoadReportFromBytes_NDJSONNoVersionOrEventType(t *testing.T) {
	t.Parallel()

	// Single-line JSON object without "version" or "event_type" — should
	// default to NDJSON format detection.
	input := []byte(
		`{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","phase":"after","container_id":"c","scope_id":"s","scope_name":"s","service_name":"x"}` + "\n",
	)

	_, format, err := auditlog.LoadReportFromBytes(input, auditlog.FormatAuto)
	if err != nil {
		t.Fatalf("LoadReportFromBytes: %v", err)
	}

	if format != auditlog.FormatNDJSON {
		t.Errorf("format: want NDJSON, got %v", format)
	}
}

func TestLoadReportFromReader_EmptyNDJSON(t *testing.T) {
	t.Parallel()

	_, _, err := auditlog.LoadReportFromReader(strings.NewReader(""), auditlog.FormatNDJSON)
	if !errors.Is(err, auditlog.ErrEmpty) {
		t.Errorf("expected ErrEmpty, got %v", err)
	}
}

// --- Fuzz target ---

package auditlog_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestReadEvents_NoTrailingNewline(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := plugin.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	// Remove trailing newline if present.
	input := strings.TrimRight(buf.String(), "\n")
	if input == buf.String() {
		input += `{"sequence":999,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"extra"}`
	}

	events, err := auditlog.ReadEvents(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadEvents without trailing newline: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("expected events")
	}
}

func TestReadEvents_OnlyBlankLines(t *testing.T) {
	t.Parallel()

	_, err := auditlog.ReadEvents(strings.NewReader("\n\n\n"))
	if !errors.Is(err, auditlog.ErrNoEvents) {
		t.Errorf("expected ErrNoEvents, got %v", err)
	}
}

func TestLoadReport_WithForcedJSONFormat(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := plugin.WriteReportJSON(&buf)
	if err != nil {
		t.Fatalf("WriteReportJSON: %v", err)
	}

	report, format, err := auditlog.LoadReportFromReader(&buf, auditlog.FormatJSON)
	if err != nil {
		t.Fatalf("LoadReportFromReader with FormatJSON: %v", err)
	}

	if format != auditlog.FormatJSON {
		t.Errorf("format: want JSON, got %v", format)
	}

	if len(report.Services) != 1 {
		t.Errorf("service count: want 1, got %d", len(report.Services))
	}
}

func TestLoadReport_WithForcedNDJSONFormat(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := plugin.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	report, format, err := auditlog.LoadReportFromReader(&buf, auditlog.FormatNDJSON)
	if err != nil {
		t.Fatalf("LoadReportFromReader with FormatNDJSON: %v", err)
	}

	if format != auditlog.FormatNDJSON {
		t.Errorf("format: want NDJSON, got %v", format)
	}

	if len(report.Services) != 1 {
		t.Errorf("service count: want 1, got %d", len(report.Services))
	}
}

func TestLoadReportFromBytes_AutoDetectJSON(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := plugin.WriteReportJSON(&buf)
	if err != nil {
		t.Fatalf("WriteReportJSON: %v", err)
	}

	report, format, err := auditlog.LoadReportFromBytes(buf.Bytes(), auditlog.FormatAuto)
	if err != nil {
		t.Fatalf("LoadReportFromBytes: %v", err)
	}

	if format != auditlog.FormatJSON {
		t.Errorf("format: want JSON, got %v", format)
	}

	if len(report.Services) != 1 {
		t.Errorf("service count: want 1, got %d", len(report.Services))
	}
}

func TestLoadReportFromBytes_AutoDetectNDJSON(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := plugin.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	report, format, err := auditlog.LoadReportFromBytes(buf.Bytes(), auditlog.FormatAuto)
	if err != nil {
		t.Fatalf("LoadReportFromBytes: %v", err)
	}

	if format != auditlog.FormatNDJSON {
		t.Errorf("format: want NDJSON, got %v", format)
	}

	if len(report.Services) != 1 {
		t.Errorf("service count: want 1, got %d", len(report.Services))
	}
}

func TestLoadReportFromBytes_EmptyInput(t *testing.T) {
	t.Parallel()

	_, _, err := auditlog.LoadReportFromBytes([]byte{}, auditlog.FormatAuto)
	if !errors.Is(err, auditlog.ErrEmpty) {
		t.Errorf("expected ErrEmpty, got %v", err)
	}
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

func TestLoadReportFromNDJSON_EmptyReader(t *testing.T) {
	t.Parallel()

	_, err := auditlog.LoadReportFromNDJSON(strings.NewReader(""))
	if !errors.Is(err, auditlog.ErrEmpty) {
		t.Errorf("expected ErrEmpty, got %v", err)
	}
}

func TestLoadReportWithFormat_Option(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	dir := t.TempDir()
	path := dir + "/report.json"

	err := plugin.ExportToFile(path)
	if err != nil {
		t.Fatalf("ExportToFile: %v", err)
	}

	report, _, err := auditlog.LoadReport(path, auditlog.WithFormat(auditlog.FormatJSON))
	if err != nil {
		t.Fatalf("LoadReport with WithFormat: %v", err)
	}

	if len(report.Services) != 1 {
		t.Errorf("service count: want 1, got %d", len(report.Services))
	}
}

func TestReplayEvents_RegistrationOnly(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")

	var buf bytes.Buffer

	err := plugin.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	events, err := auditlog.ReadEvents(&buf)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	// Registered but never invoked.
	svc := findServiceByName(t, report, "db")
	if svc == nil {
		t.Fatal("db service not found")
	}

	if svc.Status != auditlog.ServiceStatusRegistered {
		t.Errorf("status: want %q, got %q", auditlog.ServiceStatusRegistered, svc.Status)
	}
}

func TestReplayEvents_PreservesContainerID(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjectorWithID("my-container")
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := plugin.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	events, err := auditlog.ReadEvents(&buf)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	if report.ContainerID != "my-container" {
		t.Errorf("container ID: want %q, got %q", "my-container", report.ContainerID)
	}
}

func TestReadEvents_OversizedLine(t *testing.T) {
	t.Parallel()

	huge := strings.Repeat("x", 2<<20) // 2MB, exceeds 1MB limit

	_, err := auditlog.ReadEvents(strings.NewReader(huge))
	if err == nil {
		t.Fatal("expected error for oversized line")
	}
}

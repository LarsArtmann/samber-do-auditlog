package auditlog_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

func TestReplayEvents_EmptyInput(t *testing.T) {
	t.Parallel()

	_, err := auditlog.ReplayEvents(nil)
	if err == nil {
		t.Fatal("expected error for empty events")
	}
}

func TestReplayEvents_SingleRegistration(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	report := plugin.Report()

	// Round-trip: export events as NDJSON, read back, replay.
	var buf bytes.Buffer

	err := plugin.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	events, err := auditlog.ReadEvents(&buf)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	replayed, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	assertReportServiceCount(t, replayed)
	assertReportValid(t, replayed, "replayed")

	if !replayed.Reconstructed {
		t.Error("expected Reconstructed=true on replayed report")
	}

	// Events should round-trip exactly.
	if len(replayed.Events) != len(report.Events) {
		t.Errorf("event count: original=%d, replayed=%d", len(report.Events), len(replayed.Events))
	}
}

func TestReplayEvents_DependencyChain(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	provideCache(injector, "cache")
	provideUserServiceWithDeps(injector, "users", "db", "cache")
	do.MustInvokeNamed[*UserService](injector, "users")

	var buf bytes.Buffer
	if err := plugin.WriteEventsNDJSON(&buf); err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	events, err := auditlog.ReadEvents(&buf)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	replayed, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	// UserService should depend on both db and cache.
	users := findServiceByName(t, replayed, "users")
	if users == nil {
		t.Fatal("users service not found")
	}

	if len(users.Dependencies) != 2 {
		t.Errorf("users deps: want 2, got %d", len(users.Dependencies))
	}

	// db and cache should list users as a dependent.
	db := findServiceByName(t, replayed, "db")
	if db == nil {
		t.Fatal("db service not found")
	}

	if len(db.Dependents) != 1 {
		t.Errorf("db dependents: want 1, got %d", len(db.Dependents))
	}
}

func TestReplayEvents_HealthCheckCount(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjectorWithID("test-hc")
	provideHealthyDB(injector, "db", "dsn")
	do.MustInvokeNamed[*HealthyDB](injector, "db")

	hcErrors := plugin.RecordHealthCheck(injector)
	for _, hcErr := range hcErrors {
		if hcErr != nil {
			t.Fatalf("RecordHealthCheck error: %v", hcErr)
		}
	}

	var buf bytes.Buffer
	if err := plugin.WriteEventsNDJSON(&buf); err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	events, err := auditlog.ReadEvents(&buf)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	replayed, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, replayed, "db")
	if svc == nil {
		t.Fatal("db not found")
	}

	if svc.HealthCheckCount != 1 {
		t.Errorf("health check count: want 1, got %d", svc.HealthCheckCount)
	}
}

func TestReadEvents_Roundtrip(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	provideCache(injector, "cache")
	do.MustInvokeNamed[*Database](injector, "db")
	do.MustInvokeNamed[*Cache](injector, "cache")

	var buf bytes.Buffer
	if err := plugin.WriteEventsNDJSON(&buf); err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	original := plugin.Events()

	events, err := auditlog.ReadEvents(&buf)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	if len(events) != len(original) {
		t.Fatalf("event count: original=%d, read=%d", len(original), len(events))
	}

	for i, evt := range events {
		if evt.ServiceName != original[i].ServiceName {
			t.Errorf("event %d: service name mismatch: %q vs %q",
				i, evt.ServiceName, original[i].ServiceName)
		}

		if evt.EventType != original[i].EventType {
			t.Errorf("event %d: type mismatch: %q vs %q",
				i, evt.EventType, original[i].EventType)
		}
	}
}

func TestReadEvents_EmptyInput(t *testing.T) {
	t.Parallel()

	_, err := auditlog.ReadEvents(strings.NewReader(""))
	if !errors.Is(err, auditlog.ErrEmpty) {
		t.Errorf("expected ErrEmpty, got %v", err)
	}
}

func TestReadEvents_BlankLinesSkipped(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer
	if err := plugin.WriteEventsNDJSON(&buf); err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	// Insert blank lines between events.
	lines := strings.Split(buf.String(), "\n")

	var polluted strings.Builder

	for _, line := range lines {
		if line != "" {
			polluted.WriteString(line)
			polluted.WriteString("\n")
			polluted.WriteString("\n")
		}
	}

	events, err := auditlog.ReadEvents(strings.NewReader(polluted.String()))
	if err != nil {
		t.Fatalf("ReadEvents with blank lines: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("expected events despite blank lines")
	}
}

func TestReadEvents_MalformedJSON(t *testing.T) {
	t.Parallel()

	input := `{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db"}
{BROKEN JSON}`

	_, err := auditlog.ReadEvents(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}

	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error should reference line 2: %v", err)
	}
}

func TestLoadReport_JSONFile(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	if err := plugin.ExportToFile(path); err != nil {
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

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	dir := t.TempDir()
	path := filepath.Join(dir, "events.ndjson")

	if err := plugin.ExportEventsToNDJSON(path); err != nil {
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

func TestLoadReportFromJSON_DirectBytes(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer
	if err := plugin.WriteReportJSON(&buf); err != nil {
		t.Fatalf("WriteReportJSON: %v", err)
	}

	report, err := auditlog.LoadReportFromJSON(buf.Bytes())
	if err != nil {
		t.Fatalf("LoadReportFromJSON: %v", err)
	}

	assertReportServiceCount(t, report)
}

func TestLoadReportFromReader_AutoDetectJSON(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer
	if err := plugin.WriteReportJSON(&buf); err != nil {
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

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer
	if err := plugin.WriteEventsNDJSON(&buf); err != nil {
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
	if !errors.Is(err, auditlog.ErrEmpty) {
		t.Errorf("expected ErrEmpty, got %v", err)
	}
}

func TestLoadReportFromReader_ForcedJSON(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer
	if err := plugin.WriteReportJSON(&buf); err != nil {
		t.Fatalf("WriteReportJSON: %v", err)
	}

	report, format, err := auditlog.LoadReportFromReader(&buf, auditlog.FormatJSON)
	if err != nil {
		t.Fatalf("LoadReportFromReader: %v", err)
	}

	if format != auditlog.FormatJSON {
		t.Errorf("format: want JSON, got %v", format)
	}

	assertReportServiceCount(t, report)
}

func TestLoadReportFromReader_ForcedNDJSON(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer
	if err := plugin.WriteEventsNDJSON(&buf); err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	report, format, err := auditlog.LoadReportFromReader(&buf, auditlog.FormatNDJSON)
	if err != nil {
		t.Fatalf("LoadReportFromReader: %v", err)
	}

	if format != auditlog.FormatNDJSON {
		t.Errorf("format: want NDJSON, got %v", format)
	}

	assertReportServiceCount(t, report)
}

func TestReplayEvents_FullLifecycle(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	provideCache(injector, "cache")
	provideUserServiceWithDeps(injector, "users", "db", "cache")

	do.MustInvokeNamed[*UserService](injector, "users")

	_ = injector.Shutdown()

	var buf bytes.Buffer
	if err := plugin.WriteEventsNDJSON(&buf); err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	events, err := auditlog.ReadEvents(&buf)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	replayed, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	assertReportValid(t, replayed, "replayed")

	// Check shutdown status propagated.
	db := findServiceByName(t, replayed, "db")
	if db == nil {
		t.Fatal("db not found")
	}

	if db.Status != auditlog.ServiceStatusShutdown {
		t.Errorf("db status: want %q, got %q", auditlog.ServiceStatusShutdown, db.Status)
	}
}

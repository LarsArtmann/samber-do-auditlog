package auditlog_test

import (
	"bufio"
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// --- Replay engine tests ---

// mkEvent constructs an auditlog.Event for replay tests. Reduces verbose
// struct literals to a single call with the most-commonly-varied fields
// as parameters; remaining fields (DurationMs, Error) are set on the
// returned value by the caller when needed.
func mkEvent(
	seq int,
	ts time.Time,
	eventType auditlog.EventType,
	phase auditlog.Phase,
	scopeID, scopeName, serviceName, containerID string,
	svcType auditlog.ProviderType,
) auditlog.Event {
	return auditlog.Event{
		ServiceRef:  auditlog.ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName},
		Sequence:    seq,
		Timestamp:   ts,
		EventType:   eventType,
		Phase:       phase,
		ContainerID: containerID,
		ServiceType: svcType,
	}
}

// rootRef returns a ServiceRef for a root-scope service.
func rootRef(serviceName string) auditlog.ServiceRef {
	return auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: serviceName}
}

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
	assertReportValid(t, replayed, "single registration replayed")

	if !replayed.Reconstructed {
		t.Error("expected Reconstructed=true on replayed report")
	}

	if len(replayed.Events) != len(plugin.Report().Events) {
		t.Errorf("event count mismatch")
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

	svc := findServiceByName(t, report, "db")
	if svc == nil {
		t.Fatal("db service not found")
	}

	if svc.Status != auditlog.ServiceStatusRegistered {
		t.Errorf("status: want %q, got %q", auditlog.ServiceStatusRegistered, svc.Status)
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

	users := findServiceByName(t, replayed, "users")
	if users == nil {
		t.Fatal("users service not found")
	}

	if len(users.Dependencies) != 2 {
		t.Errorf("users deps: want 2, got %d", len(users.Dependencies))
	}

	db := findServiceByName(t, replayed, "db")
	if db == nil {
		t.Fatal("db service not found")
	}

	if len(db.Dependents) != 1 {
		t.Errorf("db dependents: want 1, got %d", len(db.Dependents))
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

func TestReplayEvents_ContainerIDFromEvents(t *testing.T) {
	t.Parallel()

	events := []auditlog.Event{
		{
			ServiceRef: auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:   1, Timestamp: time.Now(),
			EventType: auditlog.EventTypeRegistration, Phase: auditlog.PhaseAfter,
			ContainerID: "from-event", ServiceType: auditlog.ProviderTypeLazy,
		},
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	if report.ContainerID != "from-event" {
		t.Errorf("container ID: want %q, got %q", "from-event", report.ContainerID)
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

	svc := findServiceByName(t, replayed, "db")
	if svc == nil {
		t.Fatal("db not found")
	}

	if svc.HealthCheckCount != 1 {
		t.Errorf("health check count: want 1, got %d", svc.HealthCheckCount)
	}
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

	assertReportValid(t, replayed, "full lifecycle replayed")

	db := findServiceByName(t, replayed, "db")
	if db == nil {
		t.Fatal("db not found")
	}

	if db.Status != auditlog.ServiceStatusShutdown {
		t.Errorf("db status: want %q, got %q", auditlog.ServiceStatusShutdown, db.Status)
	}
}

func TestReplayEvents_ShutdownErrorPreserved(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideCrashing(injector, "crasher")
	do.MustInvokeNamed[*CrashingService](injector, "crasher")

	_ = injector.Shutdown()

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

	svc := findServiceByName(t, replayed, "crasher")
	if svc == nil {
		t.Fatal("crasher not found")
	}

	if svc.Status != auditlog.ServiceStatusShutdownError {
		t.Errorf("status: want %q, got %q", auditlog.ServiceStatusShutdownError, svc.Status)
	}
}

func TestReplayEvents_ManualShutdownWithoutBefore(t *testing.T) {
	t.Parallel()

	dur := 5.5
	errMsg := "shutdown failed"

	events := []auditlog.Event{
		{
			ServiceRef: auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:   1, Timestamp: time.Now(),
			EventType: auditlog.EventTypeRegistration, Phase: auditlog.PhaseAfter,
			ContainerID: "test", ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef: auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:   2, Timestamp: time.Now(),
			EventType: auditlog.EventTypeShutdown, Phase: auditlog.PhaseAfter,
			ContainerID: "test", ServiceType: auditlog.ProviderTypeLazy,
			DurationMs: &dur, Error: &errMsg,
		},
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "svc")
	if svc == nil {
		t.Fatal("svc not found")
	}

	if svc.Status != auditlog.ServiceStatusShutdownError {
		t.Errorf("status: want %q, got %q", auditlog.ServiceStatusShutdownError, svc.Status)
	}
}

func TestReplayEvents_ManualHealthCheckNewService(t *testing.T) {
	t.Parallel()

	// Health check event for a service that was never registered.
	events := []auditlog.Event{
		{
			ServiceRef: auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "ghost"},
			Sequence:   1, Timestamp: time.Now(),
			EventType: auditlog.EventTypeHealthCheck, Phase: auditlog.PhaseAfter,
			ContainerID: "test",
		},
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "ghost")
	if svc == nil {
		t.Fatal("ghost not found")
	}

	if svc.HealthCheckCount != 1 {
		t.Errorf("health check count: want 1, got %d", svc.HealthCheckCount)
	}
}

func TestReplayEvents_RegistrationOverwriteType(t *testing.T) {
	t.Parallel()

	// Two registration-after events for the same service — the second
	// should update the service type.
	events := []auditlog.Event{
		{
			ServiceRef: auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:   1, Timestamp: time.Now(),
			EventType: auditlog.EventTypeRegistration, Phase: auditlog.PhaseAfter,
			ContainerID: "test", ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef: auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:   2, Timestamp: time.Now(),
			EventType: auditlog.EventTypeRegistration, Phase: auditlog.PhaseAfter,
			ContainerID: "test", ServiceType: auditlog.ProviderTypeEager,
		},
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "svc")
	if svc == nil {
		t.Fatal("svc not found")
	}

	if svc.ServiceType != auditlog.ProviderTypeEager {
		t.Errorf("service type: want %q, got %q", auditlog.ProviderTypeEager, svc.ServiceType)
	}
}

func TestReplayEvents_MultiScopeDeterministic(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjectorWithID("multi-scope")

	provideDB(injector, "root-db", "postgres://root")
	provideCache(injector, "root-cache")

	childScope := injector.Scope("drivers")
	provideUserServiceWithDB(childScope, "driver-service", "root-db")

	do.MustInvokeNamed[*Database](injector, "root-db")
	do.MustInvokeNamed[*Cache](injector, "root-cache")
	do.MustInvokeNamed[*UserService](childScope, "driver-service")

	var buf bytes.Buffer

	err := plugin.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

	events, err := auditlog.ReadEvents(&buf)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	report1, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents (1): %v", err)
	}

	report2, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents (2): %v", err)
	}

	if report1.ScopeCount != report2.ScopeCount {
		t.Errorf("scope count differs: %d vs %d", report1.ScopeCount, report2.ScopeCount)
	}

	if report1.ScopeTree.ID != report2.ScopeTree.ID {
		t.Errorf("root scope ID differs")
	}

	if report1.ScopeCount < 2 {
		t.Errorf("expected >=2 scopes, got %d", report1.ScopeCount)
	}

	driverSvc := findServiceByName(t, report1, "driver-service")
	if driverSvc == nil {
		t.Fatal("driver-service not found")
	}

	foundDep := false

	for _, dep := range driverSvc.Dependencies {
		if dep.ServiceName == "root-db" {
			foundDep = true
		}
	}

	if !foundDep {
		t.Error("driver-service should depend on root-db")
	}
}

func TestReplayEvents_OutOfOrderStackPop(t *testing.T) {
	t.Parallel()

	// Interleaved invocations: A starts, B starts, B ends, A ends.
	// The stack pop for A is NOT the last element (non-LIFO path).
	events := []auditlog.Event{
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "a"},
			Sequence:    1,
			Timestamp:   time.Now(),
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "b"},
			Sequence:    2,
			Timestamp:   time.Now(),
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "a"},
			Sequence:    3,
			Timestamp:   time.Now(),
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "b"},
			Sequence:    4,
			Timestamp:   time.Now(),
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		// B finishes first (LIFO pop), then A finishes (non-LIFO: index < len-1).
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "b"},
			Sequence:    5,
			Timestamp:   time.Now(),
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "a"},
			Sequence:    6,
			Timestamp:   time.Now(),
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svcA := findServiceByName(t, report, "a")
	if svcA == nil {
		t.Fatal("a not found")
	}

	if svcA.InvocationCount != 1 {
		t.Errorf("a invocation count: want 1, got %d", svcA.InvocationCount)
	}
}

// --- NDJSON reader tests ---

func TestReadEvents_Roundtrip(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	provideCache(injector, "cache")
	do.MustInvokeNamed[*Database](injector, "db")
	do.MustInvokeNamed[*Cache](injector, "cache")

	var buf bytes.Buffer

	err := plugin.WriteEventsNDJSON(&buf)
	if err != nil {
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
}

func TestReadEvents_EmptyInput(t *testing.T) {
	t.Parallel()

	_, err := auditlog.ReadEvents(strings.NewReader(""))
	if !errors.Is(err, auditlog.ErrEmpty) {
		t.Errorf("expected ErrEmpty, got %v", err)
	}
}

func TestReadEvents_OnlyBlankLines(t *testing.T) {
	t.Parallel()

	_, err := auditlog.ReadEvents(strings.NewReader("\n\n\n"))
	if !errors.Is(err, auditlog.ErrNoEvents) {
		t.Errorf("expected ErrNoEvents, got %v", err)
	}
}

func TestReadEvents_BlankLinesSkipped(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := plugin.WriteEventsNDJSON(&buf)
	if err != nil {
		t.Fatalf("WriteEventsNDJSON: %v", err)
	}

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

func TestReadEvents_OversizedLine(t *testing.T) {
	t.Parallel()

	huge := strings.Repeat("x", 2<<20)

	_, err := auditlog.ReadEvents(strings.NewReader(huge))
	if !errors.Is(err, auditlog.ErrOversizedLine) {
		t.Errorf("expected ErrOversizedLine, got %v", err)
	}
}

func TestReadEvents_NoTrailingNewline(t *testing.T) {
	t.Parallel()

	input := `{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db"}`

	events, err := auditlog.ReadEvents(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("event count: want 1, got %d", len(events))
	}
}

func TestReadEvents_LeadingWhitespace(t *testing.T) {
	t.Parallel()

	input := "  " + `{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db"}` + "\n"

	events, err := auditlog.ReadEvents(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("event count: want 1, got %d", len(events))
	}
}

func TestReadEvents_CarriageReturns(t *testing.T) {
	t.Parallel()

	// Line with trailing \r (Windows-style).
	input := `{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db"}` + "\r\n"

	events, err := auditlog.ReadEvents(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadEvents with \\r\\n: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("event count: want 1, got %d", len(events))
	}
}

// --- Loader API tests ---

func TestLoadReport_JSONFile(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	err := plugin.ExportToFile(path)
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

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	dir := t.TempDir()
	path := filepath.Join(dir, "events.ndjson")

	err := plugin.ExportEventsToNDJSON(path)
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

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	err := plugin.ExportToFile(path)
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

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := plugin.WriteReportJSON(&buf)
	if err != nil {
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

	err := plugin.WriteReportJSON(&buf)
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

	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db", "postgres://localhost")
	do.MustInvokeNamed[*Database](injector, "db")

	var buf bytes.Buffer

	err := plugin.WriteEventsNDJSON(&buf)
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
	if !errors.Is(err, auditlog.ErrEmpty) {
		t.Errorf("expected ErrEmpty, got %v", err)
	}
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

	assertReportServiceCount(t, report)
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

	assertReportServiceCount(t, report)
}

func TestLoadReportFromBytes_EmptyInput(t *testing.T) {
	t.Parallel()

	_, _, err := auditlog.LoadReportFromBytes(nil, auditlog.FormatAuto)
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

	if report.Version != "0.2.0" {
		t.Errorf("version: want %q, got %q", "0.2.0", report.Version)
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

func TestLoadReportFromNDJSON_EmptyReader(t *testing.T) {
	t.Parallel()

	_, err := auditlog.LoadReportFromNDJSON(strings.NewReader(""))
	if !errors.Is(err, auditlog.ErrEmpty) {
		t.Errorf("expected ErrEmpty, got %v", err)
	}
}

// --- Fuzz target ---

func FuzzReadEvents(f *testing.F) {
	f.Add(
		[]byte(
			`{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db"}` + "\n",
		),
	)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic on arbitrary input.
		events, err := auditlog.ReadEvents(bytes.NewReader(data))
		if err != nil {
			return
		}

		// If events were parsed, replay should not panic either.
		_, _ = auditlog.ReplayEvents(events)
	})
}

// --- Diff coverage tests ---

func TestDiff_MultipleAddedRemoved(t *testing.T) {
	t.Parallel()

	pluginA, injectorA := newPluginAndInjector()
	provideDB(injectorA, "db-a", "postgres://a")
	provideCache(injectorA, "cache-a")
	provideString(injectorA, "str-a", "value-a")

	reportA := pluginA.Report()

	pluginB, injectorB := newPluginAndInjector()
	provideDB(injectorB, "db-b", "postgres://b")
	provideCache(injectorB, "cache-b")
	provideString(injectorB, "str-b", "value-b")
	provideString(injectorB, "extra", "extra")

	reportB := pluginB.Report()

	diff := reportA.Diff(reportB)

	if diff.IsEmpty() {
		t.Fatal("expected non-empty diff")
	}

	if len(diff.AddedServices) < 3 {
		t.Errorf("expected >=3 added, got %d", len(diff.AddedServices))
	}

	if len(diff.RemovedServices) < 3 {
		t.Errorf("expected >=3 removed, got %d", len(diff.RemovedServices))
	}
}

func TestDiff_MultipleChanged(t *testing.T) {
	t.Parallel()

	ref1 := auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc-a"}
	ref2 := auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc-b"}
	now := time.Now()

	eventsA := []auditlog.Event{
		{
			ServiceRef:  ref1,
			Sequence:    1,
			Timestamp:   now,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref2,
			Sequence:    2,
			Timestamp:   now,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref1,
			Sequence:    3,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  ref1,
			Sequence:    4,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref2,
			Sequence:    5,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  ref2,
			Sequence:    6,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref1,
			Sequence:    7,
			Timestamp:   now,
			EventType:   auditlog.EventTypeHealthCheck,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
		},
		{
			ServiceRef:  ref2,
			Sequence:    8,
			Timestamp:   now,
			EventType:   auditlog.EventTypeHealthCheck,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
		},
	}

	eventsB := []auditlog.Event{
		{
			ServiceRef:  ref1,
			Sequence:    1,
			Timestamp:   now,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref2,
			Sequence:    2,
			Timestamp:   now,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref1,
			Sequence:    3,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  ref1,
			Sequence:    4,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  ref2,
			Sequence:    5,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  ref2,
			Sequence:    6,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
	}

	reportA, err := auditlog.ReplayEvents(eventsA)
	if err != nil {
		t.Fatalf("ReplayEvents A: %v", err)
	}

	reportB, err := auditlog.ReplayEvents(eventsB)
	if err != nil {
		t.Fatalf("ReplayEvents B: %v", err)
	}

	diff := reportA.Diff(reportB)

	if len(diff.ChangedServices) < 2 {
		t.Errorf("expected >=2 changed services, got %d", len(diff.ChangedServices))
	}
}

// --- Test helpers ---

type errorReader struct{}

func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, errReaderFailed
}

var errReaderFailed = errors.New("simulated read failure")

// Ensure bufio is used (referenced in oversized line tests via scanner internals).
var _ = bufio.ErrTooLong

// --- Targeted branch coverage tests ---

// TestReplayEvents_NonLIFOStackPop exercises the non-LIFO branch in
// applyInvocationAfter where i < len(stack)-1 (middle-of-stack pop).
func TestReplayEvents_NonLIFOStackPop(t *testing.T) {
	t.Parallel()

	now := time.Now()

	// Stack: push A, push B. Pop A FIRST (not last), then B.
	// This triggers the append(stack[:i], stack[i+1:]...) path.
	events := []auditlog.Event{
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "a"},
			Sequence:    1,
			Timestamp:   now,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "b"},
			Sequence:    2,
			Timestamp:   now,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "a"},
			Sequence:    3,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "b"},
			Sequence:    4,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		// A finishes while B is still on stack — pops from middle.
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "a"},
			Sequence:    5,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "b"},
			Sequence:    6,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svcA := findServiceByName(t, report, "a")
	if svcA == nil {
		t.Fatal("a not found")
	}

	if svcA.InvocationCount != 1 {
		t.Errorf("a invocation count: want 1, got %d", svcA.InvocationCount)
	}
}

// TestReplayEvents_DoubleInvocation covers the firstInvokedAt-already-set
// branch in applyInvocationAfter.
func TestReplayEvents_DoubleInvocation(t *testing.T) {
	t.Parallel()

	now := time.Now()

	dur := 3.3

	events := []auditlog.Event{
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:    1,
			Timestamp:   now,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		// First invocation.
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:    2,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:    3,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
			DurationMs:  &dur,
		},
		// Second invocation — firstInvokedAt already set, firstBuildDurationMs already set.
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:    4,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:    5,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
			DurationMs:  &dur,
		},
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "svc")
	if svc == nil {
		t.Fatal("svc not found")
	}

	if svc.InvocationCount != 2 {
		t.Errorf("invocation count: want 2, got %d", svc.InvocationCount)
	}
}

// TestReplayEvents_ShutdownWithMatchingBefore covers the shutdownStart
// path in applyShutdownAfter where a matching before event exists.
func TestReplayEvents_ShutdownWithMatchingBefore(t *testing.T) {
	t.Parallel()

	t0 := time.Now()
	t1 := t0.Add(5 * time.Millisecond)

	events := []auditlog.Event{
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:    1,
			Timestamp:   t0,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:    2,
			Timestamp:   t0,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:    3,
			Timestamp:   t0,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
		// Shutdown with matching before event.
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:    4,
			Timestamp:   t0,
			EventType:   auditlog.EventTypeShutdown,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:    5,
			Timestamp:   t1,
			EventType:   auditlog.EventTypeShutdown,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "svc")
	if svc == nil {
		t.Fatal("svc not found")
	}

	if svc.Status != auditlog.ServiceStatusShutdown {
		t.Errorf("status: want %q, got %q", auditlog.ServiceStatusShutdown, svc.Status)
	}

	if svc.ShutdownDurationMs == nil {
		t.Error("expected non-nil shutdown duration from matching before")
	}
}

// TestReplayEvents_InvocationWithoutRegistration covers the !ok branch
// in applyInvocationAfter where a service is invoked without prior registration.
func TestReplayEvents_InvocationWithoutRegistration(t *testing.T) {
	t.Parallel()

	now := time.Now()
	dur := 2.0

	events := []auditlog.Event{
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "ghost"},
			Sequence:    1,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "ghost"},
			Sequence:    2,
			Timestamp:   now,
			EventType:   auditlog.EventTypeInvocation,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
			DurationMs:  &dur,
		},
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "ghost")
	if svc == nil {
		t.Fatal("ghost not found")
	}

	if svc.InvocationCount != 1 {
		t.Errorf("invocation count: want 1, got %d", svc.InvocationCount)
	}
}

// TestReplayEvents_ShutdownWithoutRegistration covers the !ok branch
// in applyShutdownAfter where a service is shut down without prior registration.
func TestReplayEvents_ShutdownWithoutRegistration(t *testing.T) {
	t.Parallel()

	now := time.Now()

	events := []auditlog.Event{
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "phantom"},
			Sequence:    1,
			Timestamp:   now,
			EventType:   auditlog.EventTypeShutdown,
			Phase:       auditlog.PhaseBefore,
			ContainerID: "c",
		},
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "phantom"},
			Sequence:    2,
			Timestamp:   now.Add(time.Millisecond),
			EventType:   auditlog.EventTypeShutdown,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "c",
			ServiceType: auditlog.ProviderTypeLazy,
		},
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	svc := findServiceByName(t, report, "phantom")
	if svc == nil {
		t.Fatal("phantom not found")
	}

	if svc.Status != auditlog.ServiceStatusShutdown {
		t.Errorf("status: want %q, got %q", auditlog.ServiceStatusShutdown, svc.Status)
	}
}

// TestReplayEvents_EmptyContainerID covers the containerID fallback
// path in ReplayEvents.
func TestReplayEvents_EmptyContainerID(t *testing.T) {
	t.Parallel()

	now := time.Now()

	events := []auditlog.Event{
		{
			ServiceRef:  auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:    1,
			Timestamp:   now,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseAfter,
			ContainerID: "",
			ServiceType: auditlog.ProviderTypeLazy,
		},
	}

	report, err := auditlog.ReplayEvents(events)
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}

	// ContainerID should be empty (from the event).
	if report.ContainerID != "" {
		t.Errorf("expected empty container ID, got %q", report.ContainerID)
	}
}

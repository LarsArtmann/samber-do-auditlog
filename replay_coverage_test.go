package auditlog_test

import (
	"bytes"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

// TestReplayEvents_MultiScopeFullLifecycle exercises the broadest set of replay
// code paths: multiple scopes, dependency chains across scopes, shutdown with
// errors, health checks, and existing-service-overwrite paths.
func TestReplayEvents_MultiScopeFullLifecycle(t *testing.T) {
	t.Parallel()

	plugin, injector := newPluginAndInjectorWithID("multi-scope")

	// Root scope services.
	provideDB(injector, "root-db", "postgres://root")
	provideCache(injector, "root-cache")

	// Child scope with cross-scope dependency on root-db.
	childScope := injector.Scope("drivers")
	provideUserServiceWithDB(childScope, "driver-service", "root-db")
	provideFailing(childScope, "unreliable")

	// Invoke root services.
	do.MustInvokeNamed[*Database](injector, "root-db")
	do.MustInvokeNamed[*Cache](injector, "root-cache")

	// Invoke child scope services (cross-scope dep on root-db).
	do.MustInvokeNamed[*UserService](childScope, "driver-service")

	_, invokeErr := do.InvokeNamed[*Database](childScope, "unreliable")
	if invokeErr != nil {
		// expected — provider returns error
	}

	// Health checks.
	hcErrors := plugin.RecordHealthCheck(injector)
	for _, hcErr := range hcErrors {
		if hcErr != nil {
			t.Fatalf("unexpected health check error: %v", hcErr)
		}
	}

	// Shutdown.
	_ = injector.Shutdown()

	// Round-trip through NDJSON.
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

	assertReportValid(t, report, "multi-scope replayed")

	if !report.Reconstructed {
		t.Error("expected Reconstructed=true")
	}

	if report.ContainerID != "multi-scope" {
		t.Errorf("container ID: want %q, got %q", "multi-scope", report.ContainerID)
	}

	// Multiple scopes should produce a non-trivial scope tree.
	if report.ScopeCount < 2 {
		t.Errorf("expected >=2 scopes, got %d", report.ScopeCount)
	}

	// At least 4 services across scopes.
	if report.ServiceCount < 4 {
		t.Errorf("expected >=4 services, got %d", report.ServiceCount)
	}

	// driver-service should depend on root-db (cross-scope dep).
	driverSvc := findServiceByName(t, report, "driver-service")
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

	// root-db should have driver-service as dependent.
	rootDB := findServiceByName(t, report, "root-db")
	if rootDB == nil {
		t.Fatal("root-db not found")
	}

	foundDependent := false

	for _, dep := range rootDB.Dependents {
		if dep.ServiceName == "driver-service" {
			foundDependent = true
		}
	}

	if !foundDependent {
		t.Error("root-db should have driver-service as dependent")
	}

	// Unreliable service should be in invocation_error status.
	unreliable := findServiceByName(t, report, "unreliable")
	if unreliable == nil {
		t.Fatal("unreliable not found")
	}

	if unreliable.Status != auditlog.ServiceStatusInvocationError {
		t.Errorf("unreliable status: want %q, got %q",
			auditlog.ServiceStatusInvocationError, unreliable.Status)
	}

	// Shutdown events should be present.
	if report.TotalShutdownDurationMs <= 0 {
		t.Error("expected non-zero shutdown duration")
	}
}

func TestReplayEvents_ScopeOrderingDeterministic(t *testing.T) {
	t.Parallel()

	// Run replay twice and verify the scope tree is identical.
	plugin, injector := newPluginAndInjector()
	provideDB(injector, "db1", "postgres://1")

	childScope := injector.Scope("child-a")
	provideCache(childScope, "cache1")

	do.MustInvokeNamed[*Database](injector, "db1")
	do.MustInvokeNamed[*Cache](childScope, "cache1")

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
		t.Errorf("root scope ID differs: %q vs %q", report1.ScopeTree.ID, report2.ScopeTree.ID)
	}
}

func TestReadEvents_LeadingWhitespace(t *testing.T) {
	t.Parallel()

	input := "  " + `{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db","service_type":"lazy"}` + "\n"

	events, err := auditlog.ReadEvents(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("event count: want 1, got %d", len(events))
	}
}

func TestReadEvents_TabSeparatedBlankLine(t *testing.T) {
	t.Parallel()

	input := "\t\n" + `{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db"}` + "\n"

	events, err := auditlog.ReadEvents(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("event count: want 1, got %d", len(events))
	}
}

func TestLoadReportFromReader_ReaderError(t *testing.T) {
	t.Parallel()

	_, _, err := auditlog.LoadReportFromReader(&errorReader{}, auditlog.FormatJSON)
	if err == nil {
		t.Fatal("expected error from failing reader")
	}
}

// errorReader is an io.Reader that always returns an error.
type errorReader struct{}

func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, errReaderFailed
}

var errReaderFailed = bytesError("simulated read failure")

func bytesError(msg string) error {
	return &simpleError{msg: msg}
}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }

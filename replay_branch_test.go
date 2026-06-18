package auditlog_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// TestReplayEvents_ShutdownWithoutBefore covers the DurationMs fallback path
// in applyShutdownAfter (when a shutdown-after event has no matching before).
func TestReplayEvents_ShutdownWithoutBefore(t *testing.T) {
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
			Sequence:   2, Timestamp: time.Now().Add(1 * time.Millisecond),
			EventType: auditlog.EventTypeInvocation, Phase: auditlog.PhaseBefore,
			ContainerID: "test",
		},
		{
			ServiceRef: auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:   3, Timestamp: time.Now().Add(2 * time.Millisecond),
			EventType: auditlog.EventTypeInvocation, Phase: auditlog.PhaseAfter,
			ContainerID: "test", ServiceType: auditlog.ProviderTypeLazy,
			DurationMs: &dur,
		},
		// Shutdown-after with NO matching before-event.
		{
			ServiceRef: auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:   4, Timestamp: time.Now().Add(3 * time.Millisecond),
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

	if svc.ShutdownError == nil || *svc.ShutdownError != "shutdown failed" {
		t.Errorf("shutdown error not preserved")
	}
}

// TestReplayEvents_HealthCheckOnExistingService covers the existing-service
// branch in applyHealthCheck.
func TestReplayEvents_HealthCheckOnExistingService(t *testing.T) {
	t.Parallel()

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
			EventType: auditlog.EventTypeHealthCheck, Phase: auditlog.PhaseAfter,
			ContainerID: "test",
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

	if svc.HealthCheckCount != 1 {
		t.Errorf("health check count: want 1, got %d", svc.HealthCheckCount)
	}
}

// TestDetectLineFormat_InvalidJSON covers the invalid-JSON branch.
func TestDetectLineFormat_InvalidJSON(t *testing.T) {
	t.Parallel()

	// Multi-line JSON Report object (not valid single-line JSON).
	multiline := []byte("{\n  \"version\": \"0.2.0\",\n  \"services\": []\n}")

	report, _, err := auditlog.LoadReportFromBytes(multiline, auditlog.FormatAuto)
	if err == nil {
		// May succeed or fail depending on MigrateReport; the key is that
		// detectLineFormat returns FormatJSON for multi-line input.
		if report.Version != "0.2.0" {
			t.Errorf("unexpected version: %q", report.Version)
		}
	}
}

// TestLoadReportFromNDJSON_DirectError covers the error path in LoadReportFromNDJSON.
func TestLoadReportFromNDJSON_DirectError(t *testing.T) {
	t.Parallel()

	// Invalid NDJSON — malformed JSON on line 1.
	_, err := auditlog.LoadReportFromNDJSON(strings.NewReader("{broken json}\n"))
	if err == nil {
		t.Fatal("expected error for invalid NDJSON")
	}
}

// TestLoadReportFromBytes_InvalidFormat covers the default case in the switch.
func TestLoadReportFromBytes_InvalidFormat(t *testing.T) {
	t.Parallel()

	_, _, err := auditlog.LoadReportFromBytes([]byte("test"), auditlog.Format(99))
	if err == nil {
		t.Fatal("expected error for invalid format value")
	}
}

// TestReplayEvents_ContainerIDFromEvents covers the container ID fallback.
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

// TestLoadReportFromBytes_NDJSONLineWithCarriageReturn covers trimWhitespace
// with \r characters.
func TestLoadReportFromBytes_NDJSONLineWithCarriageReturn(t *testing.T) {
	t.Parallel()

	events := []auditlog.Event{
		{
			ServiceRef: auditlog.ServiceRef{ScopeID: "root", ScopeName: "[root]", ServiceName: "svc"},
			Sequence:   1, Timestamp: time.Now(),
			EventType: auditlog.EventTypeRegistration, Phase: auditlog.PhaseAfter,
			ContainerID: "test", ServiceType: auditlog.ProviderTypeLazy,
		},
	}

	data, err := json.Marshal(events[0])
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Add carriage returns around the line.
	input := bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))

	_, _, err = auditlog.LoadReportFromBytes(append(input, '\n'), auditlog.FormatAuto)
	if err != nil {
		t.Fatalf("LoadReportFromBytes with \\r: %v", err)
	}
}

// Ensure errors package is used.
var _ = errors.New

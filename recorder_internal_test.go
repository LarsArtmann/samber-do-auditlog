package auditlog

import (
	"testing"
)

// TestNewRecorder verifies the exported constructor initializes a usable
// Recorder with the supplied container ID and event callback.
func TestNewRecorder(t *testing.T) {
	t.Parallel()

	var callbackCalled bool

	r := NewRecorder("recorder-test", func(_ Event) {
		callbackCalled = true
	})

	if r == nil {
		t.Fatal("NewRecorder returned nil")
	}

	if r.containerID != "recorder-test" {
		t.Errorf("containerID: want %q, got %q", "recorder-test", r.containerID)
	}

	if r.onEvent == nil {
		t.Error("onEvent callback was not stored")
	}

	if r.EventsCount() != 0 {
		t.Errorf("EventsCount: want 0, got %d", r.EventsCount())
	}

	if r.DroppedEventCount() != 0 {
		t.Errorf("DroppedEventCount: want 0, got %d", r.DroppedEventCount())
	}

	// Exercise the callback path to confirm it fires.
	r.onEvent(Event{ContainerID: "recorder-test"})

	if !callbackCalled {
		t.Error("onEvent callback was not invoked")
	}

	// BuildReport should produce a valid empty report.
	report := r.BuildReport()

	if err := report.Validate(); err != nil {
		t.Errorf("empty report failed validation: %v", err)
	}

	if report.ContainerID != "recorder-test" {
		t.Errorf("report.ContainerID: want %q, got %q", "recorder-test", report.ContainerID)
	}
}

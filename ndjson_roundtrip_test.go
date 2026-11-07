package auditlog_test

import (
	"bytes"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestNDJSONStreamer_BasicRoundTrip(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	streamer := auditlog.NewNDJSONStreamer(&buf)

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	events := []auditlog.Event{
		mkEvent(
			1,
			now,
			auditlog.EventTypeRegistration,
			auditlog.PhaseAfter,
			"db",
			"test-container",
			auditlog.ProviderTypeLazy,
		),
		mkEvent(
			2,
			now.Add(time.Millisecond),
			auditlog.EventTypeInvocation,
			auditlog.PhaseBefore,
			"db",
			"test-container",
			auditlog.ProviderTypeLazy,
		),
		mkEvent(
			3,
			now.Add(2*time.Millisecond),
			auditlog.EventTypeInvocation,
			auditlog.PhaseAfter,
			"db",
			"test-container",
			auditlog.ProviderTypeLazy,
		),
	}

	for _, evt := range events {
		streamer.OnEvent(evt)
	}

	err := streamer.Flush()
	if err != nil {
		t.Fatalf("Flush: %v", err)
	}

	readBack, err := auditlog.ReadEvents(&buf)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	if len(readBack) != len(events) {
		t.Fatalf("expected %d events, got %d", len(events), len(readBack))
	}

	for i, evt := range readBack {
		if evt.Sequence != events[i].Sequence {
			t.Errorf("event %d: expected seq %d, got %d", i, events[i].Sequence, evt.Sequence)
		}

		if evt.EventType != events[i].EventType {
			t.Errorf("event %d: expected type %v, got %v", i, events[i].EventType, evt.EventType)
		}
	}
}

func TestNDJSONStreamer_AutoFlush(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	streamer := auditlog.NewNDJSONStreamer(&buf, auditlog.WithAutoFlush())

	evt := mkEvent(
		1,
		time.Now(),
		auditlog.EventTypeRegistration,
		auditlog.PhaseAfter,
		"svc",
		"c",
		auditlog.ProviderTypeLazy,
	)
	streamer.OnEvent(evt)

	if buf.Len() == 0 {
		t.Fatal("expected buffered data after OnEvent with AutoFlush, got empty buffer")
	}
}

func TestNDJSONStreamer_BufferSize(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	streamer := auditlog.NewNDJSONStreamer(&buf, auditlog.WithStreamBufferSize(4096))

	evt := mkEvent(
		1,
		time.Now(),
		auditlog.EventTypeRegistration,
		auditlog.PhaseAfter,
		"svc",
		"c",
		auditlog.ProviderTypeLazy,
	)
	streamer.OnEvent(evt)

	if buf.Len() != 0 {
		t.Fatalf("expected no data before Flush, got %d bytes", buf.Len())
	}

	err := streamer.Flush()
	if err != nil {
		t.Fatalf("Flush: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected data after Flush, got empty buffer")
	}
}

func TestNDJSONStreamer_BufferSizeZeroIgnored(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	streamer := auditlog.NewNDJSONStreamer(&buf, auditlog.WithStreamBufferSize(0))

	evt := mkEvent(
		1,
		time.Now(),
		auditlog.EventTypeRegistration,
		auditlog.PhaseAfter,
		"svc",
		"c",
		auditlog.ProviderTypeLazy,
	)
	streamer.OnEvent(evt)

	err := streamer.Flush()
	if err != nil {
		t.Fatalf("Flush: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected data after Flush with default buffer")
	}
}

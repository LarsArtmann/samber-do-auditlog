package auditlog_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestNDJSONStreamer_Close(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	streamer := auditlog.NewNDJSONStreamer(&buf)

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

	err := streamer.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected data after Close, got empty buffer")
	}

	before := buf.Len()

	streamer.OnEvent(evt)

	if buf.Len() != before {
		t.Fatalf("expected no new data after Close, got %d -> %d bytes", before, buf.Len())
	}
}

func TestNDJSONStreamer_CloseIdempotent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	streamer := auditlog.NewNDJSONStreamer(&buf)

	err1 := streamer.Close()
	err2 := streamer.Close()

	if err1 != nil {
		t.Fatalf("first Close: %v", err1)
	}

	if err2 != nil {
		t.Fatalf("second Close: %v", err2)
	}
}

func TestNDJSONStreamer_CloseWithFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "stream.ndjson")

	streamer, err := auditlog.CreateNDJSONStreamer(path)
	if err != nil {
		t.Fatalf("CreateNDJSONStreamer: %v", err)
	}

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

	err = streamer.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("expected file to contain NDJSON data")
	}

	streamer.OnEvent(evt)

	dataAfter, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after: %v", err)
	}

	if len(dataAfter) != len(data) {
		t.Fatalf("expected file to be unchanged after Close, got %d -> %d bytes", len(data), len(dataAfter))
	}
}

func TestNDJSONStreamer_ErrOnFailingWriter(t *testing.T) {
	t.Parallel()

	streamer := auditlog.NewNDJSONStreamer(failingWriter{})

	streamer.OnEvent(
		mkEvent(
			1,
			time.Now(),
			auditlog.EventTypeRegistration,
			auditlog.PhaseAfter,
			"svc",
			"c",
			auditlog.ProviderTypeLazy,
		),
	)

	err := streamer.Flush()
	if err == nil {
		t.Fatal("expected Flush to return error from failing writer")
	}

	if streamer.Err() == nil {
		t.Error("Err() = nil, want non-nil after failing write")
	}
}

func TestNDJSONStreamer_ErrNilOnSuccess(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	streamer := auditlog.NewNDJSONStreamer(&buf)

	streamer.OnEvent(
		mkEvent(
			1,
			time.Now(),
			auditlog.EventTypeRegistration,
			auditlog.PhaseAfter,
			"svc",
			"c",
			auditlog.ProviderTypeLazy,
		),
	)

	err := streamer.Flush()
	if err != nil {
		t.Fatalf("Flush: %v", err)
	}

	if streamer.Err() != nil {
		t.Errorf("Err() = %v, want nil", streamer.Err())
	}
}

func TestNDJSONStreamer_DropsEventsAfterError(t *testing.T) {
	t.Parallel()

	streamer := auditlog.NewNDJSONStreamer(failingWriter{})

	// First event triggers the error on flush (buffered writes defer IO errors).
	streamer.OnEvent(
		mkEvent(
			1,
			time.Now(),
			auditlog.EventTypeRegistration,
			auditlog.PhaseAfter,
			"svc",
			"c",
			auditlog.ProviderTypeLazy,
		),
	)

	// Flush surfaces the underlying write error.
	_ = streamer.Flush()

	if streamer.Err() == nil {
		t.Fatal("Err() = nil, want non-nil after failing write + flush")
	}

	// Subsequent events should be silently dropped (no panic).
	for i := 2; i <= 5; i++ {
		streamer.OnEvent(
			mkEvent(
				i,
				time.Now(),
				auditlog.EventTypeRegistration,
				auditlog.PhaseAfter,
				"svc",
				"c",
				auditlog.ProviderTypeLazy,
			),
		)
	}

	// Err() should remain the original error.
	if streamer.Err() == nil {
		t.Error("Err() = nil after subsequent events")
	}
}

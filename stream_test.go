package auditlog_test

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func TestNDJSONStreamer_AutoFlushError(t *testing.T) {
	t.Parallel()

	streamer := auditlog.NewNDJSONStreamer(failingWriter{}, auditlog.WithAutoFlush())

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

	if streamer.Err() == nil {
		t.Error("expected error from autoflush with failing writer")
	}
}

func TestNDJSONStreamer_CloseCloserError(t *testing.T) {
	t.Parallel()

	streamer := auditlog.NewNDJSONStreamer(&errorCloser{})

	err := streamer.Close()
	if err == nil {
		t.Fatal("expected error from closer")
	}
}

// errorCloser returns an error on both Write and Close.
type errorCloser struct{}

func (errorCloser) Write([]byte) (int, error) {
	return 0, nil
}

func (errorCloser) Close() error {
	return errWriteFailed
}

func TestNDJSONStreamer_CreateError(t *testing.T) {
	t.Parallel()

	// Invalid path (directory as file) should return an error.
	_, err := auditlog.CreateNDJSONStreamer("/nonexistent_dir/stream.ndjson")
	if err == nil {
		t.Fatal("expected error creating streamer with invalid path")
	}
}

func TestNDJSONStreamer_ConcurrentSafe(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	streamer := auditlog.NewNDJSONStreamer(&buf)

	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)

		go func(seq int) {
			defer wg.Done()

			streamer.OnEvent(
				mkEvent(
					seq,
					time.Now(),
					auditlog.EventTypeRegistration,
					auditlog.PhaseAfter,
					"svc",
					"c",
					auditlog.ProviderTypeLazy,
				),
			)
		}(i + 1)
	}

	wg.Wait()

	err := streamer.Flush()
	if err != nil {
		t.Fatalf("Flush: %v", err)
	}

	events, err := auditlog.ReadEvents(&buf)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	if len(events) != 50 {
		t.Fatalf("expected 50 events, got %d", len(events))
	}
}

func TestNDJSONStreamer_OutputMatchesBatchEncoding(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	events := []auditlog.Event{
		mkEvent(1, now, auditlog.EventTypeRegistration, auditlog.PhaseAfter, "db", "c", auditlog.ProviderTypeLazy),
		mkEvent(
			2,
			now.Add(time.Millisecond),
			auditlog.EventTypeInvocation,
			auditlog.PhaseBefore,
			"db",
			"c",
			auditlog.ProviderTypeLazy,
		),
	}

	// Stream path.
	var streamBuf bytes.Buffer

	streamer := auditlog.NewNDJSONStreamer(&streamBuf)

	for _, evt := range events {
		streamer.OnEvent(evt)
	}

	if err := streamer.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	// Batch path (replicates writeEventsNDJSON encoding exactly).
	var batchBuf bytes.Buffer

	batchEnc := jsontext.NewEncoder(&batchBuf)
	for _, evt := range events {
		if err := json.MarshalEncode(batchEnc, evt); err != nil {
			t.Fatalf("batch encode event %d: %v", evt.Sequence, err)
		}
	}

	if streamBuf.String() != batchBuf.String() {
		t.Errorf("stream output != batch encoding\nstream:\n%s\nbatch:\n%s", streamBuf.String(), batchBuf.String())
	}
}

func TestNDJSONStreamer_WithFlushInterval_Bounds(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	const interval = 250 * time.Millisecond

	streamer := auditlog.NewNDJSONStreamer(&buf, auditlog.WithFlushInterval(interval))

	for i := range 50 {
		streamer.OnEvent(auditlog.Event{
			Sequence:    i + 1,
			EventType:   auditlog.EventTypeRegistration,
			Phase:       auditlog.PhaseBefore,
			ServiceName: auditlog.ServiceName(fmt.Sprintf("burst-%d", i)),
		})
	}

	if buf.Len() != 0 {
		t.Errorf("WithFlushInterval should buffer within the interval; got %d bytes written", buf.Len())
	}

	time.Sleep(interval + 50*time.Millisecond)

	streamer.OnEvent(auditlog.Event{
		Sequence:    51,
		EventType:   auditlog.EventTypeRegistration,
		Phase:       auditlog.PhaseBefore,
		ServiceName: "trigger",
	})

	if buf.Len() == 0 {
		t.Fatal("expected events to flush after interval elapsed")
	}

	lines := strings.Count(buf.String(), "\n")

	if lines < 51 {
		t.Errorf("expected >= 51 flushed lines, got %d", lines)
	}

	if err := streamer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestNDJSONStreamer_WithFlushInterval_IgnoredForZeroAndNegative(t *testing.T) {
	t.Parallel()

	for _, d := range []time.Duration{0, -1 * time.Second} {
		t.Run(fmt.Sprintf("d=%s", d), func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			streamer := auditlog.NewNDJSONStreamer(&buf, auditlog.WithFlushInterval(d))

			streamer.OnEvent(auditlog.Event{
				Sequence:    1,
				EventType:   auditlog.EventTypeRegistration,
				Phase:       auditlog.PhaseBefore,
				ServiceName: "x",
			})

			if buf.Len() != 0 {
				t.Errorf("expected buffered (no flush) for ignored duration %s; got %d bytes", d, buf.Len())
			}

			if err := streamer.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}
		})
	}
}

func TestNDJSONStreamer_WithAutoFlushTakesPrecedence(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	streamer := auditlog.NewNDJSONStreamer(
		&buf,
		auditlog.WithFlushInterval(time.Hour),
		auditlog.WithAutoFlush(),
	)

	streamer.OnEvent(auditlog.Event{
		Sequence:    1,
		EventType:   auditlog.EventTypeRegistration,
		Phase:       auditlog.PhaseBefore,
		ServiceName: "x",
	})

	if buf.Len() == 0 {
		t.Error("WithAutoFlush should take precedence and flush immediately")
	}

	if err := streamer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

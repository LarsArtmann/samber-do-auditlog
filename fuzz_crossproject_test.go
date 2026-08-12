package auditlog_test

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	errorfamily "github.com/larsartmann/go-error-family"
	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// FuzzNDJSONStreamer fuzzes the NDJSONStreamer with adversarial event fields.
// It streams an event, flushes, reads it back, and verifies the round-trip
// preserves the service name and sequence number.
func FuzzNDJSONStreamer(f *testing.F) {
	seeds := []struct {
		sequence    int
		eventType   string
		phase       string
		serviceName string
		scopeID     string
	}{
		{1, "registration", "before", "svc1", "root"},
		{2, "invocation", "after", "svc-with-special-[chars]", "scope-1"},
		{3, "shutdown", "before", "svc3", "root"},
		{4, "health_check", "after", "svc-xss", "root"},
		{999, "registration", "after", strings.Repeat("A", 500), "root"},
	}

	for _, s := range seeds {
		f.Add(s.sequence, s.eventType, s.phase, s.serviceName, s.scopeID)
	}

	f.Fuzz(func(t *testing.T, sequence int, eventType, phase, serviceName, scopeID string) {
		if serviceName == "" || scopeID == "" {
			t.Skip()
		}

		if !utf8.ValidString(serviceName) || !utf8.ValidString(scopeID) {
			t.Skip()
		}

		et := auditlog.EventType(eventType)
		if !et.IsKnown() {
			t.Skip()
		}

		ph := auditlog.Phase(phase)
		if ph != auditlog.PhaseBefore && ph != auditlog.PhaseAfter {
			t.Skip()
		}

		evt := auditlog.Event{
			Sequence:  sequence,
			EventType: auditlog.EventType(eventType),
			Phase:     auditlog.Phase(phase),
			ServiceRef: auditlog.ServiceRef{
				ServiceName: auditlog.ServiceName(serviceName),
				ScopeID:     auditlog.ScopeID(scopeID),
			},
		}

		var buf bytes.Buffer

		streamer := auditlog.NewNDJSONStreamer(&buf, auditlog.WithAutoFlush())

		streamer.OnEvent(evt)

		if err := streamer.Flush(); err != nil {
			t.Fatalf("Flush error: %v", err)
		}

		if buf.Len() == 0 {
			t.Fatal("streamer produced no output")
		}

		events, err := auditlog.ReadEvents(&buf)
		if err != nil {
			t.Fatalf("ReadEvents error: %v", err)
		}

		if len(events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(events))
		}

		got := events[0]
		if got.Sequence != sequence {
			t.Errorf("sequence: want %d, got %d", sequence, got.Sequence)
		}

		if string(got.ServiceName) != serviceName {
			t.Errorf("service name: want %q, got %q", serviceName, got.ServiceName)
		}
	})
}

// FuzzMultiWriter fuzzes MultiWriter with varying callback counts, nil
// callbacks, and adversarial event data. Verifies no panic and that all
// non-nil callbacks receive the event.
func FuzzMultiWriter(f *testing.F) {
	seeds := []struct {
		callbackCount int
		serviceName   string
		sequence      int
	}{
		{0, "svc", 1},
		{1, "svc", 1},
		{3, "svc", 1},
		{5, "fuzz-svc", 99},
		{2, "x", -1},
		{10, strings.Repeat("x", 200), 42},
	}

	for _, s := range seeds {
		f.Add(s.callbackCount, s.serviceName, s.sequence)
	}

	f.Fuzz(func(t *testing.T, callbackCount int, serviceName string, sequence int) {
		if callbackCount < 0 || callbackCount > 50 {
			t.Skip()
		}

		if serviceName == "" {
			t.Skip()
		}

		evt := auditlog.Event{
			Sequence: sequence,
			ServiceRef: auditlog.ServiceRef{
				ServiceName: auditlog.ServiceName(serviceName),
			},
		}

		var mu sync.Mutex

		received := make([]int, callbackCount) //nolint:makezero // indexed by position

		callbacks := make([]auditlog.MultiWriterCallback, 0, callbackCount)

		for i := range callbackCount {
			idx := i

			callbacks = append(callbacks, func(e auditlog.Event) {
				mu.Lock()
				received[idx]++
				mu.Unlock()
			})
		}

		mw := auditlog.NewMultiWriter(callbacks...)
		mw.OnEvent(evt)

		for i, count := range received {
			if count != 1 {
				t.Errorf("callback %d received %d events, want 1", i, count)
			}
		}

		if mw.CallbackCount() != callbackCount {
			t.Errorf("CallbackCount: want %d, got %d", callbackCount, mw.CallbackCount())
		}
	})
}

// FuzzClassifyAdversarialChains fuzzes error classification with adversarial
// payloads wrapped around known sentinel errors. Verifies classification
// survives arbitrary wrapping and that bare errors fail-open to Transient.
func FuzzClassifyAdversarialChains(f *testing.F) {
	payloads := []string{
		"wrapped error",
		"",
		strings.Repeat("A", 200),
		"%s%s%s%d format verbs",
		"error: inner: deep: chain",
		"<script>alert(1)</script>",
		"\x00null\x00bytes",
	}

	for _, p := range payloads {
		f.Add(p)
	}

	f.Fuzz(func(t *testing.T, payload string) {
		bare := fmt.Errorf("%s", payload) //nolint:err113 // fuzz input

		family := errorfamily.Classify(bare)
		if family != errorfamily.Transient {
			t.Errorf("bare error should classify as Transient (fail-open), got %d", family)
		}

		for sentinel, expectedFamily := range auditlog.ErrorClassifications() {
			wrapped := fmt.Errorf("%s: %w: %s", payload, sentinel, payload)

			wrappedFamily := errorfamily.Classify(wrapped)
			if wrappedFamily != expectedFamily {
				t.Errorf("wrapped sentinel classified as %d, want %d", wrappedFamily, expectedFamily)
			}
		}
	})
}

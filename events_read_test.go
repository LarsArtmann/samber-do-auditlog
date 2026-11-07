package auditlog_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/samber/do/v2"
)

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
	assertErrIs(t, err, auditlog.ErrEmpty, "expected ErrEmpty")
}

func TestReadEvents_OnlyBlankLines(t *testing.T) {
	t.Parallel()

	_, err := auditlog.ReadEvents(strings.NewReader("\n\n\n"))
	assertErrIs(t, err, auditlog.ErrNoEvents, "expected ErrNoEvents")
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
	assertErrIs(t, err, auditlog.ErrOversizedLine, "expected ErrOversizedLine")
}

func TestReadEvents_NoTrailingNewline(t *testing.T) {
	t.Parallel()

	input := `{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db"}`

	events, err := auditlog.ReadEvents(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	assertLen(t, "event count", events, 1)
}

func TestReadEvents_LeadingWhitespace(t *testing.T) {
	t.Parallel()

	input := "  " + `{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db"}` + "\n"

	events, err := auditlog.ReadEvents(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	assertLen(t, "event count", events, 1)
}

func TestReadEvents_CarriageReturns(t *testing.T) {
	t.Parallel()

	// Line with trailing \r (Windows-style).
	input := `{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db"}` + "\r\n"

	events, err := auditlog.ReadEvents(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadEvents with \\r\\n: %v", err)
	}

	assertLen(t, "event count", events, 1)
}

// --- Loader API tests ---

func TestReadEvents_RejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantSubstr string
	}{
		{
			name:       "UnknownEventType",
			input:      `{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"bogus","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db"}`,
			wantSubstr: "unknown event_type",
		},
		{
			name:       "UnknownPhase",
			input:      `{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"mid","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db"}`,
			wantSubstr: "unknown phase",
		},
		{
			name:       "UnknownProviderType",
			input:      `{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db","service_type":"singularity"}`,
			wantSubstr: "unknown provider_type",
		},
		{
			name:       "EmptyProviderTypeAllowed",
			input:      `{"sequence":1,"timestamp":"2026-01-01T00:00:00Z","event_type":"registration","phase":"after","container_id":"x","scope_id":"s","scope_name":"s","service_name":"db"}`,
			wantSubstr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := auditlog.ReadEvents(strings.NewReader(tt.input))
			if tt.wantSubstr == "" {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}

				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error should mention %q, got: %v", tt.wantSubstr, err)
			}
		})
	}
}

func TestReplayEvents_RejectsUnknownEnums(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name string
		evt  auditlog.Event
	}{
		{
			name: "UnknownEventType",
			evt:  mkEvent(1, now, auditlog.EventType("singularity"), auditlog.PhaseAfter, "db", "c1", ""),
		},
		{
			name: "UnknownPhase",
			evt:  mkEvent(1, now, auditlog.EventTypeRegistration, auditlog.Phase("mid"), "db", "c1", ""),
		},
		{
			name: "UnknownProviderType",
			evt: mkEvent(1, now, auditlog.EventTypeRegistration, auditlog.PhaseAfter,
				"db", "c1", auditlog.ProviderType("quantum")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := auditlog.ReplayEvents([]auditlog.Event{tt.evt})
			if !errors.Is(err, auditlog.ErrReplayValidationFailed) {
				t.Errorf("expected ErrReplayValidationFailed, got %v", err)
			}
		})
	}
}

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

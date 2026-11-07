package live_test

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
)

func TestServer_GracefulShutdown(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ts := httptest.NewServer(server)
	defer ts.Close()

	ctx := t.Context()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/debug/di/api/health", nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET health: %v", err)
	}

	_ = resp.Body.Close()
}

func TestServer_ClientCount(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	if server.ClientCount() != 0 {
		t.Errorf("expected 0 clients initially, got %d", server.ClientCount())
	}
}

// --- Hub Unit Tests ---

func TestHub_SubscribeUnsubscribe(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	ch := hub.Subscribe()

	if hub.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", hub.ClientCount())
	}

	hub.Unsubscribe(ch)

	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients after unsubscribe, got %d", hub.ClientCount())
	}
}

func TestHub_OnEventDelivery(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	evt := auditlog.Event{
		Sequence:    42,
		ScopeID:     "root",
		ScopeName:   "[root]",
		ServiceName: "test",
	}

	hub.OnEvent(evt)

	select {
	case received := <-ch:
		var parsed struct {
			Sequence int `json:"sequence"`
		}
		if err := json.Unmarshal([]byte(received.Data), &parsed); err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}

		if parsed.Sequence != 42 {
			t.Errorf("expected sequence 42, got %d", parsed.Sequence)
		}

		if received.Event != "event" {
			t.Errorf("expected event type %q, got %q", "event", received.Event)
		}

		if received.ID.IsZero() {
			t.Error("expected non-zero event ID for replay support")
		}

	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestHub_SignalComplete(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	hub.SignalComplete()

	select {
	case <-hub.Done():
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for done signal")
	}

	if !hub.IsComplete() {
		t.Error("expected IsComplete() to be true")
	}
}

func TestHub_BufferOverflow(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	for i := range 200 {
		hub.OnEvent(auditlog.Event{Sequence: i})
	}

	received := 0

	for {
		select {
		case <-ch:
			received++
		default:
			goto done
		}
	}

done:
	if received != 128 {
		t.Errorf("expected 128 (buffer size), got %d", received)
	}
}

func TestHub_UnsubscribeUnknownChannel(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	// Unsubscribing a never-subscribed channel should be a no-op (no panic).
	ch := make(chan sse.Event, 1)
	hub.Unsubscribe(ch)
}

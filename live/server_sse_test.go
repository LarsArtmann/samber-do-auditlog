package live_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-sse/ssetest"
	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
	"github.com/samber/do/v2"
)

func sseConnect(t *testing.T, url string) (*ssetest.StreamReader, func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose // closed via returned cleanup
	if err != nil {
		t.Fatalf("connect SSE: %v", err)
	}

	cleanup := func() {
		cancel()

		_ = resp.Body.Close()
	}

	return ssetest.NewStreamReader(resp.Body), cleanup
}

func sseConnectWithLastID(t *testing.T, url, lastEventID string) (*ssetest.StreamReader, func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	req.Header.Set("Last-Event-ID", lastEventID)

	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose // closed via returned cleanup
	if err != nil {
		t.Fatalf("connect SSE: %v", err)
	}

	cleanup := func() {
		cancel()

		_ = resp.Body.Close()
	}

	return ssetest.NewStreamReader(resp.Body), cleanup
}

func skipSnapshot(sr *ssetest.StreamReader) {
	// The datastar snapshot sends: 1 patch-signals + N patch-elements events.
	// #container-id is always the last fragment. Skip until we consume it.
	for {
		evt, err := sr.Next()
		if err != nil {
			return
		}

		if strings.Contains(evt.Data(), "selector #container-id") {
			return
		}
	}
}

func readSSEEvent(sr *ssetest.StreamReader, eventName string) (string, bool) {
	for {
		evt, err := sr.Next()
		if err != nil {
			return "", false
		}

		if evt.Type == eventName {
			return evt.Data(), true
		}
	}
}

func readUntilService(sr *ssetest.StreamReader, serviceName string) bool {
	for {
		evt, err := sr.Next()
		if err != nil {
			return false
		}

		if strings.Contains(evt.Data(), serviceName) {
			return true
		}
	}
}

// assertSSEDetectsService creates a server with an injector, connects SSE,
// provides a service, and asserts the service name appears in a datastar
// patch-elements event. Shared by TestServer_SSE_LiveEventDelivery and
// TestServer_SSE_EventBroadcast.
func assertSSEDetectsService(t *testing.T, containerID, serviceName string, value any) {
	t.Helper()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: auditlog.ContainerID(containerID),
		OnEvent:     hub.OnEvent,
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	server := live.NewServer(hub, plugin, live.Config{})
	injector := do.NewWithOpts(plugin.Opts())

	ts := httptest.NewServer(server)
	defer ts.Close()

	sr, closeSSE := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE()

	skipSnapshot(sr)

	do.ProvideNamedValue(injector, serviceName, value)

	found := false

	for range 20 {
		data, ok := readSSEEvent(sr, "datastar-patch-elements")
		if !ok {
			break
		}

		if strings.Contains(data, serviceName) {
			found = true

			break
		}
	}

	if !found {
		t.Fatalf("did not receive live update containing %s", serviceName)
	}
}

func TestServer_SSE_SnapshotOnConnect(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	server.OnEvent(auditlog.Event{
		ScopeID:     "root",
		ScopeName:   "[root]",
		ServiceName: "db",
		Sequence:    1,
		EventType:   auditlog.EventTypeRegistration,
		Phase:       auditlog.PhaseAfter,
	})

	ts := httptest.NewServer(server)
	defer ts.Close()

	sr, closeSSE := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE()

	data, found := readSSEEvent(sr, "datastar-patch-signals")
	if !found {
		t.Fatal("did not receive datastar-patch-signals event")
	}

	if !strings.Contains(data, "connStatus") {
		t.Errorf("snapshot signals should contain connStatus: %s", data[:min(200, len(data))])
	}
}

func TestServer_SSE_LiveEventDelivery(t *testing.T) {
	t.Parallel()

	assertSSEDetectsService(t, "live-event-test", "cache", "cache-value")
}

func TestServer_SSE_CompleteEvent(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ts := httptest.NewServer(server)
	defer ts.Close()

	sr, closeSSE := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE()

	skipSnapshot(sr)

	server.SignalComplete()

	// Complete sends a full snapshot, then a final signal patch with complete:true.
	// Read datastar-patch-signals events until we find the one with complete.
	found := false

	for range 20 {
		data, ok := readSSEEvent(sr, "datastar-patch-signals")
		if !ok {
			break
		}

		if strings.Contains(data, `"complete":true`) {
			found = true

			break
		}
	}

	if !found {
		t.Fatal("did not receive complete signal")
	}
}

func TestServer_SSE_FanOut(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "fanout-test",
		OnEvent:     hub.OnEvent,
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	server := live.NewServer(hub, plugin, live.Config{})
	injector := do.NewWithOpts(plugin.Opts())

	ts := httptest.NewServer(server)
	defer ts.Close()

	sr1, closeSSE1 := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE1()

	sr2, closeSSE2 := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE2()

	skipSnapshot(sr1)
	skipSnapshot(sr2)

	do.ProvideNamedValue(injector, "fanout-svc", "fanout-value")

	if !readUntilService(sr1, "fanout-svc") {
		t.Error("client 1 did not receive fanout event")
	}

	if !readUntilService(sr2, "fanout-svc") {
		t.Error("client 2 did not receive fanout event")
	}
}

package live_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
)

func newTestServer(t *testing.T) *live.Server {
	t.Helper()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "test-container",
		OnEvent:     hub.OnEvent,
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}


	server := live.NewServer(hub, plugin, live.Config{})

	return server
}

func TestServer_DashboardHTML(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ctx := t.Context()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/debug/di/", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	for _, want := range []string{"<!DOCTYPE html>", "samber-do-auditlog", "LIVE"} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard HTML missing %q", want)
		}
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content-type, got %s", ct)
	}

	if !strings.Contains(body, `__LIVE_PREFIX="/debug/di"`) {
		t.Error("dashboard HTML missing prefix JS variable")
	}
}

func TestServer_HealthEndpoint(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ctx := t.Context()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/debug/di/api/health", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	for _, want := range []string{`"status"`, `"ok"`, `"clients"`, `"events"`} {
		if !strings.Contains(body, want) {
			t.Errorf("health response missing %q: %s", want, body)
		}
	}
}

func TestServer_ReportEndpoint(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ctx := t.Context()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/debug/di/api/report", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	if !strings.Contains(body, `"container_id"`) {
		t.Errorf("report response missing container_id: %s", body[:min(200, len(body))])
	}
}

func TestServer_NotFound(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ctx := t.Context()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestServer_NewConvenience(t *testing.T) {
	t.Parallel()

	server, plugin, err := live.New(auditlog.Config{
		ContainerID: "convenience-test",
	}, live.Config{Addr: ":0"})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if server == nil {
		t.Fatal("server is nil")
	}

	if plugin == nil {
		t.Fatal("plugin is nil")
	}
}

func TestServer_CustomPrefix(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "prefix-test",
		OnEvent:     hub.OnEvent,
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}


	server := live.NewServer(hub, plugin, live.Config{Prefix: "/my/debug"})

	ts := httptest.NewServer(server)
	defer ts.Close()

	ctx := t.Context()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/my/debug/api/health", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Error("health response missing ok")
	}

	req2 := httptest.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/", nil)
	rec2 := httptest.NewRecorder()
	server.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusNotFound {
		t.Fatalf("root without prefix should 404, got %d", rec2.Code)
	}
}

func TestServer_RootPrefix(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "root-test",
		OnEvent:     hub.OnEvent,
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}


	server := live.NewServer(hub, plugin, live.Config{Prefix: "/"})

	ctx := t.Context()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `__LIVE_PREFIX="/"`) {
		t.Error("dashboard HTML missing root prefix JS variable")
	}
}

// --- SSE Tests (use httptest.NewServer for real HTTP streaming) ---

func sseConnect(t *testing.T, url string) (*bufio.Scanner, func()) {
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

	return bufio.NewScanner(resp.Body), cleanup
}

func skipSnapshot(scanner *bufio.Scanner) {
	for scanner.Scan() {
		if scanner.Text() == "" {
			break
		}
	}
}

func readSSEEvent(scanner *bufio.Scanner, eventName string) (string, bool) {
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: "+eventName) {
			scanner.Scan()

			dataLine := scanner.Text()
			data, found := strings.CutPrefix(dataLine, "data: ")

			if found {
				return data, true
			}
		}
	}

	return "", false
}

func readUntilService(scanner *bufio.Scanner, serviceName string) bool {
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), serviceName) {
			return true
		}
	}

	return false
}

func TestServer_SSE_SnapshotOnConnect(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	server.OnEvent(auditlog.Event{
		ServiceRef: auditlog.ServiceRef{
			ScopeID:     "root",
			ScopeName:   "[root]",
			ServiceName: "db",
		},
		Sequence:  1,
		EventType: auditlog.EventTypeRegistration,
		Phase:     auditlog.PhaseAfter,
	})

	ts := httptest.NewServer(server)
	defer ts.Close()

	scanner, closeSSE := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE()

	data, found := readSSEEvent(scanner, "snapshot")
	if !found {
		t.Fatal("did not receive snapshot event")
	}

	if !strings.Contains(data, `"report"`) {
		t.Errorf("snapshot should contain report field: %s", data[:min(200, len(data))])
	}
}

func TestServer_SSE_LiveEventDelivery(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ts := httptest.NewServer(server)
	defer ts.Close()

	scanner, closeSSE := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE()

	skipSnapshot(scanner)

	server.OnEvent(auditlog.Event{
		ServiceRef: auditlog.ServiceRef{
			ScopeID:     "root",
			ScopeName:   "[root]",
			ServiceName: "cache",
		},
		Sequence:  1,
		EventType: auditlog.EventTypeRegistration,
		Phase:     auditlog.PhaseAfter,
	})

	data, found := readSSEEvent(scanner, "event")
	if !found {
		t.Fatal("did not receive live event")
	}

	if !strings.Contains(data, "cache") {
		t.Errorf("live event should contain cache: %s", data)
	}
}

func TestServer_SSE_CompleteEvent(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ts := httptest.NewServer(server)
	defer ts.Close()

	scanner, closeSSE := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE()

	skipSnapshot(scanner)

	server.SignalComplete()

	_, found := readSSEEvent(scanner, "complete")
	if !found {
		t.Fatal("did not receive complete event")
	}
}

func TestServer_SSE_FanOut(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ts := httptest.NewServer(server)
	defer ts.Close()

	scanner1, closeSSE1 := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE1()

	scanner2, closeSSE2 := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE2()

	skipSnapshot(scanner1)
	skipSnapshot(scanner2)

	server.OnEvent(auditlog.Event{
		ServiceRef: auditlog.ServiceRef{
			ScopeID:     "root",
			ScopeName:   "[root]",
			ServiceName: "fanout-svc",
		},
		Sequence:  1,
		EventType: auditlog.EventTypeRegistration,
		Phase:     auditlog.PhaseAfter,
	})

	if !readUntilService(scanner1, "fanout-svc") {
		t.Error("client 1 did not receive fanout event")
	}

	if !readUntilService(scanner2, "fanout-svc") {
		t.Error("client 2 did not receive fanout event")
	}
}

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

	sub := hub.Subscribe()
	if sub == nil {
		t.Fatal("Subscribe returned nil")
	}

	if hub.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", hub.ClientCount())
	}

	hub.Unsubscribe(sub.ID())

	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients after unsubscribe, got %d", hub.ClientCount())
	}
}

func TestHub_OnEventDelivery(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	sub := hub.Subscribe()
	defer hub.Unsubscribe(sub.ID())

	evt := auditlog.Event{
		Sequence: 42,
		ServiceRef: auditlog.ServiceRef{
			ScopeID:     "root",
			ScopeName:   "[root]",
			ServiceName: "test",
		},
	}

	hub.OnEvent(evt)

	select {
	case received := <-sub.Events():
		var parsed struct {
			Sequence int `json:"sequence"`
		}
		if err := json.Unmarshal(received, &parsed); err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}

		if parsed.Sequence != 42 {
			t.Errorf("expected sequence 42, got %d", parsed.Sequence)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestHub_SignalComplete(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	sub := hub.Subscribe()
	defer hub.Unsubscribe(sub.ID())

	hub.SignalComplete()

	select {
	case <-sub.Done():
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

	sub := hub.Subscribe()
	defer hub.Unsubscribe(sub.ID())

	for i := range 200 {
		hub.OnEvent(auditlog.Event{Sequence: i})
	}

	received := 0

	for {
		select {
		case <-sub.Events():
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

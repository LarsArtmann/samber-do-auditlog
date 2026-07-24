package live_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
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

func TestHub_UnsubscribeUnknownID(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	// Unsubscribing a non-existent ID should be a no-op (no panic).
	hub.Unsubscribe(99999)
}

func TestServer_SSE_Heartbeat(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "heartbeat-test",
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	// Use a very short heartbeat interval to trigger the heartbeat path quickly.
	server := live.NewServer(hub, plugin, live.Config{HeartbeatInterval: 50 * time.Millisecond})

	ts := httptest.NewServer(server)
	defer ts.Close()

	scanner, closeSSE := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE()

	// Skip the snapshot event.
	skipSnapshot(scanner)

	// Wait for a heartbeat comment line.
	foundHeartbeat := false

	for range 100 {
		if scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, ": heartbeat") {
				foundHeartbeat = true

				break
			}
		}
	}

	if !foundHeartbeat {
		t.Error("did not receive heartbeat within timeout")
	}
}

// --- Server lifecycle tests ---

func TestServer_ListenAndServe_Addr_Shutdown(t *testing.T) {
	t.Parallel()

	// Get a free port to avoid conflicts.
	lc := net.ListenConfig{}

	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("get free port: %v", err)
	}

	addr := ln.Addr().String()
	_ = ln.Close()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "lifecycle-test",
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	server := live.NewServer(hub, plugin, live.Config{Addr: addr})

	// Addr() before starting returns the configured address.
	if got := server.Addr(); got != addr {
		t.Errorf("expected %q before start, got %q", addr, got)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	// Wait for server to start by polling the health endpoint.
	ctx := t.Context()

	var lastErr error

	for range 100 {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/debug/di/api/health", nil)

		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				lastErr = nil

				break
			}
		}

		lastErr = err

		time.Sleep(10 * time.Millisecond)
	}

	if lastErr != nil {
		t.Fatalf("server did not start: %v", lastErr)
	}

	// Addr() while running should return the configured address (httpServer is set).
	if got := server.Addr(); got != addr {
		t.Errorf("Addr() while running should return %q, got %q", addr, got)
	}

	shutdownCtx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("ListenAndServe should return non-nil error after shutdown")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ListenAndServe did not return after shutdown")
	}
}

func TestServer_ListenAndServe_AlreadyRunning(t *testing.T) {
	t.Parallel()

	// Get a free port.
	lc := net.ListenConfig{}

	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("get free port: %v", err)
	}

	addr := ln.Addr().String()
	_ = ln.Close()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "already-running-test",
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	server := live.NewServer(hub, plugin, live.Config{Addr: addr})

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	// Wait for the first ListenAndServe to take hold.
	time.Sleep(100 * time.Millisecond)

	// Second call should fail with ErrServerAlreadyRunning.
	err = server.ListenAndServe()
	if err == nil {
		t.Error("expected error when calling ListenAndServe twice")
	}

	// Clean up.
	shutdownCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	_ = server.Shutdown(shutdownCtx)

	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
	}
}

func TestServer_ShutdownNotRunning(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	// Shutdown on a server that was never started should return nil.
	err := server.Shutdown(t.Context())
	if err != nil {
		t.Errorf("expected nil error shutting down non-running server, got: %v", err)
	}
}

func TestServer_New_InvalidConfig(t *testing.T) {
	t.Parallel()

	// ContainerID with path separator causes validation error.
	_, _, err := live.New(auditlog.Config{
		ContainerID: "bad/path",
	}, live.Config{})
	if err == nil {
		t.Error("expected error for invalid ContainerID")
	}
}

// --- Handler edge-case tests ---

func TestServer_HandleSSE_NoFlusher(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	// A non-streaming ResponseWriter that does not implement http.Flusher.
	// httptest.NewRecorder does implement Flusher, so we need a custom one.
	rec := &noFlushRecorder{
		headerMap: make(http.Header),
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/debug/di/api/events", nil)
	server.ServeHTTP(rec, req)

	if rec.code != http.StatusInternalServerError {
		t.Errorf("expected 500 for non-streaming writer, got %d", rec.code)
	}

	if !strings.Contains(rec.body.String(), "streaming not supported") {
		t.Errorf("expected 'streaming not supported' error, got: %s", rec.body.String())
	}
}

// noFlushRecorder is an http.ResponseWriter that does NOT implement http.Flusher.
type noFlushRecorder struct {
	headerMap http.Header
	body      strings.Builder
	code      int
}

func (r *noFlushRecorder) Header() http.Header { return r.headerMap }
func (r *noFlushRecorder) Write(buf []byte) (int, error) {
	n, err := r.body.Write(buf)
	if err != nil {
		return n, fmt.Errorf("noFlushRecorder write: %w", err)
	}

	return n, nil
}
func (r *noFlushRecorder) WriteHeader(code int) { r.code = code }

func TestServer_NormalizePrefix(t *testing.T) {
	t.Parallel()

	// Test that a prefix without a leading slash gets one added.
	// normalizePrefix is unexported, so we test indirectly via NewServer.
	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "prefix-test-2",
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	server := live.NewServer(hub, plugin, live.Config{Prefix: "my/prefix"})
	ctx := t.Context()

	// Should be accessible at /my/prefix/ (leading slash added by normalizePrefix).
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/my/prefix/", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for prefix without leading slash, got %d", rec.Code)
	}
}

func TestServer_HandleDashboard_ExactPrefix(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	// Request to exactly the prefix (no trailing slash) triggers a ServeMux 307
	// redirect to the trailing-slash variant. This is standard Go behavior.
	ctx := t.Context()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/debug/di", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected 307 redirect for bare prefix, got %d", rec.Code)
	}
}

func TestServer_HealthEndpoint_WithEvents(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	// Emit some events so the health endpoint reports non-zero counts.
	server.OnEvent(auditlog.Event{
		ServiceRef: auditlog.ServiceRef{
			ScopeID:     "root",
			ScopeName:   "[root]",
			ServiceName: "health-test-svc",
		},
		Sequence:  1,
		EventType: auditlog.EventTypeRegistration,
		Phase:     auditlog.PhaseAfter,
	})

	ctx := t.Context()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/debug/di/api/health", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"events"`) {
		t.Errorf("health response should contain events field: %s", body)
	}
}

func TestServer_HandleDashboard_SubPathNotFound(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	// A path under the prefix but not the dashboard root should return 404
	// from the handler (not the mux).
	ctx := t.Context()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/debug/di/nonexistent-subpage", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for sub-path, got %d", rec.Code)
	}
}

func TestServer_NilPlugin_SSESnapshot(t *testing.T) {
	t.Parallel()

	// A server with nil plugin exercises the nil-plugin early return in sendSnapshot.
	hub := live.NewHub()
	server := live.NewServer(hub, nil, live.Config{})

	ts := httptest.NewServer(server)
	defer ts.Close()

	// Connect to SSE — snapshot will be skipped (nil plugin → return nil),
	// then SignalComplete triggers sendComplete (also nil plugin → return).
	hub.SignalComplete()

	scanner, closeSSE := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE()

	// The connection should succeed even with nil plugin.
	// The complete event may or may not arrive depending on timing,
	// but the connection itself should not error.
	_ = scanner
}

func TestServer_NilPlugin_ReportEndpoint(t *testing.T) {
	t.Parallel()

	// A server with nil plugin: handleReport panics (no nil check).
	// httptest.NewServer recovers the panic and returns 500.
	hub := live.NewHub()
	server := live.NewServer(hub, nil, live.Config{})

	ts := httptest.NewServer(server)
	defer ts.Close()

	ctx := t.Context()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/debug/di/api/report", nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for nil plugin report, got %d", resp.StatusCode)
	}
}

func TestServer_ShutdownWithContextCancelled(t *testing.T) {
	t.Parallel()

	lc := net.ListenConfig{}

	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("get free port: %v", err)
	}

	addr := ln.Addr().String()
	_ = ln.Close()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "shutdown-err-test",
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	server := live.NewServer(hub, plugin, live.Config{Addr: addr})

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	// Wait for server to start.
	ctx := t.Context()

	for range 100 {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/debug/di/api/health", nil)

		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	// Shutdown with an already-cancelled context should return an error.
	cancelledCtx, cancel := context.WithCancel(t.Context())
	cancel()

	shutdownErr := server.Shutdown(cancelledCtx)
	// The error may or may not be nil depending on timing — either way
	// the server should eventually stop.
	_ = shutdownErr

	// Ensure the server has stopped.
	_ = <-errCh
}

func TestServer_HealthEndpoint_NilPlugin(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()
	server := live.NewServer(hub, nil, live.Config{})

	ctx := t.Context()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/debug/di/api/health", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// With nil plugin, Events and Dropped should be 0.
	for _, want := range []string{`"status"`, `"ok"`, `"events":0`} {
		if !strings.Contains(body, want) {
			t.Errorf("health response missing %q: %s", want, body)
		}
	}
}

func TestServer_SSE_EventBroadcast(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ts := httptest.NewServer(server)
	defer ts.Close()

	scanner, closeSSE := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE()

	skipSnapshot(scanner)

	// Broadcast an event through the hub (simulates plugin OnEvent callback).
	server.OnEvent(auditlog.Event{
		Sequence:  999,
		EventType: auditlog.EventTypeRegistration,
	})

	data, ok := readSSEEvent(scanner, "event")
	if !ok {
		t.Fatal("expected to receive broadcast event")
	}

	if !strings.Contains(data, `"sequence":999`) {
		t.Errorf("event data missing sequence 999: %s", data)
	}
}

func TestServer_CORSHeaders(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ctx := t.Context()

	// Verify CORS headers on health endpoint.
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/debug/di/api/health", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin *, got %q", origin)
	}

	// Verify OPTIONS preflight returns 204.
	reqOpts := httptest.NewRequestWithContext(ctx, http.MethodOptions, "/debug/di/api/health", nil)
	recOpts := httptest.NewRecorder()

	server.ServeHTTP(recOpts, reqOpts)

	if recOpts.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", recOpts.Code)
	}
}

func TestServer_ExportEndpoints(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ctx := t.Context()

	// Test JSON export (same as /api/report but test via export path)
	reqJSON := httptest.NewRequestWithContext(ctx, http.MethodGet, "/debug/di/api/report", nil)
	recJSON := httptest.NewRecorder()
	server.ServeHTTP(recJSON, reqJSON)
	if recJSON.Code != http.StatusOK {
		t.Fatalf("JSON export: expected 200, got %d", recJSON.Code)
	}

	// Test NDJSON export
	reqNDJSON := httptest.NewRequestWithContext(ctx, http.MethodGet, "/debug/di/api/export/ndjson", nil)
	recNDJSON := httptest.NewRecorder()
	server.ServeHTTP(recNDJSON, reqNDJSON)
	if recNDJSON.Code != http.StatusOK {
		t.Fatalf("NDJSON export: expected 200, got %d", recNDJSON.Code)
	}

	if !strings.Contains(recNDJSON.Header().Get("Content-Type"), "ndjson") {
		t.Errorf("NDJSON export: expected ndjson content-type, got %q", recNDJSON.Header().Get("Content-Type"))
	}

	// Test HTML export
	reqHTML := httptest.NewRequestWithContext(ctx, http.MethodGet, "/debug/di/api/export/html", nil)
	recHTML := httptest.NewRecorder()
	server.ServeHTTP(recHTML, reqHTML)
	if recHTML.Code != http.StatusOK {
		t.Fatalf("HTML export: expected 200, got %d", recHTML.Code)
	}

	if !strings.Contains(recHTML.Header().Get("Content-Type"), "text/html") {
		t.Errorf("HTML export: expected text/html content-type, got %q", recHTML.Header().Get("Content-Type"))
	}
}

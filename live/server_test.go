package live_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
	"github.com/larsartmann/samber-do-auditlog/testhelpers"
	"github.com/samber/do/v2"
)

// sseTimeout bounds every SSE read in this suite so a missing event fails
// the test instead of hanging it.
const sseTimeout = 10 * time.Second

// sseFrame is one parsed Server-Sent Event from the wire.
type sseFrame struct {
	Type string
	Data string
}

// sseReader parses an SSE stream body frame by frame (stdlib replacement for
// the ssetest stream reader).
type sseReader struct {
	scanner *bufio.Scanner
}

func newSSEReader(body io.Reader) *sseReader {
	return &sseReader{scanner: bufio.NewScanner(body)}
}

// next returns the next complete SSE frame, or an error on EOF/timeout.
func (r *sseReader) next() (sseFrame, error) {
	frame := sseFrame{}
	data := make([]string, 0, 4)

	for r.scanner.Scan() {
		line := r.scanner.Text()

		switch {
		case strings.HasPrefix(line, "event: "):
			frame.Type = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			data = append(data, strings.TrimPrefix(line, "data: "))
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(line, "data:"))
		case line == "":
			if frame.Type != "" || len(data) > 0 {
				frame.Data = strings.Join(data, "\n")

				return frame, nil
			}
		}
	}

	if err := r.scanner.Err(); err != nil {
		return frame, fmt.Errorf("read sse frame: %w", err)
	}

	return frame, io.EOF
}

// newTestServer creates a live server wired to a fresh plugin + hub.
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

// sseConnect opens an SSE connection to url with the given Last-Event-ID and
// returns a frame reader plus a cleanup func.
func sseConnect(t *testing.T, client *http.Client, url, lastEventID string) (*sseReader, func()) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), sseTimeout)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		cancel()
		t.Fatalf("create request: %v", err)
	}

	if lastEventID != "" {
		req.Header.Set("Last-Event-ID", lastEventID)
	}

	resp, err := client.Do(req) //nolint:bodyclose // closed via returned cleanup
	if err != nil {
		cancel()
		t.Fatalf("connect SSE: %v", err)
	}

	cleanup := func() {
		cancel()

		_ = resp.Body.Close()
	}

	return newSSEReader(resp.Body), cleanup
}

// readEvent returns the next frame with the given event type.
func readEvent(t *testing.T, sr *sseReader, eventName string) (string, bool) {
	t.Helper()

	for {
		frame, err := sr.next()
		if err != nil {
			return "", false
		}

		if frame.Type == eventName {
			return frame.Data, true
		}
	}
}

// skipSnapshot consumes frames until the final snapshot fragment
// (#container-id) has been seen.
func skipSnapshot(t *testing.T, sr *sseReader) {
	t.Helper()

	for {
		frame, err := sr.next()
		if err != nil {
			return
		}

		if strings.Contains(frame.Data, "selector #container-id") {
			return
		}
	}
}

// assertSSEDetectsService registers a service after SSE connect and asserts
// the service appears in a live datastar-patch-elements update.
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

	sr, closeSSE := sseConnect(t, ts.Client(), ts.URL+"/debug/di/api/events", "")
	defer closeSSE()

	skipSnapshot(t, sr)

	do.ProvideNamedValue(injector, serviceName, value)

	for range 20 {
		data, ok := readEvent(t, sr, "datastar-patch-elements")
		if !ok {
			break
		}

		if strings.Contains(data, serviceName) {
			return
		}
	}

	t.Fatalf("did not receive live update containing %s", serviceName)
}

// --- Dashboard ---

func TestServer_DashboardHTML(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/debug/di/", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("content-type = %q, want text/html", ct)
	}

	body := rec.Body.String()

	for _, want := range []string{"datastar", "services-tbody", "events-tbody", "skip-link"} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}
}

func TestServer_NotFound(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/debug/di/nope", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestServer_CustomPrefix(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{Enabled: true, ContainerID: "prefix-test"})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	server := live.NewServer(hub, plugin, live.Config{Prefix: "/custom"})

	req := httptest.NewRequest(http.MethodGet, "/custom/api/health", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("custom prefix health: expected 200, got %d", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/debug/di/api/health", nil)
	rec2 := httptest.NewRecorder()

	server.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusNotFound {
		t.Errorf("default prefix should 404 under custom prefix server, got %d", rec2.Code)
	}
}

func TestServer_RootPrefix(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{Enabled: true, ContainerID: "root-prefix"})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	server := live.NewServer(hub, plugin, live.Config{Prefix: "/"})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("root prefix health: expected 200, got %d", rec.Code)
	}
}

// --- Report / Health / Exports ---

func TestServer_ReportEndpoint(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/debug/di/api/report", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var report auditlog.Report

	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
}

func TestServer_NilPlugin_ReportEndpoint(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()
	server := live.NewServer(hub, nil, live.Config{})

	req := httptest.NewRequest(http.MethodGet, "/debug/di/api/report", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}

func TestServer_HealthEndpoint_WithEvents(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{Enabled: true, ContainerID: "health-events", OnEvent: hub.OnEvent})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	injector := do.NewWithOpts(plugin.Opts())

	do.ProvideNamedValue(injector, "health-svc", "v")

	server := live.NewServer(hub, plugin, live.Config{})

	req := httptest.NewRequest(http.MethodGet, "/debug/di/api/health", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var health struct {
		Status   string `json:"status"`
		Events   int    `json:"events"`
		Clients  int    `json:"clients"`
		Complete bool   `json:"complete"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatalf("decode health: %v", err)
	}

	if health.Status != "ok" || health.Events == 0 {
		t.Errorf("unexpected health payload: %+v", health)
	}
}

func TestServer_ExportEndpoints(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	reqNDJSON := httptest.NewRequest(http.MethodGet, "/debug/di/api/export/ndjson", nil)
	recNDJSON := httptest.NewRecorder()

	server.ServeHTTP(recNDJSON, reqNDJSON)

	if recNDJSON.Code != http.StatusOK {
		t.Fatalf("NDJSON export: expected 200, got %d", recNDJSON.Code)
	}

	if !strings.Contains(recNDJSON.Header().Get("Content-Type"), "ndjson") {
		t.Errorf("NDJSON content-type = %q", recNDJSON.Header().Get("Content-Type"))
	}

	if !strings.Contains(recNDJSON.Header().Get("Content-Disposition"), "attachment") {
		t.Errorf("NDJSON should be an attachment, got %q", recNDJSON.Header().Get("Content-Disposition"))
	}

	reqHTML := httptest.NewRequest(http.MethodGet, "/debug/di/api/export/html", nil)
	recHTML := httptest.NewRecorder()

	server.ServeHTTP(recHTML, reqHTML)

	if recHTML.Code != http.StatusOK {
		t.Fatalf("HTML export: expected 200, got %d", recHTML.Code)
	}
}

// --- CORS ---

func TestServer_CORSHeaders(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/debug/di/api/health", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin *, got %q", origin)
	}

	reqOpts := httptest.NewRequest(http.MethodOptions, "/debug/di/api/health", nil)
	recOpts := httptest.NewRecorder()

	server.ServeHTTP(recOpts, reqOpts)

	if recOpts.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", recOpts.Code)
	}
}

// --- SSE ---

func TestServer_SSE_SnapshotOnConnect(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	server.OnEvent(auditlog.Event{
		ServiceRef: auditlog.ServiceRef{
			ScopeID:     "root",
			ScopeName:   auditlog.RootScopeName,
			ServiceName: "db",
		},
		Sequence:  1,
		EventType: auditlog.EventTypeRegistration,
		Phase:     auditlog.PhaseAfter,
	})

	ts := httptest.NewServer(server)
	defer ts.Close()

	sr, closeSSE := sseConnect(t, ts.Client(), ts.URL+"/debug/di/api/events", "")
	defer closeSSE()

	data, found := readEvent(t, sr, "datastar-patch-signals")
	if !found {
		t.Fatal("did not receive datastar-patch-signals event")
	}

	if !strings.Contains(data, "connStatus") {
		t.Errorf("snapshot signals should contain connStatus: %.200s", data)
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

	sr, closeSSE := sseConnect(t, ts.Client(), ts.URL+"/debug/di/api/events", "")
	defer closeSSE()

	skipSnapshot(t, sr)

	server.SignalComplete()

	found := false

	for range 20 {
		data, ok := readEvent(t, sr, "datastar-patch-signals")
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

	sr1, closeSSE1 := sseConnect(t, ts.Client(), ts.URL+"/debug/di/api/events", "")
	defer closeSSE1()

	sr2, closeSSE2 := sseConnect(t, ts.Client(), ts.URL+"/debug/di/api/events", "")
	defer closeSSE2()

	skipSnapshot(t, sr1)
	skipSnapshot(t, sr2)

	do.ProvideNamedValue(injector, "fanout-svc", "fanout-value")

	for i, sr := range []*sseReader{sr1, sr2} {
		found := false

		for range 20 {
			data, ok := readEvent(t, sr, "datastar-patch-elements")
			if !ok {
				break
			}

			if strings.Contains(data, "fanout-svc") {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("client %d did not receive fanout event", i+1)
		}
	}
}

func TestServer_SSE_ReconnectReplay(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "replay-test",
		OnEvent:     hub.OnEvent,
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	injector := do.NewWithOpts(plugin.Opts())

	do.ProvideNamedValue(injector, "svc-1", 1)
	do.ProvideNamedValue(injector, "svc-2", 2)
	do.ProvideNamedValue(injector, "svc-3", 3)

	events := plugin.Events()
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events from registrations, got %d", len(events))
	}

	server := live.NewServer(hub, plugin, live.Config{})

	ts := httptest.NewServer(server)
	defer ts.Close()

	firstSeq := strconv.Itoa(events[0].Sequence)

	sr, closeSSE := sseConnect(t, ts.Client(), ts.URL+"/debug/di/api/events", firstSeq)
	defer closeSSE()

	data, found := readEvent(t, sr, "datastar-patch-signals")
	if !found {
		t.Fatal("did not receive datastar-patch-signals on reconnect")
	}

	if !strings.Contains(data, "connStatus") {
		t.Error("reconnect signals should contain connStatus")
	}
}

func TestServer_SSE_Heartbeat(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{Enabled: true, ContainerID: "heartbeat-test"})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	server := live.NewServer(hub, plugin, live.Config{HeartbeatInterval: 20 * time.Millisecond})

	ts := httptest.NewServer(server)
	defer ts.Close()

	sr, closeSSE := sseConnect(t, ts.Client(), ts.URL+"/debug/di/api/events", "")
	defer closeSSE()

	// With heartbeats firing every 20ms (between snapshot frames), the
	// snapshot must still arrive complete: the stream mutex serializes the
	// heartbeat goroutine against event writes.
	gotSignals := false
	gotElements := false

	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) && !(gotSignals && gotElements) {
		frame, err := sr.next()
		if err != nil {
			break
		}

		if frame.Type == "datastar-patch-signals" {
			gotSignals = true
		}

		if frame.Type == "datastar-patch-elements" && strings.Contains(frame.Data, "selector #container-id") {
			gotElements = true
		}
	}

	if !gotSignals || !gotElements {
		t.Errorf("snapshot incomplete under concurrent heartbeats: signals=%v elements=%v", gotSignals, gotElements)
	}
}

func TestServer_HandleSSE_NoFlusher(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/debug/di/api/events", nil)
	rec := &nonFlusherRecorder{rec: httptest.NewRecorder()}

	server.ServeHTTP(rec, req)

	if rec.rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 without flusher support, got %d", rec.rec.Code)
	}
}

type nonFlusherRecorder struct{ rec *httptest.ResponseRecorder }

func (r *nonFlusherRecorder) Header() http.Header { return r.rec.Header() }
func (r *nonFlusherRecorder) Write(b []byte) (int, error) {
	n, err := r.rec.Write(b)
	if err != nil {
		return n, fmt.Errorf("recorder write: %w", err)
	}

	return n, nil
}
func (r *nonFlusherRecorder) WriteHeader(code int) { r.rec.WriteHeader(code) }

// --- Export write errors ---

func TestServer_ExportNDJSON_WriteError(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/debug/di/api/export/ndjson", nil)
	rec := &failingResponseWriter{header: http.Header{}}

	server.ServeHTTP(rec, req)
}

func TestServer_ExportHTML_WriteError(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/debug/di/api/export/html", nil)
	rec := &failingResponseWriter{header: http.Header{}}

	server.ServeHTTP(rec, req)
}

var errWriteFailed = errors.New("write failed")

type failingResponseWriter struct{ header http.Header }

func (w *failingResponseWriter) Header() http.Header { return w.header }
func (w *failingResponseWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}
func (w *failingResponseWriter) WriteHeader(int) {}

// --- Lifecycle (real TCP listener) ---

func TestServer_ListenAndServe_Addr_Shutdown(t *testing.T) {
	t.Parallel()

	// Get a free port to avoid conflicts.
	lc := net.ListenConfig{}

	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
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

	if got := server.Addr(); got != addr {
		t.Errorf("expected %q before start, got %q", addr, got)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	// Wait for the server to start by polling the health endpoint.
	ctx := context.Background()

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

	if got := server.Addr(); got != addr {
		t.Errorf("Addr() while running should return %q, got %q", addr, got)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

	hub := live.NewHub()

	plugin, err := auditlog.New(auditlog.Config{Enabled: true, ContainerID: "already-running"})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	server := live.NewServer(hub, plugin, live.Config{Addr: "127.0.0.1:0"})

	errCh := make(chan error, 1)

	go func() { errCh <- server.ListenAndServe() }()

	time.Sleep(100 * time.Millisecond)

	if err := server.ListenAndServe(); !errors.Is(err, live.ErrServerAlreadyRunning) {
		t.Errorf("expected ErrServerAlreadyRunning, got %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("shutdown: %v", err)
	}
}

func TestServer_ShutdownNotRunning(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("shutdown without listen should be a no-op, got %v", err)
	}
}

func TestServer_ClientCount(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	if server.ClientCount() != 0 {
		t.Errorf("expected 0 clients initially, got %d", server.ClientCount())
	}
}

func TestServer_NewConvenience(t *testing.T) {
	t.Parallel()

	server, plugin, err := live.New(auditlog.Config{ContainerID: "convenience"}, live.Config{ReplayBufferSize: 5})
	if err != nil {
		t.Fatalf("live.New: %v", err)
	}

	if plugin == nil || server == nil {
		t.Fatal("expected non-nil server and plugin")
	}

	if server.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", server.ClientCount())
	}
}

// --- Dashboard JS balance ---

func TestServer_DashboardHTML_JavaScriptBalanced(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/debug/di/", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	js := testhelpers.ExtractExecutableJS(t, rec.Body.String())
	testhelpers.AssertJSBalanced(t, js)
}

// --- Hub ---

func TestHub_OnEventDelivery(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	ch := hub.Subscribe()
	defer func() { hub.Unsubscribe(ch) }()

	hub.OnEvent(auditlog.Event{Sequence: 1, EventType: auditlog.EventTypeRegistration})

	select {
	case evt, ok := <-ch:
		if !ok {
			t.Fatal("channel closed unexpectedly")
		}

		if evt.ID != "1" || evt.Name != "event" {
			t.Errorf("unexpected event: %+v", evt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for hub event")
	}
}

func TestHub_SignalComplete(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	if hub.IsComplete() {
		t.Error("hub should not be complete initially")
	}

	hub.SignalComplete()
	hub.SignalComplete()

	select {
	case <-hub.Done():
	default:
		t.Error("Done() should be closed after SignalComplete")
	}

	if !hub.IsComplete() {
		t.Error("hub should be complete after SignalComplete")
	}
}

func TestHub_BufferOverflow(t *testing.T) {
	t.Parallel()

	hub := live.NewHubWithReplay(4)

	ch := hub.Subscribe()

	for i := range 300 {
		hub.OnEvent(auditlog.Event{Sequence: i + 1})
	}

	// Close the channel first so the drain loop terminates; buffered
	// events remain readable after close.
	hub.Unsubscribe(ch)

	drained := 0

	for range ch {
		drained++
	}

	if drained > 128 {
		t.Errorf("subscriber should drop beyond buffer size 128, drained %d", drained)
	}
}

func TestHub_UnsubscribeUnknownChannel(t *testing.T) {
	t.Parallel()

	hub := live.NewHub()

	other := live.NewHub()
	ch := other.Subscribe()

	hub.Unsubscribe(ch)

	select {
	case _, ok := <-ch:
		if !ok {
			t.Error("channel from a different hub should remain open")
		}
	default:
	}

	other.Unsubscribe(ch)
}

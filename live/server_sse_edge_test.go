package live_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
)

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
		ScopeID:     "root",
		ScopeName:   "[root]",
		ServiceName: "health-test-svc",
		Sequence:    1,
		EventType:   auditlog.EventTypeRegistration,
		Phase:       auditlog.PhaseAfter,
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

	if !strings.Contains(body, `"buffer_size"`) {
		t.Errorf("health response should contain buffer_size field: %s", body)
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

	sr, closeSSE := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE()

	// The connection should succeed even with nil plugin.
	// The complete event may or may not arrive depending on timing,
	// but the connection itself should not error.
	_ = sr
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

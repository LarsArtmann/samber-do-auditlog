package live_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
	"github.com/larsartmann/samber-do-auditlog/testhelpers"
	"github.com/samber/do/v2"
)

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
}

func TestServer_DashboardHTML_JavaScriptBalanced(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	ctx := t.Context()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/debug/di/", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	js := testhelpers.ExtractExecutableJS(t, rec.Body.String())
	if len(js) < 100 {
		t.Fatalf("extracted JS too short (%d bytes)", len(js))
	}

	testhelpers.AssertJSBalanced(t, js)
}

// failingBodyRecorder wraps httptest.ResponseRecorder but fails every Write,
// letting tests exercise the export handlers' write-error branches.
type failingBodyRecorder struct {
	httptest.ResponseRecorder
}

func (f *failingBodyRecorder) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

var errWriteFailed error = writeError("simulated write failure")

type writeError string

func (e writeError) Error() string { return string(e) }

// newTestServerWithEvents returns a server whose plugin has recorded events,
// so export handlers have data to write.
func newTestServerWithEvents(t *testing.T) *live.Server {
	t.Helper()

	plugin, err := auditlog.New(auditlog.Config{
		Enabled:     true,
		ContainerID: "export-test",
	})
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	injector := do.NewWithOpts(plugin.Opts())
	do.ProvideNamed(injector, "svc", func(do.Injector) (*strings.Builder, error) {
		return &strings.Builder{}, nil
	})

	if _, err := do.InvokeNamed[*strings.Builder](injector, "svc"); err != nil {
		t.Fatalf("invoke: %v", err)
	}

	return live.NewServer(live.NewHub(), plugin, live.Config{})
}

func TestServer_ExportNDJSON_WriteError(t *testing.T) {
	t.Parallel()

	server := newTestServerWithEvents(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/debug/di/api/export/ndjson", nil)
	rec := &failingBodyRecorder{}

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on write failure, got %d", rec.Code)
	}
}

func TestServer_ExportHTML_WriteError(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/debug/di/api/export/html", nil)
	rec := &failingBodyRecorder{}

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on write failure, got %d", rec.Code)
	}
}

func TestServer_ReportWithoutPlugin(t *testing.T) {
	t.Parallel()

	// A server constructed without a plugin answers 503 on data endpoints
	// (requirePlugin path) instead of panicking.
	server := live.NewServer(live.NewHub(), nil, live.Config{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/debug/di/api/report", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 without plugin, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

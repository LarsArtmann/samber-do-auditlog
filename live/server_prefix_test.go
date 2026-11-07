package live_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
)

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

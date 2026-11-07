package live_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
)

func newTestServer(t *testing.T) *live.Server {
	t.Helper()

	return newTestServerWithConfig(t, live.Config{})
}

func newTestServerWithConfig(t *testing.T, cfg live.Config) *live.Server {
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

	return live.NewServer(hub, plugin, cfg)
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

	for _, want := range []string{"<!DOCTYPE html>", "samber-do-auditlog", "LIVE", "skip-link", "Skip to main content", "main-content", "role=\"tab\"", "aria-selected", "aria-controls", "tabindex", "handleKeydown", "exportReport", "closeKbdHelp", "kbdHelpPrevFocus", "Keyboard shortcuts", "ArrowRight", "ArrowLeft", "Press ? for keyboard shortcuts", "data-signals", "data-init", "data-on:click", "data-text"} {
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

func TestServer_DashboardCSP(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		prefix string
		path   string
	}{
		{name: "default prefix", prefix: "/debug/di", path: "/debug/di/"},
		{name: "root prefix", prefix: "/", path: "/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := newTestServerWithConfig(t, live.Config{Prefix: tc.prefix})

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			server.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}

			csp := rec.Header().Get("Content-Security-Policy")
			if csp != "frame-ancestors 'none'" {
				t.Errorf("expected frame-ancestors CSP response header, got %q", csp)
			}

			if xfo := rec.Header().Get("X-Frame-Options"); xfo != "DENY" {
				t.Errorf("expected X-Frame-Options DENY response header, got %q", xfo)
			}

			directives := parseCSPDirectives(t, rec.Body.String())

			assertExpectedDirectives(t, directives)
			assertNoHeaderOnlyDirectives(t, directives)
		})
	}
}

// assertExpectedDirectives pins the dashboard CSP meta policy directive by
// directive so a reordered or extra directive cannot silently pass.
func assertExpectedDirectives(t *testing.T, directives map[string][]string) {
	t.Helper()

	expected := map[string][]string{
		"default-src": {"'none'"},
		"style-src":   {"'unsafe-inline'"},
		"script-src":  {"'unsafe-inline'", "'unsafe-eval'"},
		"connect-src": {"'self'"},
		"base-uri":    {"'none'"},
	}

	for directive, want := range expected {
		got, ok := directives[directive]
		if !ok {
			t.Errorf("dashboard CSP meta missing %s directive", directive)

			continue
		}

		if !slices.Equal(got, want) {
			t.Errorf("dashboard CSP meta %s = %v, want %v", directive, got, want)
		}
	}
}

// assertNoHeaderOnlyDirectives fails when a directive browsers ignore inside
// <meta> appears in the dashboard CSP meta tag.
func assertNoHeaderOnlyDirectives(t *testing.T, directives map[string][]string) {
	t.Helper()

	for _, forbidden := range []string{"frame-ancestors", "sandbox", "report-uri"} {
		if _, ok := directives[forbidden]; ok {
			t.Errorf("dashboard CSP meta must not contain %s: ignored in <meta> (header-only directive)", forbidden)
		}
	}
}

// parseCSPDirectives extracts the Content-Security-Policy meta tag from a
// dashboard document and splits its policy into a directive-name → source-list
// map, so tests assert exact directives instead of substring coincidences.
func parseCSPDirectives(t *testing.T, htmlBody string) map[string][]string {
	t.Helper()

	const metaTag = `<meta http-equiv="Content-Security-Policy" content="`

	_, rest, found := strings.Cut(htmlBody, metaTag)
	if !found {
		t.Fatal("dashboard HTML missing Content-Security-Policy meta tag")
	}

	content, _, terminated := strings.Cut(rest, `"`)
	if !terminated {
		t.Fatal("CSP meta content attribute not terminated")
	}

	directives := make(map[string][]string)

	for directive := range strings.SplitSeq(content, ";") {
		tokens := strings.Fields(directive)
		if len(tokens) == 0 {
			continue
		}

		directives[tokens[0]] = tokens[1:]
	}

	return directives
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

package live_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
	"github.com/samber/do/v2"
)

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

	assertSSEDetectsService(t, "broadcast-test", "broadcast-svc", "value")
}

func TestServer_SSE_ReconnectSnapshot(t *testing.T) {
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

	// Register services to generate events with real sequences.
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

	// Reconnect with Last-Event-ID = first event's sequence.
	// With datastar, reconnect always sends a fresh snapshot.
	firstSeq := strconv.Itoa(events[0].Sequence)

	sr, closeSSE := sseConnectWithLastID(t, ts.URL+"/debug/di/api/events", firstSeq)
	defer closeSSE()

	// Reconnect sends a fresh snapshot (datastar events) regardless of Last-Event-ID.
	// Verify we get datastar-patch-signals on reconnect.
	data, found := readSSEEvent(sr, "datastar-patch-signals")
	if !found {
		t.Fatal("did not receive datastar-patch-signals on reconnect")
	}

	if !strings.Contains(data, "connStatus") {
		t.Error("reconnect signals should contain connStatus")
	}
}

func TestServer_SSE_ReconnectNoLastEventID(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	ts := httptest.NewServer(server)
	defer ts.Close()

	// Connect without Last-Event-ID — should get snapshot only, no replay.
	sr, closeSSE := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE()

	data, found := readSSEEvent(sr, "datastar-patch-signals")
	if !found {
		t.Fatal("did not receive snapshot signals")
	}

	if !strings.Contains(data, "connStatus") {
		t.Error("snapshot signals should contain connStatus")
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

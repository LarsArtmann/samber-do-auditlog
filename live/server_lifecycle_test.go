package live_test

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/live"
)

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

	sr, closeSSE := sseConnect(t, ts.URL+"/debug/di/api/events")
	defer closeSSE()

	// Skip the snapshot event.
	skipSnapshot(sr)

	// Heartbeats are SSE comment lines (: heartbeat), which StreamReader
	// ignores. Use a raw scanner on a separate connection to detect them.
	ctx2, cancel2 := context.WithCancel(t.Context())
	defer cancel2()

	req2, err := http.NewRequestWithContext(ctx2, http.MethodGet, ts.URL+"/debug/di/api/events", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("connect SSE: %v", err)
	}

	defer func() { _ = resp2.Body.Close() }()

	scanner := bufio.NewScanner(resp2.Body)

	// Skip the snapshot (read until blank line dispatches the frame).
	for scanner.Scan() {
		if scanner.Text() == "" {
			break
		}
	}

	err = scanner.Err()
	if err != nil {
		t.Fatalf("read snapshot frame: %v", err)
	}

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

	err = scanner.Err()
	if err != nil {
		t.Fatalf("read heartbeat lines: %v", err)
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

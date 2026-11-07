package forward_test

import (
	"context"
	"encoding/json/v2"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

func startFakeCollector(t *testing.T) (*fakeCollector, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "audit-runs.sock")

	return startFakeCollectorAtPath(t, path), path
}

func startFakeCollectorAtPath(t *testing.T, path string) *fakeCollector {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir socket dir: %v", err)
	}

	var lc net.ListenConfig

	listener, err := lc.Listen(context.Background(), "unix", path)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}

	collector := &fakeCollector{listener: listener}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /events", func(writer http.ResponseWriter, request *http.Request) {
		var envelope map[string]any
		if err := json.UnmarshalRead(request.Body, &envelope); err != nil {
			writer.WriteHeader(http.StatusBadRequest)

			return
		}

		collector.mu.Lock()
		collector.envelopes = append(collector.envelopes, envelope)
		collector.mu.Unlock()

		writer.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /health", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})

	collector.server = &http.Server{Handler: mux, ReadHeaderTimeout: time.Second} //nolint:exhaustruct // test double

	collector.wg.Go(func() {
		_ = collector.server.Serve(listener)
	})

	t.Cleanup(func() {
		_ = collector.server.Close()
		collector.wg.Wait()
	})

	return collector
}

func (c *fakeCollector) awaitEnvelopes(t *testing.T, want int) []map[string]any {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		count := len(c.envelopes)
		c.mu.Unlock()

		if count >= want {
			c.mu.Lock()
			defer c.mu.Unlock()

			return append([]map[string]any(nil), c.envelopes...)
		}

		time.Sleep(25 * time.Millisecond)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	t.Fatalf("collector saw %d envelopes, want %d", len(c.envelopes), want)

	return nil
}

func diEvent(runID string, sequence int, eventType auditlog.EventType, phase auditlog.Phase) auditlog.Event {
	return auditlog.Event{
		ScopeID:     "scope-1",
		ScopeName:   auditlog.RootScopeName,
		ServiceName: "database",
		RunID:       auditlog.RunID(runID),
		Sequence:    sequence,
		Timestamp:   time.Now().UTC(),
		EventType:   eventType,
		Phase:       phase,
	}
}

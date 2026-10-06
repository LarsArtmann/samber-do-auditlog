package forward_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/forward"
)

// fakeCollector accepts ingest POSTs on a unix socket and records them.
type fakeCollector struct {
	mu        sync.Mutex
	envelopes []map[string]any

	listener net.Listener
	server   *http.Server
	wg       sync.WaitGroup
}

func startFakeCollector(t *testing.T) (*fakeCollector, string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "audit-runs.sock")

	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}

	collector := &fakeCollector{listener: listener}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /events", func(writer http.ResponseWriter, request *http.Request) {
		var envelope map[string]any
		if err := json.NewDecoder(request.Body).Decode(&envelope); err != nil {
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
	collector.wg.Add(1)

	go func() {
		defer collector.wg.Done()
		_ = collector.server.Serve(listener)
	}()

	t.Cleanup(func() {
		_ = collector.server.Close()
		collector.wg.Wait()
	})

	return collector, path
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
		ServiceRef: auditlog.ServiceRef{
			ScopeID:     "scope-1",
			ScopeName:   auditlog.RootScopeName,
			ServiceName: "database",
		},
		RunID:     auditlog.RunID(runID),
		Sequence:  sequence,
		Timestamp: time.Now().UTC(),
		EventType: eventType,
		Phase:     phase,
	}
}

func TestForwarder_UnixRoundTripAndCompletionDerivation(t *testing.T) {
	collector, path := startFakeCollector(t)

	fwd := forward.NewWithTarget("unix://"+path, "test-app")
	if !fwd.Enabled() {
		t.Fatal("explicit unix target must enable the forwarder")
	}

	t.Cleanup(func() { _ = fwd.Shutdown(context.Background()) })

	fwd.OnEvent(diEvent("run-1", 1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))
	fwd.OnEvent(diEvent("run-1", 2, auditlog.EventTypeInvocation, auditlog.PhaseAfter))
	fwd.OnEvent(diEvent("run-1", 3, auditlog.EventTypeShutdown, auditlog.PhaseAfter))

	envelopes := collector.awaitEnvelopes(t, 1)
	if len(envelopes) != 1 {
		t.Fatalf("expected one batch envelope, got %d", len(envelopes))
	}

	first := envelopes[0]
	if first["kind"] != "di" || first["sourceId"] != "test-app" || first["runId"] != "run-1" {
		t.Fatalf("envelope identity wrong: %+v", first)
	}

	if first["complete"] != true {
		t.Fatalf("root-scope shutdown must derive complete=true: %+v", first)
	}

	events, ok := first["events"].([]any)
	if !ok || len(events) != 3 {
		t.Fatalf("batch must carry 3 events: %+v", first["events"])
	}

	// The events must ride the upstream snake_case tags verbatim.
	firstEvent, _ := events[0].(map[string]any)
	if firstEvent["event_type"] != "invocation" || firstEvent["service_name"] != "database" {
		t.Fatalf("event JSON must keep the auditlog wire tags: %+v", firstEvent)
	}
}

func TestForwarder_ShutdownFlushes(t *testing.T) {
	collector, path := startFakeCollector(t)

	fwd := forward.NewWithTarget(path, "test-app") // bare path = unix target
	if !fwd.Enabled() {
		t.Fatal("bare absolute path must enable the forwarder")
	}

	fwd.OnEvent(diEvent("run-2", 1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := fwd.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	envelopes := collector.awaitEnvelopes(t, 1)
	if envelopes[0]["runId"] != "run-2" {
		t.Fatalf("shutdown must flush buffered events: %+v", envelopes[0])
	}
}

func TestForwarder_ExplicitCompleteAfterEvents(t *testing.T) {
	collector, path := startFakeCollector(t)

	fwd := forward.NewWithTarget("unix://"+path, "test-app")

	t.Cleanup(func() { _ = fwd.Shutdown(context.Background()) })

	fwd.OnEvent(diEvent("run-3", 1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))
	fwd.Complete()

	envelopes := collector.awaitEnvelopes(t, 2)
	if envelopes[0]["runId"] != "run-3" || envelopes[0]["complete"] != false {
		t.Fatalf("event batch must carry the events uncompleted: %+v", envelopes[0])
	}

	if envelopes[1]["complete"] != true || envelopes[1]["runId"] != "run-3" {
		t.Fatalf("explicit Complete() must mark the last seen run: %+v", envelopes[1])
	}
}

func TestForwarder_OffModes(t *testing.T) {
	fwd := forward.NewWithTarget("off", "test-app")
	if fwd.Enabled() {
		t.Fatal("off target must disable forwarding")
	}

	// All methods must be safe no-ops.
	fwd.OnEvent(diEvent("run-x", 1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))
	fwd.Complete()

	if err := fwd.Shutdown(context.Background()); err != nil {
		t.Fatalf("disabled shutdown: %v", err)
	}

	if fwd.Dropped() != 0 {
		t.Fatalf("disabled forwarder must not drop: %d", fwd.Dropped())
	}

	// Default (empty target) with a dead conventional socket stays off.
	_ = os.Unsetenv("XDG_RUNTIME_DIR")

	fwd = forward.NewWithTarget("", "test-app")
	if fwd.Enabled() {
		t.Fatal("dead default socket must disable forwarding")
	}
}

func TestForwarder_DropOldestUnderBurst(t *testing.T) {
	// A collector that never reads fast enough is modeled by a target with
	// a full buffer: burst more events than channelCap without a flush.
	path := filepath.Join(t.TempDir(), "missing.sock")

	fwd := forward.NewWithTarget("unix://"+path, "burst-app")
	if !fwd.Enabled() {
		t.Fatal("explicit target enables even when the socket is dead")
	}

	t.Cleanup(func() { _ = fwd.Shutdown(context.Background()) })

	for i := range 5000 {
		fwd.OnEvent(diEvent("burst", i+1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))
	}

	if fwd.Dropped() == 0 {
		t.Fatal("a burst beyond the buffer cap must report drops")
	}
}

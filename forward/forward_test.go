package forward_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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

	path := filepath.Join(t.TempDir(), "audit-runs.sock")

	return startFakeCollectorAtPath(t, path), path
}

func startFakeCollectorAtPath(t *testing.T, path string) *fakeCollector {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir socket dir: %v", err)
	}

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

	// Default (empty target) with a dead conventional socket stays ARMED:
	// delivery starts whenever the socket answers (see the activation test).
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	fwd = forward.NewWithTarget("", "test-app")
	if !fwd.Enabled() {
		t.Fatal("dead default socket must stay armed, not disable forwarding")
	}

	if err := fwd.Shutdown(context.Background()); err != nil {
		t.Fatalf("armed shutdown: %v", err)
	}
}

func TestForwarder_AutoTargetActivatesWhenSocketAppears(t *testing.T) {
	// The conventional socket path is pinned to a SHORT temp runtime dir —
	// unix socket paths must stay under the kernel's 108-byte sun_path limit.
	runtimeDir, err := os.MkdirTemp("", "fwd-xdg-*")
	if err != nil {
		t.Fatalf("temp runtime dir: %v", err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(runtimeDir) })

	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)

	fwd := forward.NewWithTarget("", "late-app")
	if !fwd.Enabled() {
		t.Fatal("empty target must arm the forwarder")
	}

	t.Cleanup(func() { _ = fwd.Shutdown(context.Background()) })

	// Events buffered while PapDashboard is still down…
	fwd.OnEvent(diEvent("run-late", 1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))

	// …and the collector appears later at the conventional socket.
	collector := startFakeCollectorAtPath(t, filepath.Join(runtimeDir, "papdashboard", "audit-runs.sock"))

	// Pending events re-probe at flush cadence, so the buffered event must
	// arrive without any further action.
	envelopes := collector.awaitEnvelopes(t, 1)
	if envelopes[0]["runId"] != "run-late" || envelopes[0]["sourceId"] != "late-app" {
		t.Fatalf("activation must flush buffered events: %+v", envelopes[0])
	}

	// Live events keep flowing through the now-active target.
	fwd.OnEvent(diEvent("run-late", 2, auditlog.EventTypeInvocation, auditlog.PhaseAfter))
	fwd.Complete()

	all := collector.awaitEnvelopes(t, 3)

	last := all[len(all)-1]
	if last["complete"] != true || last["runId"] != "run-late" {
		t.Fatalf("explicit complete must flow after activation: %+v", last)
	}
}

func TestForwarder_HTTPTargetBearerKey(t *testing.T) {
	var gotAuth, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuth = request.Header.Get("Authorization")
		gotPath = request.URL.Path

		writer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	t.Setenv("DO_AUDITLOG_FORWARD_API_KEY", "secret-key")

	fwd := forward.NewWithTarget(server.URL, "http-app")
	if !fwd.Enabled() {
		t.Fatal("http target must enable the forwarder")
	}

	t.Cleanup(func() { _ = fwd.Shutdown(context.Background()) })

	fwd.OnEvent(diEvent("run-http", 1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))
	_ = fwd.Shutdown(context.Background())

	if gotAuth != "Bearer secret-key" {
		t.Fatalf("http target must send the bearer key, got %q", gotAuth)
	}

	if gotPath != "/events" {
		t.Fatalf("http target must POST the ingest path, got %q", gotPath)
	}
}

func TestForwarder_FanOutToUnixAndHTTP(t *testing.T) {
	collector, socketPath := startFakeCollector(t)

	httpSink := make(chan string, 8)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		httpSink <- string(body)

		writer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	fwd := forward.NewWithTarget("unix://"+socketPath+","+server.URL, "fan-app")
	if !fwd.Enabled() {
		t.Fatal("fan-out spec must enable the forwarder")
	}

	t.Cleanup(func() { _ = fwd.Shutdown(context.Background()) })

	fwd.OnEvent(diEvent("run-fan", 1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))
	_ = fwd.Shutdown(context.Background())

	envelopes := collector.awaitEnvelopes(t, 1)
	if envelopes[0]["runId"] != "run-fan" {
		t.Fatalf("unix leg must receive the batch: %+v", envelopes[0])
	}

	select {
	case body := <-httpSink:
		if !strings.Contains(body, `"runId":"run-fan"`) {
			t.Fatalf("http leg must receive the batch: %s", body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("http leg received nothing")
	}
}

func TestForwarder_PostFailuresCountedAndLogged(t *testing.T) {
	var logBuf bytes.Buffer

	original := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, nil)))
	t.Cleanup(func() { slog.SetDefault(original) })

	failures := atomic.Int64{}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		failures.Add(1)
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	fwd := forward.NewWithTarget(server.URL, "flaky-app")

	t.Cleanup(func() { _ = fwd.Shutdown(context.Background()) })

	for i := range 3 {
		fwd.OnEvent(diEvent("run-flaky", i+1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))
		time.Sleep(300 * time.Millisecond) // let flushes hit the failing target
	}

	if fwd.Failed() == 0 {
		t.Fatal("failed POSTs must be counted")
	}

	if !strings.Contains(logBuf.String(), "delivery failing") {
		t.Fatalf("failure transition must be logged once, got: %s", logBuf.String())
	}

	transitionLogs := strings.Count(logBuf.String(), "delivery failing")
	if transitionLogs != 1 {
		t.Fatalf("repeat failures must not re-log the transition, got %d: %s", transitionLogs, logBuf.String())
	}
}

func TestForwarder_BatchMaxChunksEnvelopes(t *testing.T) {
	collector, socketPath := startFakeCollector(t)

	t.Setenv("DO_AUDITLOG_FORWARD_BATCH_MAX", "2")

	fwd := forward.NewWithTarget("unix://"+socketPath, "chunk-app")

	for i := range 5 {
		fwd.OnEvent(diEvent("run-chunk", i+1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))
	}

	_ = fwd.Shutdown(context.Background())

	envelopes := collector.awaitEnvelopes(t, 3)
	if len(envelopes) < 3 {
		t.Fatalf("batch max 2 over 5 events must chunk into 3 envelopes, got %d", len(envelopes))
	}

	total := 0

	for _, envelope := range envelopes {
		events, _ := envelope["events"].([]any)
		total += len(events)
	}

	if total != 5 {
		t.Fatalf("chunks must carry every event exactly once, got %d of 5", total)
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

func TestDefaultSocketPath(t *testing.T) {
	runtimeDir := t.TempDir()

	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	if got, want := forward.DefaultSocketPath(), filepath.Join(runtimeDir, "papdashboard", "audit-runs.sock"); got != want {
		t.Fatalf("XDG set: got %q want %q", got, want)
	}

	t.Setenv("XDG_RUNTIME_DIR", "")
	if got, want := forward.DefaultSocketPath(), filepath.Join(os.TempDir(), "papdashboard", "audit-runs.sock"); got != want {
		t.Fatalf("empty XDG must fall back to the temp dir: got %q want %q", got, want)
	}
}

func TestNewWithTarget_SpecParsing(t *testing.T) {
	// Isolate the conventional socket: armed auto targets probe it in the
	// background and must never touch a developer's live PapDashboard.
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	cases := []struct {
		spec    string
		enabled bool
	}{
		{"OFF", false},
		{"Disabled", false},
		{"   ", true},                           // whitespace-only arms the default socket
		{"off,unix:///tmp/fwd-spec.sock", true}, // off entries drop, the rest stay
		{"unix://a.sock,,unix://b.sock", true},  // empty segments are skipped
		{"auto,http://127.0.0.1:1", true},       // junk entries arm the auto target
	}

	for _, tc := range cases {
		fwd := forward.NewWithTarget(tc.spec, "spec-app")

		if fwd.Enabled() != tc.enabled {
			t.Errorf("spec %q: Enabled() = %v, want %v", tc.spec, fwd.Enabled(), tc.enabled)
		}

		if err := fwd.Shutdown(context.Background()); err != nil {
			t.Errorf("spec %q shutdown: %v", tc.spec, err)
		}
	}
}

func TestNewWithTarget_SourceFallbackChain(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	// Garbage and out-of-range knobs exercise envInt's fallback and clamp
	// branches while the envelope assertions below run.
	t.Setenv("DO_AUDITLOG_FORWARD_BATCH_MAX", "not-a-number")
	t.Setenv("DO_AUDITLOG_FORWARD_FLUSH_MS", "1") // below-min clamps to 16ms: exercise the clamp branch without slowing the test

	// newSinkServer returns a collector URL whose every accepted envelope
	// body lands on the returned channel.
	newSinkServer := func(t *testing.T) (string, <-chan string) {
		t.Helper()

		sink := make(chan string, 4)

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			body, _ := io.ReadAll(request.Body)

			select {
			case sink <- string(body):
			default:
			}

			writer.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		return server.URL, sink
	}

	sourceOf := func(t *testing.T, payload string) string {
		t.Helper()

		var env map[string]any

		if err := json.Unmarshal([]byte(payload), &env); err != nil {
			t.Fatalf("envelope %q: %v", payload, err)
		}

		source, _ := env["sourceId"].(string)

		return source
	}

	waitFor := func(t *testing.T, sink <-chan string, want, description string) {
		t.Helper()

		select {
		case payload := <-sink:
			if got := sourceOf(t, payload); got != want {
				t.Fatalf("%s: got source %q want %q", description, got, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%s: no envelope arrived", description)
		}
	}

	// 1. A non-empty argument wins over a SET env override.
	t.Setenv("DO_AUDITLOG_FORWARD_SOURCE", "env-app")

	url, sink := newSinkServer(t)

	fwd := forward.NewWithTarget(url, "explicit-app")
	fwd.OnEvent(diEvent("run-src", 1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))
	waitFor(t, sink, "explicit-app", "explicit argument must beat the env override")

	if err := fwd.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	// 2. Empty argument: the env override applies.
	url, sink = newSinkServer(t)

	fwd2 := forward.NewWithTarget(url, "")
	fwd2.OnEvent(diEvent("run-src", 1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))
	waitFor(t, sink, "env-app", "empty argument must use the env override")

	if err := fwd2.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	// 3. Both empty: the executable name identifies the process.
	t.Setenv("DO_AUDITLOG_FORWARD_SOURCE", "")

	url, sink = newSinkServer(t)

	fwd3 := forward.NewWithTarget(url, "")
	fwd3.OnEvent(diEvent("run-src", 1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}

	waitFor(t, sink, filepath.Base(executable), "empty argument and env must use the executable name")

	if err := fwd3.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestForwarder_FailThenRecoverSkipsInactive(t *testing.T) {
	// "auto" arms the conventional socket (isolated to a dead temp dir), so
	// postAll must skip it while delivering to the live HTTP target. The
	// HTTP target 500s the first batch (counted, never retried in place)
	// and recovers on the next one.
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	var attempts atomic.Int64

	sink := make(chan string, 4)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if attempts.Add(1) == 1 {
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}

		body, _ := io.ReadAll(request.Body)

		select {
		case sink <- string(body):
		default:
		}

		writer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	fwd := forward.NewWithTarget("auto,"+server.URL, "mixed-app")
	t.Cleanup(func() { _ = fwd.Shutdown(context.Background()) })

	fwd.OnEvent(diEvent("run-rec", 1, auditlog.EventTypeInvocation, auditlog.PhaseBefore))

	time.Sleep(300 * time.Millisecond) // land event 2 in a later flush than the failed one

	fwd.OnEvent(diEvent("run-rec", 2, auditlog.EventTypeInvocation, auditlog.PhaseAfter))

	select {
	case payload := <-sink:
		var env map[string]any

		if err := json.Unmarshal([]byte(payload), &env); err != nil {
			t.Fatalf("envelope %q: %v", payload, err)
		}

		if env["runId"] != "run-rec" {
			t.Fatalf("recovered delivery carried the wrong run: %+v", env)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("delivery never recovered after the 500")
	}

	if fwd.Failed() == 0 {
		t.Fatal("the 500 must count as a delivery failure")
	}
}

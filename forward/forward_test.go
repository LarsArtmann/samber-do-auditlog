package forward_test

import (
	"context"
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
	// unix socket paths must stay under the kernel's 108-byte sun_path limit
	// (t.TempDir() derives from the test name and overflows it).
	runtimeDir, err := os.MkdirTemp("", "fwd-xdg-*") //nolint:usetesting // sun_path 108-byte limit needs a short prefix
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

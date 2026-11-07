package forward_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/forward"
)

func TestDefaultSocketPath(t *testing.T) {
	runtimeDir := t.TempDir()

	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)

	want := filepath.Join(runtimeDir, "papdashboard", "audit-runs.sock")
	if got := forward.DefaultSocketPath(); got != want {
		t.Fatalf("XDG set: got %q want %q", got, want)
	}

	t.Setenv("XDG_RUNTIME_DIR", "")

	want = filepath.Join(os.TempDir(), "papdashboard", "audit-runs.sock")
	if got := forward.DefaultSocketPath(); got != want {
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
	t.Setenv("DO_AUDITLOG_FORWARD_FLUSH_MS", "1") // below-min clamps to 16ms: clamp branch, fast test

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

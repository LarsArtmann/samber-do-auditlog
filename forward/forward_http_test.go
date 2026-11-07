package forward_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
	"github.com/larsartmann/samber-do-auditlog/forward"
)

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

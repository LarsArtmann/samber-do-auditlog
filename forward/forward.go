// Package forward streams audit events to a PapDashboard audit-run
// collector: batches events per run, POSTs them over a unix socket (zero
// config — the conventional local socket is probed until it appears) or
// HTTP, and marks run completion. Stdlib-only on purpose: this is a PUBLIC
// repo and must stay fetchable through proxy.golang.org.
//
// Zero-code wiring: live.New auto-attaches an enabled Forwarder to the
// audit pipeline; plain-plugin consumers compose it themselves:
//
//	fwd := forward.New("") // source defaults to the executable name
//	if fwd.Enabled() {
//		plugin, _ := auditlog.New(auditlog.Config{
//			OnEvent: auditlog.NewMultiWriter(hub.OnEvent, fwd.OnEvent).OnEvent,
//		})
//		defer fwd.Shutdown(context.Background())
//	}
//
// Target selection (DO_AUDITLOG_FORWARD_TARGET, comma-separated for
// fan-out to multiple collectors):
//
//	unset          arm the conventional socket
//	              ($XDG_RUNTIME_DIR/papdashboard/audit-runs.sock) and probe
//	              until it answers — events seen while it is down stay
//	              buffered, so a process that boots BEFORE PapDashboard
//	              still forwards its early run events once it appears
//	off, disabled  never forward
//	unix:///path   explicit unix socket
//	/path          bare absolute path is a unix socket
//	http(s)://…    remote collector (pair with DO_AUDITLOG_FORWARD_API_KEY)
//
// Delivery is best-effort: failed POSTs are counted (Failed), logged on
// state change, and never retried in place — the collector dedups by
// (run_id, sequence), so a later batch safely re-delivers anything lost.
package forward

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// Env knobs: target selection, HTTP bearer key, the source label the
// collector's UI shows for this process, and the batching bounds.
const (
	EnvTarget   = "DO_AUDITLOG_FORWARD_TARGET"
	EnvAPIKey   = "DO_AUDITLOG_FORWARD_API_KEY"
	EnvSource   = "DO_AUDITLOG_FORWARD_SOURCE"
	EnvBatchMax = "DO_AUDITLOG_FORWARD_BATCH_MAX"
	EnvFlushMs  = "DO_AUDITLOG_FORWARD_FLUSH_MS"
)

// Batching bounds: the flusher wakes on the earlier of the ticker or a full
// batch; the channel cap bounds memory under burst load (drop-oldest).
const (
	defaultFlushInterval  = 250 * time.Millisecond
	defaultMaxBatchEvents = 200
	channelCap            = 4096

	// dialTimeout bounds socket connection establishment (probe + dial).
	dialTimeout = 150 * time.Millisecond

	// requestTimeout bounds one POST (the collector answers fast; a stuck
	// socket must not stall the flusher for long).
	requestTimeout = 5 * time.Second

	// ingestPath is the collector's ingest route (TCP and socket-local).
	ingestPath = "/events"

	// unixHost routes socket URLs through the custom dialer.
	unixHost = "papdashboard.internal"

	// reprobeInterval is how often an idle armed target re-probes the
	// conventional socket. A target with events pending re-probes at flush
	// cadence instead, so activation after a PapDashboard boot lands within
	// one flush tick.
	reprobeInterval = 30 * time.Second

	// logEveryFails throttles repeat failure logs while a target stays down.
	logEveryFails = 100
)

// envelope is the PapDashboard ingest contract (kind "di"). Events ride
// auditlog.Event's own snake_case JSON tags verbatim — the collector's
// RawEvent is a structural superset of them.
type envelope struct {
	Kind     string           `json:"kind"`
	SourceID string           `json:"sourceId"`
	RunID    string           `json:"runId"`
	Events   []auditlog.Event `json:"events"`
	Complete bool             `json:"complete"`
}

// targetKind distinguishes how a target becomes deliverable.
type targetKind uint8

const (
	// targetUnix is an explicit unix socket, active from construction.
	targetUnix targetKind = iota
	// targetHTTP is a remote http(s) collector, active from construction.
	targetHTTP
	// targetAuto is the conventional socket, armed now and activated by
	// probing (flush-cadence while events pend, reprobeInterval when idle).
	targetAuto
)

// target is one ingest destination. All fields are owned by the flusher
// goroutine once constructed; resolveTargets fills the immutable identity.
type target struct {
	kind       targetKind
	url        string // ingest URL ("" until an auto target activates)
	socketPath string
	client     *http.Client

	active    bool
	lastProbe time.Time
	fails     int64 // consecutive failed POSTs
	failing   bool  // last POST failed (drives log-on-transition)
}

// Forwarder batches auditlog events per run and forwards them to one or
// more PapDashboard collectors (fan-out). OnEvent never blocks: events
// land in a bounded channel; overflow drops the OLDEST pending event (a
// live view values freshness over completeness — the local live dashboard
// keeps everything).
type Forwarder struct {
	targets   []target
	apiKey    string
	sourceID  string
	batchMax  int
	flushInterval time.Duration

	events   chan auditlog.Event
	complete chan string
	lastRun  atomic.Value // string: most recent RunID seen
	dropped  atomic.Int64
	failed   atomic.Int64
	closed   atomic.Bool

	closeOnce sync.Once
	stop      chan struct{} // closed by Shutdown to signal the flusher
	stopped   chan struct{} // closed by the flusher after its final flush
}

// New resolves the target list from the environment and returns a
// Forwarder. sourceID identifies this process in the collector's UI (""
// falls back to DO_AUDITLOG_FORWARD_SOURCE, then the executable name). The
// returned Forwarder is always non-nil; when every target is off it is
// disabled and every method is a safe no-op. An unset target ARMS the
// conventional socket: Enabled() is true and delivery starts whenever the
// socket answers.
func New(sourceID string) *Forwarder {
	return NewWithTarget(os.Getenv(EnvTarget), sourceID)
}

// NewWithTarget builds a Forwarder for an explicit target spec (same
// syntax as DO_AUDITLOG_FORWARD_TARGET, comma-separated for fan-out; ""
// arms the default socket).
func NewWithTarget(spec, sourceID string) *Forwarder {
	targets := resolveTargets(spec)
	if len(targets) == 0 {
		return &Forwarder{}
	}

	if sourceID == "" {
		sourceID = os.Getenv(EnvSource)
	}

	if sourceID == "" {
		if executable, err := os.Executable(); err == nil {
			sourceID = filepath.Base(executable)
		} else {
			sourceID = "unknown"
		}
	}

	f := &Forwarder{
		targets:   targets,
		apiKey:    os.Getenv(EnvAPIKey),
		sourceID:  sourceID,
		batchMax:     envInt(EnvBatchMax, defaultMaxBatchEvents, 1, 8192),
		flushInterval: time.Duration(envInt(EnvFlushMs, int(defaultFlushInterval.Milliseconds()), 16, 60_000)) * time.Millisecond,
		events:    make(chan auditlog.Event, channelCap),
		complete:  make(chan string, 16),
		stop:      make(chan struct{}),
		stopped:   make(chan struct{}),
	}

	go f.loop()

	return f
}

// resolveTargets maps a comma-separated target spec onto delivery targets.
// "off"/"disabled" entries drop out; an entirely empty spec arms exactly
// one auto target. Unknown junk falls through to the auto target (same
// tolerance as a bare spec).
func resolveTargets(spec string) []target {
	if strings.TrimSpace(spec) == "" {
		return []target{autoTarget()}
	}

	var targets []target

	for _, raw := range strings.Split(spec, ",") {
		one := strings.TrimSpace(raw)
		if one == "" {
			continue
		}

		if t, ok := resolveTarget(one); ok {
			targets = append(targets, t)
		}
	}

	return targets
}

// resolveTarget maps one target entry onto a target, or (!ok) for "off".
func resolveTarget(spec string) (target, bool) {
	switch {
	case strings.EqualFold(spec, "off"), strings.EqualFold(spec, "disabled"):
		return target{}, false

	case strings.HasPrefix(spec, "unix://"):
		path := strings.TrimPrefix(spec, "unix://")
		return unixTarget(path), true

	case strings.HasPrefix(spec, "http://"), strings.HasPrefix(spec, "https://"):
		return target{
			kind:   targetHTTP,
			url:    strings.TrimRight(spec, "/") + ingestPath,
			client: &http.Client{Timeout: requestTimeout},
			active: true,
		}, true

	case strings.HasPrefix(spec, "/"):
		return unixTarget(spec), true

	default:
		return autoTarget(), true
	}
}

func autoTarget() target {
	return target{kind: targetAuto, socketPath: DefaultSocketPath()}
}

func unixTarget(path string) target {
	return target{
		kind:       targetUnix,
		url:        "http://" + unixHost + ingestPath,
		socketPath: path,
		client:     unixClient(path),
		active:     true,
	}
}

// DefaultSocketPath is the conventional PapDashboard audit-run socket,
// mirroring the server's default (runtime dir first, /tmp fallback).
func DefaultSocketPath() string {
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, "papdashboard", "audit-runs.sock")
	}

	return filepath.Join(os.TempDir(), "papdashboard", "audit-runs.sock")
}

// unixClient builds an HTTP client whose dialer ignores the synthetic host
// and connects to the given socket path.
func unixClient(socketPath string) *http.Client {
	return &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialer := &net.Dialer{Timeout: dialTimeout}

				return dialer.DialContext(ctx, "unix", socketPath)
			},
			MaxIdleConnsPerHost: 4,
			IdleConnTimeout:     30 * time.Second,
		},
	}
}

// probeSocket reports whether a unix socket answers a dial.
func probeSocket(path string) bool {
	conn, err := net.DialTimeout("unix", path, dialTimeout)
	if err != nil {
		return false
	}
	_ = conn.Close()

	return true
}

// envInt reads a bounded integer knob; missing or invalid values fall back
// to the default, out-of-range values clamp to the bounds.
func envInt(name string, fallback, minValue, maxValue int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return min(max(value, minValue), maxValue)
}

// Enabled reports whether any target is armed (an armed auto target counts
// as enabled — delivery starts when its socket answers).
func (f *Forwarder) Enabled() bool { return len(f.targets) > 0 }

// Dropped returns how many events were dropped from a full buffer.
func (f *Forwarder) Dropped() int64 { return f.dropped.Load() }

// Failed returns how many batch POSTs failed across all targets.
func (f *Forwarder) Failed() int64 { return f.failed.Load() }

// OnEvent enqueues one event (auditlog.Config.OnEvent-compatible). It never
// blocks: a full buffer drops the oldest pending event.
func (f *Forwarder) OnEvent(evt auditlog.Event) {
	if !f.Enabled() || f.closed.Load() {
		return
	}

	if runID := string(evt.RunID); runID != "" {
		f.lastRun.Store(runID)
	}

	select {
	case f.events <- evt:
		return
	default:
	}

	// Buffer full: drop the oldest, then retry once.
	select {
	case <-f.events:
		f.dropped.Add(1)
	default:
	}

	select {
	case f.events <- evt:
	default:
		f.dropped.Add(1)
	}
}

// Complete marks the most recently seen run as finished (explicit terminal
// marker). The collector also derives completion from root-scope shutdown
// events, so this is belt-and-braces for explicit lifecycle ends.
func (f *Forwarder) Complete() {
	if !f.Enabled() || f.closed.Load() {
		return
	}

	runID, _ := f.lastRun.Load().(string)
	if runID == "" {
		return
	}

	select {
	case f.complete <- runID:
	default:
	}
}

// Shutdown flushes everything still buffered and stops the flusher. Safe to
// call multiple times; blocks until the final flush finished or ctx ends.
func (f *Forwarder) Shutdown(ctx context.Context) error {
	if !f.Enabled() {
		return nil
	}

	f.closeOnce.Do(func() {
		f.closed.Store(true)
		close(f.stop)
	})

	select {
	case <-f.stopped:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("forwarder shutdown: %w", ctx.Err())
	}
}

// loop is the flusher: ticker-batched sends plus a final shutdown flush.
func (f *Forwarder) loop() {
	defer close(f.stopped)

	pending := make([]auditlog.Event, 0, f.batchMax)
	completed := make([]string, 0, 8)

	ticker := time.NewTicker(f.flushInterval)
	defer ticker.Stop()

	drain := func() {
		for {
			select {
			case evt := <-f.events:
				pending = append(pending, evt)
			case runID := <-f.complete:
				completed = append(completed, runID)
			default:
				return
			}
		}
	}

	for {
		select {
		case <-f.stop:
			drain()
			f.flush(pending, completed)

			return
		case <-ticker.C:
			drain()
			f.flush(pending, completed)
			pending = pending[:0]
			completed = completed[:0]
		}
	}
}

// flush activates due targets, groups the buffer by run id, and POSTs one
// envelope per (run, batch-chunk) to every active target, then any
// explicit completion markers.
func (f *Forwarder) flush(buffer []auditlog.Event, completed []string) {
	f.activateDue(len(buffer) > 0)

	if len(buffer) == 0 && len(completed) == 0 {
		return
	}

	byRun := map[string][]auditlog.Event{}
	order := []string{}

	for _, evt := range buffer {
		runID := string(evt.RunID)
		if _, seen := byRun[runID]; !seen {
			order = append(order, runID)
		}

		byRun[runID] = append(byRun[runID], evt)
	}

	sentComplete := map[string]bool{}

	for _, runID := range order {
		events := byRun[runID]
		sentComplete[runID] = isRootShutdown(events[len(events)-1])

		for start := 0; start < len(events); start += f.batchMax {
			end := min(start+f.batchMax, len(events))
			chunk := events[start:end]

			f.postAll(envelope{
				Kind: "di", SourceID: f.sourceID, RunID: runID, Events: chunk,
				Complete: sentComplete[runID] && end == len(events),
			})
		}
	}

	for _, runID := range completed {
		if sentComplete[runID] {
			continue // completion already rode the event batch
		}

		f.postAll(envelope{Kind: "di", SourceID: f.sourceID, RunID: runID, Complete: true})
	}
}

// postAll sends one envelope to every active target (fan-out).
func (f *Forwarder) postAll(payload envelope) {
	for i := range f.targets {
		if !f.targets[i].active {
			continue
		}

		f.post(&f.targets[i], payload)
	}
}

// activateDue probes armed auto targets. A target with events pending
// re-probes at flush cadence so a PapDashboard boot lands within one tick;
// an idle target throttles to reprobeInterval.
func (f *Forwarder) activateDue(pendingEvents bool) {
	now := time.Now()

	for i := range f.targets {
		t := &f.targets[i]
		if t.active || t.kind != targetAuto {
			continue
		}

		wait := reprobeInterval
		if pendingEvents {
			wait = f.flushInterval
		}

		if now.Sub(t.lastProbe) < wait {
			continue
		}

		t.lastProbe = now
		if !probeSocket(t.socketPath) {
			continue
		}

		t.url = "http://" + unixHost + ingestPath
		t.client = unixClient(t.socketPath)
		t.active = true

		slog.Info("auditlog forward: target activated", "socket", t.socketPath)
	}
}

// post sends one batch to one target; failures increment the counters and
// log on state change (never per failure — a dead collector would spam).
func (f *Forwarder) post(t *target, payload envelope) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(body))
	if err != nil {
		return
	}

	request.Header.Set("Content-Type", "application/json")

	if f.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+f.apiKey)
	}

	response, err := t.client.Do(request)
	if err != nil {
		f.deliveryFailed(t, err)
		return
	}

	defer func() { _, _ = io.Copy(io.Discard, response.Body) }() //nolint:errcheck // drain for keep-alive

	if response.StatusCode >= http.StatusBadRequest {
		f.deliveryFailed(t, fmt.Errorf("collector answered %s", response.Status))
		return
	}

	f.deliveryRecovered(t)
}

// deliveryFailed records one failed POST: bump counters, log on transition
// into failure and then only every logEveryFails-th consecutive failure.
func (f *Forwarder) deliveryFailed(t *target, err error) {
	t.fails++
	f.failed.Add(1)

	switch {
	case !t.failing:
		t.failing = true
		slog.Warn("auditlog forward: delivery failing", "target", targetLabel(t), "err", err)
	case t.fails%logEveryFails == 0:
		slog.Warn("auditlog forward: delivery still failing", "target", targetLabel(t), "consecutive", t.fails)
	}
}

// deliveryRecovered clears the failure state and logs the recovery once.
func (f *Forwarder) deliveryRecovered(t *target) {
	if t.failing {
		slog.Info("auditlog forward: delivery recovered", "target", targetLabel(t))
	}

	t.failing = false
	t.fails = 0
}

// targetLabel names a target in log lines (URL for http, path for unix).
func targetLabel(t *target) string {
	if t.kind == targetHTTP {
		return t.url
	}

	return t.socketPath
}

// isRootShutdown mirrors the collector's completion derivation: the root
// scope's after-shutdown marks the container run complete.
func isRootShutdown(evt auditlog.Event) bool {
	return evt.EventType == auditlog.EventTypeShutdown &&
		evt.Phase == auditlog.PhaseAfter &&
		(evt.ScopeName == "" || evt.ScopeName == auditlog.RootScopeName)
}

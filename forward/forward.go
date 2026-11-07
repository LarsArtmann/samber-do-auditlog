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
	"errors"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// Env knobs: target selection, HTTP bearer key, the source label the
// collector's UI shows for this process, and the batching bounds.
const (
	EnvTarget   = "DO_AUDITLOG_FORWARD_TARGET"
	EnvAPIKey   = "DO_AUDITLOG_FORWARD_API_KEY" //nolint:gosec // env-var NAME, not a credential
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

	// envInt bounds for the batching knobs (ceiling keeps one burst bounded
	// in memory; the flush floor keeps the ticker from spinning hot).
	envBatchMaxCeiling = 8192
	envFlushMinMs      = 16
	envFlushMaxMs      = 60_000

	// Completion queue caps: runs are few (one per root scope), so these are
	// generosity, not load-bearing.
	completionChanCap = 16
	completedCap      = 8

	// HTTP client tuning for small, fast POSTs to a local collector.
	httpMaxIdleConnsPerHost = 4
	httpIdleConnTimeout     = 30 * time.Second
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

// errCollectorStatus marks non-2xx collector responses; the concrete status
// text rides along via %w wrapping at the delivery site.
var errCollectorStatus = errors.New("collector answered non-2xx")

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
	targets       []target
	apiKey        string
	sourceID      string
	batchMax      int
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

func (f *Forwarder) Enabled() bool { return len(f.targets) > 0 }

// Dropped returns how many events were dropped from a full buffer.
func (f *Forwarder) Dropped() int64 { return f.dropped.Load() }

// Failed returns how many batch POSTs failed across all targets.
func (f *Forwarder) Failed() int64 { return f.failed.Load() }

func (f *Forwarder) loop() {
	defer close(f.stopped)

	pending := make([]auditlog.Event, 0, f.batchMax)
	completed := make([]string, 0, completedCap)

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

func isRootShutdown(evt auditlog.Event) bool {
	return evt.EventType == auditlog.EventTypeShutdown &&
		evt.Phase == auditlog.PhaseAfter &&
		(evt.ScopeName == "" || evt.ScopeName == auditlog.RootScopeName)
}

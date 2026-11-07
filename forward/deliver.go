package forward

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

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

	defer func() { _, _ = io.Copy(io.Discard, response.Body) }()

	if response.StatusCode >= http.StatusBadRequest {
		f.deliveryFailed(t, fmt.Errorf("%w: %s", errCollectorStatus, response.Status))

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

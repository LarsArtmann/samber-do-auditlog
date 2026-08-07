package auditlog

import "sync"

// MultiWriterCallback is the per-event callback signature accepted by
// [MultiWriter]. It matches the [Config.OnEvent] / [NDJSONStreamer.OnEvent]
// signature exactly so sinks can be composed without adapter lambdas.
//
// Callbacks are invoked synchronously on the goroutine that calls OnEvent and
// must not panic: a panicking callback propagates the panic to the caller
// (the same contract [Recorder] places on its own OnEvent). Callbacks that
// perform slow work should hand the event to their own goroutine.
type MultiWriterCallback func(Event)

// MultiWriter fans each event out to multiple OnEvent-style callbacks.
//
// Use it when a single container lifecycle needs to drive multiple consumers
// simultaneously — for example, streaming to an NDJSON file AND a live SSE
// hub AND an OpenTelemetry bridge — without composing them manually at the
// call site:
//
//	mw := auditlog.NewMultiWriter(streamer.OnEvent, hub.OnEvent)
//	plugin, _ := auditlog.New(auditlog.Config{OnEvent: mw.OnEvent})
//
// MultiWriter is safe for concurrent use by multiple goroutines: the internal
// mutex serializes the fan-out so callbacks never see a half-written event
// interleaving from another goroutine. This matches the concurrency guarantee
// of [NDJSONStreamer.OnEvent].
//
// Callbacks are fixed at construction time. There is no Add/Remove API —
// callers who need dynamic membership should swap the MultiWriter via a
// single atomic pointer (not provided here; out of scope for the common case
// of static composition).
type MultiWriter struct {
	mu        sync.Mutex
	callbacks []MultiWriterCallback
}

// NewMultiWriter returns a MultiWriter that invokes every fn for each event.
// At least one callback must be supplied; an empty list returns nil and the
// caller should treat that as a no-op sink (nil MultiWriter methods are safe).
func NewMultiWriter(fn ...MultiWriterCallback) *MultiWriter {
	if len(fn) == 0 {
		return nil
	}

	return &MultiWriter{callbacks: fn}
}

// OnEvent invokes every registered callback with evt, in registration order.
// It matches the [Config.OnEvent] signature so a MultiWriter can be wired
// directly as an auditlog OnEvent callback.
//
// Holds an internal mutex during fan-out so callbacks see no concurrent
// interleaving from other goroutines calling OnEvent. Nil callbacks are
// skipped.
func (m *MultiWriter) OnEvent(evt Event) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, fn := range m.callbacks {
		if fn == nil {
			continue
		}

		fn(evt)
	}
}

// CallbackCount returns the number of registered callbacks. Useful for
// debugging and tests.
func (m *MultiWriter) CallbackCount() int {
	if m == nil {
		return 0
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.callbacks)
}

package auditlog

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/samber/do/v2"
)

const (
	// microsPerMs converts microseconds to milliseconds.
	microsPerMs = 1000.0
	// initialEventCapacity is the starting capacity for the events slice.
	initialEventCapacity = 1024
	// initialDepsCapacity is the initial capacity for a service's dependency map.
	initialDepsCapacity = 2
)

type stackEntry struct {
	scopeID     ScopeID
	scopeName   string
	serviceName ServiceName
	start       time.Time
}

// svcKey is a zero-allocation map key for looking up services by scope + name.
// Unlike a concatenated string key, it requires no heap allocation.
type svcKey struct {
	scopeID ScopeID
	name    ServiceName
}

// serviceKey produces the canonical string key for a service within a scope.
// Used only by public API surfaces (ReportIndex) that need string keys.
func serviceKey(scopeID ScopeID, serviceName ServiceName) string {
	return string(scopeID) + "/" + string(serviceName)
}

type serviceRecord struct {
	scopeID              ScopeID
	scopeName            string
	serviceName          ServiceName
	serviceType          ProviderType
	registeredAt         time.Time
	firstInvokedAt       *time.Time
	invocationCount      int
	invocationOrder      int
	firstBuildDurationMs *float64
	dependencies         map[svcKey]struct{}
	shutdownAt           *time.Time
	shutdownDurationMs   *float64
	invocationError      *string
	shutdownError        *string
	lastHealthCheckAt    *time.Time
	healthCheckError     *string
	healthCheckCount     int
}

type scopeMeta struct {
	id       ScopeID
	name     string
	parentID ScopeID
	ref      *do.Scope
}

// newSequenceCounter returns a fresh counter slot for sequence generation.
// Using a per-recorder counter keeps the package free of global state and
// avoids cross-test interference.
func newSequenceCounter() *int64 {
	counter := int64(0)

	return &counter
}

// Recorder captures DI lifecycle events in-memory with minimal overhead.
//
// # Locking Protocol
//
// All mutable state is protected by a single sync.RWMutex (mu), except the
// onEvent callback which has its own onEventMu (see field comment):
//
//	Write path:  mu.Lock()   : all hook methods (OnBefore*, OnAfter*, RecordHealthCheck)
//	Read path:   mu.RLock()  : BuildReport, Events, EventsCount, ResolveServiceScope
//
// The invocation counter (invocationSeq) is a plain int64 guarded by mu
// (all mutations happen while mu is held). Sequence numbers use a separate
// per-recorder counter with the same discipline.
//
// The onEvent callback is always called outside the lock to prevent user code from
// blocking or deadlocking the recorder.
//
// # Critical: enrichCapabilities and do.ExplainInjector
//
// BuildReport copies the scopes map under mu.RLock, then releases the lock BEFORE calling
// enrichCapabilities. This is mandatory because do.ExplainInjector acquires internal
// samber/do locks that would deadlock if called from inside any hook (which holds mu).
type Recorder struct {
	mu       sync.RWMutex
	events   []Event
	services map[svcKey]*serviceRecord
	scopes   map[ScopeID]scopeMeta
	stack    []stackEntry

	// shutdownStart stores per-service shutdown start times for duration calc.
	shutdownStart map[svcKey]time.Time

	sequence      *int64
	invocationSeq int64
	containerID   ContainerID
	runID         RunID

	// onEventMu guards onEvent so the callback can be replaced after
	// creation via setOnEvent (Plugin.SetOnEvent). It is deliberately
	// separate from mu: the callback is invoked outside mu (see the
	// locking protocol above), so it needs its own lock. fireEvent takes
	// RLock per event; setOnEvent takes Lock only when a caller swaps
	// the callback.
	onEventMu sync.RWMutex
	onEvent   func(Event)

	// maxEvents caps the events slice. When > 0, new events are dropped
	// (counter incremented) after this many events are stored.
	maxEvents     int
	droppedEvents int64
}

// NewRecorder creates a new event recorder.
func NewRecorder(containerID ContainerID, runID RunID, onEvent func(Event)) *Recorder {
	return &Recorder{ //nolint:exhaustruct
		mu:            sync.RWMutex{},
		events:        make([]Event, 0, initialEventCapacity),
		services:      make(map[svcKey]*serviceRecord),
		scopes:        make(map[ScopeID]scopeMeta),
		shutdownStart: make(map[svcKey]time.Time),
		sequence:      newSequenceCounter(),
		containerID:   containerID,
		runID:         runID,
		onEvent:       onEvent,
	}
}

func (r *Recorder) nextSequence() int {
	*r.sequence++

	return int(*r.sequence)
}

// recordScopeLocked records scope metadata. Caller must hold r.mu.
func (r *Recorder) recordScopeLocked(scopeID ScopeID, scopeName string, scope *do.Scope) {
	if _, ok := r.scopes[scopeID]; ok {
		return
	}

	meta := scopeMeta{id: scopeID, name: scopeName, parentID: "", ref: scope}
	if ancestors := scope.Ancestors(); len(ancestors) > 0 {
		meta.parentID = ScopeID(ancestors[0].ID())
	}

	r.scopes[scopeID] = meta
}

// serviceTypeForLocked returns the recorded provider type for a service, or empty if
// the service has not been recorded yet. Caller must hold r.mu.
func (r *Recorder) serviceTypeForLocked(key svcKey) ProviderType {
	if rec, ok := r.services[key]; ok {
		return rec.serviceType
	}

	return ""
}

// appendEventLocked appends an event to the events slice, respecting the MaxEvents cap.
// When the cap is reached, the event is dropped and the dropped counter is incremented.
// Caller must hold r.mu.
func (r *Recorder) appendEventLocked(evt Event) {
	if r.maxEvents > 0 && len(r.events) >= r.maxEvents {
		r.droppedEvents++

		return
	}

	r.events = append(r.events, evt)
}

// DroppedEventCount returns the number of events dropped due to MaxEvents cap.
func (r *Recorder) DroppedEventCount() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.droppedEvents
}

// Events returns a defensive copy of all captured events.
func (r *Recorder) Events() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return append([]Event(nil), r.events...)
}

// EventsCount returns the number of captured events without copying the slice.
func (r *Recorder) EventsCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.events)
}

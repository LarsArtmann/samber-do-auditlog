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
	scopeID     string
	scopeName   string
	serviceName string
	start       time.Time
}

// serviceKey produces the canonical map key for a service within a scope.
func serviceKey(scopeID, serviceName string) string {
	return scopeID + "/" + serviceName
}

type serviceRecord struct {
	scopeID              string
	scopeName            string
	serviceName          string
	serviceType          ProviderType
	registeredAt         time.Time
	firstInvokedAt       *time.Time
	invocationCount      int
	invocationOrder      int
	firstBuildDurationMs *float64
	dependencies         map[string]struct{}
	shutdownAt           *time.Time
	shutdownDurationMs   *float64
	invocationError      *string
	shutdownError        *string
	lastHealthCheckAt    *time.Time
	healthCheckError     *string
	healthCheckCount     int
}

type scopeMeta struct {
	id       string
	name     string
	parentID string
	ref      *do.Scope
}

// newSequenceCounter returns a fresh atomic counter for sequence generation.
// Using a per-recorder counter keeps the package free of global state and
// avoids cross-test interference.
func newSequenceCounter() *atomic.Int64 {
	var counter atomic.Int64

	return &counter
}

// Recorder captures DI lifecycle events in-memory with minimal overhead.
//
// # Locking Protocol
//
// All mutable state is protected by a single sync.RWMutex (mu):
//
//	Write path:  mu.Lock()   — all hook methods (OnBefore*, OnAfter*, RecordHealthCheck)
//	Read path:   mu.RLock()  — BuildReport, Events, EventsCount, ResolveServiceScope
//
// The invocation counter (invocationSeq) uses atomic.Int64, eliminating a separate mutex.
// Sequence numbers use a separate per-recorder atomic.Int64, also lock-free.
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
	services map[string]*serviceRecord
	scopes   map[string]scopeMeta
	stack    []stackEntry

	// shutdownStart stores per-service shutdown start times for duration calc.
	shutdownStart map[string]time.Time

	sequence      *atomic.Int64
	invocationSeq atomic.Int64
	containerID   string
	onEvent       func(Event)
}

// NewRecorder creates a new event recorder.
func NewRecorder(containerID string, onEvent func(Event)) *Recorder {
	return &Recorder{ //nolint:exhaustruct
		mu:            sync.RWMutex{},
		events:        make([]Event, 0, initialEventCapacity),
		services:      make(map[string]*serviceRecord),
		scopes:        make(map[string]scopeMeta),
		shutdownStart: make(map[string]time.Time),
		sequence:      newSequenceCounter(),
		containerID:   containerID,
		onEvent:       onEvent,
	}
}

func (r *Recorder) nextSequence() int {
	return int(r.sequence.Add(1))
}

// recordScopeLocked records scope metadata. Caller must hold r.mu.
func (r *Recorder) recordScopeLocked(scopeID, scopeName string, scope *do.Scope) {
	if _, ok := r.scopes[scopeID]; ok {
		return
	}

	meta := scopeMeta{id: scopeID, name: scopeName, parentID: "", ref: scope}
	if ancestors := scope.Ancestors(); len(ancestors) > 0 {
		meta.parentID = ancestors[0].ID()
	}

	r.scopes[scopeID] = meta
}

// serviceTypeForLocked returns the recorded provider type for a service, or empty if
// the service has not been recorded yet. Caller must hold r.mu.
func (r *Recorder) serviceTypeForLocked(key string) ProviderType {
	if rec, ok := r.services[key]; ok {
		return rec.serviceType
	}

	return ""
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

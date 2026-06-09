package auditlog

import (
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/samber/do/v2"
)

const (
	// microsecondsPerMillisecond converts microseconds to milliseconds.
	microsecondsPerMillisecond = 1000.0
	// initialEventCapacity is the starting capacity for the events slice.
	initialEventCapacity = 1024
)

type stackEntry struct {
	scopeID     string
	scopeName   string
	serviceName string
	start       time.Time
}

type serviceRecord struct {
	scopeID         string
	scopeName       string
	serviceName     string
	registeredAt    time.Time
	firstInvokedAt  *time.Time
	invocationCount int
	invocationOrder int
	firstBuildDurationMs *float64
	dependencies    map[string]struct{}
	shutdownAt      *time.Time
	invocationError *string
	shutdownError   *string
}

type scopeMeta struct {
	id       string
	name     string
	parentID string
}

// newSequenceCounter returns a fresh atomic counter for sequence generation.
// Using a per-recorder counter keeps the package free of global state and
// avoids cross-test interference.
func newSequenceCounter() *atomic.Int64 {
	var counter atomic.Int64

	return &counter
}

// Recorder captures DI lifecycle events in-memory with minimal overhead.
type Recorder struct {
	mu       sync.RWMutex
	events   []Event
	services map[string]*serviceRecord
	scopes   map[string]scopeMeta

	stackMu sync.Mutex
	stack   []stackEntry

	invocationMu    sync.Mutex
	invocationIndex int

	sequence *atomic.Int64
}

// NewRecorder creates a new event recorder.
func NewRecorder() *Recorder {
	return &Recorder{
		mu:              sync.RWMutex{},
		events:          make([]Event, 0, initialEventCapacity),
		services:        make(map[string]*serviceRecord),
		scopes:          make(map[string]scopeMeta),
		stackMu:         sync.Mutex{},
		stack:           nil,
		invocationMu:    sync.Mutex{},
		invocationIndex: 0,
		sequence:        newSequenceCounter(),
	}
}

func (r *Recorder) nextSequence() int {
	return int(r.sequence.Add(1))
}

func (r *Recorder) recordScope(scope *do.Scope) {
	scopeID := scope.ID()

	r.mu.Lock()
	if _, ok := r.scopes[scopeID]; ok {
		r.mu.Unlock()

		return
	}

	meta := scopeMeta{id: scopeID, name: scope.Name(), parentID: ""}
	if ancestors := scope.Ancestors(); len(ancestors) > 0 {
		meta.parentID = ancestors[0].ID()
	}

	r.scopes[scopeID] = meta
	r.mu.Unlock()
}

// newRegistrationEvent builds an Event struct with all fields initialized.
// Centralizing the construction ensures exhaustruct sees every field.
func newRegistrationEvent(seq int, now time.Time, phase Phase, scope *do.Scope, serviceName string) Event {
	return Event{
		Sequence:    seq,
		Timestamp:   now,
		EventType:   EventTypeRegistration,
		Phase:       phase,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
		DurationMs:  nil,
		Error:       nil,
	}
}

// newInvocationEvent builds an Event struct for an invocation phase.
func newInvocationEvent(
	seq int,
	now time.Time,
	phase Phase,
	scope *do.Scope,
	serviceName string,
	dur *float64,
	errStr *string,
) Event {
	return Event{
		Sequence:    seq,
		Timestamp:   now,
		EventType:   EventTypeInvocation,
		Phase:       phase,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
		DurationMs:  dur,
		Error:       errStr,
	}
}

// newShutdownEvent builds an Event struct for a shutdown phase.
func newShutdownEvent(seq int, now time.Time, phase Phase, scope *do.Scope, serviceName string, errStr *string) Event {
	return Event{
		Sequence:    seq,
		Timestamp:   now,
		EventType:   EventTypeShutdown,
		Phase:       phase,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
		DurationMs:  nil,
		Error:       errStr,
	}
}

// newServiceRecord constructs a serviceRecord with all fields set.
func newServiceRecord(scope *do.Scope, serviceName string, now time.Time) *serviceRecord {
	return &serviceRecord{
		scopeID:         scope.ID(),
		scopeName:       scope.Name(),
		serviceName:     serviceName,
		registeredAt:    now,
		firstInvokedAt:  nil,
		invocationCount: 0,
		invocationOrder: 0,
		firstBuildDurationMs: nil,
		dependencies:    make(map[string]struct{}),
		shutdownAt:      nil,
		invocationError: nil,
		shutdownError:   nil,
	}
}

func (r *Recorder) OnBeforeRegistration(scope *do.Scope, serviceName string) {
	r.recordScope(scope)
	r.addEvent(newRegistrationEvent(r.nextSequence(), time.Now(), PhaseBefore, scope, serviceName))
}

func (r *Recorder) OnAfterRegistration(scope *do.Scope, serviceName string) {
	now := time.Now()
	key := scope.ID() + "/" + serviceName

	r.mu.Lock()
	if _, ok := r.services[key]; !ok {
		r.services[key] = newServiceRecord(scope, serviceName, now)
	}
	r.mu.Unlock()
	r.addEvent(newRegistrationEvent(r.nextSequence(), now, PhaseAfter, scope, serviceName))
}

func (r *Recorder) OnBeforeInvocation(scope *do.Scope, serviceName string) {
	r.recordScope(scope)

	now := time.Now()

	r.stackMu.Lock()
	if len(r.stack) > 0 {
		parent := r.stack[len(r.stack)-1]
		parentKey := parent.scopeID + "/" + parent.serviceName
		depKey := scope.ID() + "/" + serviceName

		r.mu.Lock()
		if rec, ok := r.services[parentKey]; ok {
			rec.dependencies[depKey] = struct{}{}
		}
		r.mu.Unlock()
	}

	r.stack = append(r.stack, stackEntry{
		scopeID:     scope.ID(),
		scopeName:   scope.Name(),
		serviceName: serviceName,
		start:       now,
	})
	r.stackMu.Unlock()

	r.addEvent(newInvocationEvent(r.nextSequence(), now, PhaseBefore, scope, serviceName, nil, nil))
}

func (r *Recorder) OnAfterInvocation(scope *do.Scope, serviceName string, err error) {
	now := time.Now()
	durationMs := r.popInvocationDuration(scope, serviceName, now)
	errStr := errorToStringPtr(err)
	r.addEvent(newInvocationEvent(r.nextSequence(), now, PhaseAfter, scope, serviceName, durationMs, errStr))
	r.recordInvocationResult(scope, serviceName, now, durationMs, errStr)
}

// popInvocationDuration finds and pops the matching stack frame, returning the
// elapsed duration in milliseconds (nil if no frame matched).
func (r *Recorder) popInvocationDuration(scope *do.Scope, serviceName string, now time.Time) *float64 {
	var durationMs *float64

	r.stackMu.Lock()
	defer r.stackMu.Unlock()

	for i, frame := range slices.Backward(r.stack) {
		if frame.serviceName == serviceName && frame.scopeID == scope.ID() {
			d := float64(now.Sub(frame.start).Microseconds()) / microsecondsPerMillisecond
			durationMs = &d

			r.stack = append(r.stack[:i], r.stack[i+1:]...)

			break
		}
	}

	return durationMs
}

// recordInvocationResult updates the per-service aggregate after an invocation.
func (r *Recorder) recordInvocationResult(
	scope *do.Scope,
	serviceName string,
	now time.Time,
	durationMs *float64,
	errStr *string,
) {
	key := scope.ID() + "/" + serviceName

	r.mu.Lock()
	defer r.mu.Unlock()

	rec, ok := r.services[key]
	if !ok {
		rec = newServiceRecord(scope, serviceName, now)
		r.services[key] = rec
	}

	rec.invocationCount++

	if rec.firstInvokedAt == nil {
		rec.firstInvokedAt = &now

		r.invocationMu.Lock()
		rec.invocationOrder = r.invocationIndex
		r.invocationIndex++
		r.invocationMu.Unlock()
	}

	if durationMs != nil && rec.firstBuildDurationMs == nil {
		rec.firstBuildDurationMs = durationMs
	}

	if errStr != nil {
		rec.invocationError = errStr
	}
}

func (r *Recorder) OnBeforeShutdown(scope *do.Scope, serviceName string) {
	r.addEvent(newShutdownEvent(r.nextSequence(), time.Now(), PhaseBefore, scope, serviceName, nil))
}

func (r *Recorder) OnAfterShutdown(scope *do.Scope, serviceName string, err error) {
	now := time.Now()
	errStr := errorToStringPtr(err)

	r.addEvent(newShutdownEvent(r.nextSequence(), now, PhaseAfter, scope, serviceName, errStr))

	key := scope.ID() + "/" + serviceName

	r.mu.Lock()
	if rec, ok := r.services[key]; ok {
		rec.shutdownAt = &now
		rec.shutdownError = errStr
	}
	r.mu.Unlock()
}

func (r *Recorder) addEvent(e Event) {
	r.mu.Lock()
	r.events = append(r.events, e)
	r.mu.Unlock()
}

// errorToStringPtr converts an error to a heap-allocated string pointer.
// Returns nil when err is nil so we don't emit empty error fields in events.
func errorToStringPtr(err error) *string {
	if err == nil {
		return nil
	}

	msg := err.Error()

	return &msg
}

// BuildReport assembles a machine-readable Report from all captured events.
func (r *Recorder) BuildReport(containerID string) Report {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dependents := buildDependentsMapLocked(r.services)

	services := make([]ServiceInfo, 0, len(r.services))
	for _, rec := range r.services {
		deps := make([]DependencyRef, 0, len(rec.dependencies))
		for depKey := range rec.dependencies {
			if depRec, ok := r.services[depKey]; ok {
				deps = append(deps, DependencyRef{
					ScopeName:   depRec.scopeName,
					ServiceName: depRec.serviceName,
				})
			}
		}

		key := rec.scopeID + "/" + rec.serviceName
		svcDependents := dependents[key]

		services = append(services, ServiceInfo{
			ServiceName:     rec.serviceName,
			ScopeID:         rec.scopeID,
			ScopeName:       rec.scopeName,
			RegisteredAt:    rec.registeredAt,
			FirstInvokedAt:  rec.firstInvokedAt,
			InvocationCount: rec.invocationCount,
			InvocationOrder: rec.invocationOrder,
			FirstBuildDurationMs: rec.firstBuildDurationMs,
			Dependencies:    deps,
			Dependents:      svcDependents,
			ShutdownAt:      rec.shutdownAt,
			ShutdownError:   rec.shutdownError,
			InvocationError: rec.invocationError,
		})
	}

	scopeTree := r.buildScopeTreeLocked()

	return Report{
		ContainerID:  containerID,
		ExportedAt:   time.Now(),
		EventCount:   len(r.events),
		ServiceCount: len(services),
		Events:       append([]Event(nil), r.events...),
		Services:     services,
		ScopeTree:    scopeTree,
	}
}

func buildDependentsMapLocked(services map[string]*serviceRecord) map[string][]DependencyRef {
	dependents := make(map[string][]DependencyRef)

	for _, rec := range services {
		for depKey := range rec.dependencies {
			if _, ok := services[depKey]; ok {
				dependents[depKey] = append(dependents[depKey], DependencyRef{
					ScopeName:   rec.scopeName,
					ServiceName: rec.serviceName,
				})
			}
		}
	}

	return dependents
}

func (r *Recorder) buildScopeTreeLocked() ScopeNode {
	var root scopeMeta

	hasRoot := false

	for _, meta := range r.scopes {
		if meta.parentID == "" {
			root = meta
			hasRoot = true

			break
		}
	}

	if !hasRoot && len(r.scopes) > 0 {
		for _, meta := range r.scopes {
			root = meta

			break
		}
	}

	scopeServices := make(map[string][]string)
	for _, rec := range r.services {
		scopeServices[rec.scopeID] = append(scopeServices[rec.scopeID], rec.serviceName)
	}

	var build func(parentID string) []ScopeNode

	build = func(parentID string) []ScopeNode {
		var children []ScopeNode

		for _, meta := range r.scopes {
			if meta.parentID == parentID {
				children = append(children, ScopeNode{
					ID:       meta.id,
					Name:     meta.name,
					Services: scopeServices[meta.id],
					Children: build(meta.id),
				})
			}
		}

		return children
	}

	return ScopeNode{
		ID:       root.id,
		Name:     root.name,
		Services: scopeServices[root.id],
		Children: build(root.id),
	}
}

// Events returns a defensive copy of all captured events.
func (r *Recorder) Events() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return append([]Event(nil), r.events...)
}

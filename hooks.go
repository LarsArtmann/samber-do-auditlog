package auditlog

import (
	"slices"
	"time"

	"github.com/samber/do/v2"
)

// --- Event/Record constructors (used by hooks and health check) ---

// newEventFromRef builds an Event from a ServiceRef.
func newEventFromRef(
	seq int,
	now time.Time,
	eventType EventType,
	phase Phase,
	ref ServiceRef,
	containerID string,
	svcType ProviderType,
	dur *float64,
	errStr *string,
) Event {
	return Event{
		Sequence:    seq,
		Timestamp:   now,
		EventType:   eventType,
		Phase:       phase,
		ContainerID: containerID,
		ServiceRef:  ref,
		ServiceType: svcType,
		DurationMs:  dur,
		Error:       errStr,
	}
}

// getOrCreateServiceRecord returns the existing serviceRecord for evt, or
// creates a new one when absent using metadata from evt. Used by the replay
// path where an Event is already available.
func getOrCreateServiceRecord(
	services map[svcKey]*serviceRecord,
	evt Event,
) *serviceRecord {
	key := svcKey{scopeID: evt.ScopeID, name: evt.ServiceName}

	if rec, ok := services[key]; ok {
		return rec
	}

	rec := newServiceRecordCore(evt.ScopeID, evt.ScopeName, evt.ServiceName, evt.ServiceType, evt.Timestamp)
	services[key] = rec

	return rec
}

// popStackFrame removes the most recent matching stack frame via backward
// LIFO search. Returns the updated slice, the removed frame, and true if
// found. Used by both the live Recorder path (under r.mu) and the replay
// path (no lock).
func popStackFrame(stack []stackEntry, scopeID, serviceName string) ([]stackEntry, stackEntry, bool) {
	for i, frame := range slices.Backward(stack) {
		if frame.serviceName == serviceName && frame.scopeID == scopeID {
			if i == len(stack)-1 {
				stack = stack[:i]
			} else {
				stack = append(stack[:i], stack[i+1:]...)
			}

			return stack, frame, true
		}
	}

	return stack, stackEntry{}, false //nolint:exhaustruct // zero-value sentinel for not-found
}

// recordDependencyFromStack inspects the current invocation stack and, if
// non-empty, records depKey as a dependency of the top-of-stack service.
// Used by both the live Recorder path (under r.mu) and the replay path
// (no lock), so it takes the maps explicitly.
func recordDependencyFromStack(
	stack []stackEntry,
	services map[svcKey]*serviceRecord,
	depKey svcKey,
) {
	if len(stack) == 0 {
		return
	}

	parent := stack[len(stack)-1]
	parentKey := svcKey{scopeID: parent.scopeID, name: parent.serviceName}

	rec, ok := services[parentKey]
	if !ok {
		return
	}

	if rec.dependencies == nil {
		rec.dependencies = make(map[svcKey]struct{}, initialDepsCapacity)
	}

	rec.dependencies[depKey] = struct{}{}
}

// newServiceRecordCore constructs a serviceRecord with lazy deps map.
func newServiceRecordCore(scopeID, scopeName, serviceName string, svcType ProviderType, now time.Time) *serviceRecord {
	return &serviceRecord{
		scopeID:              scopeID,
		scopeName:            scopeName,
		serviceName:          serviceName,
		serviceType:          svcType,
		registeredAt:         now,
		firstInvokedAt:       nil,
		invocationCount:      0,
		invocationOrder:      0,
		firstBuildDurationMs: nil,
		dependencies:         nil,
		shutdownAt:           nil,
		shutdownDurationMs:   nil,
		invocationError:      nil,
		shutdownError:        nil,
		lastHealthCheckAt:    nil,
		healthCheckError:     nil,
		healthCheckCount:     0,
	}
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

// inferServiceType uses do.ExplainNamedService to determine the provider type
// (lazy, eager, transient, alias). Returns empty string if unknown.
func inferServiceType(scope *do.Scope, serviceName string) ProviderType {
	desc, ok := do.ExplainNamedService(scope, serviceName)
	if !ok {
		return ""
	}

	return ProviderType(desc.ServiceType)
}

// --- Hook methods (single-lock hot path) ---

func (r *Recorder) OnBeforeRegistration(scope *do.Scope, serviceName string) {
	scopeID := scope.ID()
	scopeName := scope.Name()
	now := time.Now()
	seq := r.nextSequence()

	ref := ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName}
	evt := newEventFromRef(seq, now, EventTypeRegistration, PhaseBefore, ref, r.containerID, "", nil, nil)

	r.mu.Lock()
	r.recordScopeLocked(scopeID, scopeName, scope)
	r.appendEventLocked(evt)
	r.mu.Unlock()

	if r.onEvent != nil {
		r.onEvent(evt)
	}
}

func (r *Recorder) OnAfterRegistration(scope *do.Scope, serviceName string) {
	scopeID := scope.ID()
	scopeName := scope.Name()
	now := time.Now()
	key := svcKey{scopeID: scopeID, name: serviceName}
	seq := r.nextSequence()

	r.mu.Lock()

	rec, ok := r.services[key]

	var svcType ProviderType

	if !ok {
		svcType = inferServiceType(scope, serviceName)
		rec = newServiceRecordCore(scopeID, scopeName, serviceName, svcType, now)
		r.services[key] = rec
	} else {
		svcType = rec.serviceType
	}

	ref := ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName}
	evt := newEventFromRef(seq, now, EventTypeRegistration, PhaseAfter, ref, r.containerID, svcType, nil, nil)
	r.appendEventLocked(evt)

	r.mu.Unlock()

	if r.onEvent != nil {
		r.onEvent(evt)
	}
}

func (r *Recorder) OnBeforeInvocation(scope *do.Scope, serviceName string) {
	scopeID := scope.ID()
	scopeName := scope.Name()
	now := time.Now()
	depKey := svcKey{scopeID: scopeID, name: serviceName}
	seq := r.nextSequence()

	r.mu.Lock()

	r.recordScopeLocked(scopeID, scopeName, scope)

	recordDependencyFromStack(r.stack, r.services, depKey)

	r.stack = append(r.stack, stackEntry{
		scopeID:     scopeID,
		scopeName:   scopeName,
		serviceName: serviceName,
		start:       now,
	})

	// Look up service type from existing record.
	svcType := r.serviceTypeForLocked(depKey)

	ref := ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName}
	evt := newEventFromRef(seq, now, EventTypeInvocation, PhaseBefore, ref, r.containerID, svcType, nil, nil)
	r.appendEventLocked(evt)

	r.mu.Unlock()

	if r.onEvent != nil {
		r.onEvent(evt)
	}
}

func (r *Recorder) OnAfterInvocation(scope *do.Scope, serviceName string, err error) {
	scopeID := scope.ID()
	scopeName := scope.Name()
	now := time.Now()
	errStr := errorToStringPtr(err)
	key := svcKey{scopeID: scopeID, name: serviceName}
	seq := r.nextSequence()

	r.mu.Lock()

	// Pop matching stack frame (LIFO fast path).
	var durationMs *float64

	newStack, frame, found := popStackFrame(r.stack, scopeID, serviceName)
	if found {
		r.stack = newStack
		d := float64(now.Sub(frame.start).Microseconds()) / microsPerMs
		durationMs = &d
	}

	// Look up service type.
	svcType := r.serviceTypeForLocked(key)

	ref := ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName}
	evt := newEventFromRef(seq, now, EventTypeInvocation, PhaseAfter, ref, r.containerID, svcType, durationMs, errStr)
	r.appendEventLocked(evt)

	r.updateInvocationAggregate(scopeID, scopeName, serviceName, now, svcType, durationMs, errStr)

	r.mu.Unlock()

	if r.onEvent != nil {
		r.onEvent(evt)
	}
}

// updateInvocationAggregate updates the per-service aggregate after an invocation.
// Caller must hold r.mu.
func (r *Recorder) updateInvocationAggregate(
	scopeID, scopeName, serviceName string,
	now time.Time,
	svcType ProviderType,
	durationMs *float64,
	errStr *string,
) {
	key := svcKey{scopeID: scopeID, name: serviceName}

	rec, ok := r.services[key]
	if !ok {
		rec = newServiceRecordCore(scopeID, scopeName, serviceName, svcType, now)
		r.services[key] = rec
	}

	rec.invocationCount++

	if rec.firstInvokedAt == nil {
		rec.firstInvokedAt = &now
		rec.invocationOrder = int(r.invocationSeq.Add(1)) - 1
	}

	if durationMs != nil && rec.firstBuildDurationMs == nil {
		rec.firstBuildDurationMs = durationMs
	}

	if errStr != nil {
		rec.invocationError = errStr
	}
}

func (r *Recorder) OnBeforeShutdown(scope *do.Scope, serviceName string) {
	scopeID := scope.ID()
	scopeName := scope.Name()
	now := time.Now()
	key := svcKey{scopeID: scopeID, name: serviceName}
	seq := r.nextSequence()

	r.mu.Lock()

	r.recordScopeLocked(scopeID, scopeName, scope)
	r.shutdownStart[key] = now

	svcType := r.serviceTypeForLocked(key)

	ref := ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName}
	evt := newEventFromRef(seq, now, EventTypeShutdown, PhaseBefore, ref, r.containerID, svcType, nil, nil)
	r.appendEventLocked(evt)

	r.mu.Unlock()

	if r.onEvent != nil {
		r.onEvent(evt)
	}
}

func (r *Recorder) OnAfterShutdown(scope *do.Scope, serviceName string, err error) {
	scopeID := scope.ID()
	scopeName := scope.Name()
	now := time.Now()
	errStr := errorToStringPtr(err)
	key := svcKey{scopeID: scopeID, name: serviceName}
	seq := r.nextSequence()

	r.mu.Lock()

	// Resolve shutdown duration.
	start, hasStart := r.shutdownStart[key]
	if hasStart {
		delete(r.shutdownStart, key)
	}

	var shutdownDur *float64

	if hasStart {
		d := float64(now.Sub(start).Microseconds()) / microsPerMs
		shutdownDur = &d
	}

	svcType := r.serviceTypeForLocked(key)

	ref := ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName}
	evt := newEventFromRef(seq, now, EventTypeShutdown, PhaseAfter, ref, r.containerID, svcType, nil, errStr)
	r.appendEventLocked(evt)

	if rec, ok := r.services[key]; ok {
		rec.shutdownAt = &now
		rec.shutdownDurationMs = shutdownDur
		rec.shutdownError = errStr
	}

	r.mu.Unlock()

	if r.onEvent != nil {
		r.onEvent(evt)
	}
}

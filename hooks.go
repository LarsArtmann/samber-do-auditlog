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
	containerID ContainerID,
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
func popStackFrame(stack []stackEntry, scopeID ScopeID, serviceName ServiceName) ([]stackEntry, stackEntry, bool) {
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
func newServiceRecordCore(scopeID ScopeID, scopeName string, serviceName ServiceName, svcType ProviderType, now time.Time) *serviceRecord {
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
func inferServiceType(scope *do.Scope, serviceName ServiceName) ProviderType {
	desc, ok := do.ExplainNamedService(scope, string(serviceName))
	if !ok {
		return ""
	}

	return ProviderType(desc.ServiceType)
}

// --- Hook methods (single-lock hot path) ---

// fireEvent invokes the onEvent callback if configured. Always called
// outside the mutex to avoid blocking the hot path.
func (r *Recorder) fireEvent(evt Event) {
	if r.onEvent != nil {
		r.onEvent(evt)
	}
}

// publishLockedEvent finalizes a hook: appends evt under r.mu, releases the
// lock, and fires the onEvent callback outside the lock. Centralizes the
// 3-line "append + unlock + fire" closing that every before/after hook shares.
// Caller must hold r.mu; on return the lock is released.
func (r *Recorder) publishLockedEvent(evt Event) {
	r.appendEventLocked(evt)
	r.mu.Unlock()
	r.fireEvent(evt)
}

// hookContext bundles the per-hook preamble that every before/after
// invocation+shutdown hook shares: scope identifiers, current time, and a
// monotonically increasing sequence number. Centralizing this avoids the
// repeated 5-line `scopeID := ...; scopeName := ...; now := ...; key := ...;
// seq := ...` that previously appeared in 4 hook methods.
type hookContext struct {
	scopeID     ScopeID
	scopeName   string
	serviceName ServiceName
	key         svcKey
	now         time.Time
	seq         int
}

// beginBeforeHook builds a hookContext for a before-hook (no error to
// pre-stringify). Caller is responsible for r.mu.Lock() and for invoking
// recordScopeLocked after acquiring the lock.
func (r *Recorder) beginBeforeHook(scope *do.Scope, serviceName string) hookContext {
	svcName := ServiceName(serviceName)
	scopeID := ScopeID(scope.ID())

	return hookContext{
		scopeID:     scopeID,
		scopeName:   scope.Name(),
		serviceName: svcName,
		key:         svcKey{scopeID: scopeID, name: svcName},
		now:         time.Now(),
		seq:         r.nextSequence(),
	}
}

// beginLockedBeforeHook combines beginBeforeHook with r.mu.Lock() and
// recordScopeLocked so the 3 before-hooks that share this exact preamble
// (registration, invocation, shutdown) don't repeat it inline. Caller
// inherits r.mu and must release it (typically via publishLockedEvent).
func (r *Recorder) beginLockedBeforeHook(scope *do.Scope, serviceName string) hookContext {
	ctx := r.beginBeforeHook(scope, serviceName)

	r.mu.Lock()
	r.recordScopeLocked(ctx.scopeID, ctx.scopeName, scope)

	return ctx
}

// beginAfterHook builds a hookContext for an after-hook, pre-stringifying the
// error so callers can pass it straight into newEventFromRef. Caller is
// responsible for r.mu.Lock().
func (r *Recorder) beginAfterHook(scope *do.Scope, serviceName string, err error) (hookContext, *string) {
	svcName := ServiceName(serviceName)
	scopeID := ScopeID(scope.ID())

	return hookContext{
		scopeID:     scopeID,
		scopeName:   scope.Name(),
		serviceName: svcName,
		key:         svcKey{scopeID: scopeID, name: svcName},
		now:         time.Now(),
		seq:         r.nextSequence(),
	}, errorToStringPtr(err)
}

func (r *Recorder) OnBeforeRegistration(scope *do.Scope, serviceName string) {
	ctx := r.beginLockedBeforeHook(scope, serviceName)

	ref := ServiceRef{ScopeID: ctx.scopeID, ScopeName: ctx.scopeName, ServiceName: ctx.serviceName}
	evt := newEventFromRef(ctx.seq, ctx.now, EventTypeRegistration, PhaseBefore, ref, r.containerID, "", nil, nil)
	r.publishLockedEvent(evt)
}

func (r *Recorder) OnAfterRegistration(scope *do.Scope, serviceName string) {
	ctx := r.beginBeforeHook(scope, serviceName)

	r.mu.Lock()

	rec, ok := r.services[ctx.key]

	var svcType ProviderType

	if !ok {
		svcType = inferServiceType(scope, serviceName)
		rec = newServiceRecordCore(ctx.scopeID, ctx.scopeName, ctx.serviceName, svcType, ctx.now)
		r.services[ctx.key] = rec
	} else {
		svcType = rec.serviceType
	}

	ref := ServiceRef{ScopeID: ctx.scopeID, ScopeName: ctx.scopeName, ServiceName: ctx.serviceName}
	evt := newEventFromRef(ctx.seq, ctx.now, EventTypeRegistration, PhaseAfter, ref, r.containerID, svcType, nil, nil)
	r.publishLockedEvent(evt)
}

func (r *Recorder) OnBeforeInvocation(scope *do.Scope, serviceName string) {
	ctx := r.beginLockedBeforeHook(scope, serviceName)

	recordDependencyFromStack(r.stack, r.services, ctx.key)

	r.stack = append(r.stack, stackEntry{
		scopeID:     ctx.scopeID,
		scopeName:   ctx.scopeName,
		serviceName: ctx.serviceName,
		start:       ctx.now,
	})

	// Look up service type from existing record.
	svcType := r.serviceTypeForLocked(ctx.key)

	ref := ServiceRef{ScopeID: ctx.scopeID, ScopeName: ctx.scopeName, ServiceName: ctx.serviceName}
	evt := newEventFromRef(ctx.seq, ctx.now, EventTypeInvocation, PhaseBefore, ref, r.containerID, svcType, nil, nil)
	r.publishLockedEvent(evt)
}

func (r *Recorder) OnAfterInvocation(scope *do.Scope, serviceName string, err error) {
	ctx, errStr := r.beginAfterHook(scope, serviceName, err)

	r.mu.Lock()

	// Pop matching stack frame (LIFO fast path).
	var durationMs *float64

	newStack, frame, found := popStackFrame(r.stack, ctx.scopeID, ctx.serviceName)
	if found {
		r.stack = newStack
		d := float64(ctx.now.Sub(frame.start).Microseconds()) / microsPerMs
		durationMs = &d
	}

	// Look up service type.
	svcType := r.serviceTypeForLocked(ctx.key)

	ref := ServiceRef{ScopeID: ctx.scopeID, ScopeName: ctx.scopeName, ServiceName: ctx.serviceName}
	evt := newEventFromRef(
		ctx.seq, ctx.now, EventTypeInvocation, PhaseAfter,
		ref, r.containerID, svcType, durationMs, errStr,
	)
	r.appendEventLocked(evt)

	r.updateInvocationAggregate(ctx.scopeID, ctx.scopeName, ctx.serviceName, ctx.now, svcType, durationMs, errStr)

	r.mu.Unlock()

	r.fireEvent(evt)
}

// updateInvocationAggregate updates the per-service aggregate after an invocation.
// Caller must hold r.mu.
func (r *Recorder) updateInvocationAggregate(
	scopeID ScopeID, scopeName string, serviceName ServiceName,
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
	ctx := r.beginLockedBeforeHook(scope, serviceName)
	r.shutdownStart[ctx.key] = ctx.now

	svcType := r.serviceTypeForLocked(ctx.key)

	ref := ServiceRef{ScopeID: ctx.scopeID, ScopeName: ctx.scopeName, ServiceName: ctx.serviceName}
	evt := newEventFromRef(ctx.seq, ctx.now, EventTypeShutdown, PhaseBefore, ref, r.containerID, svcType, nil, nil)
	r.publishLockedEvent(evt)
}

func (r *Recorder) OnAfterShutdown(scope *do.Scope, serviceName string, err error) {
	ctx, errStr := r.beginAfterHook(scope, serviceName, err)

	r.mu.Lock()

	// Resolve shutdown duration.
	start, hasStart := r.shutdownStart[ctx.key]
	if hasStart {
		delete(r.shutdownStart, ctx.key)
	}

	var shutdownDur *float64

	if hasStart {
		d := float64(ctx.now.Sub(start).Microseconds()) / microsPerMs
		shutdownDur = &d
	}

	svcType := r.serviceTypeForLocked(ctx.key)

	ref := ServiceRef{ScopeID: ctx.scopeID, ScopeName: ctx.scopeName, ServiceName: ctx.serviceName}
	evt := newEventFromRef(ctx.seq, ctx.now, EventTypeShutdown, PhaseAfter, ref, r.containerID, svcType, nil, errStr)
	r.appendEventLocked(evt)

	if rec, ok := r.services[ctx.key]; ok {
		rec.shutdownAt = &ctx.now
		rec.shutdownDurationMs = shutdownDur
		rec.shutdownError = errStr
	}

	r.mu.Unlock()

	r.fireEvent(evt)
}

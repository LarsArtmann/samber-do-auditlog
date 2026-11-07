package auditlog

import (
	"slices"
	"time"

	"github.com/samber/do/v2"
)

// newEventFromRef builds an Event from a ServiceRef.
func newEventFromRef(
	seq int,
	now time.Time,
	eventType EventType,
	phase Phase,
	ref ServiceRef,
	containerID ContainerID,
	runID RunID,
	svcType ProviderType,
	dur *float64,
	errStr *string,
) Event {
	return Event{
		RunID:       runID,
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

	return stack, stackEntry{}, false //nolint:exhaustruct_v5 // zero-value sentinel for not-found
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
func newServiceRecordCore(
	scopeID ScopeID,
	scopeName string,
	serviceName ServiceName,
	svcType ProviderType,
	now time.Time,
) *serviceRecord {
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

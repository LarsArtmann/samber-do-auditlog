package auditlog

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

// errReplayValidationFailed indicates the replayed report failed internal validation.
var errReplayValidationFailed = errors.New("replayed report failed validation")

// replayState holds the mutable state accumulated by replaying events.
// It mirrors the Recorder's internal state but without any concurrency
// primitives — replay is a single-threaded, pure transformation.
type replayState struct {
	services      map[svcKey]*serviceRecord
	scopes        map[string]replayScopeMeta
	stack         []stackEntry
	shutdownStart map[svcKey]time.Time
	containerID   string
}

// replayScopeMeta is scope metadata without the live *do.Scope reference.
// The parentID is inferred (first-seen scope becomes root; all others
// parented to it) because events carry only scope_id/scope_name, not
// the parent-child relationship from the original container.
type replayScopeMeta struct {
	id       string
	name     string
	parentID string
}

// newReplayState initializes an empty replay state.
func newReplayState() *replayState {
	return &replayState{ //nolint:exhaustruct
		services:      make(map[svcKey]*serviceRecord),
		scopes:        make(map[string]replayScopeMeta),
		shutdownStart: make(map[svcKey]time.Time),
	}
}

// ReplayEvents reconstructs a Report from a flat event stream.
//
// This is the inverse of the hook-based recording path: instead of live
// *do.Scope callbacks mutating a Recorder, it processes already-captured
// events to rebuild the same serviceRecord/scope state, then assembles
// a Report via the same buildReportFromCore finalizer.
//
// Limitations (documented, not fixable without additional data):
//   - IsHealthchecker and IsShutdowner are always false (they require a
//     live do.ExplainInjector call on a *do.Scope reference).
//   - Scope tree hierarchy is flattened (events carry scope_id/scope_name
//     but not parent_id). The first-seen scope becomes root; all others
//     are its direct children.
//   - DurationMs values are taken from the events themselves, not recomputed
//     from wall-clock time.
//
// The returned Report has Reconstructed=true so consumers can detect
// these limitations.
func ReplayEvents(events []Event) (Report, error) {
	if len(events) == 0 {
		return Report{}, fmt.Errorf("%w: no events to replay", errReplayValidationFailed)
	}

	state := newReplayState()

	for _, evt := range events {
		applyEvent(evt, state)
	}

	services := buildReplayServices(state)
	scopeTree := buildReplayScopeTree(state)
	containerID := state.containerID

	if containerID == "" && len(events) > 0 {
		containerID = events[0].ContainerID
	}

	report := buildReportFromCore(
		SchemaVersion,
		containerID,
		time.Now(),
		0, // replayed reports have no dropped events
		slices.Clone(events),
		services,
		scopeTree,
	)
	report.Reconstructed = true

	err := report.Validate()
	if err != nil {
		return report, fmt.Errorf("%w: %w", errReplayValidationFailed, err)
	}

	return report, nil
}

// applyEvent dispatches a single event to the appropriate state mutation.
func applyEvent(evt Event, state *replayState) {
	if state.containerID == "" {
		state.containerID = evt.ContainerID
	}

	state.recordScope(evt.ScopeID, evt.ScopeName)

	key := svcKey{scopeID: evt.ScopeID, name: evt.ServiceName}

	switch evt.EventType {
	case EventTypeRegistration:
		if evt.Phase == PhaseAfter {
			state.applyRegistrationAfter(evt, key)
		}

	case EventTypeInvocation:
		switch evt.Phase {
		case PhaseBefore:
			state.applyInvocationBefore(evt, key)
		case PhaseAfter:
			state.applyInvocationAfter(evt, key)
		}

	case EventTypeShutdown:
		switch evt.Phase {
		case PhaseBefore:
			state.shutdownStart[key] = evt.Timestamp
		case PhaseAfter:
			state.applyShutdownAfter(evt, key)
		}

	case EventTypeHealthCheck:
		if evt.Phase == PhaseAfter {
			state.applyHealthCheck(evt, key)
		}
	}
}

// recordScope records a scope if not already seen. The first scope becomes
// root (parentID=""); all subsequent scopes are parented to the root.
func (state *replayState) recordScope(scopeID, scopeName string) {
	if _, exists := state.scopes[scopeID]; exists {
		return
	}

	rootID := state.firstScopeID()

	if rootID == "" {
		state.scopes[scopeID] = replayScopeMeta{
			id:       scopeID,
			name:     scopeName,
			parentID: "",
		}

		return
	}

	state.scopes[scopeID] = replayScopeMeta{
		id:       scopeID,
		name:     scopeName,
		parentID: rootID,
	}
}

// firstScopeID returns the ID of the first-recorded scope, or "" if none yet.
func (state *replayState) firstScopeID() string {
	if len(state.scopes) == 0 {
		return ""
	}

	sorted := sortedReplayScopes(state.scopes)

	return sorted[0].id
}

func (state *replayState) applyRegistrationAfter(evt Event, key svcKey) {
	rec, ok := state.services[key]
	if !ok {
		rec = newServiceRecordCore(evt.ScopeID, evt.ScopeName, evt.ServiceName, evt.ServiceType, evt.Timestamp)
		state.services[key] = rec
	} else {
		rec.serviceType = evt.ServiceType
	}
}

func (state *replayState) applyInvocationBefore(evt Event, key svcKey) {
	if len(state.stack) > 0 {
		parent := state.stack[len(state.stack)-1]
		parentKey := svcKey{scopeID: parent.scopeID, name: parent.serviceName}

		if rec, ok := state.services[parentKey]; ok {
			if rec.dependencies == nil {
				rec.dependencies = make(map[svcKey]struct{}, initialDepsCapacity)
			}

			rec.dependencies[key] = struct{}{}
		}
	}

	state.stack = append(state.stack, stackEntry{
		scopeID:     evt.ScopeID,
		scopeName:   evt.ScopeName,
		serviceName: evt.ServiceName,
		start:       evt.Timestamp,
	})
}

func (state *replayState) applyInvocationAfter(evt Event, key svcKey) {
	for i, frame := range slices.Backward(state.stack) {
		if frame.serviceName == evt.ServiceName && frame.scopeID == evt.ScopeID {
			if i == len(state.stack)-1 {
				state.stack = state.stack[:i]
			} else {
				state.stack = append(state.stack[:i], state.stack[i+1:]...)
			}

			break
		}
	}

	rec, ok := state.services[key]
	if !ok {
		rec = newServiceRecordCore(evt.ScopeID, evt.ScopeName, evt.ServiceName, evt.ServiceType, evt.Timestamp)
		state.services[key] = rec
	}

	if evt.ServiceType != "" {
		rec.serviceType = evt.ServiceType
	}

	rec.invocationCount++

	if rec.firstInvokedAt == nil {
		t := evt.Timestamp
		rec.firstInvokedAt = &t
		rec.invocationOrder = rec.invocationCount - 1
	}

	if evt.DurationMs != nil && rec.firstBuildDurationMs == nil {
		d := *evt.DurationMs
		rec.firstBuildDurationMs = &d
	}

	if evt.Error != nil {
		rec.invocationError = evt.Error
	}
}

func (state *replayState) applyShutdownAfter(evt Event, key svcKey) {
	rec, ok := state.services[key]
	if !ok {
		rec = newServiceRecordCore(evt.ScopeID, evt.ScopeName, evt.ServiceName, evt.ServiceType, evt.Timestamp)
		state.services[key] = rec
	}

	t := evt.Timestamp
	rec.shutdownAt = &t

	if start, hasStart := state.shutdownStart[key]; hasStart {
		d := float64(evt.Timestamp.Sub(start).Microseconds()) / microsPerMs
		rec.shutdownDurationMs = &d

		delete(state.shutdownStart, key)
	} else if evt.DurationMs != nil {
		d := *evt.DurationMs
		rec.shutdownDurationMs = &d
	}

	if evt.Error != nil {
		rec.shutdownError = evt.Error
	}
}

func (state *replayState) applyHealthCheck(evt Event, key svcKey) {
	rec, ok := state.services[key]
	if !ok {
		rec = newServiceRecordCore(evt.ScopeID, evt.ScopeName, evt.ServiceName, evt.ServiceType, evt.Timestamp)
		state.services[key] = rec
	}

	t := evt.Timestamp
	rec.lastHealthCheckAt = &t
	rec.healthCheckError = evt.Error
	rec.healthCheckCount++
}

// buildReplayServices assembles sorted ServiceInfo from replay state.
func buildReplayServices(state *replayState) []ServiceInfo {
	dependents := buildDependentsMapLocked(state.services)

	services := make([]ServiceInfo, 0, len(state.services))

	for _, rec := range state.services {
		deps := buildReplayDeps(rec, state.services)
		key := svcKey{scopeID: rec.scopeID, name: rec.serviceName}
		svcDependents := dependents[key]

		sortDepRefs(svcDependents)

		svc := serviceRecordToInfo(rec)
		svc.Dependencies = deps
		svc.Dependents = svcDependents
		services = append(services, svc)
	}

	slices.SortFunc(services, func(a, b ServiceInfo) int {
		return compareByName(a.ServiceRef, b.ServiceRef)
	})

	return services
}

// buildReplayDeps builds sorted dependency refs from a replay service record.
func buildReplayDeps(rec *serviceRecord, services map[svcKey]*serviceRecord) []ServiceRef {
	if len(rec.dependencies) == 0 {
		return nil
	}

	deps := make([]ServiceRef, 0, len(rec.dependencies))
	for depKey := range rec.dependencies {
		if depRec, ok := services[depKey]; ok {
			deps = append(deps, ServiceRef{
				ScopeID:     depRec.scopeID,
				ScopeName:   depRec.scopeName,
				ServiceName: depRec.serviceName,
			})
		}
	}

	sortDepRefs(deps)

	return deps
}

// buildReplayScopeTree builds a ScopeNode tree from replay scopes.
func buildReplayScopeTree(state *replayState) ScopeNode {
	sorted := sortedReplayScopes(state.scopes)
	if len(sorted) == 0 {
		return ScopeNode{} //nolint:exhaustruct
	}

	var root replayScopeMeta

	hasRoot := false

	for _, meta := range sorted {
		if meta.parentID == "" {
			root = meta
			hasRoot = true

			break
		}
	}

	if !hasRoot {
		root = sorted[0]
	}

	scopeServices := make(map[string][]string)
	for _, rec := range state.services {
		scopeServices[rec.scopeID] = append(scopeServices[rec.scopeID], rec.serviceName)
	}

	for id, names := range scopeServices {
		slices.Sort(names)
		scopeServices[id] = names
	}

	var build func(parentID string) []ScopeNode

	build = func(parentID string) []ScopeNode {
		var children []ScopeNode

		for _, meta := range sorted {
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
		Children: sortScopeNodes(build(root.id)),
	}
}

func sortedReplayScopes(scopes map[string]replayScopeMeta) []replayScopeMeta {
	result := make([]replayScopeMeta, 0, len(scopes))
	for _, meta := range scopes {
		result = append(result, meta)
	}

	slices.SortFunc(result, func(a, b replayScopeMeta) int {
		if a.id < b.id {
			return -1
		}

		if a.id > b.id {
			return 1
		}

		return 0
	})

	return result
}

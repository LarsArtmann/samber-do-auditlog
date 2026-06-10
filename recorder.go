package auditlog

import (
	"cmp"
	"maps"
	"slices"
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
// All mutable state is protected by a single RWMutex (mu), which reduces lock
// acquisition overhead from 2-4 acquisitions per hook to exactly 1. The
// invocation counter uses an atomic, eliminating a separate mutex.
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

// inferServiceType uses do.ExplainNamedService to determine the provider type
// (lazy, eager, transient, alias). Returns empty string if unknown.
func inferServiceType(scope *do.Scope, serviceName string) ProviderType {
	desc, ok := do.ExplainNamedService(scope, serviceName)
	if !ok {
		return ""
	}

	return ProviderType(desc.ServiceType)
}

// enrichCapabilities populates IsHealthchecker and IsShutdowner on each ServiceInfo
// by calling do.ExplainInjector on each stored scope reference. Must be called
// outside the recorder mutex to avoid deadlocking with samber/do's internal locks.
func enrichCapabilities(scopes map[string]scopeMeta, services []ServiceInfo) {
	for _, meta := range scopes {
		if meta.ref == nil {
			continue
		}

		output := do.ExplainInjector(meta.ref)
		svcMap := buildCapabilityMap(output.DAG)

		for i := range services {
			if services[i].ScopeID != meta.id {
				continue
			}

			caps, ok := svcMap[services[i].ServiceName]
			if ok {
				services[i].IsHealthchecker = caps[0]
				services[i].IsShutdowner = caps[1]
			}
		}
	}
}

func buildCapabilityMap(scopes []do.ExplainInjectorScopeOutput) map[string][2]bool {
	result := make(map[string][2]bool)

	for _, s := range scopes {
		for _, svc := range s.Services {
			result[svc.ServiceName] = [2]bool{svc.IsHealthchecker, svc.IsShutdowner}
		}

		maps.Copy(result, buildCapabilityMap(s.Children))
	}

	return result
}

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

// --- Hook methods (single-lock hot path) ---

func (r *Recorder) OnBeforeRegistration(scope *do.Scope, serviceName string) {
	scopeID := scope.ID()
	scopeName := scope.Name()
	now := time.Now()
	seq := r.nextSequence()

	evt := Event{
		Sequence:    seq,
		Timestamp:   now,
		EventType:   EventTypeRegistration,
		Phase:       PhaseBefore,
		ContainerID: r.containerID,
		ServiceRef:  ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName},
		ServiceType: "",
		DurationMs:  nil,
		Error:       nil,
	}

	r.mu.Lock()
	r.recordScopeLocked(scopeID, scopeName, scope)
	r.events = append(r.events, evt)
	r.mu.Unlock()

	if r.onEvent != nil {
		r.onEvent(evt)
	}
}

func (r *Recorder) OnAfterRegistration(scope *do.Scope, serviceName string) {
	scopeID := scope.ID()
	scopeName := scope.Name()
	now := time.Now()
	key := serviceKey(scopeID, serviceName)
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

	evt := Event{
		Sequence:    seq,
		Timestamp:   now,
		EventType:   EventTypeRegistration,
		Phase:       PhaseAfter,
		ContainerID: r.containerID,
		ServiceRef:  ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName},
		ServiceType: svcType,
		DurationMs:  nil,
		Error:       nil,
	}
	r.events = append(r.events, evt)

	r.mu.Unlock()

	if r.onEvent != nil {
		r.onEvent(evt)
	}
}

func (r *Recorder) OnBeforeInvocation(scope *do.Scope, serviceName string) {
	scopeID := scope.ID()
	scopeName := scope.Name()
	now := time.Now()
	depKey := serviceKey(scopeID, serviceName)
	seq := r.nextSequence()

	r.mu.Lock()

	r.recordScopeLocked(scopeID, scopeName, scope)

	// Infer dependency from invocation stack.
	if len(r.stack) > 0 {
		parent := r.stack[len(r.stack)-1]
		parentKey := serviceKey(parent.scopeID, parent.serviceName)

		if rec, ok := r.services[parentKey]; ok {
			if rec.dependencies == nil {
				rec.dependencies = make(map[string]struct{}, initialDepsCapacity)
			}

			rec.dependencies[depKey] = struct{}{}
		}
	}

	r.stack = append(r.stack, stackEntry{
		scopeID:     scopeID,
		scopeName:   scopeName,
		serviceName: serviceName,
		start:       now,
	})

	// Look up service type from existing record.
	var svcType ProviderType

	if rec, ok := r.services[depKey]; ok {
		svcType = rec.serviceType
	}

	evt := Event{
		Sequence:    seq,
		Timestamp:   now,
		EventType:   EventTypeInvocation,
		Phase:       PhaseBefore,
		ContainerID: r.containerID,
		ServiceRef:  ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName},
		ServiceType: svcType,
		DurationMs:  nil,
		Error:       nil,
	}
	r.events = append(r.events, evt)

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
	key := serviceKey(scopeID, serviceName)
	seq := r.nextSequence()

	r.mu.Lock()

	// Pop matching stack frame (LIFO fast path).
	var durationMs *float64

	for i, frame := range slices.Backward(r.stack) {
		if frame.serviceName == serviceName && frame.scopeID == scopeID {
			d := float64(now.Sub(frame.start).Microseconds()) / microsPerMs
			durationMs = &d

			if i == len(r.stack)-1 {
				r.stack = r.stack[:i]
			} else {
				r.stack = append(r.stack[:i], r.stack[i+1:]...)
			}

			break
		}
	}

	// Look up service type.
	var svcType ProviderType

	if rec, ok := r.services[key]; ok {
		svcType = rec.serviceType
	}

	evt := Event{
		Sequence:    seq,
		Timestamp:   now,
		EventType:   EventTypeInvocation,
		Phase:       PhaseAfter,
		ContainerID: r.containerID,
		ServiceRef:  ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName},
		ServiceType: svcType,
		DurationMs:  durationMs,
		Error:       errStr,
	}
	r.events = append(r.events, evt)

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
	key := serviceKey(scopeID, serviceName)

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
	key := serviceKey(scopeID, serviceName)
	seq := r.nextSequence()

	r.mu.Lock()

	r.recordScopeLocked(scopeID, scopeName, scope)
	r.shutdownStart[key] = now

	svcType := ProviderType("")

	if rec, ok := r.services[key]; ok {
		svcType = rec.serviceType
	}

	evt := Event{
		Sequence:    seq,
		Timestamp:   now,
		EventType:   EventTypeShutdown,
		Phase:       PhaseBefore,
		ContainerID: r.containerID,
		ServiceRef:  ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName},
		ServiceType: svcType,
		DurationMs:  nil,
		Error:       nil,
	}
	r.events = append(r.events, evt)

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
	key := serviceKey(scopeID, serviceName)
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

	svcType := ProviderType("")

	if rec, ok := r.services[key]; ok {
		svcType = rec.serviceType
	}

	evt := Event{
		Sequence:    seq,
		Timestamp:   now,
		EventType:   EventTypeShutdown,
		Phase:       PhaseAfter,
		ContainerID: r.containerID,
		ServiceRef:  ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName},
		ServiceType: svcType,
		DurationMs:  nil,
		Error:       errStr,
	}
	r.events = append(r.events, evt)

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

// errorToStringPtr converts an error to a heap-allocated string pointer.
// Returns nil when err is nil so we don't emit empty error fields in events.
func errorToStringPtr(err error) *string {
	if err == nil {
		return nil
	}

	msg := err.Error()

	return &msg
}

// computeServiceStatus derives the lifecycle status from a service record.
// Invocation errors take priority over shutdown errors.
func computeServiceStatus(rec *serviceRecord) ServiceStatus {
	if rec.invocationError != nil {
		return ServiceStatusInvocationError
	}

	if rec.shutdownError != nil {
		return ServiceStatusShutdownError
	}

	if rec.shutdownAt != nil {
		return ServiceStatusShutdown
	}

	if rec.firstInvokedAt != nil {
		return ServiceStatusActive
	}

	return ServiceStatusRegistered
}

// BuildReport assembles a machine-readable Report from all captured events.
func (r *Recorder) BuildReport() Report {
	r.mu.RLock()
	services := r.buildServicesLocked()
	scopeTree := r.buildScopeTreeLocked()
	events := append([]Event(nil), r.events...)
	scopeCount := len(r.scopes)
	scopesCopy := make(map[string]scopeMeta, len(r.scopes))
	maps.Copy(scopesCopy, r.scopes)

	r.mu.RUnlock()

	enrichCapabilities(scopesCopy, services)

	return Report{
		Version:                 SchemaVersion,
		ContainerID:             r.containerID,
		ExportedAt:              time.Now(),
		EventCount:              len(events),
		ServiceCount:            len(services),
		ScopeCount:              scopeCount,
		TotalBuildDurationMs:    sumBuildMs(services),
		TotalShutdownDurationMs: sumShutdownMs(services),
		ShutdownSucceeded:       noShutdownErrors(services),
		HealthCheckSucceeded:    allHealthChecksPassed(services),
		HealthCheckedCount:      countHealthChecked(services),
		Events:                  events,
		Services:                services,
		ScopeTree:               scopeTree,
	}
}

// buildServicesLocked assembles sorted ServiceInfo from the recorded data.
// Must be called with r.mu held for reading.
func (r *Recorder) buildServicesLocked() []ServiceInfo {
	dependents := buildDependentsMapLocked(r.services)

	services := make([]ServiceInfo, 0, len(r.services))
	for _, rec := range r.services {
		deps := r.buildDepsLocked(rec)

		key := serviceKey(rec.scopeID, rec.serviceName)
		svcDependents := dependents[key]

		sortDepRefs(svcDependents)

		services = append(services, ServiceInfo{
			ServiceRef: ServiceRef{
				ServiceName: rec.serviceName,
				ScopeID:     rec.scopeID,
				ScopeName:   rec.scopeName,
			},
			Status:               computeServiceStatus(rec),
			ServiceType:          rec.serviceType,
			RegisteredAt:         rec.registeredAt,
			FirstInvokedAt:       rec.firstInvokedAt,
			InvocationCount:      rec.invocationCount,
			InvocationOrder:      rec.invocationOrder,
			FirstBuildDurationMs: rec.firstBuildDurationMs,
			Dependencies:         deps,
			Dependents:           svcDependents,
			ShutdownAt:           rec.shutdownAt,
			ShutdownDurationMs:   rec.shutdownDurationMs,
			ShutdownError:        rec.shutdownError,
			InvocationError:      rec.invocationError,
			IsHealthchecker:      false,
			IsShutdowner:         false,
			LastHealthCheckAt:    rec.lastHealthCheckAt,
			HealthCheckError:     rec.healthCheckError,
			HealthCheckCount:     rec.healthCheckCount,
		})
	}

	slices.SortFunc(services, func(a, b ServiceInfo) int {
		return compareByName(a.ServiceRef, b.ServiceRef)
	})

	return services
}

// buildDepsLocked builds sorted dependency refs for a service record.
// Must be called with r.mu held for reading.
func (r *Recorder) buildDepsLocked(rec *serviceRecord) []ServiceRef {
	if len(rec.dependencies) == 0 {
		return nil
	}

	deps := make([]ServiceRef, 0, len(rec.dependencies))
	for depKey := range rec.dependencies {
		if depRec, ok := r.services[depKey]; ok {
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

func sortDepRefs(refs []ServiceRef) {
	slices.SortFunc(refs, compareByName)
}

func compareByName(a, b ServiceRef) int {
	return cmp.Or(
		cmp.Compare(a.ScopeName, b.ScopeName),
		cmp.Compare(a.ServiceName, b.ServiceName),
	)
}

func buildDependentsMapLocked(services map[string]*serviceRecord) map[string][]ServiceRef {
	dependents := make(map[string][]ServiceRef)

	for _, rec := range services {
		for depKey := range rec.dependencies {
			if _, ok := services[depKey]; ok {
				dependents[depKey] = append(dependents[depKey], ServiceRef{
					ScopeID:     rec.scopeID,
					ScopeName:   rec.scopeName,
					ServiceName: rec.serviceName,
				})
			}
		}
	}

	return dependents
}

func (r *Recorder) buildScopeTreeLocked() ScopeNode {
	sortedScopes := sortedScopesLocked(r.scopes)

	var root scopeMeta

	hasRoot := false

	for _, meta := range sortedScopes {
		if meta.parentID == "" {
			root = meta
			hasRoot = true

			break
		}
	}

	if !hasRoot && len(sortedScopes) > 0 {
		root = sortedScopes[0]
	}

	scopeServices := make(map[string][]string)
	for _, rec := range r.services {
		scopeServices[rec.scopeID] = append(scopeServices[rec.scopeID], rec.serviceName)
	}

	for id, names := range scopeServices {
		slices.Sort(names)
		scopeServices[id] = names
	}

	var build func(parentID string) []ScopeNode

	build = func(parentID string) []ScopeNode {
		var children []ScopeNode

		for _, meta := range sortedScopes {
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

func sortedScopesLocked(scopes map[string]scopeMeta) []scopeMeta {
	result := make([]scopeMeta, 0, len(scopes))

	for _, meta := range scopes {
		result = append(result, meta)
	}

	slices.SortFunc(result, func(a, b scopeMeta) int {
		return cmp.Compare(a.id, b.id)
	})

	return result
}

func sortScopeNodes(nodes []ScopeNode) []ScopeNode {
	slices.SortFunc(nodes, func(a, b ScopeNode) int {
		return cmp.Compare(a.Name, b.Name)
	})

	for i := range nodes {
		nodes[i].Children = sortScopeNodes(nodes[i].Children)
	}

	return nodes
}

func sumDurationField(services []ServiceInfo, get func(ServiceInfo) *float64) float64 {
	total := 0.0

	for _, s := range services {
		if v := get(s); v != nil {
			total += *v
		}
	}

	return total
}

func sumBuildMs(services []ServiceInfo) float64 {
	return sumDurationField(services, func(s ServiceInfo) *float64 { return s.FirstBuildDurationMs })
}

func sumShutdownMs(services []ServiceInfo) float64 {
	return sumDurationField(services, func(s ServiceInfo) *float64 { return s.ShutdownDurationMs })
}

func noShutdownErrors(services []ServiceInfo) bool {
	for _, s := range services {
		if s.ShutdownError != nil {
			return false
		}
	}

	return true
}

func countHealthChecked(services []ServiceInfo) int {
	count := 0

	for _, s := range services {
		if s.HealthCheckCount > 0 {
			count++
		}
	}

	return count
}

func allHealthChecksPassed(services []ServiceInfo) bool {
	checked := 0

	for _, s := range services {
		if s.HealthCheckCount > 0 {
			checked++

			if s.HealthCheckError != nil {
				return false
			}
		}
	}

	return checked > 0
}

// RecordHealthCheck records a single health check result for a service.
func (r *Recorder) RecordHealthCheck(scopeID, scopeName, serviceName string, err error) {
	now := time.Now()
	errStr := errorToStringPtr(err)
	seq := r.nextSequence()

	ref := ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName}

	r.mu.Lock()

	svcType := ProviderType("")
	key := serviceKey(scopeID, serviceName)

	if rec, ok := r.services[key]; ok {
		svcType = rec.serviceType
	}

	evt := newEventFromRef(
		seq, now, EventTypeHealthCheck, PhaseAfter,
		ref, r.containerID, svcType, nil, errStr,
	)
	r.events = append(r.events, evt)

	rec, ok := r.services[key]
	if !ok {
		rec = newServiceRecordCore(scopeID, scopeName, serviceName, "", now)
		r.services[key] = rec
	}

	rec.lastHealthCheckAt = &now
	rec.healthCheckError = errStr
	rec.healthCheckCount++

	r.mu.Unlock()

	if r.onEvent != nil {
		r.onEvent(evt)
	}
}

// ResolveServiceScope finds the scope metadata for a service by name.
// Returns (scopeID, scopeName, true) if found, or ("", "", false) otherwise.
func (r *Recorder) ResolveServiceScope(injector do.Injector, serviceName string) (string, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	injectorScopeID := injector.ID()
	if rec, ok := r.services[serviceKey(injectorScopeID, serviceName)]; ok {
		return rec.scopeID, rec.scopeName, true
	}

	if scope, ok := injector.(*do.Scope); ok {
		for _, ancestor := range scope.Ancestors() {
			if rec, ok := r.services[serviceKey(ancestor.ID(), serviceName)]; ok {
				return rec.scopeID, rec.scopeName, true
			}
		}
	}

	return "", "", false
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

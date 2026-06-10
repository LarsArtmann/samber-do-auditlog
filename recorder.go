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
type Recorder struct {
	mu       sync.RWMutex
	events   []Event
	services map[string]*serviceRecord
	scopes   map[string]scopeMeta

	stackMu sync.Mutex
	stack   []stackEntry

	invocationMu    sync.Mutex
	invocationIndex int

	sequence    *atomic.Int64
	containerID string
	onEvent     func(Event)

	shutdownMu    sync.Mutex
	shutdownStart map[string]time.Time
}

// NewRecorder creates a new event recorder.
func NewRecorder(containerID string, onEvent func(Event)) *Recorder {
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
		containerID:     containerID,
		onEvent:         onEvent,
		shutdownMu:      sync.Mutex{},
		shutdownStart:   make(map[string]time.Time),
	}
}

func (r *Recorder) nextSequence() int {
	return int(r.sequence.Add(1))
}

func (r *Recorder) recordScope(scope *do.Scope) {
	scopeID := scope.ID()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.scopes[scopeID]; ok {
		return
	}

	meta := scopeMeta{id: scopeID, name: scope.Name(), parentID: "", ref: scope}
	if ancestors := scope.Ancestors(); len(ancestors) > 0 {
		meta.parentID = ancestors[0].ID()
	}

	r.scopes[scopeID] = meta
}

// scopeKey produces the canonical map key for a service within a scope.
func scopeKey(scope *do.Scope, serviceName string) string {
	return serviceKey(scope.ID(), serviceName)
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

// newEvent builds an Event struct with all fields initialized.
func newEvent(
	seq int,
	now time.Time,
	eventType EventType,
	phase Phase,
	scope *do.Scope,
	serviceName string,
	containerID string,
	dur *float64,
	errStr *string,
) Event {
	return newEventFromRef(seq, now, eventType, phase, ServiceRef{
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
	}, containerID, inferServiceType(scope, serviceName), dur, errStr)
}

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

// newServiceRecord constructs a serviceRecord with all fields set.
func newServiceRecord(scope *do.Scope, serviceName string, now time.Time) *serviceRecord {
	return &serviceRecord{
		scopeID:              scope.ID(),
		scopeName:            scope.Name(),
		serviceName:          serviceName,
		serviceType:          inferServiceType(scope, serviceName),
		registeredAt:         now,
		firstInvokedAt:       nil,
		invocationCount:      0,
		invocationOrder:      0,
		firstBuildDurationMs: nil,
		dependencies:         make(map[string]struct{}),
		shutdownAt:           nil,
		shutdownDurationMs:   nil,
		invocationError:      nil,
		shutdownError:        nil,
		lastHealthCheckAt:    nil,
		healthCheckError:     nil,
		healthCheckCount:     0,
	}
}

func newServiceRecordFromMeta(scopeID, scopeName, serviceName string, now time.Time) *serviceRecord {
	return &serviceRecord{
		scopeID:              scopeID,
		scopeName:            scopeName,
		serviceName:          serviceName,
		serviceType:          "",
		registeredAt:         now,
		firstInvokedAt:       nil,
		invocationCount:      0,
		invocationOrder:      0,
		firstBuildDurationMs: nil,
		dependencies:         make(map[string]struct{}),
		shutdownAt:           nil,
		shutdownDurationMs:   nil,
		invocationError:      nil,
		shutdownError:        nil,
		lastHealthCheckAt:    nil,
		healthCheckError:     nil,
		healthCheckCount:     0,
	}
}

func (r *Recorder) OnBeforeRegistration(scope *do.Scope, serviceName string) {
	r.recordScope(scope)
	r.addEvent(
		newEvent(
			r.nextSequence(),
			time.Now(),
			EventTypeRegistration,
			PhaseBefore,
			scope,
			serviceName,
			r.containerID,
			nil,
			nil,
		),
	)
}

func (r *Recorder) OnAfterRegistration(scope *do.Scope, serviceName string) {
	now := time.Now()
	key := scopeKey(scope, serviceName)

	r.mu.Lock()
	if _, ok := r.services[key]; !ok {
		r.services[key] = newServiceRecord(scope, serviceName, now)
	}
	r.mu.Unlock()
	r.addEvent(
		newEvent(r.nextSequence(), now, EventTypeRegistration, PhaseAfter, scope, serviceName, r.containerID, nil, nil),
	)
}

func (r *Recorder) OnBeforeInvocation(scope *do.Scope, serviceName string) {
	r.recordScope(scope)

	now := time.Now()
	depKey := scopeKey(scope, serviceName)

	r.stackMu.Lock()
	if len(r.stack) > 0 {
		parent := r.stack[len(r.stack)-1]
		parentKey := serviceKey(parent.scopeID, parent.serviceName)

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

	r.addEvent(
		newEvent(r.nextSequence(), now, EventTypeInvocation, PhaseBefore, scope, serviceName, r.containerID, nil, nil),
	)
}

func (r *Recorder) OnAfterInvocation(scope *do.Scope, serviceName string, err error) {
	now := time.Now()
	durationMs := r.popInvocationDuration(scope, serviceName, now)
	errStr := errorToStringPtr(err)
	r.addEvent(
		newEvent(
			r.nextSequence(),
			now,
			EventTypeInvocation,
			PhaseAfter,
			scope,
			serviceName,
			r.containerID,
			durationMs,
			errStr,
		),
	)
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
			d := float64(now.Sub(frame.start).Microseconds()) / microsPerMs
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
	key := scopeKey(scope, serviceName)

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
	r.recordScope(scope)

	now := time.Now()
	key := scopeKey(scope, serviceName)

	r.shutdownMu.Lock()
	r.shutdownStart[key] = now
	r.shutdownMu.Unlock()

	r.addEvent(
		newEvent(r.nextSequence(), now, EventTypeShutdown, PhaseBefore, scope, serviceName, r.containerID, nil, nil),
	)
}

func (r *Recorder) OnAfterShutdown(scope *do.Scope, serviceName string, err error) {
	now := time.Now()
	errStr := errorToStringPtr(err)

	r.addEvent(
		newEvent(r.nextSequence(), now, EventTypeShutdown, PhaseAfter, scope, serviceName, r.containerID, nil, errStr),
	)

	key := scopeKey(scope, serviceName)

	r.shutdownMu.Lock()

	start, ok := r.shutdownStart[key]
	if ok {
		delete(r.shutdownStart, key)
	}
	r.shutdownMu.Unlock()

	var shutdownDur *float64

	if ok {
		d := float64(now.Sub(start).Microseconds()) / microsPerMs
		shutdownDur = &d
	}

	r.mu.Lock()
	if rec, ok := r.services[key]; ok {
		rec.shutdownAt = &now
		rec.shutdownDurationMs = shutdownDur
		rec.shutdownError = errStr
	}
	r.mu.Unlock()
}

func (r *Recorder) addEvent(evt Event) {
	r.mu.Lock()
	r.events = append(r.events, evt)
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

	r.addEvent(newEventFromRef(
		seq, now, EventTypeHealthCheck, PhaseAfter,
		ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName},
		r.containerID, "", nil, errStr,
	))

	r.mu.Lock()
	defer r.mu.Unlock()

	key := serviceKey(scopeID, serviceName)

	rec, ok := r.services[key]
	if !ok {
		rec = newServiceRecordFromMeta(scopeID, scopeName, serviceName, now)
		r.services[key] = rec
	}

	rec.lastHealthCheckAt = &now
	rec.healthCheckError = errStr
	rec.healthCheckCount++
}

// ResolveServiceScope finds the scope metadata for a service by name.
// Returns (scopeID, scopeName, true) if found, or ("", "", false) otherwise.
func (r *Recorder) ResolveServiceScope(injector do.Injector, serviceName string) (string, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check by injector's scope ID first (handles both RootScope and Scope).
	injectorScopeID := injector.ID()
	if rec, ok := r.services[serviceKey(injectorScopeID, serviceName)]; ok {
		return rec.scopeID, rec.scopeName, true
	}

	// Walk ancestor scopes (only relevant for child scopes).
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

package auditlog

import (
	"cmp"
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

func (e stackEntry) key() string {
	return e.scopeID + "/" + e.serviceName
}

type serviceRecord struct {
	scopeID              string
	scopeName            string
	serviceName          string
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
}

func (r *serviceRecord) key() string {
	return r.scopeID + "/" + r.serviceName
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

	sequence    *atomic.Int64
	containerID string

	shutdownMu    sync.Mutex
	shutdownStart map[string]time.Time
}

// NewRecorder creates a new event recorder.
func NewRecorder(containerID string) *Recorder {
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

	meta := scopeMeta{id: scopeID, name: scope.Name(), parentID: ""}
	if ancestors := scope.Ancestors(); len(ancestors) > 0 {
		meta.parentID = ancestors[0].ID()
	}

	r.scopes[scopeID] = meta
}

// scopeKey produces the canonical map key for a service within a scope.
func scopeKey(scope *do.Scope, serviceName string) string {
	return scope.ID() + "/" + serviceName
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
	return Event{
		Sequence:    seq,
		Timestamp:   now,
		EventType:   eventType,
		Phase:       phase,
		ContainerID: containerID,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
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
		parentKey := parent.key()

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
	defer r.mu.RUnlock()

	services := r.buildServicesLocked()
	scopeTree := r.buildScopeTreeLocked()

	return Report{
		Version:      SchemaVersion,
		ContainerID:  r.containerID,
		ExportedAt:   time.Now(),
		EventCount:   len(r.events),
		ServiceCount: len(services),
		Events:       append([]Event(nil), r.events...),
		Services:     services,
		ScopeTree:    scopeTree,
	}
}

// buildServicesLocked assembles sorted ServiceInfo from the recorded data.
// Must be called with r.mu held for reading.
func (r *Recorder) buildServicesLocked() []ServiceInfo {
	dependents := buildDependentsMapLocked(r.services)

	services := make([]ServiceInfo, 0, len(r.services))
	for _, rec := range r.services {
		deps := r.buildDepsLocked(rec)

		key := rec.key()
		svcDependents := dependents[key]

		sortDepRefs(svcDependents)

		services = append(services, ServiceInfo{
			ServiceName:          rec.serviceName,
			ScopeID:              rec.scopeID,
			ScopeName:            rec.scopeName,
			Status:               computeServiceStatus(rec),
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
		})
	}

	slices.SortFunc(services, func(a, b ServiceInfo) int {
		return cmp.Or(
			cmp.Compare(a.ScopeName, b.ScopeName),
			cmp.Compare(a.ServiceName, b.ServiceName),
		)
	})

	return services
}

// buildDepsLocked builds sorted dependency refs for a service record.
// Must be called with r.mu held for reading.
func (r *Recorder) buildDepsLocked(rec *serviceRecord) []DependencyRef {
	deps := make([]DependencyRef, 0, len(rec.dependencies))
	for depKey := range rec.dependencies {
		if depRec, ok := r.services[depKey]; ok {
			deps = append(deps, DependencyRef{
				ScopeID:     depRec.scopeID,
				ScopeName:   depRec.scopeName,
				ServiceName: depRec.serviceName,
			})
		}
	}

	sortDepRefs(deps)

	return deps
}

func sortDepRefs(refs []DependencyRef) {
	slices.SortFunc(refs, func(a, b DependencyRef) int {
		return cmp.Or(
			cmp.Compare(a.ScopeName, b.ScopeName),
			cmp.Compare(a.ServiceName, b.ServiceName),
		)
	})
}

func buildDependentsMapLocked(services map[string]*serviceRecord) map[string][]DependencyRef {
	dependents := make(map[string][]DependencyRef)

	for _, rec := range services {
		for depKey := range rec.dependencies {
			if _, ok := services[depKey]; ok {
				dependents[depKey] = append(dependents[depKey], DependencyRef{
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

	for id, names := range scopeServices {
		slices.Sort(names)
		scopeServices[id] = names
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
		Children: sortScopeNodes(build(root.id)),
	}
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

// Events returns a defensive copy of all captured events.
func (r *Recorder) Events() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return append([]Event(nil), r.events...)
}

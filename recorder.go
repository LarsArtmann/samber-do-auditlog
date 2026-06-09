package auditlog

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/samber/do/v2"
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
	serviceType     ServiceType
	registeredAt    time.Time
	firstInvokedAt  *time.Time
	invocationCount int
	invocationOrder int
	buildDurationMs *float64
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

var sequenceCounter atomic.Int64

func nextSequence() int {
	return int(sequenceCounter.Add(1))
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
}

// NewRecorder creates a new event recorder.
func NewRecorder() *Recorder {
	return &Recorder{
		events:   make([]Event, 0, 1024),
		services: make(map[string]*serviceRecord),
		scopes:   make(map[string]scopeMeta),
	}
}

func (r *Recorder) recordScope(scope *do.Scope) {
	id := scope.ID()
	r.mu.Lock()
	if _, ok := r.scopes[id]; ok {
		r.mu.Unlock()
		return
	}
	meta := scopeMeta{id: id, name: scope.Name()}
	if ancestors := scope.Ancestors(); len(ancestors) > 0 {
		meta.parentID = ancestors[0].ID()
	}
	r.scopes[id] = meta
	r.mu.Unlock()
}

func (r *Recorder) OnBeforeRegistration(scope *do.Scope, serviceName string) {
	r.recordScope(scope)
	r.addEvent(Event{
		Sequence:    nextSequence(),
		Timestamp:   time.Now(),
		EventType:   EventTypeRegistration,
		Phase:       PhaseBefore,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
	})
}

func (r *Recorder) OnAfterRegistration(scope *do.Scope, serviceName string) {
	now := time.Now()
	key := scope.ID() + "/" + serviceName
	r.mu.Lock()
	if _, ok := r.services[key]; !ok {
		r.services[key] = &serviceRecord{
			scopeID:      scope.ID(),
			scopeName:    scope.Name(),
			serviceName:  serviceName,
			serviceType:  inferServiceType(scope, serviceName),
			registeredAt: now,
			dependencies: make(map[string]struct{}),
		}
	}
	r.mu.Unlock()
	r.addEvent(Event{
		Sequence:    nextSequence(),
		Timestamp:   now,
		EventType:   EventTypeRegistration,
		Phase:       PhaseAfter,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
	})
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

	r.addEvent(Event{
		Sequence:    nextSequence(),
		Timestamp:   now,
		EventType:   EventTypeInvocation,
		Phase:       PhaseBefore,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
	})
}

func (r *Recorder) OnAfterInvocation(scope *do.Scope, serviceName string, err error) {
	now := time.Now()

	var durationMs *float64
	r.stackMu.Lock()
	for i := len(r.stack) - 1; i >= 0; i-- {
		if r.stack[i].serviceName == serviceName && r.stack[i].scopeID == scope.ID() {
			d := float64(now.Sub(r.stack[i].start).Microseconds()) / 1000.0
			durationMs = &d
			r.stack = append(r.stack[:i], r.stack[i+1:]...)
			break
		}
	}
	r.stackMu.Unlock()

	var errStr *string
	if err != nil {
		s := err.Error()
		errStr = &s
	}

	r.addEvent(Event{
		Sequence:    nextSequence(),
		Timestamp:   now,
		EventType:   EventTypeInvocation,
		Phase:       PhaseAfter,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
		DurationMs:  durationMs,
		Error:       errStr,
	})

	key := scope.ID() + "/" + serviceName
	r.mu.Lock()
	rec, ok := r.services[key]
	if !ok {
		rec = &serviceRecord{
			scopeID:      scope.ID(),
			scopeName:    scope.Name(),
			serviceName:  serviceName,
			serviceType:  inferServiceType(scope, serviceName),
			dependencies: make(map[string]struct{}),
		}
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
	if durationMs != nil {
		if rec.buildDurationMs == nil || *durationMs > *rec.buildDurationMs {
			rec.buildDurationMs = durationMs
		}
	}
	if err != nil {
		rec.invocationError = errStr
	}
	r.mu.Unlock()
}

func (r *Recorder) OnBeforeShutdown(scope *do.Scope, serviceName string) {
	r.addEvent(Event{
		Sequence:    nextSequence(),
		Timestamp:   time.Now(),
		EventType:   EventTypeShutdown,
		Phase:       PhaseBefore,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
	})
}

func (r *Recorder) OnAfterShutdown(scope *do.Scope, serviceName string, err error) {
	now := time.Now()

	var errStr *string
	if err != nil {
		s := err.Error()
		errStr = &s
	}

	r.addEvent(Event{
		Sequence:    nextSequence(),
		Timestamp:   now,
		EventType:   EventTypeShutdown,
		Phase:       PhaseAfter,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
		Error:       errStr,
	})

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

func inferServiceType(_ *do.Scope, _ string) ServiceType {
	return ServiceTypeUnknown
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
			ServiceType:     rec.serviceType,
			RegisteredAt:    rec.registeredAt,
			FirstInvokedAt:  rec.firstInvokedAt,
			InvocationCount: rec.invocationCount,
			InvocationOrder: rec.invocationOrder,
			BuildDurationMs: rec.buildDurationMs,
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

// Events returns a copy of all captured events.
func (r *Recorder) Events() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]Event(nil), r.events...)
}

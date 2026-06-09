package auditlog

import (
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
)

type stackEntry struct {
	scopeID     string
	scopeName   string
	serviceName string
	start       time.Time
}

type serviceRecord struct {
	scopeID            string
	scopeName          string
	serviceName        string
	registeredAt       time.Time
	firstInvokedAt     *time.Time
	invocationCount    int
	buildDurationMs    *float64
	dependencies       map[string]struct{}
	shutdownAt         *time.Time
	shutdownDurationMs *float64
	invocationError    *string
	shutdownError      *string
}

type scopeMeta struct {
	id       string
	name     string
	parentID string
}

// Recorder captures DI lifecycle events in-memory with minimal overhead.
type Recorder struct {
	mu       sync.RWMutex
	events   []Event
	services map[string]*serviceRecord
	scopes   map[string]scopeMeta

	stackMu sync.Mutex
	stack   []stackEntry
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
	defer r.mu.Unlock()

	if _, ok := r.scopes[id]; ok {
		return
	}

	meta := scopeMeta{id: id, name: scope.Name()}
	if ancestors := scope.Ancestors(); len(ancestors) > 0 {
		meta.parentID = ancestors[0].ID()
	}

	r.scopes[id] = meta
}

// OnBeforeRegistration implements the do hook.
func (r *Recorder) OnBeforeRegistration(scope *do.Scope, serviceName string) {
	r.recordScope(scope)

	now := time.Now()
	r.addEvent(Event{
		ID:          uuid.Must(uuid.NewRandom()).String(),
		Timestamp:   now,
		EventType:   EventTypeRegistration,
		Phase:       PhaseBefore,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
	})
}

// OnAfterRegistration implements the do hook.
func (r *Recorder) OnAfterRegistration(scope *do.Scope, serviceName string) {
	now := time.Now()
	key := scope.ID() + "/" + serviceName

	r.mu.Lock()
	if _, ok := r.services[key]; !ok {
		r.services[key] = &serviceRecord{
			scopeID:      scope.ID(),
			scopeName:    scope.Name(),
			serviceName:  serviceName,
			registeredAt: now,
			dependencies: make(map[string]struct{}),
		}
	}
	r.mu.Unlock()
	r.addEvent(Event{
		ID:          uuid.Must(uuid.NewRandom()).String(),
		Timestamp:   now,
		EventType:   EventTypeRegistration,
		Phase:       PhaseAfter,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
	})
}

// OnBeforeInvocation implements the do hook.
func (r *Recorder) OnBeforeInvocation(scope *do.Scope, serviceName string) {
	r.recordScope(scope)

	now := time.Now()

	r.stackMu.Lock()
	// If there is a parent on the invocation stack, the current service
	// is a dependency of that parent.
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
		ID:          uuid.Must(uuid.NewRandom()).String(),
		Timestamp:   now,
		EventType:   EventTypeInvocation,
		Phase:       PhaseBefore,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
	})
}

// OnAfterInvocation implements the do hook.
func (r *Recorder) OnAfterInvocation(scope *do.Scope, serviceName string, err error) {
	now := time.Now()

	var durationMs *float64

	r.stackMu.Lock()
	for i, v := range slices.Backward(r.stack) {
		if v.serviceName == serviceName && v.scopeID == scope.ID() {
			d := float64(now.Sub(v.start).Microseconds()) / 1000.0
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
		ID:          uuid.Must(uuid.NewRandom()).String(),
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
			dependencies: make(map[string]struct{}),
		}
		r.services[key] = rec
	}

	rec.invocationCount++
	if rec.firstInvokedAt == nil {
		rec.firstInvokedAt = &now
	}

	if durationMs != nil {
		// The first (build) invocation is typically the slowest.
		// Cached retrievals are near-instant, so we keep the max.
		if rec.buildDurationMs == nil || *durationMs > *rec.buildDurationMs {
			rec.buildDurationMs = durationMs
		}
	}

	if err != nil {
		rec.invocationError = errStr
	}
	r.mu.Unlock()
}

// OnBeforeShutdown implements the do hook.
func (r *Recorder) OnBeforeShutdown(scope *do.Scope, serviceName string) {
	now := time.Now()
	r.addEvent(Event{
		ID:          uuid.Must(uuid.NewRandom()).String(),
		Timestamp:   now,
		EventType:   EventTypeShutdown,
		Phase:       PhaseBefore,
		ScopeID:     scope.ID(),
		ScopeName:   scope.Name(),
		ServiceName: serviceName,
	})
}

// OnAfterShutdown implements the do hook.
func (r *Recorder) OnAfterShutdown(scope *do.Scope, serviceName string, err error) {
	now := time.Now()

	var errStr *string

	if err != nil {
		s := err.Error()
		errStr = &s
	}

	r.addEvent(Event{
		ID:          uuid.Must(uuid.NewRandom()).String(),
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

// BuildReport assembles a machine-readable Report from all captured events.
func (r *Recorder) BuildReport(containerID string) Report {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]ServiceInfo, 0, len(r.services))
	for _, rec := range r.services {
		deps := make([]string, 0, len(rec.dependencies))
		for depKey := range rec.dependencies {
			if depRec, ok := r.services[depKey]; ok {
				deps = append(deps, depRec.scopeName+"/"+depRec.serviceName)
			}
		}

		services = append(services, ServiceInfo{
			ServiceName:        rec.serviceName,
			ScopeID:            rec.scopeID,
			ScopeName:          rec.scopeName,
			RegisteredAt:       rec.registeredAt,
			FirstInvokedAt:     rec.firstInvokedAt,
			InvocationCount:    rec.invocationCount,
			BuildDurationMs:    rec.buildDurationMs,
			Dependencies:       deps,
			ShutdownAt:         rec.shutdownAt,
			ShutdownDurationMs: rec.shutdownDurationMs,
			InvocationError:    rec.invocationError,
			ShutdownError:      rec.shutdownError,
		})
	}

	scopeTree := r.buildScopeTreeLocked()

	return Report{
		ContainerID:  containerID,
		CreatedAt:    time.Now(),
		ExportedAt:   time.Now(),
		EventCount:   len(r.events),
		ServiceCount: len(services),
		Events:       append([]Event(nil), r.events...),
		Services:     services,
		ScopeTree:    scopeTree,
	}
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

	var build func(parentID string) []ScopeNode

	build = func(parentID string) []ScopeNode {
		var children []ScopeNode

		for _, meta := range r.scopes {
			if meta.parentID == parentID {
				children = append(children, ScopeNode{
					ID:       meta.id,
					Name:     meta.name,
					Children: build(meta.id),
				})
			}
		}

		return children
	}

	return ScopeNode{
		ID:       root.id,
		Name:     root.name,
		Children: build(root.id),
	}
}

// Events returns a copy of all captured events.
func (r *Recorder) Events() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return append([]Event(nil), r.events...)
}

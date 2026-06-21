package auditlog

import (
	"time"

	"github.com/samber/do/v2"
)

// scopeAncestorWalker matches any scope type that can report its ancestor
// scopes. Using an interface assertion instead of a concrete type check makes
// ResolveServiceScope robust against *do.RootScope and future wrapper types.
type scopeAncestorWalker interface {
	Ancestors() []*do.Scope
}

// RecordHealthCheck records a single health check result for a service.
func (r *Recorder) RecordHealthCheck(scopeID, scopeName, serviceName string, err error) {
	now := time.Now()
	errStr := errorToStringPtr(err)
	seq := r.nextSequence()

	ref := ServiceRef{ScopeID: scopeID, ScopeName: scopeName, ServiceName: serviceName}

	r.mu.Lock()

	key := svcKey{scopeID: scopeID, name: serviceName}

	svcType := ProviderType("")

	rec, ok := r.services[key]
	if !ok {
		rec = newServiceRecordCore(scopeID, scopeName, serviceName, "", now)
		r.services[key] = rec
	} else {
		svcType = rec.serviceType
	}

	evt := newEventFromRef(
		seq, now, EventTypeHealthCheck, PhaseAfter,
		ref, r.containerID, svcType, nil, errStr,
	)
	r.appendEventLocked(evt)

	rec.lastHealthCheckAt = &now
	rec.healthCheckError = errStr
	rec.healthCheckCount++

	r.mu.Unlock()

	r.fireEvent(evt)
}

// ResolveServiceScope finds the scope metadata for a service by name.
// Returns (scopeID, scopeName, true) if found, or ("", "", false) otherwise.
func (r *Recorder) ResolveServiceScope(injector do.Injector, serviceName string) (string, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	injectorScopeID := injector.ID()
	if rec, ok := r.services[svcKey{scopeID: injectorScopeID, name: serviceName}]; ok {
		return rec.scopeID, rec.scopeName, true
	}

	if walker, ok := injector.(scopeAncestorWalker); ok {
		for _, ancestor := range walker.Ancestors() {
			if rec, ok := r.services[svcKey{scopeID: ancestor.ID(), name: serviceName}]; ok {
				return rec.scopeID, rec.scopeName, true
			}
		}
	}

	return "", "", false
}

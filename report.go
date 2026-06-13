package auditlog

import "time"

// Report is a consolidated, machine-readable snapshot of the audit log.
type Report struct {
	Version                 string    `json:"version"`
	ContainerID             string    `json:"container_id"`
	ExportedAt              time.Time `json:"exported_at"`
	EventCount              int       `json:"event_count"`
	ServiceCount            int       `json:"service_count"`
	ScopeCount              int       `json:"scope_count"`
	TotalBuildDurationMs    float64   `json:"total_build_duration_ms"`
	TotalShutdownDurationMs float64   `json:"total_shutdown_duration_ms"`
	ShutdownSucceeded       bool      `json:"shutdown_succeeded"`
	// HealthCheckSucceeded is true when at least one service was health-checked
	// and all passed. It is false when no health checks ran (HealthCheckedCount == 0)
	// or when any service failed its check.
	HealthCheckSucceeded bool          `json:"health_check_succeeded"`
	HealthCheckedCount   int           `json:"health_checked_count"`
	Events               []Event       `json:"events,omitempty"`
	Services             []ServiceInfo `json:"services"`
	ScopeTree            ScopeNode     `json:"scope_tree"`
}

// ServiceByName returns the first ServiceInfo matching the given exact service name.
// Returns nil if no service matches. For scoped lookup, use ServiceByRef.
func (r Report) ServiceByName(name string) *ServiceInfo {
	for i := range r.Services {
		if r.Services[i].ServiceName == name {
			return &r.Services[i]
		}
	}

	return nil
}

// ServiceByRef returns the ServiceInfo matching the given scope ID and service name.
// Returns nil if no service matches.
func (r Report) ServiceByRef(scopeID, serviceName string) *ServiceInfo {
	for i := range r.Services {
		if r.Services[i].ScopeID == scopeID && r.Services[i].ServiceName == serviceName {
			return &r.Services[i]
		}
	}

	return nil
}

// ServicesByScope returns all services in the given scope.
func (r Report) ServicesByScope(scopeID string) []ServiceInfo {
	var result []ServiceInfo

	for _, s := range r.Services {
		if s.ScopeID == scopeID {
			result = append(result, s)
		}
	}

	return result
}

// EventsByService returns all events for the given service name.
func (r Report) EventsByService(serviceName string) []Event {
	var result []Event

	for _, e := range r.Events {
		if e.ServiceName == serviceName {
			result = append(result, e)
		}
	}

	return result
}

// EventsByType returns all events matching the given event type.
func (r Report) EventsByType(t EventType) []Event {
	var result []Event

	for _, e := range r.Events {
		if e.EventType == t {
			result = append(result, e)
		}
	}

	return result
}

// FailedServices returns all services with invocation or shutdown errors.
func (r Report) FailedServices() []ServiceInfo {
	var failed []ServiceInfo

	for _, s := range r.Services {
		if s.Status.IsError() {
			failed = append(failed, s)
		}
	}

	return failed
}

// UnhealthyServices returns all services with a health check error.
func (r Report) UnhealthyServices() []ServiceInfo {
	var unhealthy []ServiceInfo

	for _, s := range r.Services {
		if s.HealthCheckError != nil {
			unhealthy = append(unhealthy, s)
		}
	}

	return unhealthy
}

// EventsByRef returns all events for the given scope ID and service name.
func (r Report) EventsByRef(scopeID, serviceName string) []Event {
	var result []Event

	for _, e := range r.Events {
		if e.ScopeID == scopeID && e.ServiceName == serviceName {
			result = append(result, e)
		}
	}

	return result
}

// ReportIndex provides O(1) lookups into a Report.
// Build it once with report.Index() and reuse it for multiple queries.
type ReportIndex struct {
	ByName       map[string]*ServiceInfo
	ByRef        map[string]*ServiceInfo
	ByScope      map[string][]ServiceInfo
	EventsByName map[string][]Event
	EventsByRef  map[string][]Event
	EventsByType map[EventType][]Event
}

// Index builds a lookup index for O(1) report queries.
// Useful when performing multiple lookups on the same report.
func (r Report) Index() ReportIndex {
	idx := ReportIndex{
		ByName:       make(map[string]*ServiceInfo, len(r.Services)),
		ByRef:        make(map[string]*ServiceInfo, len(r.Services)),
		ByScope:      make(map[string][]ServiceInfo),
		EventsByName: make(map[string][]Event),
		EventsByRef:  make(map[string][]Event),
		EventsByType: make(map[EventType][]Event),
	}

	for i := range r.Services {
		svc := &r.Services[i]
		idx.ByName[svc.ServiceName] = svc
		idx.ByRef[serviceKey(svc.ScopeID, svc.ServiceName)] = svc
		idx.ByScope[svc.ScopeID] = append(idx.ByScope[svc.ScopeID], *svc)
	}

	for _, event := range r.Events {
		idx.EventsByName[event.ServiceName] = append(
			idx.EventsByName[event.ServiceName], event,
		)
		idx.EventsByRef[serviceKey(event.ScopeID, event.ServiceName)] = append(
			idx.EventsByRef[serviceKey(event.ScopeID, event.ServiceName)], event,
		)
		idx.EventsByType[event.EventType] = append(
			idx.EventsByType[event.EventType], event,
		)
	}

	return idx
}

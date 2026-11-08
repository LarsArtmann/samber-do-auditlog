package auditlog

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
)

// ServiceByName returns the first ServiceInfo matching the given exact service name.
// Returns nil if no service matches. For scoped lookup, use ServiceByRef.
func (r Report) ServiceByName(name ServiceName) *ServiceInfo {
	for i := range r.Services {
		if r.Services[i].ServiceName == name {
			return &r.Services[i]
		}
	}

	return nil
}

// ServiceByRef returns the ServiceInfo matching the given scope ID and service name.
// Returns nil if no service matches.
func (r Report) ServiceByRef(scopeID ScopeID, serviceName ServiceName) *ServiceInfo {
	for i := range r.Services {
		if r.Services[i].ScopeID == scopeID && r.Services[i].ServiceName == serviceName {
			return &r.Services[i]
		}
	}

	return nil
}

// filterInto returns the items for which keep returns true.
// The result is non-nil even when empty.
func filterInto[T any](items []T, keep func(T) bool) []T {
	out := make([]T, 0, len(items))

	for _, item := range items {
		if keep(item) {
			out = append(out, item)
		}
	}

	return out
}

// ServicesByScope returns all services in the given scope.
func (r Report) ServicesByScope(scopeID ScopeID) []ServiceInfo {
	return filterInto(r.Services, func(s ServiceInfo) bool { return s.ScopeID == scopeID })
}

// EventsByService returns all events for the given service name.
func (r Report) EventsByService(serviceName ServiceName) []Event {
	return filterInto(r.Events, func(e Event) bool { return e.ServiceName == serviceName })
}

// EventsByType returns all events matching the given event type.
func (r Report) EventsByType(t EventType) []Event {
	return filterInto(r.Events, func(e Event) bool { return e.EventType == t })
}

// FailedServices returns all services with invocation or shutdown errors.
func (r Report) FailedServices() []ServiceInfo {
	return filterInto(r.Services, func(s ServiceInfo) bool { return s.Status.IsError() })
}

// UnhealthyServices returns all services with a health check error.
func (r Report) UnhealthyServices() []ServiceInfo {
	return filterInto(r.Services, func(s ServiceInfo) bool { return s.HealthCheckError != nil })
}

// EventsByRef returns all events for the given scope ID and service name.
func (r Report) EventsByRef(scopeID ScopeID, serviceName ServiceName) []Event {
	return filterInto(r.Events, func(e Event) bool {
		return e.ScopeID == scopeID && e.ServiceName == serviceName
	})
}

// ReportIndex provides O(1) lookups into a Report.
// Build it once with report.Index() and reuse it for multiple queries.
type ReportIndex struct {
	ByName       map[ServiceName]*ServiceInfo
	ByRef        map[string]*ServiceInfo
	ByScope      map[ScopeID][]ServiceInfo
	EventsByName map[ServiceName][]Event
	EventsByRef  map[string][]Event
	EventsByType map[EventType][]Event
}

// Index builds a lookup index for O(1) report queries.
// Useful when performing multiple lookups on the same report.
func (r Report) Index() ReportIndex {
	idx := ReportIndex{
		ByName:       make(map[ServiceName]*ServiceInfo, len(r.Services)),
		ByRef:        make(map[string]*ServiceInfo, len(r.Services)),
		ByScope:      make(map[ScopeID][]ServiceInfo),
		EventsByName: make(map[ServiceName][]Event),
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

		key := serviceKey(event.ScopeID, event.ServiceName)
		idx.EventsByRef[key] = append(idx.EventsByRef[key], event)

		idx.EventsByType[event.EventType] = append(
			idx.EventsByType[event.EventType], event,
		)
	}

	return idx
}

// WriteNDJSON writes every event as a line-delimited JSON object (NDJSON).
// Operates directly on the Report.Events slice without a defensive copy,
// unlike Plugin.WriteEventsNDJSON which copies first.
func (r Report) WriteNDJSON(writer io.Writer) error {
	return writeEventsNDJSON(writer, r.Events)
}

// WriteJSON writes the full report as indented JSON to the writer.
func (r Report) WriteJSON(writer io.Writer) error {
	enc := jsontext.NewEncoder(writer, jsontext.WithIndent("  "))

	err := json.MarshalEncode(enc, r)
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}

	return nil
}

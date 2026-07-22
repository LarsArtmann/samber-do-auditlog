package auditlog

import "time"

// reportFilter holds parsed filter criteria for Report.Filtered.
type reportFilter struct {
	serviceNames map[ServiceName]struct{}
	serviceTypes map[ProviderType]struct{}
	eventTypes   map[EventType]struct{}
	scopeIDs     map[ScopeID]struct{}
	timeFrom     *time.Time
	timeTo       *time.Time
}

// ReportOption is a functional option for filtering a Report.
type ReportOption func(*reportFilter)

// WithServicesByName filters the report to only include services with the given names.
func WithServicesByName(names ...ServiceName) ReportOption {
	return func(filter *reportFilter) {
		if filter.serviceNames == nil {
			filter.serviceNames = make(map[ServiceName]struct{}, len(names))
		}

		for _, name := range names {
			filter.serviceNames[name] = struct{}{}
		}
	}
}

// WithServicesByType filters the report to only include services with the given provider type.
func WithServicesByType(providerType ProviderType) ReportOption {
	return func(filter *reportFilter) {
		if filter.serviceTypes == nil {
			filter.serviceTypes = make(map[ProviderType]struct{})
		}

		filter.serviceTypes[providerType] = struct{}{}
	}
}

// WithEventsByType filters the report to only include events with the given event type.
func WithEventsByType(eventType EventType) ReportOption {
	return func(filter *reportFilter) {
		if filter.eventTypes == nil {
			filter.eventTypes = make(map[EventType]struct{})
		}

		filter.eventTypes[eventType] = struct{}{}
	}
}

// WithTimeRange filters the report to only include events within the given time range.
func WithTimeRange(from, to time.Time) ReportOption {
	return func(filter *reportFilter) {
		filter.timeFrom = &from
		filter.timeTo = &to
	}
}

// WithScope filters the report to only include services and events in the given scope.
func WithScope(scopeID ScopeID) ReportOption {
	return func(filter *reportFilter) {
		if filter.scopeIDs == nil {
			filter.scopeIDs = make(map[ScopeID]struct{})
		}

		filter.scopeIDs[scopeID] = struct{}{}
	}
}

func newReportFilter(opts ...ReportOption) *reportFilter {
	filter := &reportFilter{} //nolint:exhaustruct

	for _, opt := range opts {
		opt(filter)
	}

	return filter
}

func (filter *reportFilter) matchService(svc ServiceInfo) bool {
	if len(filter.serviceNames) > 0 {
		if _, ok := filter.serviceNames[svc.ServiceName]; !ok {
			return false
		}
	}

	if len(filter.serviceTypes) > 0 {
		if _, ok := filter.serviceTypes[svc.ServiceType]; !ok {
			return false
		}
	}

	if len(filter.scopeIDs) > 0 {
		if _, ok := filter.scopeIDs[svc.ScopeID]; !ok {
			return false
		}
	}

	return true
}

func (filter *reportFilter) matchEvent(evt Event) bool {
	if len(filter.eventTypes) > 0 {
		if _, ok := filter.eventTypes[evt.EventType]; !ok {
			return false
		}
	}

	if len(filter.scopeIDs) > 0 {
		if _, ok := filter.scopeIDs[evt.ScopeID]; !ok {
			return false
		}
	}

	if filter.timeFrom != nil && evt.Timestamp.Before(*filter.timeFrom) {
		return false
	}

	if filter.timeTo != nil && evt.Timestamp.After(*filter.timeTo) {
		return false
	}

	return true
}

// Filtered returns a new Report with the given filters applied.
// Services and events that don't match any filter are removed.
// Summary fields (counts, durations) are recomputed from the filtered data.
// The scope tree is pruned to only include scopes with matching services.
func (r Report) Filtered(opts ...ReportOption) Report {
	filter := newReportFilter(opts...)

	filteredServices := make([]ServiceInfo, 0, len(r.Services))

	for _, svc := range r.Services {
		if filter.matchService(svc) {
			filteredServices = append(filteredServices, svc)
		}
	}

	filteredEvents := make([]Event, 0, len(r.Events))

	for _, evt := range r.Events {
		if filter.matchEvent(evt) {
			filteredEvents = append(filteredEvents, evt)
		}
	}

	scopeTree, _ := pruneScopeTree(r.ScopeTree, filteredServices)

	return buildReportFromCore(
		r.Version,
		r.ContainerID,
		r.ExportedAt,
		r.DroppedEventCount,
		filteredEvents,
		filteredServices,
		scopeTree,
	)
}

// pruneScopeTree rebuilds the scope tree from the original tree,
// keeping only nodes that have at least one service in the filtered set.
// Returns the pruned tree and the count of remaining scope nodes.
func pruneScopeTree(original ScopeNode, filteredServices []ServiceInfo) (ScopeNode, int) {
	allowed := make(map[ScopeID]map[ServiceName]struct{}, len(filteredServices))
	for _, svc := range filteredServices {
		if allowed[svc.ScopeID] == nil {
			allowed[svc.ScopeID] = make(map[ServiceName]struct{})
		}

		allowed[svc.ScopeID][svc.ServiceName] = struct{}{}
	}

	pruned, count := pruneScopeTreeRecursive(original, allowed)

	return pruned, count
}

func pruneScopeTreeRecursive(node ScopeNode, allowed map[ScopeID]map[ServiceName]struct{}) (ScopeNode, int) {
	var filteredServices []ServiceName

	if svcSet, ok := allowed[node.ID]; ok {
		for _, name := range node.Services {
			if _, has := svcSet[name]; has {
				filteredServices = append(filteredServices, name)
			}
		}
	}

	var filteredChildren []ScopeNode

	count := 0

	for _, child := range node.Children {
		prunedChild, childCount := pruneScopeTreeRecursive(child, allowed)
		if childCount > 0 {
			filteredChildren = append(filteredChildren, prunedChild)
			count += childCount
		}
	}

	if len(filteredServices) > 0 || count > 0 {
		return ScopeNode{
			ID:       node.ID,
			Name:     node.Name,
			Services: filteredServices,
			Children: filteredChildren,
		}, count + 1
	}

	return ScopeNode{ //nolint:exhaustruct
		ID:   "",
		Name: "",
	}, 0
}

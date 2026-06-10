package auditlog

import (
	"fmt"
	"time"
)

// SchemaVersion is the current report schema version.
const SchemaVersion = "0.2.0"

// RootScopeName is the canonical name for the root scope in samber/do v2.
const RootScopeName = "[root]"

// EventType categorizes audit log events.
type EventType string

const (
	EventTypeRegistration EventType = "registration"
	EventTypeInvocation   EventType = "invocation"
	EventTypeShutdown     EventType = "shutdown"
	EventTypeHealthCheck  EventType = "health_check"
)

// Phase indicates whether an event is the start or end of an operation.
type Phase string

const (
	PhaseBefore Phase = "before"
	PhaseAfter  Phase = "after"
)

// ProviderType describes how a service was registered in the DI container.
type ProviderType string

const (
	ProviderTypeLazy      ProviderType = "lazy"
	ProviderTypeEager     ProviderType = "eager"
	ProviderTypeTransient ProviderType = "transient"
	ProviderTypeAlias     ProviderType = "alias"
)

// String returns the provider type name.
func (p ProviderType) String() string { return string(p) }

// IsKnown returns true if the provider type is a recognized value.
func (p ProviderType) IsKnown() bool {
	switch p {
	case ProviderTypeLazy, ProviderTypeEager, ProviderTypeTransient, ProviderTypeAlias:
		return true
	default:
		return false
	}
}

// Icon returns the samber/do canonical emoji for this provider type.
func (p ProviderType) Icon() string {
	switch p {
	case ProviderTypeLazy:
		return "\U0001F634"
	case ProviderTypeEager:
		return "\U0001F501"
	case ProviderTypeTransient:
		return "\U0001F3ED"
	case ProviderTypeAlias:
		return "\U0001F517"
	default:
		return ""
	}
}

// ServiceStatus describes the lifecycle state of a service.
type ServiceStatus string

const (
	ServiceStatusRegistered      ServiceStatus = "registered"
	ServiceStatusActive          ServiceStatus = "active"
	ServiceStatusInvocationError ServiceStatus = "invocation_error"
	ServiceStatusShutdown        ServiceStatus = "shutdown"
	ServiceStatusShutdownError   ServiceStatus = "shutdown_error"
)

// IsError returns true if the service has an invocation or shutdown error.
func (s ServiceStatus) IsError() bool {
	return s == ServiceStatusInvocationError || s == ServiceStatusShutdownError
}

// ServiceRef identifies a service within a specific scope.
// Embedded in Event and ServiceInfo for JSON flattening.
type ServiceRef struct {
	ScopeID     string `json:"scope_id"`
	ScopeName   string `json:"scope_name"`
	ServiceName string `json:"service_name"`
}

// String returns a human-readable "scope/name" format for the service reference.
// Root scope services return just the service name.
func (r ServiceRef) String() string {
	if !r.IsRoot() {
		return fmt.Sprintf("%s/%s", r.ScopeName, r.ServiceName)
	}

	return r.ServiceName
}

// IsRoot returns true if the service belongs to the root scope.
func (r ServiceRef) IsRoot() bool {
	return r.ScopeName == "" || r.ScopeName == RootScopeName
}

// Event is a single, timestamped observation from the DI container lifecycle.
type Event struct {
	ServiceRef

	Sequence    int          `json:"sequence"`
	Timestamp   time.Time    `json:"timestamp"`
	EventType   EventType    `json:"event_type"`
	Phase       Phase        `json:"phase"`
	ServiceType ProviderType `json:"service_type,omitempty"`
	ContainerID string       `json:"container_id"`
	DurationMs  *float64     `json:"duration_ms,omitempty"`
	Error       *string      `json:"error,omitempty"`
}

// IsRegistration returns true if the event is a registration event.
func (e Event) IsRegistration() bool { return e.EventType == EventTypeRegistration }

// IsInvocation returns true if the event is an invocation event.
func (e Event) IsInvocation() bool { return e.EventType == EventTypeInvocation }

// IsShutdown returns true if the event is a shutdown event.
func (e Event) IsShutdown() bool { return e.EventType == EventTypeShutdown }

// IsHealthCheck returns true if the event is a health check event.
func (e Event) IsHealthCheck() bool { return e.EventType == EventTypeHealthCheck }

// IsBefore returns true if the event is the start (before) phase of an operation.
func (e Event) IsBefore() bool { return e.Phase == PhaseBefore }

// IsAfter returns true if the event is the end (after) phase of an operation.
func (e Event) IsAfter() bool { return e.Phase == PhaseAfter }

// HasError returns true if the event recorded an error.
func (e Event) HasError() bool { return e.Error != nil }

// Duration returns the event duration in milliseconds, or 0 if unavailable.
func (e Event) Duration() float64 {
	if e.DurationMs == nil {
		return 0
	}

	return *e.DurationMs
}

// ServiceInfo aggregates all observed data for a single service.
type ServiceInfo struct {
	ServiceRef

	Status               ServiceStatus `json:"status"`
	ServiceType          ProviderType  `json:"service_type"`
	RegisteredAt         time.Time     `json:"registered_at"`
	FirstInvokedAt       *time.Time    `json:"first_invoked_at,omitempty"`
	InvocationCount      int           `json:"invocation_count"`
	InvocationOrder      int           `json:"invocation_order"`
	FirstBuildDurationMs *float64      `json:"first_build_duration_ms,omitempty"`
	Dependencies         []ServiceRef  `json:"dependencies,omitempty"`
	Dependents           []ServiceRef  `json:"dependents,omitempty"`
	ShutdownAt           *time.Time    `json:"shutdown_at,omitempty"`
	ShutdownDurationMs   *float64      `json:"shutdown_duration_ms,omitempty"`
	ShutdownError        *string       `json:"shutdown_error,omitempty"`
	InvocationError      *string       `json:"invocation_error,omitempty"`
	IsHealthchecker      bool          `json:"is_healthchecker"`
	IsShutdowner         bool          `json:"is_shutdowner"`

	LastHealthCheckAt *time.Time `json:"last_health_check_at,omitempty"`
	HealthCheckError  *string    `json:"health_check_error,omitempty"`
	HealthCheckCount  int        `json:"health_check_count"`
}

// Uptime returns the duration since the service was registered.
func (s ServiceInfo) Uptime() time.Duration {
	return time.Since(s.RegisteredAt)
}

// HasHealthError returns true if the service has a health check error.
func (s ServiceInfo) HasHealthError() bool { return s.HealthCheckError != nil }

// ScopeNode represents the scope hierarchy for visualization.
type ScopeNode struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Services []string    `json:"services,omitempty"`
	Children []ScopeNode `json:"children,omitempty"`
}

// Report is a consolidated, machine-readable snapshot of the audit log.
type Report struct {
	Version                 string        `json:"version"`
	ContainerID             string        `json:"container_id"`
	ExportedAt              time.Time     `json:"exported_at"`
	EventCount              int           `json:"event_count"`
	ServiceCount            int           `json:"service_count"`
	ScopeCount              int           `json:"scope_count"`
	TotalBuildDurationMs    float64       `json:"total_build_duration_ms"`
	TotalShutdownDurationMs float64       `json:"total_shutdown_duration_ms"`
	ShutdownSucceeded       bool          `json:"shutdown_succeeded"`
	HealthCheckSucceeded    bool          `json:"health_check_succeeded"`
	HealthCheckedCount      int           `json:"health_checked_count"`
	Events                  []Event       `json:"events,omitempty"`
	Services                []ServiceInfo `json:"services"`
	ScopeTree               ScopeNode     `json:"scope_tree"`
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

// reportFilter holds parsed filter criteria for Report.Filtered.
type reportFilter struct {
	serviceNames map[string]struct{}
	serviceTypes map[ProviderType]struct{}
	eventTypes   map[EventType]struct{}
	scopeIDs     map[string]struct{}
	timeFrom     *time.Time
	timeTo       *time.Time
}

// ReportOption is a functional option for filtering a Report.
type ReportOption func(*reportFilter)

// WithServicesByName filters the report to only include services with the given names.
func WithServicesByName(names ...string) ReportOption {
	return func(filter *reportFilter) {
		if filter.serviceNames == nil {
			filter.serviceNames = make(map[string]struct{}, len(names))
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
func WithScope(scopeID string) ReportOption {
	return func(filter *reportFilter) {
		if filter.scopeIDs == nil {
			filter.scopeIDs = make(map[string]struct{})
		}

		filter.scopeIDs[scopeID] = struct{}{}
	}
}

func newReportFilter(opts ...ReportOption) *reportFilter {
	filter := &reportFilter{
		serviceNames: nil,
		serviceTypes: nil,
		eventTypes:   nil,
		scopeIDs:     nil,
		timeFrom:     nil,
		timeTo:       nil,
	}

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

	scopeTree, scopeCount := pruneScopeTree(r.ScopeTree, filteredServices)

	return Report{
		Version:                 r.Version,
		ContainerID:             r.ContainerID,
		ExportedAt:              r.ExportedAt,
		EventCount:              len(filteredEvents),
		ServiceCount:            len(filteredServices),
		ScopeCount:              scopeCount,
		TotalBuildDurationMs:    sumBuildMs(filteredServices),
		TotalShutdownDurationMs: sumShutdownMs(filteredServices),
		ShutdownSucceeded:       noShutdownErrors(filteredServices),
		HealthCheckSucceeded:    allHealthChecksPassed(filteredServices),
		HealthCheckedCount:      countHealthChecked(filteredServices),
		Events:                  filteredEvents,
		Services:                filteredServices,
		ScopeTree:               scopeTree,
	}
}

// pruneScopeTree rebuilds the scope tree from the original tree,
// keeping only nodes that have at least one service in the filtered set.
// Returns the pruned tree and the count of remaining scope nodes.
func pruneScopeTree(original ScopeNode, filteredServices []ServiceInfo) (ScopeNode, int) {
	allowed := make(map[string]map[string]struct{}, len(filteredServices))
	for _, svc := range filteredServices {
		if allowed[svc.ScopeID] == nil {
			allowed[svc.ScopeID] = make(map[string]struct{})
		}
		allowed[svc.ScopeID][svc.ServiceName] = struct{}{}
	}

	pruned, count := pruneScopeTreeRecursive(original, allowed)

	return pruned, count
}

func pruneScopeTreeRecursive(node ScopeNode, allowed map[string]map[string]struct{}) (ScopeNode, int) {
	var filteredServices []string
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

	return ScopeNode{}, 0
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

	for _, e := range r.Events {
		idx.EventsByName[e.ServiceName] = append(idx.EventsByName[e.ServiceName], e)
		idx.EventsByRef[serviceKey(e.ScopeID, e.ServiceName)] = append(idx.EventsByRef[serviceKey(e.ScopeID, e.ServiceName)], e)
		idx.EventsByType[e.EventType] = append(idx.EventsByType[e.EventType], e)
	}

	return idx
}

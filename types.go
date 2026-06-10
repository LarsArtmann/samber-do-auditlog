package auditlog

import (
	"fmt"
	"time"
)

// SchemaVersion is the current report schema version.
const SchemaVersion = "0.2.0"

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

func (r ServiceRef) String() string {
	if r.ScopeName != "" && r.ScopeName != "[root]" {
		return fmt.Sprintf("%s/%s", r.ScopeName, r.ServiceName)
	}

	return r.ServiceName
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

func (e Event) IsRegistration() bool { return e.EventType == EventTypeRegistration }
func (e Event) IsInvocation() bool   { return e.EventType == EventTypeInvocation }
func (e Event) IsShutdown() bool     { return e.EventType == EventTypeShutdown }
func (e Event) IsHealthCheck() bool  { return e.EventType == EventTypeHealthCheck }
func (e Event) IsBefore() bool       { return e.Phase == PhaseBefore }
func (e Event) IsAfter() bool        { return e.Phase == PhaseAfter }

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
// Returns nil if no service matches.
func (r Report) ServiceByName(name string) *ServiceInfo {
	for i := range r.Services {
		if r.Services[i].ServiceName == name {
			return &r.Services[i]
		}
	}

	return nil
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

package auditlog

import "fmt"

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

// Label returns the human-readable display label for this event type.
func (e EventType) Label() string {
	switch e {
	case EventTypeRegistration:
		return "Registration"
	case EventTypeInvocation:
		return "Invocation"
	case EventTypeShutdown:
		return "Shutdown"
	case EventTypeHealthCheck:
		return "Health"
	default:
		return ""
	}
}

// Color returns the CSS color token for this event type, used in the HTML visualization.
func (e EventType) Color() string {
	switch e {
	case EventTypeRegistration:
		return "var(--accent)"
	case EventTypeInvocation:
		return "var(--success)"
	case EventTypeShutdown:
		return "var(--warning)"
	case EventTypeHealthCheck:
		return "var(--info)"
	default:
		return ""
	}
}

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

// Label returns the human-readable display label for this provider type.
func (p ProviderType) Label() string {
	switch p {
	case ProviderTypeLazy:
		return "Lazy"
	case ProviderTypeEager:
		return "Eager"
	case ProviderTypeTransient:
		return "Transient"
	case ProviderTypeAlias:
		return "Alias"
	default:
		return ""
	}
}

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

// Icon returns the display emoji for this service status.
func (s ServiceStatus) Icon() string {
	switch s {
	case ServiceStatusRegistered:
		return "\u26AA"
	case ServiceStatusActive:
		return "\U0001F7E2"
	case ServiceStatusShutdown:
		return "\U0001F535"
	case ServiceStatusInvocationError, ServiceStatusShutdownError:
		return "\U0001F534"
	default:
		return ""
	}
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

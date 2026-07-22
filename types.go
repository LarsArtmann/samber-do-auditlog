package auditlog

import "fmt"

// SchemaVersion is the current report schema version.
const SchemaVersion = "0.2.0"

// RootScopeName is the canonical name for the root scope in samber/do v2.
const RootScopeName = "[root]"

// ContainerID identifies a DI container instance. Used in Config, Report, and Event.
type ContainerID string

// ScopeID identifies a scope within the DI hierarchy.
type ScopeID string

// ServiceName identifies a named service within a scope.
type ServiceName string

// EventType categorizes audit log events.
type EventType string

const (
	EventTypeRegistration EventType = "registration"
	EventTypeInvocation   EventType = "invocation"
	EventTypeShutdown     EventType = "shutdown"
	EventTypeHealthCheck  EventType = "health_check"
)

// eventTypeMeta pairs the human-readable label with the CSS color token.
type eventTypeMeta struct {
	label string
	color string
}

// CSS color tokens used by the HTML visualization. Centralized as constants so
// the goconst linter recognizes them as repeated literals across files.
const (
	cssColorAccent  = "var(--accent)"
	cssColorSuccess = "var(--success)"
	cssColorWarning = "var(--warning)"
	cssColorInfo    = "var(--info)"
)

// eventTypeMetaTable maps each known EventType to its display metadata.
// Read-only lookup table — treated as a constant, never mutated at runtime.
//
//nolint:gochecknoglobals // read-only enum metadata table, not mutable shared state
var eventTypeMetaTable = map[EventType]eventTypeMeta{
	EventTypeRegistration: {label: "Registration", color: cssColorAccent},
	EventTypeInvocation:   {label: "Invocation", color: cssColorSuccess},
	EventTypeShutdown:     {label: "Shutdown", color: cssColorWarning},
	EventTypeHealthCheck:  {label: "Health", color: cssColorInfo},
}

// Label returns the human-readable display label for this event type.
func (e EventType) Label() string {
	return eventTypeMetaTable[e].label
}

// IsKnown returns true if the event type is a recognized value.
func (e EventType) IsKnown() bool {
	_, ok := eventTypeMetaTable[e]

	return ok
}

// Color returns the CSS color token for this event type, used in the HTML visualization.
func (e EventType) Color() string {
	return eventTypeMetaTable[e].color
}

// Phase indicates whether an event is the start or end of an operation.
type Phase string

const (
	PhaseBefore Phase = "before"
	PhaseAfter  Phase = "after"
)

// IsKnown returns true if the phase is a recognized value.
func (p Phase) IsKnown() bool {
	return p == PhaseBefore || p == PhaseAfter
}

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

// providerMeta pairs the human-readable label with the samber/do canonical emoji.
type providerMeta struct {
	label string
	icon  string
}

// providerTypeMeta maps each known ProviderType to its display metadata.
// The metadata builder in metadata.go reads from this via BuildTypeMetadata;
// Label and Icon methods on ProviderType look up here as well.
// Read-only lookup table — treated as a constant, never mutated at runtime.
//
//nolint:gochecknoglobals // read-only enum metadata table, not mutable shared state
var providerTypeMeta = map[ProviderType]providerMeta{
	ProviderTypeLazy:      {label: "Lazy", icon: "\U0001F634"},
	ProviderTypeEager:     {label: "Eager", icon: "\U0001F501"},
	ProviderTypeTransient: {label: "Transient", icon: "\U0001F3ED"},
	ProviderTypeAlias:     {label: "Alias", icon: "\U0001F517"},
}

// Label returns the human-readable display label for this provider type.
func (p ProviderType) Label() string {
	return providerTypeMeta[p].label
}

// IsKnown returns true if the provider type is a recognized value.
func (p ProviderType) IsKnown() bool {
	_, ok := providerTypeMeta[p]

	return ok
}

// Icon returns the samber/do canonical emoji for this provider type.
func (p ProviderType) Icon() string {
	return providerTypeMeta[p].icon
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

// serviceStatusIcons maps ServiceStatus to its display emoji.
// Read-only lookup table — treated as a constant, never mutated at runtime.
//
//nolint:gochecknoglobals // read-only enum metadata table, not mutable shared state
var serviceStatusIcons = map[ServiceStatus]string{
	ServiceStatusRegistered:      "\u26AA",
	ServiceStatusActive:          "\U0001F7E2",
	ServiceStatusShutdown:        "\U0001F535",
	ServiceStatusInvocationError: "\U0001F534",
	ServiceStatusShutdownError:   "\U0001F534",
}

// IsKnown returns true if the service status is a recognized value.
func (s ServiceStatus) IsKnown() bool {
	_, ok := serviceStatusIcons[s]

	return ok
}

// Icon returns the display emoji for this service status.
func (s ServiceStatus) Icon() string {
	return serviceStatusIcons[s]
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

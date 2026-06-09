package auditlog

import "time"

// EventType categorizes audit log events.
type EventType string

const (
	EventTypeRegistration EventType = "registration"
	EventTypeInvocation   EventType = "invocation"
	EventTypeShutdown     EventType = "shutdown"
)

// Phase indicates whether an event is the start or end of an operation.
type Phase string

const (
	PhaseBefore Phase = "before"
	PhaseAfter  Phase = "after"
)

// Event is a single, timestamped observation from the DI container lifecycle.
type Event struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	EventType   EventType `json:"event_type"`
	Phase       Phase     `json:"phase"`
	ScopeID     string    `json:"scope_id"`
	ScopeName   string    `json:"scope_name"`
	ServiceName string    `json:"service_name"`
	DurationMs  *float64  `json:"duration_ms,omitempty"`
	Error       *string   `json:"error,omitempty"`
}

// ServiceInfo aggregates all observed data for a single service.
type ServiceInfo struct {
	ServiceName        string    `json:"service_name"`
	ScopeID            string    `json:"scope_id"`
	ScopeName          string    `json:"scope_name"`
	RegisteredAt       time.Time `json:"registered_at"`
	FirstInvokedAt     *time.Time `json:"first_invoked_at,omitempty"`
	InvocationCount    int       `json:"invocation_count"`
	BuildDurationMs    *float64  `json:"build_duration_ms,omitempty"`
	Dependencies       []string  `json:"dependencies,omitempty"`
	ShutdownAt         *time.Time `json:"shutdown_at,omitempty"`
	ShutdownDurationMs *float64  `json:"shutdown_duration_ms,omitempty"`
	InvocationError    *string   `json:"invocation_error,omitempty"`
	ShutdownError      *string   `json:"shutdown_error,omitempty"`
}

// ScopeNode represents the scope hierarchy for visualization.
type ScopeNode struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Children []ScopeNode `json:"children,omitempty"`
}

// Report is a consolidated, machine-readable snapshot of the audit log.
type Report struct {
	ContainerID    string        `json:"container_id"`
	CreatedAt      time.Time     `json:"created_at"`
	ExportedAt     time.Time     `json:"exported_at"`
	EventCount     int           `json:"event_count"`
	ServiceCount   int           `json:"service_count"`
	Events         []Event       `json:"events,omitempty"`
	Services       []ServiceInfo `json:"services"`
	ScopeTree      ScopeNode     `json:"scope_tree"`
}

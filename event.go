package auditlog

import "time"

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

package auditlog

import "time"

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
func (s *ServiceInfo) Uptime() time.Duration {
	return time.Since(s.RegisteredAt)
}

// HasHealthError returns true if the service has a health check error.
func (s *ServiceInfo) HasHealthError() bool { return s.HealthCheckError != nil }

// DeriveStatus computes the lifecycle status from the service's own error
// pointers and invocation/shutdown timestamps. This is the canonical
// derivation — the stored Status field should always be populated via this
// method so it can never drift from the underlying data.
func (s *ServiceInfo) DeriveStatus() ServiceStatus {
	return deriveServiceStatus(s.InvocationError, s.ShutdownError, s.ShutdownAt, s.FirstInvokedAt)
}

// RederiveStatus sets Status to the value of DeriveStatus() in place.
// Use this on *ServiceInfo to repair stale or hand-edited statuses so
// the report always passes Validate().
func (s *ServiceInfo) RederiveStatus() {
	s.Status = s.DeriveStatus()
}

// ScopeNode represents the scope hierarchy for visualization.
type ScopeNode struct {
	ID       ScopeID       `json:"id"`
	Name     string        `json:"name"`
	Services []ServiceName `json:"services,omitempty"`
	Children []ScopeNode   `json:"children,omitempty"`
}

package auditlog

import "time"

// ServiceIdentity holds the immutable identity of a service: its scope
// reference and provider type. Embedded in ServiceInfo.
type ServiceIdentity struct {
	ServiceRef

	ServiceType ProviderType `json:"service_type"`
}

// ServiceLifecycle holds the lifecycle state of a service: registration,
// invocation, and shutdown data including errors and durations. Embedded
// in ServiceInfo.
type ServiceLifecycle struct {
	Status               ServiceStatus `json:"status"`
	RegisteredAt         time.Time     `json:"registered_at"`
	FirstInvokedAt       *time.Time    `json:"first_invoked_at,omitempty"`
	InvocationCount      int           `json:"invocation_count"`
	InvocationOrder      int           `json:"invocation_order"`
	FirstBuildDurationMs *float64      `json:"first_build_duration_ms,omitempty"`
	ShutdownAt           *time.Time    `json:"shutdown_at,omitempty"`
	ShutdownDurationMs   *float64      `json:"shutdown_duration_ms,omitempty"`
	ShutdownError        *string       `json:"shutdown_error,omitempty"`
	InvocationError      *string       `json:"invocation_error,omitempty"`
	IsShutdowner         bool          `json:"is_shutdowner"`
}

// ServiceHealth holds health-check data for a service. Embedded in
// ServiceInfo.
type ServiceHealth struct {
	IsHealthchecker   bool       `json:"is_healthchecker"`
	LastHealthCheckAt *time.Time `json:"last_health_check_at,omitempty"`
	HealthCheckError  *string    `json:"health_check_error,omitempty"`
	HealthCheckCount  int        `json:"health_check_count"`
}

// ServiceGraph holds the dependency relationships of a service: which
// services it depends on and which services depend on it. Embedded in
// ServiceInfo.
type ServiceGraph struct {
	Dependencies []ServiceRef `json:"dependencies,omitempty"`
	Dependents   []ServiceRef `json:"dependents,omitempty"`
}

// ServiceInfo aggregates all observed data for a single service. The four
// embedded structs (ServiceIdentity, ServiceLifecycle, ServiceHealth,
// ServiceGraph) keep related fields grouped while preserving flat field
// access and flat JSON output via Go's struct embedding.
type ServiceInfo struct {
	ServiceIdentity
	ServiceLifecycle
	ServiceHealth
	ServiceGraph
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

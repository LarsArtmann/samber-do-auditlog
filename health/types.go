package health

// Status is the roll-up health status of a check or the overall response.
type Status string

const (
	// StatusPass means the service or system is healthy.
	StatusPass Status = "pass"
	// StatusFail means the service or system is unhealthy.
	StatusFail Status = "fail"
	// StatusWarn means the service is degraded but functional.
	// Used for non-critical service failures in readiness responses.
	StatusWarn Status = "warn"
)

// Check is the per-service health result.
type Check struct {
	// Status is the health status of this individual check.
	Status Status `json:"status"`
	// Error contains the failure message when Status is not pass.
	Error string `json:"error,omitempty"`
}

// Response is the aggregate health-check response served by all probe handlers.
type Response struct {
	// Status is the overall roll-up: fail if any critical service is down
	// or the probe is shutting down, warn if only non-critical services
	// are degraded, pass when all services are healthy.
	Status Status `json:"status"`
	// Version is the application version, if configured.
	Version string `json:"version,omitempty"`
	// Uptime is human-readable duration since boot.
	Uptime string `json:"uptime,omitempty"`
	// ShuttingDown is true when the probe has been marked for shutdown.
	// Readiness returns 503 when this is set; liveness stays 200.
	ShuttingDown bool `json:"shutting_down,omitempty"`
	// TotalLatencyMs is the wall-clock time spent running the health-check
	// batch. Populated by readiness and startup evaluations; always zero
	// for liveness (which performs no dependency checks).
	TotalLatencyMs int64 `json:"total_latency_ms,omitempty"`
	// Checks maps each service name to its individual result.
	Checks map[string]Check `json:"checks"`
}

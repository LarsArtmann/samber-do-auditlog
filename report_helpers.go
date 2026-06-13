package auditlog

import "time"

// --- Report aggregate helpers (used by Filtered and migration) ---

func sumDurationField(services []ServiceInfo, get func(ServiceInfo) *float64) float64 {
	total := 0.0

	for _, s := range services {
		if v := get(s); v != nil {
			total += *v
		}
	}

	return total
}

func sumBuildMs(services []ServiceInfo) float64 {
	return sumDurationField(services, func(s ServiceInfo) *float64 { return s.FirstBuildDurationMs })
}

func sumShutdownMs(services []ServiceInfo) float64 {
	return sumDurationField(services, func(s ServiceInfo) *float64 { return s.ShutdownDurationMs })
}

func noShutdownErrors(services []ServiceInfo) bool {
	for _, s := range services {
		if s.ShutdownError != nil {
			return false
		}
	}

	return true
}

func countHealthChecked(services []ServiceInfo) int {
	count := 0

	for _, s := range services {
		if s.HealthCheckCount > 0 {
			count++
		}
	}

	return count
}

// allHealthChecksPassed returns true only when at least one service was
// health-checked AND none failed. Returns false when no health checks
// ran — distinguish this case by checking HealthCheckedCount == 0.
func allHealthChecksPassed(services []ServiceInfo) bool {
	checked := 0

	for _, s := range services {
		if s.HealthCheckCount > 0 {
			checked++

			if s.HealthCheckError != nil {
				return false
			}
		}
	}

	return checked > 0
}

// --- Status derivation (single source of truth) ---

// deriveServiceStatus is the single source of truth for lifecycle status derivation.
// Priority: invocation_error > shutdown_error > shutdown > active > registered.
func deriveServiceStatus(invocationError, shutdownError *string, shutdownAt, firstInvokedAt *time.Time) ServiceStatus {
	if invocationError != nil {
		return ServiceStatusInvocationError
	}

	if shutdownError != nil {
		return ServiceStatusShutdownError
	}

	if shutdownAt != nil {
		return ServiceStatusShutdown
	}

	if firstInvokedAt != nil {
		return ServiceStatusActive
	}

	return ServiceStatusRegistered
}

// computeServiceStatus derives the lifecycle status from a service record.
func computeServiceStatus(rec *serviceRecord) ServiceStatus {
	return deriveServiceStatus(rec.invocationError, rec.shutdownError, rec.shutdownAt, rec.firstInvokedAt)
}

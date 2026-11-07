package auditlog

import (
	"maps"
	"slices"
	"time"

	"github.com/samber/do/v2"
)

// BuildReport assembles a machine-readable Report from all captured events.
func (r *Recorder) BuildReport() Report {
	r.mu.RLock()
	services := r.buildServicesLocked()
	scopeTree := r.buildScopeTreeLocked()
	events := append([]Event(nil), r.events...)
	scopesCopy := make(map[ScopeID]scopeMeta, len(r.scopes))
	maps.Copy(scopesCopy, r.scopes)

	r.mu.RUnlock()

	enrichCapabilities(scopesCopy, services)

	return buildReportFromCore(
		SchemaVersion,
		r.containerID,
		r.runID,
		time.Now(),
		r.droppedEvents.Load(),
		events,
		services,
		scopeTree,
	)
}

func sortServiceInfos(services []ServiceInfo) {
	slices.SortFunc(services, func(a, b ServiceInfo) int {
		return CompareServiceRefs(a.ServiceRef, b.ServiceRef)
	})
}

// buildServicesLocked assembles sorted ServiceInfo from the recorded data.
// Must be called with r.mu held for reading.
func (r *Recorder) buildServicesLocked() []ServiceInfo {
	return buildServicesFromMap(r.services)
}

// buildServicesFromMap assembles sorted ServiceInfo from a service record map.
// Shared by the live recording path (Recorder.buildServicesLocked) and the
// replay path (ReplayEvents) to guarantee identical assembly logic.
func buildServicesFromMap(services map[svcKey]*serviceRecord) []ServiceInfo {
	dependents := buildDependentsMapLocked(services)

	result := make([]ServiceInfo, 0, len(services))

	for _, rec := range services {
		deps := buildServiceDeps(rec, services)
		key := svcKey{scopeID: rec.scopeID, name: rec.serviceName}
		svcDependents := dependents[key]

		sortDepRefs(svcDependents)

		svc := serviceRecordToInfo(rec)
		svc.Dependencies = deps
		svc.Dependents = svcDependents
		result = append(result, svc)
	}

	sortServiceInfos(result)

	return result
}

// serviceRecordToInfo converts an internal serviceRecord to a public ServiceInfo.
// This centralizes the field mapping so any new field added to ServiceInfo only
// needs to be wired in one place. Dependencies, Dependents, IsHealthchecker,
// and IsShutdowner are left as zero values — the caller sets them after calling
// this function.
func serviceRecordToInfo(rec *serviceRecord) ServiceInfo {
	return ServiceInfo{
		ServiceName: rec.serviceName,
		ScopeID:     rec.scopeID,
		ScopeName:   rec.scopeName,
		ServiceType: rec.serviceType,
		Status: deriveServiceStatus(
			rec.invocationError, rec.shutdownError,
			rec.shutdownAt, rec.firstInvokedAt,
		),
		RegisteredAt:         rec.registeredAt,
		FirstInvokedAt:       rec.firstInvokedAt,
		InvocationCount:      rec.invocationCount,
		InvocationOrder:      rec.invocationOrder,
		FirstBuildDurationMs: rec.firstBuildDurationMs,
		ShutdownAt:           rec.shutdownAt,
		ShutdownDurationMs:   rec.shutdownDurationMs,
		ShutdownError:        rec.shutdownError,
		InvocationError:      rec.invocationError,
		IsShutdowner:         false,
		IsHealthchecker:      false,
		LastHealthCheckAt:    rec.lastHealthCheckAt,
		HealthCheckError:     rec.healthCheckError,
		HealthCheckCount:     rec.healthCheckCount,
		Dependencies:         nil,
		Dependents:           nil,
	}
}

// buildServiceDeps converts a serviceRecord's dependency map into a sorted
// slice of ServiceRef pointers, skipping any deps whose target service is
// missing. Pure function — usable from both the locked live path and the
// unlocked replay path.
func buildServiceDeps(rec *serviceRecord, services map[svcKey]*serviceRecord) []ServiceRef {
	if len(rec.dependencies) == 0 {
		return nil
	}

	deps := make([]ServiceRef, 0, len(rec.dependencies))
	for depKey := range rec.dependencies {
		if depRec, ok := services[depKey]; ok {
			deps = append(deps, depRecToRef(depRec))
		}
	}

	sortDepRefs(deps)

	return deps
}

// depRecToRef extracts a ServiceRef from a serviceRecord.
func depRecToRef(rec *serviceRecord) ServiceRef {
	return ServiceRef{
		ScopeID:     rec.scopeID,
		ScopeName:   rec.scopeName,
		ServiceName: rec.serviceName,
	}
}

func sortDepRefs(refs []ServiceRef) {
	slices.SortFunc(refs, CompareServiceRefs)
}

// buildDependentsMapLocked builds the reverse-dependency map.
func buildDependentsMapLocked(services map[svcKey]*serviceRecord) map[svcKey][]ServiceRef {
	dependents := make(map[svcKey][]ServiceRef)

	for _, rec := range services {
		for depKey := range rec.dependencies {
			if _, ok := services[depKey]; ok {
				dependents[depKey] = append(dependents[depKey], depRecToRef(rec))
			}
		}
	}

	return dependents
}

func enrichCapabilities(scopes map[ScopeID]scopeMeta, services []ServiceInfo) {
	// Sort scope iteration for deterministic output across runs.
	sorted := sortedScopes(scopes)

	for _, meta := range sorted {
		if meta.ref == nil {
			continue
		}

		output := do.ExplainInjector(meta.ref)
		svcMap := buildCapabilityMap(output.DAG)

		for i := range services {
			if services[i].ScopeID != meta.id {
				continue
			}

			caps, ok := svcMap[services[i].ServiceName]
			if ok {
				services[i].IsHealthchecker = caps.isHealthchecker
				services[i].IsShutdowner = caps.isShutdowner
			}
		}
	}
}

// capabilityFlags pairs the two boolean capabilities detected by
// do.ExplainInjector for a single service. A named struct replaces the
// previous [2]bool tuple so call sites are self-documenting.
type capabilityFlags struct {
	isHealthchecker bool
	isShutdowner    bool
}

func buildCapabilityMap(scopes []do.ExplainInjectorScopeOutput) map[ServiceName]capabilityFlags {
	result := make(map[ServiceName]capabilityFlags)
	queue := scopes

	for len(queue) > 0 {
		scope := queue[0]
		queue = queue[1:]

		for _, svc := range scope.Services {
			result[ServiceName(svc.ServiceName)] = capabilityFlags{
				isHealthchecker: svc.IsHealthchecker,
				isShutdowner:    svc.IsShutdowner,
			}
		}

		queue = append(queue, scope.Children...)
	}

	return result
}

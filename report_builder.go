package auditlog

import (
	"cmp"
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
	scopeCount := len(r.scopes)
	scopesCopy := make(map[string]scopeMeta, len(r.scopes))
	maps.Copy(scopesCopy, r.scopes)

	r.mu.RUnlock()

	enrichCapabilities(scopesCopy, services)

	return Report{
		Version:                 SchemaVersion,
		ContainerID:             r.containerID,
		ExportedAt:              time.Now(),
		EventCount:              len(events),
		ServiceCount:            len(services),
		ScopeCount:              scopeCount,
		TotalBuildDurationMs:    sumBuildMs(services),
		TotalShutdownDurationMs: sumShutdownMs(services),
		ShutdownSucceeded:       noShutdownErrors(services),
		HealthCheckSucceeded:    allHealthChecksPassed(services),
		HealthCheckedCount:      countHealthChecked(services),
		Events:                  events,
		Services:                services,
		ScopeTree:               scopeTree,
	}
}

// buildServicesLocked assembles sorted ServiceInfo from the recorded data.
// Must be called with r.mu held for reading.
func (r *Recorder) buildServicesLocked() []ServiceInfo {
	dependents := buildDependentsMapLocked(r.services)

	services := make([]ServiceInfo, 0, len(r.services))
	for _, rec := range r.services {
		deps := r.buildDepsLocked(rec)

		key := serviceKey(rec.scopeID, rec.serviceName)
		svcDependents := dependents[key]

		sortDepRefs(svcDependents)

		services = append(services, ServiceInfo{
			ServiceRef: ServiceRef{
				ServiceName: rec.serviceName,
				ScopeID:     rec.scopeID,
				ScopeName:   rec.scopeName,
			},
			Status:               computeServiceStatus(rec),
			ServiceType:          rec.serviceType,
			RegisteredAt:         rec.registeredAt,
			FirstInvokedAt:       rec.firstInvokedAt,
			InvocationCount:      rec.invocationCount,
			InvocationOrder:      rec.invocationOrder,
			FirstBuildDurationMs: rec.firstBuildDurationMs,
			Dependencies:         deps,
			Dependents:           svcDependents,
			ShutdownAt:           rec.shutdownAt,
			ShutdownDurationMs:   rec.shutdownDurationMs,
			ShutdownError:        rec.shutdownError,
			InvocationError:      rec.invocationError,
			IsHealthchecker:      false,
			IsShutdowner:         false,
			LastHealthCheckAt:    rec.lastHealthCheckAt,
			HealthCheckError:     rec.healthCheckError,
			HealthCheckCount:     rec.healthCheckCount,
		})
	}

	slices.SortFunc(services, func(a, b ServiceInfo) int {
		return compareByName(a.ServiceRef, b.ServiceRef)
	})

	return services
}

// buildDepsLocked builds sorted dependency refs for a service record.
// Must be called with r.mu held for reading.
func (r *Recorder) buildDepsLocked(rec *serviceRecord) []ServiceRef {
	if len(rec.dependencies) == 0 {
		return nil
	}

	deps := make([]ServiceRef, 0, len(rec.dependencies))
	for depKey := range rec.dependencies {
		if depRec, ok := r.services[depKey]; ok {
			deps = append(deps, ServiceRef{
				ScopeID:     depRec.scopeID,
				ScopeName:   depRec.scopeName,
				ServiceName: depRec.serviceName,
			})
		}
	}

	sortDepRefs(deps)

	return deps
}

func sortDepRefs(refs []ServiceRef) {
	slices.SortFunc(refs, compareByName)
}

func compareByName(a, b ServiceRef) int {
	return cmp.Or(
		cmp.Compare(a.ScopeName, b.ScopeName),
		cmp.Compare(a.ServiceName, b.ServiceName),
	)
}

func buildDependentsMapLocked(services map[string]*serviceRecord) map[string][]ServiceRef {
	dependents := make(map[string][]ServiceRef)

	for _, rec := range services {
		for depKey := range rec.dependencies {
			if _, ok := services[depKey]; ok {
				dependents[depKey] = append(dependents[depKey], ServiceRef{
					ScopeID:     rec.scopeID,
					ScopeName:   rec.scopeName,
					ServiceName: rec.serviceName,
				})
			}
		}
	}

	return dependents
}

func (r *Recorder) buildScopeTreeLocked() ScopeNode {
	sortedScopes := sortedScopesLocked(r.scopes)

	var root scopeMeta

	hasRoot := false

	for _, meta := range sortedScopes {
		if meta.parentID == "" {
			root = meta
			hasRoot = true

			break
		}
	}

	if !hasRoot && len(sortedScopes) > 0 {
		root = sortedScopes[0]
	}

	scopeServices := make(map[string][]string)
	for _, rec := range r.services {
		scopeServices[rec.scopeID] = append(scopeServices[rec.scopeID], rec.serviceName)
	}

	for id, names := range scopeServices {
		slices.Sort(names)
		scopeServices[id] = names
	}

	var build func(parentID string) []ScopeNode

	build = func(parentID string) []ScopeNode {
		var children []ScopeNode

		for _, meta := range sortedScopes {
			if meta.parentID == parentID {
				children = append(children, ScopeNode{
					ID:       meta.id,
					Name:     meta.name,
					Services: scopeServices[meta.id],
					Children: build(meta.id),
				})
			}
		}

		return children
	}

	return ScopeNode{
		ID:       root.id,
		Name:     root.name,
		Services: scopeServices[root.id],
		Children: sortScopeNodes(build(root.id)),
	}
}

func sortedScopesLocked(scopes map[string]scopeMeta) []scopeMeta {
	result := make([]scopeMeta, 0, len(scopes))

	for _, meta := range scopes {
		result = append(result, meta)
	}

	slices.SortFunc(result, func(a, b scopeMeta) int {
		return cmp.Compare(a.id, b.id)
	})

	return result
}

func sortScopeNodes(nodes []ScopeNode) []ScopeNode {
	slices.SortFunc(nodes, func(a, b ScopeNode) int {
		return cmp.Compare(a.Name, b.Name)
	})

	for i := range nodes {
		nodes[i].Children = sortScopeNodes(nodes[i].Children)
	}

	return nodes
}


// --- Capability enrichment (samber/do ExplainInjector) ---

// enrichCapabilities populates IsHealthchecker and IsShutdowner on each ServiceInfo
// by calling do.ExplainInjector on each stored scope reference. Must be called
// outside the recorder mutex to avoid deadlocking with samber/do's internal locks.
func enrichCapabilities(scopes map[string]scopeMeta, services []ServiceInfo) {
	// Sort scope iteration for deterministic output across runs.
	sorted := make([]scopeMeta, 0, len(scopes))
	for _, meta := range scopes {
		sorted = append(sorted, meta)
	}

	slices.SortFunc(sorted, func(a, b scopeMeta) int {
		return cmp.Compare(a.id, b.id)
	})

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
				services[i].IsHealthchecker = caps[0]
				services[i].IsShutdowner = caps[1]
			}
		}
	}
}

func buildCapabilityMap(scopes []do.ExplainInjectorScopeOutput) map[string][2]bool {
	result := make(map[string][2]bool)
	queue := scopes

	for len(queue) > 0 {
		scope := queue[0]
		queue = queue[1:]

		for _, svc := range scope.Services {
			result[svc.ServiceName] = [2]bool{svc.IsHealthchecker, svc.IsShutdowner}
		}

		queue = append(queue, scope.Children...)
	}

	return result
}

// Ensure time import is used (for BuildReport's time.Now call).
var _ = time.Now

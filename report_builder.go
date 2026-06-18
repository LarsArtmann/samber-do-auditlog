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
	scopesCopy := make(map[string]scopeMeta, len(r.scopes))
	maps.Copy(scopesCopy, r.scopes)

	r.mu.RUnlock()

	enrichCapabilities(scopesCopy, services)

	return buildReportFromCore(
		SchemaVersion,
		r.containerID,
		time.Now(),
		r.droppedEvents.Load(),
		events,
		services,
		scopeTree,
	)
}

func sortServiceInfos(services []ServiceInfo) {
	slices.SortFunc(services, func(a, b ServiceInfo) int {
		return compareByName(a.ServiceRef, b.ServiceRef)
	})
}

// buildServicesLocked assembles sorted ServiceInfo from the recorded data.
// Must be called with r.mu held for reading.
func (r *Recorder) buildServicesLocked() []ServiceInfo {
	dependents := buildDependentsMapLocked(r.services)

	services := make([]ServiceInfo, 0, len(r.services))
	for _, rec := range r.services {
		deps := r.buildDepsLocked(rec)

		key := svcKey{scopeID: rec.scopeID, name: rec.serviceName}
		svcDependents := dependents[key]

		sortDepRefs(svcDependents)

		svc := serviceRecordToInfo(rec)
		svc.Dependencies = deps
		svc.Dependents = svcDependents
		services = append(services, svc)
	}

	sortServiceInfos(services)

	return services
}

// serviceRecordToInfo converts an internal serviceRecord to a public ServiceInfo.
// This centralizes the field mapping so any new field added to ServiceInfo only
// needs to be wired in one place. Dependencies, Dependents, IsHealthchecker,
// and IsShutdowner are left as zero values — the caller sets them after calling
// this function.
func serviceRecordToInfo(rec *serviceRecord) ServiceInfo {
	return ServiceInfo{
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
		Dependencies:         nil,
		Dependents:           nil,
		ShutdownAt:           rec.shutdownAt,
		ShutdownDurationMs:   rec.shutdownDurationMs,
		ShutdownError:        rec.shutdownError,
		InvocationError:      rec.invocationError,
		IsHealthchecker:      false,
		IsShutdowner:         false,
		LastHealthCheckAt:    rec.lastHealthCheckAt,
		HealthCheckError:     rec.healthCheckError,
		HealthCheckCount:     rec.healthCheckCount,
	}
}

// buildDepsLocked builds sorted dependency refs for a service record.
// Must be called with r.mu held for reading.
func (r *Recorder) buildDepsLocked(rec *serviceRecord) []ServiceRef {
	return buildServiceDeps(rec, r.services)
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
	slices.SortFunc(refs, compareByName)
}

func compareByName(a, b ServiceRef) int {
	return cmp.Or(
		cmp.Compare(a.ScopeName, b.ScopeName),
		cmp.Compare(a.ServiceName, b.ServiceName),
	)
}

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

func (r *Recorder) buildScopeTreeLocked() ScopeNode {
	sortedScopes := sortedScopesLocked(r.scopes)

	return buildScopeTreeFromMeta(
		sortedScopes,
		scopeMetaID, scopeMetaName, scopeMetaParentID,
		scopeServicesForServices(r.services),
	)
}

// scopeMetaID, scopeMetaName, scopeMetaParentID are field accessors for scopeMeta.
func scopeMetaID(m scopeMeta) string       { return m.id }
func scopeMetaName(m scopeMeta) string     { return m.name }
func scopeMetaParentID(m scopeMeta) string { return m.parentID }

// scopeServicesForServices groups service names by their scopeID.
func scopeServicesForServices(services map[svcKey]*serviceRecord) map[string][]string {
	scopeServices := make(map[string][]string)
	for _, rec := range services {
		scopeServices[rec.scopeID] = append(scopeServices[rec.scopeID], rec.serviceName)
	}

	for id, names := range scopeServices {
		slices.Sort(names)
		scopeServices[id] = names
	}

	return scopeServices
}

// findRootScope returns the first meta with an empty parentID, or false
// if none found. Sorted iteration keeps the result deterministic.
func findRootScope[T any](sorted []T, parentOf func(T) string) (T, bool) {
	for _, meta := range sorted {
		if parentOf(meta) == "" {
			return meta, true
		}
	}

	var zero T

	return zero, false
}

// buildScopeChildren constructs the child scope tree below parentID. The
// cycle guard (metaID(meta) != parentID) prevents infinite recursion on
// self-referential entries where both IDs are empty.
func buildScopeChildren[T any](
	parentID string,
	sorted []T,
	metaID, metaName, metaParent func(T) string,
	scopeServices map[string][]string,
) []ScopeNode {
	var children []ScopeNode

	for _, meta := range sorted {
		if metaParent(meta) != parentID {
			continue
		}

		if metaID(meta) == parentID {
			continue
		}

		id := metaID(meta)

		children = append(children, ScopeNode{
			ID:       id,
			Name:     metaName(meta),
			Services: scopeServices[id],
			Children: buildScopeChildren(id, sorted, metaID, metaName, metaParent, scopeServices),
		})
	}

	return children
}

// buildScopeTreeFromMeta assembles a ScopeNode tree from sorted scope
// metadata using the provided field accessors. The first scope with an
// empty parentID is the root; remaining scopes become children of
// whichever scope matches their parentID.
func buildScopeTreeFromMeta[T any](
	sorted []T,
	metaID, metaName, metaParent func(T) string,
	scopeServices map[string][]string,
) ScopeNode {
	if len(sorted) == 0 {
		return ScopeNode{} //nolint:exhaustruct
	}

	root, ok := findRootScope(sorted, metaParent)
	if !ok {
		root = sorted[0]
	}

	id := metaID(root)

	return ScopeNode{
		ID:       id,
		Name:     metaName(root),
		Services: scopeServices[id],
		Children: sortScopeNodes(buildScopeChildren(id, sorted, metaID, metaName, metaParent, scopeServices)),
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
	sorted := sortedScopesLocked(scopes)

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

func buildCapabilityMap(scopes []do.ExplainInjectorScopeOutput) map[string]capabilityFlags {
	result := make(map[string]capabilityFlags)
	queue := scopes

	for len(queue) > 0 {
		scope := queue[0]
		queue = queue[1:]

		for _, svc := range scope.Services {
			result[svc.ServiceName] = capabilityFlags{
				isHealthchecker: svc.IsHealthchecker,
				isShutdowner:    svc.IsShutdowner,
			}
		}

		queue = append(queue, scope.Children...)
	}

	return result
}

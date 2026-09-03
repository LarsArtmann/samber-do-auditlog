package auditlog

import (
	"sort"
)

// DiffResult describes the differences between two Reports.
// All slices are nil when empty (no allocation for identical reports).
type DiffResult struct {
	// AddedServices are services present in `other` but not in `r`.
	AddedServices []ServiceRef `json:"added_services,omitempty"`
	// RemovedServices are services present in `r` but not in `other`.
	RemovedServices []ServiceRef `json:"removed_services,omitempty"`
	// ChangedServices are services present in both with different fields.
	ChangedServices []ServiceDiff `json:"changed_services,omitempty"`
	// EventCountDelta is other.EventCount - r.EventCount.
	EventCountDelta int `json:"event_count_delta"`
	// TotalBuildDurationMsDelta is other.TotalBuildDurationMs - r.TotalBuildDurationMs.
	TotalBuildDurationMsDelta float64 `json:"total_build_duration_ms_delta"`
	// TotalShutdownDurationMsDelta is other.TotalShutdownDurationMs - r.TotalShutdownDurationMs.
	TotalShutdownDurationMsDelta float64 `json:"total_shutdown_duration_ms_delta"`
}

// ServiceDiff describes changes to a service that exists in both reports.
type ServiceDiff struct {
	ServiceRef

	StatusChanged         bool `json:"status_changed"`
	InvocationCountDelta  int  `json:"invocation_count_delta"`
	HealthCheckCountDelta int  `json:"health_check_count_delta"`
	HasNewError           bool `json:"has_new_error"`
	// AddedDeps are dependency edges present in `other` but not in `r`.
	AddedDeps []ServiceRef `json:"added_deps,omitempty"`
	// RemovedDeps are dependency edges present in `r` but not in `other`.
	RemovedDeps []ServiceRef `json:"removed_deps,omitempty"`
}

// IsEmpty returns true when no differences were found.
func (d DiffResult) IsEmpty() bool {
	return len(d.AddedServices) == 0 &&
		len(d.RemovedServices) == 0 &&
		len(d.ChangedServices) == 0 &&
		d.EventCountDelta == 0 &&
		d.TotalBuildDurationMsDelta == 0 &&
		d.TotalShutdownDurationMsDelta == 0
}

// HasChanges returns true if the diff found any differences.
// This is the logical inverse of IsEmpty, provided for parity with the
// go-workflow-auditlog twin API.
func (d DiffResult) HasChanges() bool {
	return !d.IsEmpty()
}

// Diff compares this report with another and returns the structural and
// status differences. Useful for regression-testing DI graphs across deploys.
//
// The comparison key is (scope_id, service_name). Timestamps and durations are
// intentionally ignored — reported changes are added/removed services, status
// transitions, invocation/health-count deltas, error appearances, and
// per-service dependency-edge changes (ServiceDiff.AddedDeps/RemovedDeps).
func (r Report) Diff(other Report) DiffResult {
	result := DiffResult{
		AddedServices:                nil,
		RemovedServices:              nil,
		ChangedServices:              nil,
		EventCountDelta:              other.EventCount - r.EventCount,
		TotalBuildDurationMsDelta:    other.TotalBuildDurationMs - r.TotalBuildDurationMs,
		TotalShutdownDurationMsDelta: other.TotalShutdownDurationMs - r.TotalShutdownDurationMs,
	}

	rByID := indexServicesByKey(r.Services)
	otherByID := indexServicesByKey(other.Services)

	for key, prevSvc := range rByID {
		otherSvc, exists := otherByID[key]
		if !exists {
			result.RemovedServices = append(result.RemovedServices, prevSvc.ServiceRef)

			continue
		}

		diff, changed := compareService(prevSvc, otherSvc)
		if changed {
			result.ChangedServices = append(result.ChangedServices, diff)
		}
	}

	for key, otherSvc := range otherByID {
		if _, exists := rByID[key]; !exists {
			result.AddedServices = append(result.AddedServices, otherSvc.ServiceRef)
		}
	}

	sortDepRefs(result.AddedServices)
	sortDepRefs(result.RemovedServices)
	sort.Slice(result.ChangedServices, func(i, j int) bool {
		return sortServiceDiffs(result.ChangedServices[i], result.ChangedServices[j]) < 0
	})

	return result
}

func compareService(prev, other ServiceInfo) (ServiceDiff, bool) {
	prevDeps := serviceRefSet(prev.Dependencies)
	otherDeps := serviceRefSet(other.Dependencies)

	diff := ServiceDiff{
		ServiceRef:            prev.ServiceRef,
		StatusChanged:         prev.Status != other.Status,
		InvocationCountDelta:  other.InvocationCount - prev.InvocationCount,
		HealthCheckCountDelta: other.HealthCheckCount - prev.HealthCheckCount,
		HasNewError:           !prev.Status.IsError() && other.Status.IsError(),
		AddedDeps:             serviceRefsOnlyIn(otherDeps, prevDeps),
		RemovedDeps:           serviceRefsOnlyIn(prevDeps, otherDeps),
	}

	changed := diff.StatusChanged ||
		diff.InvocationCountDelta != 0 ||
		diff.HealthCheckCountDelta != 0 ||
		diff.HasNewError ||
		len(diff.AddedDeps) > 0 ||
		len(diff.RemovedDeps) > 0

	return diff, changed
}

// serviceRefSet indexes ServiceRefs by the canonical (scope_id, service_name)
// key used throughout the diff logic.
func serviceRefSet(refs []ServiceRef) map[string]ServiceRef {
	set := make(map[string]ServiceRef, len(refs))

	for _, ref := range refs {
		set[serviceKey(ref.ScopeID, ref.ServiceName)] = ref
	}

	return set
}

// serviceRefsOnlyIn returns the refs present in `a` but not in `b`, sorted
// with the canonical ServiceRef ordering. Returns nil when the difference is
// empty so JSON output stays lean.
func serviceRefsOnlyIn(a, b map[string]ServiceRef) []ServiceRef {
	var out []ServiceRef

	for key, ref := range a {
		if _, exists := b[key]; !exists {
			out = append(out, ref)
		}
	}

	sortDepRefs(out)

	return out
}

func indexServicesByKey(services []ServiceInfo) map[string]ServiceInfo {
	idx := make(map[string]ServiceInfo, len(services))

	for _, svc := range services {
		idx[serviceKey(svc.ScopeID, svc.ServiceName)] = svc
	}

	return idx
}

// CompareServiceRefs is the canonical sort ordering for ServiceRef slices:
// primary by ServiceName, secondary by ScopeID. Used by report builders and
// diff output so all ServiceRef lists are consistently ordered.
func CompareServiceRefs(a, b ServiceRef) int {
	if a.ServiceName != b.ServiceName {
		if a.ServiceName < b.ServiceName {
			return -1
		}

		return 1
	}

	if a.ScopeID != b.ScopeID {
		if a.ScopeID < b.ScopeID {
			return -1
		}

		return 1
	}

	return 0
}

func sortServiceDiffs(a, b ServiceDiff) int {
	return CompareServiceRefs(a.ServiceRef, b.ServiceRef)
}

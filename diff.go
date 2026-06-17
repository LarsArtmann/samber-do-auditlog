package auditlog

import (
	"cmp"
	"slices"
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
}

// ServiceDiff describes changes to a service that exists in both reports.
type ServiceDiff struct {
	ServiceRef

	StatusChanged         bool `json:"status_changed"`
	InvocationCountDelta  int  `json:"invocation_count_delta"`
	HealthCheckCountDelta int  `json:"health_check_count_delta"`
	HasNewError           bool `json:"has_new_error"`
}

// IsEmpty returns true when no differences were found.
func (d DiffResult) IsEmpty() bool {
	return len(d.AddedServices) == 0 &&
		len(d.RemovedServices) == 0 &&
		len(d.ChangedServices) == 0 &&
		d.EventCountDelta == 0
}

// Diff compares this report with another and returns the structural and
// status differences. Useful for regression-testing DI graphs across deploys.
//
// The comparison key is (scope_id, service_name). Timestamps and durations are
// intentionally ignored — only structural changes (added/removed services,
// dependency edges, status transitions, error appearances) are reported.
func (r Report) Diff(other Report) DiffResult {
	result := DiffResult{
		AddedServices:   nil,
		RemovedServices: nil,
		ChangedServices: nil,
		EventCountDelta: other.EventCount - r.EventCount,
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

	slices.SortFunc(result.AddedServices, sortServiceRefs)
	slices.SortFunc(result.RemovedServices, sortServiceRefs)
	slices.SortFunc(result.ChangedServices, sortServiceDiffs)

	return result
}

func compareService(prev, other ServiceInfo) (ServiceDiff, bool) {
	diff := ServiceDiff{
		ServiceRef:            prev.ServiceRef,
		StatusChanged:         prev.Status != other.Status,
		InvocationCountDelta:  other.InvocationCount - prev.InvocationCount,
		HealthCheckCountDelta: other.HealthCheckCount - prev.HealthCheckCount,
		HasNewError:           !hasError(prev) && hasError(other),
	}

	changed := diff.StatusChanged ||
		diff.InvocationCountDelta != 0 ||
		diff.HealthCheckCountDelta != 0 ||
		diff.HasNewError

	return diff, changed
}

func hasError(svc ServiceInfo) bool {
	return svc.InvocationError != nil || svc.ShutdownError != nil
}

func indexServicesByKey(services []ServiceInfo) map[string]ServiceInfo {
	idx := make(map[string]ServiceInfo, len(services))

	for _, svc := range services {
		idx[serviceKey(svc.ScopeID, svc.ServiceName)] = svc
	}

	return idx
}

func sortServiceRefs(a, b ServiceRef) int {
	if c := cmp.Compare(a.ServiceName, b.ServiceName); c != 0 {
		return c
	}

	return cmp.Compare(a.ScopeID, b.ScopeID)
}

func sortServiceDiffs(a, b ServiceDiff) int {
	return sortServiceRefs(a.ServiceRef, b.ServiceRef)
}

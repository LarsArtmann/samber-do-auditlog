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
// intentionally ignored — only structural changes (added/removed services,
// dependency edges, status transitions, error appearances) are reported.
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

	slices.SortFunc(result.AddedServices, CompareServiceRefs)
	slices.SortFunc(result.RemovedServices, CompareServiceRefs)
	slices.SortFunc(result.ChangedServices, sortServiceDiffs)

	return result
}

func compareService(prev, other ServiceInfo) (ServiceDiff, bool) {
	diff := ServiceDiff{
		ServiceRef:            prev.ServiceRef,
		StatusChanged:         prev.Status != other.Status,
		InvocationCountDelta:  other.InvocationCount - prev.InvocationCount,
		HealthCheckCountDelta: other.HealthCheckCount - prev.HealthCheckCount,
		HasNewError:           !prev.Status.IsError() && other.Status.IsError(),
	}

	changed := diff.StatusChanged ||
		diff.InvocationCountDelta != 0 ||
		diff.HealthCheckCountDelta != 0 ||
		diff.HasNewError

	return diff, changed
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
	return cmp.Or(
		cmp.Compare(a.ServiceName, b.ServiceName),
		cmp.Compare(a.ScopeID, b.ScopeID),
	)
}

func sortServiceDiffs(a, b ServiceDiff) int {
	return CompareServiceRefs(a.ServiceRef, b.ServiceRef)
}

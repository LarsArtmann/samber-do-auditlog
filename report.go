package auditlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

var (
	errReportEventCountMismatch    = errors.New("event_count does not match len(events)")
	errReportServiceCountMismatch  = errors.New("service_count does not match len(services)")
	errReportScopeCountMismatch    = errors.New("scope_count does not match scope tree")
	errReportHealthCheckedMismatch = errors.New("health_checked_count does not match services with health checks")
	errReportStatusDrift           = errors.New("service status does not match derived status")
)

// Report is a consolidated, machine-readable snapshot of the audit log.
type Report struct {
	Version                 string    `json:"version"`
	ContainerID             string    `json:"container_id"`
	ExportedAt              time.Time `json:"exported_at"`
	EventCount              int       `json:"event_count"`
	ServiceCount            int       `json:"service_count"`
	ScopeCount              int       `json:"scope_count"`
	TotalBuildDurationMs    float64   `json:"total_build_duration_ms"`
	TotalShutdownDurationMs float64   `json:"total_shutdown_duration_ms"`
	ShutdownSucceeded       bool      `json:"shutdown_succeeded"`
	// HealthCheckSucceeded is true when at least one service was health-checked
	// and all passed. It is false when no health checks ran (HealthCheckedCount == 0)
	// or when any service failed its check.
	HealthCheckSucceeded bool `json:"health_check_succeeded"`
	HealthCheckedCount   int  `json:"health_checked_count"`
	// DroppedEventCount is the number of events dropped due to Config.MaxEvents.
	// Always 0 when MaxEvents is 0 (unlimited).
	DroppedEventCount int64 `json:"dropped_event_count"`
	// Reconstructed is true when the report was built by ReplayEvents from a
	// flat event stream rather than from live container hooks. Capability
	// flags (IsHealthchecker, IsShutdowner) are always false on reconstructed
	// reports, and the scope tree may be flattened.
	Reconstructed bool          `json:"reconstructed,omitempty"`
	Events        []Event       `json:"events,omitempty"`
	Services      []ServiceInfo `json:"services"`
	ScopeTree     ScopeNode     `json:"scope_tree"`
}

// Validate checks internal consistency of the report: denormalized count fields
// must match the actual slice/tree lengths. Returns nil if consistent, or an
// error describing the first discrepancy found.
func (r Report) Validate() error {
	if r.EventCount != len(r.Events) {
		return fmt.Errorf("%w: got %d, want %d", errReportEventCountMismatch, r.EventCount, len(r.Events))
	}

	if r.ServiceCount != len(r.Services) {
		return fmt.Errorf("%w: got %d, want %d", errReportServiceCountMismatch, r.ServiceCount, len(r.Services))
	}

	treeLen := countScopeNodes(r.ScopeTree)
	if r.ScopeCount != treeLen {
		return fmt.Errorf("%w: got %d, want %d", errReportScopeCountMismatch, r.ScopeCount, treeLen)
	}

	healthChecked := 0

	for _, svc := range r.Services {
		if svc.HealthCheckCount > 0 {
			healthChecked++
		}
	}

	if r.HealthCheckedCount != healthChecked {
		return fmt.Errorf("%w: got %d, want %d", errReportHealthCheckedMismatch, r.HealthCheckedCount, healthChecked)
	}

	for _, svc := range r.Services {
		derived := svc.DeriveStatus()
		if svc.Status != derived {
			return fmt.Errorf("%w: service %q has status %q but derived status is %q",
				errReportStatusDrift, svc.ServiceName, svc.Status, derived)
		}
	}

	return nil
}

// finalizeDenormalized recomputes all aggregate count and summary fields on the
// report from its core data (Events, Services, ScopeTree). Called after any
// mutation to those core slices so the report stays self-consistent and passes
// Validate(). This is the single source of truth for denormalized fields.
func finalizeDenormalized(report *Report) {
	report.EventCount = len(report.Events)
	report.ServiceCount = len(report.Services)
	report.ScopeCount = countScopeNodes(report.ScopeTree)
	report.TotalBuildDurationMs = sumBuildMs(report.Services)
	report.TotalShutdownDurationMs = sumShutdownMs(report.Services)
	report.ShutdownSucceeded = noShutdownErrors(report.Services)
	report.HealthCheckSucceeded = allHealthChecksPassed(report.Services)
	report.HealthCheckedCount = countHealthChecked(report.Services)
}

// buildReportFromCore assembles a Report from the immutable identity/metadata
// fields and the core data slices, deriving every denormalized aggregate from
// them. This is the single construction path shared by BuildReport, Filtered,
// and MigrateReport — guaranteeing that count/summary fields can never drift
// from the underlying data.
func buildReportFromCore(
	version, containerID string,
	exportedAt time.Time,
	droppedEventCount int64,
	events []Event,
	services []ServiceInfo,
	scopeTree ScopeNode,
) Report {
	report := Report{ //nolint:exhaustruct
		Version:           version,
		ContainerID:       containerID,
		ExportedAt:        exportedAt,
		DroppedEventCount: droppedEventCount,
		Events:            events,
		Services:          services,
		ScopeTree:         scopeTree,
	}
	finalizeDenormalized(&report)

	return report
}

// NewReport constructs a validated Report from its core data: the immutable
// identity/metadata fields and the three data slices (events, services,
// scope tree). It is the public, validated counterpart to the internal
// buildReportFromCore path used by BuildReport, Filtered, MigrateReport and
// ReplayEvents.
//
// Per-service Status is re-derived from the underlying error/timestamp fields
// (via ServiceInfo.DeriveStatus), so callers never need to compute Status by
// hand and cannot construct a report whose Status would drift. All denormalized
// aggregate fields (counts, durations, health flags) are derived from the data.
//
// Returns an error if the inputs are structurally inconsistent in a way that
// cannot be repaired by re-derivation.
func NewReport(
	version, containerID string,
	exportedAt time.Time,
	events []Event,
	services []ServiceInfo,
	scopeTree ScopeNode,
) (Report, error) {
	// Re-derive per-service Status so the report is always self-consistent.
	for idx := range services {
		services[idx].Status = services[idx].DeriveStatus()
	}

	report := buildReportFromCore(
		version, containerID, exportedAt, 0, events, services, scopeTree,
	)

	err := report.Validate()
	if err != nil {
		return Report{}, fmt.Errorf("new report for containerID=%q version=%q exportedAt=%v: %w",
			containerID, version, exportedAt, err)
	}

	return report, nil
}

// countScopeNodes counts all real scope nodes in the tree (root + recursive children).
// A zero-value root (empty ID+Name, no children) counts as 0 — it represents
// an empty report where buildScopeTreeLocked returns a default ScopeNode.
func countScopeNodes(node ScopeNode) int {
	if node.ID == "" && node.Name == "" && len(node.Children) == 0 {
		return 0
	}

	count := 1

	for _, child := range node.Children {
		count += countScopeNodes(child)
	}

	return count
}

// ServiceByName returns the first ServiceInfo matching the given exact service name.
// Returns nil if no service matches. For scoped lookup, use ServiceByRef.
func (r Report) ServiceByName(name string) *ServiceInfo {
	for i := range r.Services {
		if r.Services[i].ServiceName == name {
			return &r.Services[i]
		}
	}

	return nil
}

// ServiceByRef returns the ServiceInfo matching the given scope ID and service name.
// Returns nil if no service matches.
func (r Report) ServiceByRef(scopeID, serviceName string) *ServiceInfo {
	for i := range r.Services {
		if r.Services[i].ScopeID == scopeID && r.Services[i].ServiceName == serviceName {
			return &r.Services[i]
		}
	}

	return nil
}

// ServicesByScope returns all services in the given scope.
func (r Report) ServicesByScope(scopeID string) []ServiceInfo {
	var result []ServiceInfo

	for _, s := range r.Services {
		if s.ScopeID == scopeID {
			result = append(result, s)
		}
	}

	return result
}

// EventsByService returns all events for the given service name.
func (r Report) EventsByService(serviceName string) []Event {
	var result []Event

	for _, e := range r.Events {
		if e.ServiceName == serviceName {
			result = append(result, e)
		}
	}

	return result
}

// EventsByType returns all events matching the given event type.
func (r Report) EventsByType(t EventType) []Event {
	var result []Event

	for _, e := range r.Events {
		if e.EventType == t {
			result = append(result, e)
		}
	}

	return result
}

// FailedServices returns all services with invocation or shutdown errors.
func (r Report) FailedServices() []ServiceInfo {
	var failed []ServiceInfo

	for _, s := range r.Services {
		if s.Status.IsError() {
			failed = append(failed, s)
		}
	}

	return failed
}

// UnhealthyServices returns all services with a health check error.
func (r Report) UnhealthyServices() []ServiceInfo {
	var unhealthy []ServiceInfo

	for _, s := range r.Services {
		if s.HealthCheckError != nil {
			unhealthy = append(unhealthy, s)
		}
	}

	return unhealthy
}

// EventsByRef returns all events for the given scope ID and service name.
func (r Report) EventsByRef(scopeID, serviceName string) []Event {
	var result []Event

	for _, e := range r.Events {
		if e.ScopeID == scopeID && e.ServiceName == serviceName {
			result = append(result, e)
		}
	}

	return result
}

// ReportIndex provides O(1) lookups into a Report.
// Build it once with report.Index() and reuse it for multiple queries.
type ReportIndex struct {
	ByName       map[string]*ServiceInfo
	ByRef        map[string]*ServiceInfo
	ByScope      map[string][]ServiceInfo
	EventsByName map[string][]Event
	EventsByRef  map[string][]Event
	EventsByType map[EventType][]Event
}

// Index builds a lookup index for O(1) report queries.
// Useful when performing multiple lookups on the same report.
func (r Report) Index() ReportIndex {
	idx := ReportIndex{
		ByName:       make(map[string]*ServiceInfo, len(r.Services)),
		ByRef:        make(map[string]*ServiceInfo, len(r.Services)),
		ByScope:      make(map[string][]ServiceInfo),
		EventsByName: make(map[string][]Event),
		EventsByRef:  make(map[string][]Event),
		EventsByType: make(map[EventType][]Event),
	}

	for i := range r.Services {
		svc := &r.Services[i]
		idx.ByName[svc.ServiceName] = svc
		idx.ByRef[serviceKey(svc.ScopeID, svc.ServiceName)] = svc
		idx.ByScope[svc.ScopeID] = append(idx.ByScope[svc.ScopeID], *svc)
	}

	for _, event := range r.Events {
		idx.EventsByName[event.ServiceName] = append(
			idx.EventsByName[event.ServiceName], event,
		)

		key := serviceKey(event.ScopeID, event.ServiceName)
		idx.EventsByRef[key] = append(idx.EventsByRef[key], event)

		idx.EventsByType[event.EventType] = append(
			idx.EventsByType[event.EventType], event,
		)
	}

	return idx
}

// WriteNDJSON writes every event as a line-delimited JSON object (NDJSON).
// Operates directly on the Report.Events slice without a defensive copy,
// unlike Plugin.WriteEventsNDJSON which copies first.
func (r Report) WriteNDJSON(writer io.Writer) error {
	return writeEventsNDJSON(writer, r.Events)
}

// WriteJSON writes the full report as indented JSON to the writer.
func (r Report) WriteJSON(writer io.Writer) error {
	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")

	err := enc.Encode(r)
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}

	return nil
}

package auditlog

import (
	"fmt"

	"github.com/larsartmann/go-output/daghtml"
)

// CSS color tokens used by the daghtml visualization. Centralized as constants
// so the goconst linter recognizes them as repeated literals.
const (
	dagColorAccent    = "var(--accent)"
	dagColorSuccess   = "var(--success)"
	dagColorTextMuted = "var(--text-muted)"
	dagColorError     = "var(--error)"
)

// serviceStatusDAGColor maps a ServiceStatus to the CSS color token used by the
// daghtml visualization. Matches the statusColors map in the former inline
// renderGraph() JS. Read-only lookup table — treated as a constant, never
// mutated at runtime.
//
//nolint:gochecknoglobals // read-only lookup table, not mutable shared state
var serviceStatusDAGColor = map[ServiceStatus]string{
	ServiceStatusActive:          dagColorSuccess,
	ServiceStatusRegistered:      dagColorTextMuted,
	ServiceStatusShutdown:        dagColorAccent,
	ServiceStatusInvocationError: dagColorError,
	ServiceStatusShutdownError:   dagColorError,
}

// serviceProviderDAGColor maps a ProviderType to its CSS color token.
// Read-only lookup table — treated as a constant, never mutated at runtime.
//
//nolint:gochecknoglobals // read-only lookup table, not mutable shared state
var serviceProviderDAGColor = map[ProviderType]string{
	ProviderTypeLazy:      "var(--lazy)",
	ProviderTypeEager:     "var(--eager)",
	ProviderTypeTransient: "var(--transient)",
	ProviderTypeAlias:     "var(--alias)",
}

// BuildDAGHTML converts a Report into a daghtml.DAG for the interactive graph
// renderer. Each service becomes a node; each dependency edge points from the
// dependent to its dependency. Exported for use by the live/ sub-package.
func BuildDAGHTML(report Report) daghtml.DAG {
	dag := daghtml.DAG{
		Nodes: make([]daghtml.Node, 0, len(report.Services)),
		Edges: make([]daghtml.Edge, 0),
	}

	for _, svc := range report.Services {
		label := serviceLabel(svc)

		color := serviceProviderDAGColor[svc.ServiceType]
		if color == "" {
			color = serviceStatusDAGColor[svc.Status]
		}

		if color == "" {
			color = dagColorAccent
		}

		dag.Nodes = append(dag.Nodes, daghtml.Node{
			ID:      diagramNodeID(svc.ScopeID, svc.ServiceName),
			Label:   label,
			Color:   color,
			Tooltip: buildServiceTooltip(svc),
			Error:   svc.Status.IsError(),
		})
	}

	for _, svc := range report.Services {
		fromID := diagramNodeID(svc.ScopeID, svc.ServiceName)
		for _, dep := range svc.Dependencies {
			dag.Edges = append(dag.Edges, daghtml.Edge{
				From: fromID,
				To:   diagramNodeID(dep.ScopeID, dep.ServiceName),
			})
		}
	}

	return dag
}

func buildServiceTooltip(svc ServiceInfo) string {
	tip := string(svc.ServiceName) + " | " + svc.ScopeName + " | status: " + string(svc.Status)

	tip += fmt.Sprintf(" | invocations: %d", svc.InvocationCount)
	if svc.FirstBuildDurationMs != nil {
		tip += fmt.Sprintf(" | build: %.3fms", *svc.FirstBuildDurationMs)
	}

	if svc.InvocationError != nil {
		tip += " | error: " + *svc.InvocationError
	}

	if svc.ShutdownError != nil {
		tip += " | shutdown_error: " + *svc.ShutdownError
	}

	return tip
}

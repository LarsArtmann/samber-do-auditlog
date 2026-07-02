package auditlog

import (
	"fmt"

	"github.com/larsartmann/go-output/daghtml"
)

// serviceStatusDAGColor maps a ServiceStatus to the CSS color token used by the
// daghtml visualization. Matches the statusColors map in the former inline
// renderGraph() JS.
var serviceStatusDAGColor = map[ServiceStatus]string{
	ServiceStatusActive:          "var(--success)",
	ServiceStatusRegistered:      "var(--text-muted)",
	ServiceStatusShutdown:        "var(--accent)",
	ServiceStatusInvocationError: "var(--error)",
	ServiceStatusShutdownError:   "var(--error)",
}

// serviceProviderDAGColor maps a ProviderType to its CSS color token.
var serviceProviderDAGColor = map[ProviderType]string{
	ProviderTypeLazy:      "var(--lazy)",
	ProviderTypeEager:     "var(--eager)",
	ProviderTypeTransient: "var(--transient)",
	ProviderTypeAlias:     "var(--alias)",
}

// buildDAGHTML converts a Report into a daghtml.DAG for the interactive graph
// renderer. Each service becomes a node; each dependency edge points from the
// dependent to its dependency.
func buildDAGHTML(report Report) daghtml.DAG {
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
			color = "var(--accent)"
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
	tip := svc.ServiceName + " | " + svc.ScopeName + " | status: " + string(svc.Status)

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

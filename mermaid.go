package auditlog

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// WriteMermaid writes a Mermaid flowchart representing the dependency graph.
// Each service is a node; edges point from dependent → dependency.
func (r Report) WriteMermaid(writer io.Writer) error {
	_, err := fmt.Fprintln(writer, "flowchart TD")
	if err != nil {
		return fmt.Errorf("write mermaid header: %w", err)
	}

	seen := make(map[string]struct{})

	var lines []string

	for _, svc := range r.Services {
		svcID := mermaidNodeID(svc.ScopeID, svc.ServiceName)

		if _, ok := seen[svcID]; !ok {
			label := serviceLabel(svc)

			lines = append(lines, fmt.Sprintf("    %s[%s]", svcID, label))
			seen[svcID] = struct{}{}
		}

		for _, dep := range svc.Dependencies {
			depID := mermaidNodeID(dep.ScopeID, dep.ServiceName)

			if _, ok := seen[depID]; !ok {
				label := serviceRefLabel(dep)

				lines = append(lines, fmt.Sprintf("    %s[%s]", depID, label))
				seen[depID] = struct{}{}
			}

			lines = append(lines, fmt.Sprintf("    %s --> %s", svcID, depID))
		}
	}

	slices.Sort(lines)

	unique := slices.Compact(lines)

	return writeSortedLines(writer, unique)
}

func mermaidNodeID(scopeID, serviceName string) string {
	clean := strings.NewReplacer(
		"-", "_",
		" ", "_",
		"/", "_",
		".", "_",
	).Replace(scopeID + "_" + serviceName)

	return clean
}

package auditlog

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// WritePlantUML writes a PlantUML component diagram representing the dependency graph.
// Each service is a component; edges point from dependent → dependency.
// Paste the output into any tool that renders PlantUML.
func (r Report) WritePlantUML(writer io.Writer) error {
	_, err := fmt.Fprintln(writer, "@startuml")
	if err != nil {
		return fmt.Errorf("write plantuml header: %w", err)
	}

	seen := make(map[string]struct{})

	var lines []string

	for _, svc := range r.Services {
		svcID := plantumlNodeID(svc.ScopeID, svc.ServiceName)

		if _, ok := seen[svcID]; !ok {
			label := serviceLabel(svc)

			lines = append(lines, fmt.Sprintf(`    component "%s" as %s`, label, svcID))
			seen[svcID] = struct{}{}
		}

		for _, dep := range svc.Dependencies {
			depID := plantumlNodeID(dep.ScopeID, dep.ServiceName)

			if _, ok := seen[depID]; !ok {
				label := serviceRefLabel(dep)

				lines = append(lines, fmt.Sprintf(`    component "%s" as %s`, label, depID))
				seen[depID] = struct{}{}
			}

			lines = append(lines, fmt.Sprintf("    %s --> %s", svcID, depID))
		}
	}

	slices.Sort(lines)

	unique := slices.Compact(lines)

	err = writeSortedLines(writer, unique)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(writer, "@enduml")
	if err != nil {
		return fmt.Errorf("write plantuml footer: %w", err)
	}

	return nil
}

func plantumlNodeID(scopeID, serviceName string) string {
	return strings.NewReplacer(
		"-", "_",
		" ", "_",
		"/", "_",
		".", "_",
		"*", "_",
		"[", "_",
		"]", "_",
	).Replace(scopeID + "_" + serviceName)
}

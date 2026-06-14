package auditlog

import (
	"fmt"
	"io"
)

// WritePlantUML writes a PlantUML component diagram representing the dependency graph.
// Each service is a component; edges point from dependent → dependency.
// Paste the output into any tool that renders PlantUML.
func (r Report) WritePlantUML(writer io.Writer) error {
	err := writeDiagram(writer, r, plantumlFormatter{})
	if err != nil {
		return fmt.Errorf("write plantuml diagram: %w", err)
	}

	return nil
}

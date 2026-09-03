package auditlog

import (
	"fmt"
	"io"
	"strings"
)

// WritePlantUML writes a PlantUML component diagram representing the dependency
// graph. Each service is a component; edges point from dependent -> dependency.
// The warm-amber palette is applied per-node via PlantUML color specs.
// Paste the output into any tool that renders PlantUML.
//
// Use [WithDirection] to change the layout direction (default: top-down):
//
//	report.WritePlantUML(w, auditlog.WithDirection(auditlog.DirectionRight))
func (r Report) WritePlantUML(writer io.Writer, opts ...DiagramOption) error {
	cfg := applyDiagramOpts(opts)

	nodes := buildDiagramNodes(r)
	edges := dedupDiagramEdges(buildDiagramEdges(r))

	rendered := renderPlantUML(nodes, edges)

	if cfg.hasDirection() {
		cmd := plantumlDirectionCommand(cfg.direction)
		rendered = applyPlantumlDirection(rendered, cmd)
	}

	if err := writeDiagram(writer, rendered); err != nil {
		return fmt.Errorf("write plantuml diagram: %w", err)
	}

	return nil
}

// WritePlantUMLString returns the PlantUML diagram as a string. It is a
// convenience wrapper around [Report.WritePlantUML] for use in tests, CLI
// output, and any context where a string is preferred over an [io.Writer].
func (r Report) WritePlantUMLString(opts ...DiagramOption) (string, error) {
	var buf strings.Builder

	if err := r.WritePlantUML(&buf, opts...); err != nil {
		return "", err
	}

	return buf.String(), nil
}

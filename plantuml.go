package auditlog

import (
	"fmt"
	"io"

	"github.com/larsartmann/go-output/plantuml"
)

// WritePlantUML writes a PlantUML component diagram representing the dependency
// graph. Each service is a component; edges point from dependent -> dependency.
// The warm-amber palette is applied per-node via PlantUML color specs.
// Paste the output into any tool that renders PlantUML.
func (r Report) WritePlantUML(writer io.Writer) error {
	renderer := plantuml.NewPlantUMLDiagram()
	renderer.SetNodes(buildDiagramNodes(r))
	renderer.SetEdges(buildDiagramEdges(r))
	renderer.DedupEdges()

	err := writeRendered(writer, renderer)
	if err != nil {
		return fmt.Errorf("write plantuml diagram: %w", err)
	}

	return nil
}

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
//
// Use [WithDirection] to change the layout direction (default: top-down):
//
//	report.WritePlantUML(w, auditlog.WithDirection(output.DirectionRight))
func (r Report) WritePlantUML(writer io.Writer, opts ...DiagramOption) error {
	cfg := applyDiagramOpts(opts)
	renderer := plantuml.NewPlantUMLDiagram()

	var transform func(string) string

	if cfg.hasDirection() {
		cmd := plantumlDirectionCommand(cfg.direction)
		transform = func(out string) string {
			return applyPlantumlDirection(out, cmd)
		}
	}

	err := renderGraphDiagramTransform(writer, r, renderer, transform)
	if err != nil {
		return fmt.Errorf("write plantuml diagram: %w", err)
	}

	return nil
}

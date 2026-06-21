package auditlog

import (
	"fmt"
	"io"

	"github.com/larsartmann/go-output/d2"
)

// WriteD2 writes a D2 diagram representing the dependency graph to writer.
// Each service is a node; edges point from dependent -> dependency. The diagram
// title is set to the container ID for self-documenting output, and the
// warm-amber palette is applied per-node via D2 style directives. Edges are
// deduplicated locally via dedupGraphEdges because the D2 renderer has no
// built-in DedupEdges. See also WriteMermaid, WritePlantUML, and WriteDOT for
// the other diagram formats.
func (r Report) WriteD2(writer io.Writer) error {
	renderer := d2.NewD2Diagram()
	renderer.SetTitle(r.ContainerID)
	renderer.SetNodes(buildDiagramNodes(r))
	renderer.SetEdges(dedupGraphEdges(buildDiagramEdges(r)))

	err := writeRendered(writer, renderer)
	if err != nil {
		return fmt.Errorf("write d2 diagram: %w", err)
	}

	return nil
}

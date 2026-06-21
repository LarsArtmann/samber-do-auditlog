package auditlog

import (
	"fmt"
	"io"

	"github.com/larsartmann/go-output"
	"github.com/larsartmann/go-output/d2"
)

// WriteD2 writes a D2 diagram representing the dependency graph.
// Each service is a node; edges point from dependent -> dependency.
// The warm-amber palette is applied per-node via D2 style directives.
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

// dedupGraphEdges removes duplicate edges (same from/to pair) while preserving
// order. D2's SetEdges does not deduplicate, so we do it here for consistency
// with Mermaid/PlantUML/DOT which rely on renderer-level DedupEdges.
func dedupGraphEdges(edges []output.GraphEdge) []output.GraphEdge {
	seen := make(map[string]struct{}, len(edges))
	out := make([]output.GraphEdge, 0, len(edges))

	for _, edge := range edges {
		key := edge.From.Get() + "|" + edge.To.Get()
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}

		out = append(out, edge)
	}

	return out
}

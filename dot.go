package auditlog

import (
	"fmt"
	"io"

	"github.com/larsartmann/go-output/graph"
)

// WriteDOT writes a Graphviz DOT digraph representing the dependency graph.
// Each service is a node; edges point from dependent -> dependency. The output
// is valid input for `dot -Tsvg` / `dot -Tpng`. Nodes carry the warm-amber
// palette via per-node fillcolor/color attributes.
func (r Report) WriteDOT(writer io.Writer) error {
	renderer := graph.NewDOTRenderer()
	renderer.SetGraphID("do_auditlog")
	renderer.SetRankDir(graph.RankDirLR)

	err := renderGraphDiagram(writer, r, renderer)
	if err != nil {
		return fmt.Errorf("write dot diagram: %w", err)
	}

	return nil
}

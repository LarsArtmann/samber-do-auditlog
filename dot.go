package auditlog

import (
	"fmt"
	"io"
	"strings"

	"github.com/larsartmann/go-output/graph"
)

// WriteDOT writes a Graphviz DOT digraph representing the dependency graph.
// Each service is a node; edges point from dependent -> dependency. The output
// is valid input for `dot -Tsvg` / `dot -Tpng`. Nodes carry the warm-amber
// palette via per-node fillcolor/color attributes.
//
// Use [WithDirection] to change the layout direction (default: left-to-right):
//
//	report.WriteDOT(w, auditlog.WithDirection(output.DirectionDown))
func (r Report) WriteDOT(writer io.Writer, opts ...DiagramOption) error {
	cfg := applyDiagramOpts(opts)
	renderer := graph.NewDOTRenderer()
	renderer.SetGraphID("do_auditlog")

	if cfg.hasDirection() {
		renderer.SetDirection(cfg.direction)
	} else {
		renderer.SetRankDir(graph.RankDirLR)
	}

	err := renderGraphDiagram(writer, r, renderer)
	if err != nil {
		return fmt.Errorf("write dot diagram: %w", err)
	}

	return nil
}

// WriteDOTString returns the DOT diagram as a string. It is a convenience
// wrapper around [Report.WriteDOT] for use in tests, CLI output, and any
// context where a string is preferred over an [io.Writer].
func (r Report) WriteDOTString(opts ...DiagramOption) (string, error) {
	var buf strings.Builder

	if err := r.WriteDOT(&buf, opts...); err != nil {
		return "", err
	}

	return buf.String(), nil
}

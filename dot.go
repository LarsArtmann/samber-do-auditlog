package auditlog

import (
	"fmt"
	"io"
	"strings"
)

// dotGraphID is the digraph identifier used in all DOT output.
const dotGraphID = "do_auditlog"

// WriteDOT writes a Graphviz DOT digraph representing the dependency graph.
// Each service is a node; edges point from dependent -> dependency. The output
// is valid input for `dot -Tsvg` / `dot -Tpng`. Nodes carry the warm-amber
// palette via per-node fillcolor/color attributes.
//
// Use [WithDirection] to change the layout direction (default: left-to-right):
//
//	report.WriteDOT(w, auditlog.WithDirection(auditlog.DirectionDown))
func (r Report) WriteDOT(writer io.Writer, opts ...DiagramOption) error {
	cfg := applyDiagramOpts(opts)

	rankdir := "LR"
	if cfg.hasDirection() {
		rankdir = cfg.direction.toRankDir()
	}

	nodes := buildDiagramNodes(r)
	edges := dedupDiagramEdges(buildDiagramEdges(r))

	if err := writeDiagram(writer, renderDOT(dotGraphID, rankdir, nodes, edges)); err != nil {
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

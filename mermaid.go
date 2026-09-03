package auditlog

import (
	"fmt"
	"io"
	"strings"
)

// WriteMermaid writes a Mermaid flowchart representing the dependency graph.
// Each service is a node; edges point from dependent -> dependency. The warm
// -amber palette is applied per-node via style directives.
//
// Use [WithDirection] to change the layout direction (default: TD):
//
//	report.WriteMermaid(w, auditlog.WithDirection(auditlog.DirectionRight))
func (r Report) WriteMermaid(writer io.Writer, opts ...DiagramOption) error {
	cfg := applyDiagramOpts(opts)

	nodes := buildDiagramNodes(r)
	edges := dedupDiagramEdges(buildDiagramEdges(r))

	rendered := renderMermaid(nodes, edges)

	if cfg.hasDirection() {
		keyword := mermaidDirection(cfg.direction)
		rendered = strings.Replace(rendered, "flowchart TD", "flowchart "+keyword, 1)
	}

	if err := writeDiagram(writer, rendered); err != nil {
		return fmt.Errorf("write mermaid diagram: %w", err)
	}

	return nil
}

// WriteMermaidString returns the Mermaid diagram as a string. It is a
// convenience wrapper around [Report.WriteMermaid] for use in tests, CLI
// output, and any context where a string is preferred over an [io.Writer].
func (r Report) WriteMermaidString(opts ...DiagramOption) (string, error) {
	var buf strings.Builder

	if err := r.WriteMermaid(&buf, opts...); err != nil {
		return "", err
	}

	return buf.String(), nil
}

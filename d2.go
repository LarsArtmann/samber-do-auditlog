package auditlog

import (
	"fmt"
	"io"
	"strings"
)

// WriteD2 writes a D2 diagram representing the dependency graph to writer.
// Each service is a node; edges point from dependent -> dependency. The diagram
// title is set to the container ID for self-documenting output, and the
// warm-amber palette is applied per-node via D2 style directives. See also
// WriteMermaid, WritePlantUML, and WriteDOT for the other diagram formats.
//
// Use [WithDirection] to change the layout direction (default: down):
//
//	report.WriteD2(w, auditlog.WithDirection(auditlog.DirectionRight))
func (r Report) WriteD2(writer io.Writer, opts ...DiagramOption) error {
	cfg := applyDiagramOpts(opts)

	direction := ""
	if cfg.hasDirection() {
		direction = string(cfg.direction)
	}

	nodes := buildDiagramNodes(r)
	edges := dedupDiagramEdges(buildDiagramEdges(r))

	if err := writeDiagram(writer, renderD2(string(r.ContainerID), direction, nodes, edges)); err != nil {
		return fmt.Errorf("write d2 diagram: %w", err)
	}

	return nil
}

// WriteD2String returns the D2 diagram as a string. It is a convenience
// wrapper around [Report.WriteD2] for use in tests, CLI output, and any
// context where a string is preferred over an [io.Writer].
func (r Report) WriteD2String(opts ...DiagramOption) (string, error) {
	var buf strings.Builder

	if err := r.WriteD2(&buf, opts...); err != nil {
		return "", err
	}

	return buf.String(), nil
}

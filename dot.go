package auditlog

import (
	"fmt"
	"io"
)

// WriteDOT writes a Graphviz DOT digraph representing the dependency graph.
// Each service is a node; edges point from dependent → dependency.
// The output is valid input for `dot -Tsvg` / `dot -Tpng`.
func (r Report) WriteDOT(writer io.Writer) error {
	err := writeDiagram(writer, r, dotFormatter{})
	if err != nil {
		return fmt.Errorf("write dot diagram: %w", err)
	}

	return nil
}

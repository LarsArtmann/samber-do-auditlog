package auditlog

import (
	"fmt"
	"io"
)

// WriteMermaid writes a Mermaid flowchart representing the dependency graph.
// Each service is a node; edges point from dependent → dependency.
func (r Report) WriteMermaid(writer io.Writer) error {
	err := writeDiagram(writer, r, mermaidFormatter{})
	if err != nil {
		return fmt.Errorf("write mermaid diagram: %w", err)
	}

	return nil
}

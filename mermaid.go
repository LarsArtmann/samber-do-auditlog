package auditlog

import (
	"fmt"
	"io"

	"github.com/larsartmann/go-output/graph"
)

// WriteMermaid writes a Mermaid flowchart representing the dependency graph.
// Each service is a node; edges point from dependent -> dependency. The warm
// -amber palette is applied per-node via style directives.
func (r Report) WriteMermaid(writer io.Writer) error {
	renderer := graph.NewMermaidRenderer().SetCodeFence(false)

	err := renderGraphDiagram(writer, r, renderer)
	if err != nil {
		return fmt.Errorf("write mermaid diagram: %w", err)
	}

	return nil
}

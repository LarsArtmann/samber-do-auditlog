package auditlog

import (
	"fmt"
	"io"
	"strings"

	"github.com/larsartmann/go-output/graph"
)

// WriteMermaid writes a Mermaid flowchart representing the dependency graph.
// Each service is a node; edges point from dependent -> dependency. The warm
// -amber palette is applied per-node via style directives.
//
// Use [WithDirection] to change the layout direction (default: TD):
//
//	report.WriteMermaid(w, auditlog.WithDirection(output.DirectionRight))
func (r Report) WriteMermaid(writer io.Writer, opts ...DiagramOption) error {
	cfg := applyDiagramOpts(opts)
	renderer := graph.NewMermaidRenderer().SetCodeFence(false)

	var transform func(string) string

	if cfg.hasDirection() {
		keyword := mermaidDirection(cfg.direction)
		transform = func(out string) string {
			return strings.Replace(out, "flowchart TD", "flowchart "+keyword, 1)
		}
	}

	err := renderGraphDiagramTransform(writer, r, renderer, transform)
	if err != nil {
		return fmt.Errorf("write mermaid diagram: %w", err)
	}

	return nil
}

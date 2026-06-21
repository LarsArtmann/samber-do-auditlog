package auditlog

import (
	"fmt"
	"io"

	"github.com/larsartmann/go-output"
	"github.com/larsartmann/go-output/escape"
)

// warmAmberNodeStyle is the "Container Telemetry" palette applied to every
// diagram node, matching the HTML visualization aesthetic. go-output renderers
// translate this into per-node style directives (Mermaid `style`, PlantUML
// `#color;line:...;text:...`, DOT `fillcolor`/`color`), replacing the former
// global theme headers with equivalent per-node styling.
//
//nolint:gochecknoglobals // Static theme palette, safe to share across formats.
var warmAmberNodeStyle = output.GraphStyle{
	Fill:      "#e8a838",
	Stroke:    "#4a4030",
	FontColor: "#14110d",
	FontSize:  0,
}

// diagramNodeID builds a deterministic node identifier from scopeID and
// serviceName. SlugifyID collapses separator characters (- / . * [ ] { } ( )
// and space) to underscores to preserve word boundaries, then MermaidID strips
// any remaining non-identifier rune. The result is valid across Mermaid,
// PlantUML, and DOT. Returns "node" if everything is stripped (via MermaidID).
func diagramNodeID(scopeID, serviceName string) string {
	return escape.MermaidID(escape.SlugifyID(scopeID + "_" + serviceName))
}

// newGraphNode constructs a boxed graph node with the warm-amber style applied.
// All GraphNode fields are set explicitly so the rendering is deterministic.
func newGraphNode(id, label string) output.GraphNode {
	return output.GraphNode{
		ID:       output.NewBrandedID[output.GraphNodeIDBrand](id),
		Label:    output.NewBrandedID[output.GraphNodeLabelBrand](label),
		Shape:    output.NodeShapeBox,
		Style:    warmAmberNodeStyle,
		Metadata: nil,
	}
}

// newGraphEdge constructs an unlabeled directed edge from the dependent node to
// its dependency.
func newGraphEdge(fromID, toID string) output.GraphEdge {
	return *output.NewGraphEdge(fromID, toID)
}

// buildDiagramNodes builds the deduplicated node list for the dependency graph.
// Each registered service becomes a node labeled with its provider-type icon
// (via serviceLabel); external dependencies are added as bare nodes. Nodes are
// deduplicated by ID — first occurrence wins, preserving the sorted iteration
// order of report.Services for deterministic output.
func buildDiagramNodes(report Report) []output.GraphNode {
	seen := make(map[string]struct{})
	nodes := make([]output.GraphNode, 0, len(report.Services))

	addNode := func(nodeID, label string) {
		if _, ok := seen[nodeID]; ok {
			return
		}

		seen[nodeID] = struct{}{}
		nodes = append(nodes, newGraphNode(nodeID, label))
	}

	for _, svc := range report.Services {
		fromID := diagramNodeID(svc.ScopeID, svc.ServiceName)
		addNode(fromID, serviceLabel(svc))

		for _, dep := range svc.Dependencies {
			toID := diagramNodeID(dep.ScopeID, dep.ServiceName)
			addNode(toID, dep.ServiceName)
		}
	}

	return nodes
}

// buildDiagramEdges builds the edge list for the dependency graph: one edge per
// dependent -> dependency pair. Duplicate edges (same from/to) are NOT removed
// here; the renderer's DedupEdges handles that for Mermaid/PlantUML/DOT.
// D2 uses the dedupGraphEdges helper since go-output's D2 renderer lacks
// built-in edge dedup.
func buildDiagramEdges(report Report) []output.GraphEdge {
	edges := make([]output.GraphEdge, 0, len(report.Services))

	for _, svc := range report.Services {
		fromID := diagramNodeID(svc.ScopeID, svc.ServiceName)

		for _, dep := range svc.Dependencies {
			toID := diagramNodeID(dep.ScopeID, dep.ServiceName)
			edges = append(edges, newGraphEdge(fromID, toID))
		}
	}

	return edges
}

// writeRendered renders a graph renderer to writer with consistent error
// wrapping shared by WriteMermaid, WritePlantUML, WriteDOT, and WriteD2.
func writeRendered(writer io.Writer, renderer output.Renderer) error {
	out, err := renderer.Render()
	if err != nil {
		return fmt.Errorf("render diagram: %w", err)
	}

	_, err = writer.Write([]byte(out))
	if err != nil {
		return fmt.Errorf("write diagram: %w", err)
	}

	return nil
}

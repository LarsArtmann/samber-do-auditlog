package auditlog

import (
	"fmt"
	"io"
	"strings"
)

// This file contains a self-contained port of the go-output rendering pipeline
// (escape + graph + plantuml + d2 renderers) reduced to the features the
// auditlog dependency graph uses: boxed nodes with the warm-amber palette and
// unlabeled directed edges. Output is byte-identical to the go-output v0.37.0
// renderers for these inputs, which keeps all diagram tests valid.

// warmAmberNodeStyle is the "Container Telemetry" palette applied to every
// diagram node, matching the HTML visualization aesthetic.
//
//nolint:gochecknoglobals // Static theme palette, safe to share across formats.
var warmAmberNodeStyle = diagramNodeStyle{
	Fill:      "#e8a838",
	Stroke:    "#4a4030",
	FontColor: "#14110d",
}

// diagramNodeStyle carries per-node colors shared by all four formats.
type diagramNodeStyle struct {
	Fill      string
	Stroke    string
	FontColor string
}

func (s diagramNodeStyle) isSet() bool {
	return s.Fill != "" || s.Stroke != "" || s.FontColor != ""
}

// diagramNode is a boxed graph node.
type diagramNode struct {
	id    string
	label string
	style diagramNodeStyle
}

// diagramEdge is an unlabeled directed edge from dependent to dependency.
type diagramEdge struct {
	from string
	to   string
}

// slugIDReplacer sanitizes strings for use as identifiers across diagram
// formats: identifier-hostile characters become underscores.
//
//nolint:gochecknoglobals // Reusable strings.Replacer, safe to share.
var slugIDReplacer = strings.NewReplacer(
	" ", "_",
	"-", "_",
	"/", "_",
	".", "_",
	"*", "_",
	"[", "_",
	"]", "_",
	"{", "_",
	"}", "_",
	"(", "_",
	")", "_",
)

// slugifyID sanitizes a string for use as a diagram node identifier.
func slugifyID(s string) string {
	return slugIDReplacer.Replace(s)
}

// isDiagramIdentRune reports whether r is valid in a Mermaid node identifier
// (ASCII letter, digit, or underscore).
func isDiagramIdentRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '_'
}

// mermaidID sanitizes a string for use as a Mermaid node identifier.
func mermaidID(id string) string {
	var result strings.Builder

	for _, r := range id {
		if isDiagramIdentRune(r) {
			result.WriteRune(r)
		}
	}

	if result.Len() == 0 {
		return "node"
	}

	return result.String()
}

// mermaidTextReplacer escapes brackets, braces, quotes, and newlines for
// Mermaid display labels.
//
//nolint:gochecknoglobals // Reusable strings.Replacer, safe to share.
var mermaidTextReplacer = strings.NewReplacer(
	`"`, "'",
	"[", "(",
	"]", ")",
	"{", "(",
	"}", ")",
	"\n", "<br>",
)

// mermaidText escapes special characters for Mermaid display labels.
func mermaidText(s string) string {
	return mermaidTextReplacer.Replace(s)
}

// d2Replacer escapes backslash, double quote, newline, and tab for D2 and DOT
// strings (both formats share the same escaping rules).
//
//nolint:gochecknoglobals // Reusable strings.Replacer, safe to share.
var d2Replacer = strings.NewReplacer(
	`\`, `\\`,
	`"`, `\"`,
	"\n", `\n`,
	"\t", `\t`,
)

// d2Escape escapes special characters for D2 diagram strings.
func d2Escape(s string) string {
	return d2Replacer.Replace(s)
}

// dotEscape escapes special characters for DOT/Graphviz strings.
func dotEscape(s string) string {
	return d2Replacer.Replace(s)
}

// plantumlReplacer escapes characters that break PlantUML component notation
// and edge labels.
//
//nolint:gochecknoglobals // Reusable strings.Replacer, safe to share.
var plantumlReplacer = strings.NewReplacer(
	"]", "\\]",
	"\n", "\\n",
	`\`, `\\`,
	`"`, `\"`,
)

// plantumlEscape escapes special characters for PlantUML labels and text.
func plantumlEscape(s string) string {
	return plantumlReplacer.Replace(s)
}

// d2NeedsQuoting reports whether s contains characters that are special in
// D2 syntax and would break parsing if left unquoted.
func d2NeedsQuoting(s string) bool {
	if s == "" {
		return true
	}

	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r',
			'#', ':', ';', '|',
			'{', '}', '[', ']', '(', ')',
			'"', '\\':
			return true
		}
	}

	return false
}

// d2Quote wraps s in double quotes (after D2 escaping) when s contains
// characters that require quoting. Simple identifiers are returned unquoted.
func d2Quote(s string) string {
	if d2NeedsQuoting(s) {
		return `"` + d2Escape(s) + `"`
	}

	return d2Escape(s)
}

// diagramNodeID builds a deterministic node identifier from scopeID and
// serviceName. slugifyID collapses separator characters to underscores to
// preserve word boundaries, then mermaidID strips any remaining
// non-identifier rune. The result is valid across Mermaid, PlantUML, and DOT.
func diagramNodeID(scopeID ScopeID, serviceName ServiceName) string {
	return mermaidID(slugifyID(string(scopeID) + "_" + string(serviceName)))
}

// newDiagramNode constructs a boxed graph node with the warm-amber style.
func newDiagramNode(id, label string) diagramNode {
	return diagramNode{id: id, label: label, style: warmAmberNodeStyle}
}

// buildDiagramNodes builds the deduplicated node list for the dependency graph.
// Each registered service becomes a node labeled with its provider-type icon
// (via serviceLabel); external dependencies are added as bare nodes. Nodes are
// deduplicated by ID — first occurrence wins, preserving the sorted iteration
// order of report.Services for deterministic output.
func buildDiagramNodes(report Report) []diagramNode {
	seen := make(map[string]struct{})
	nodes := make([]diagramNode, 0, len(report.Services))

	addNode := func(nodeID, label string) {
		if _, ok := seen[nodeID]; ok {
			return
		}

		seen[nodeID] = struct{}{}
		nodes = append(nodes, newDiagramNode(nodeID, label))
	}

	for _, svc := range report.Services {
		fromID := diagramNodeID(svc.ScopeID, svc.ServiceName)
		addNode(fromID, serviceLabel(svc))

		for _, dep := range svc.Dependencies {
			toID := diagramNodeID(dep.ScopeID, dep.ServiceName)
			addNode(toID, string(dep.ServiceName))
		}
	}

	return nodes
}

// buildDiagramEdges builds the edge list for the dependency graph: one edge
// per dependent -> dependency pair. Duplicate edges are NOT removed here;
// callers deduplicate via dedupDiagramEdges.
func buildDiagramEdges(report Report) []diagramEdge {
	edges := make([]diagramEdge, 0, len(report.Services))

	for _, svc := range report.Services {
		fromID := diagramNodeID(svc.ScopeID, svc.ServiceName)

		for _, dep := range svc.Dependencies {
			toID := diagramNodeID(dep.ScopeID, dep.ServiceName)
			edges = append(edges, diagramEdge{from: fromID, to: toID})
		}
	}

	return edges
}

// dedupDiagramEdges removes duplicate edges (same from/to pair) while
// preserving order. Matches go-output GraphBuilder.DedupEdges semantics.
func dedupDiagramEdges(edges []diagramEdge) []diagramEdge {
	if len(edges) <= 1 {
		return edges
	}

	seen := make(map[string]struct{}, len(edges))
	out := make([]diagramEdge, 0, len(edges))

	for _, edge := range edges {
		key := edge.from + "\x00" + edge.to
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}

		out = append(out, edge)
	}

	return out
}

// writeDiagram writes a rendered diagram to writer with consistent error
// wrapping shared by all Write* diagram entry points.
func writeDiagram(writer io.Writer, rendered string) error {
	if _, err := writer.Write([]byte(rendered)); err != nil {
		return fmt.Errorf("write diagram: %w", err)
	}

	return nil
}

// renderMermaid renders the Mermaid flowchart (no code fence, matching
// go-output MermaidRenderer with SetCodeFence(false)).
func renderMermaid(nodes []diagramNode, edges []diagramEdge) string {
	var b strings.Builder

	b.WriteString("flowchart TD\n")

	for _, node := range nodes {
		fmt.Fprintf(&b, "    %s[%s]\n", mermaidID(node.id), mermaidText(node.label))
	}

	for _, edge := range edges {
		fmt.Fprintf(&b, "    %s --> %s\n", mermaidID(edge.from), mermaidID(edge.to))
	}

	writeMermaidNodeStyles(&b, nodes)

	return b.String()
}

// writeMermaidNodeStyles emits per-node Mermaid style directives. This
// replaces the go-output renderer's classDef handling.
func writeMermaidNodeStyles(b *strings.Builder, nodes []diagramNode) {
	wroteAny := false

	for _, node := range nodes {
		parts := mermaidStyleParts(node.style)
		if len(parts) == 0 {
			continue
		}

		if !wroteAny {
			b.WriteString("\n    %% Styling\n")

			wroteAny = true
		}

		fmt.Fprintf(b, "    style %s %s\n", mermaidID(node.id), strings.Join(parts, ","))
	}
}

// mermaidStyleParts converts a diagramNodeStyle into Mermaid style pairs.
func mermaidStyleParts(s diagramNodeStyle) []string {
	var parts []string

	if s.Fill != "" {
		parts = append(parts, "fill:"+mermaidText(s.Fill))
	}

	if s.Stroke != "" {
		parts = append(parts, "stroke:"+mermaidText(s.Stroke))
	}

	if s.FontColor != "" {
		parts = append(parts, "color:"+mermaidText(s.FontColor))
	}

	return parts
}

// dotSepDefaults are the default DOT spacing attributes.
const (
	dotNodeSep = "0.5"
	dotRankSep = "0.5"
)

// renderDOT renders the Graphviz digraph, byte-identical to go-output's
// DOTRenderer for our node/edge shapes.
func renderDOT(graphID, rankdir string, nodes []diagramNode, edges []diagramEdge) string {
	var b strings.Builder

	b.WriteString("digraph ")
	b.WriteString(graphID)
	b.WriteString(" {\n")

	b.WriteString("  // Graph attributes\n")
	fmt.Fprintf(&b, "  rankdir=%s;\n", rankdir)
	b.WriteString("  splines=ortho;\n")
	fmt.Fprintf(&b, "  nodesep=%s;\n", dotNodeSep)
	fmt.Fprintf(&b, "  ranksep=%s;\n\n", dotRankSep)

	b.WriteString("  // Default node attributes\n")
	b.WriteString("  node [\n")
	b.WriteString("    shape=box\n")
	b.WriteString("    fontname=\"Helvetica\"\n")
	b.WriteString("    fontsize=12\n")
	b.WriteString("  ];\n\n")

	b.WriteString("  // Nodes\n")

	for _, node := range nodes {
		b.WriteString("  \"")
		b.WriteString(dotEscape(node.id))
		b.WriteString("\" [\n")

		b.WriteString("    label=\"")
		b.WriteString(dotEscape(node.label))
		b.WriteString("\"\n")

		fmt.Fprintf(&b, "    shape=\"%s\"\n", "box")

		if node.style.Fill != "" {
			fmt.Fprintf(&b, "    fillcolor=\"%s\"\n", dotEscape(node.style.Fill))
		}

		if node.style.Stroke != "" {
			fmt.Fprintf(&b, "    color=\"%s\"\n", dotEscape(node.style.Stroke))
		}

		b.WriteString("  ];\n")
	}

	b.WriteString("\n  // Edges\n")

	for _, edge := range edges {
		fmt.Fprintf(&b, "  \"%s\" -> \"%s\";\n", dotEscape(edge.from), dotEscape(edge.to))
	}

	b.WriteString("}\n")

	return b.String()
}

// renderPlantUML renders the PlantUML component diagram, byte-identical to
// go-output's PlantUMLDiagram for our node/edge shapes.
func renderPlantUML(nodes []diagramNode, edges []diagramEdge) string {
	var b strings.Builder

	b.WriteString("@startuml\n")
	b.WriteString("skinparam componentStyle uml2\n")
	b.WriteString("skinparam defaultFontSize 12\n\n")

	for _, node := range nodes {
		colorSpec := plantumlColorSpec(node.style)

		label := plantumlEscape(node.label)

		nodeID := slugifyID(node.id)

		if colorSpec != "" {
			fmt.Fprintf(&b, "[%s] as %s %s\n", label, nodeID, colorSpec)
		} else {
			fmt.Fprintf(&b, "[%s] as %s\n", label, nodeID)
		}
	}

	b.WriteString("\n")

	for _, edge := range edges {
		fmt.Fprintf(&b, "%s --> %s\n", slugifyID(edge.from), slugifyID(edge.to))
	}

	b.WriteString("\n@enduml")

	return b.String()
}

// plantumlColorSpec converts a diagramNodeStyle into a PlantUML color
// specification string (e.g. #e8a838;line:#4a4030;text:#14110d).
func plantumlColorSpec(s diagramNodeStyle) string {
	var parts []string

	if s.Fill != "" {
		parts = append(parts, plantumlColorValue(s.Fill))
	}

	if s.Stroke != "" {
		parts = append(parts, "line:"+plantumlColorValue(s.Stroke))
	}

	if s.FontColor != "" {
		parts = append(parts, "text:"+plantumlColorValue(s.FontColor))
	}

	result := strings.Join(parts, ";")

	if result != "" && !strings.HasPrefix(result, "#") {
		result = "#" + result
	}

	return result
}

// plantumlColorValue escapes a color value for use in a PlantUML color spec,
// also replacing the attribute separator to prevent injection.
func plantumlColorValue(s string) string {
	return strings.ReplaceAll(plantumlEscape(s), ";", "_")
}

// renderD2 renders the D2 diagram, byte-identical to go-output's D2 Diagram
// with title + per-node warm-amber styling.
func renderD2(title, direction string, nodes []diagramNode, edges []diagramEdge) string {
	var b strings.Builder

	hasConfig := direction != "" || title != ""
	if hasConfig {
		if direction != "" {
			fmt.Fprintf(&b, "direction: %s\n", direction)
		}

		if title != "" {
			fmt.Fprintf(&b, "title: {\n  label: %s\n}\n", d2Quote(title))
		}

		b.WriteString("\n")
	}

	for _, node := range nodes {
		if node.style.isSet() {
			fmt.Fprintf(&b, "%s: %s {\n", d2Quote(node.id), d2Quote(node.label))

			if node.style.Fill != "" {
				fmt.Fprintf(&b, "  style.fill: %s\n", d2Quote(node.style.Fill))
			}

			if node.style.Stroke != "" {
				fmt.Fprintf(&b, "  style.stroke: %s\n", d2Quote(node.style.Stroke))
			}

			if node.style.FontColor != "" {
				fmt.Fprintf(&b, "  style.font-color: %s\n", d2Quote(node.style.FontColor))
			}

			b.WriteString("}\n")
		} else {
			fmt.Fprintf(&b, "%s: %s\n", d2Quote(node.id), d2Quote(node.label))
		}
	}

	for _, edge := range edges {
		fmt.Fprintf(&b, "%s -> %s\n", d2Quote(edge.from), d2Quote(edge.to))
	}

	return strings.TrimRight(b.String(), "\n")
}

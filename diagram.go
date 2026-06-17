package auditlog

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// diagramFormatter turns a collected graph into a specific text format.
type diagramFormatter interface {
	Header() string
	Footer() string
	NodeID(scopeID, serviceName string) string
	NodeDecl(id, label string) string
	EdgeDecl(fromID, toID string) string
}

// diagramEntry is a single line in the generated diagram, paired with its
// sort key so declarations and edges can be deduplicated and ordered.
type diagramEntry struct {
	line string
	key  string
}

// diagramAvgLineBytes is the pre-allocation estimate for each diagram line.
const diagramAvgLineBytes = 64

// writeDiagram writes a dependency graph using the supplied formatter.
// It deduplicates nodes and edges and sorts the output for stable,
// deterministic reports. All lines are batched via strings.Builder and
// written in a single io.Writer.Write call to minimize syscalls.
func writeDiagram(writer io.Writer, report Report, formatter diagramFormatter) error {
	seen := make(map[string]struct{})

	var entries []diagramEntry

	add := func(key, line string) {
		if _, ok := seen[key]; ok {
			return
		}

		seen[key] = struct{}{}
		entries = append(entries, diagramEntry{line: line, key: key})
	}

	for _, svc := range report.Services {
		fromID := formatter.NodeID(svc.ScopeID, svc.ServiceName)
		add(fromID, formatter.NodeDecl(fromID, serviceLabel(svc)))

		for _, dep := range svc.Dependencies {
			toID := formatter.NodeID(dep.ScopeID, dep.ServiceName)
			add(toID, formatter.NodeDecl(toID, serviceRefLabel(dep)))
			add(fromID+"->"+toID, formatter.EdgeDecl(fromID, toID))
		}
	}

	slices.SortFunc(entries, func(a, b diagramEntry) int {
		return strings.Compare(a.key, b.key)
	})

	var builder strings.Builder
	builder.Grow(len(entries) * diagramAvgLineBytes)

	builder.WriteString(formatter.Header())
	builder.WriteByte('\n')

	for _, entry := range entries {
		builder.WriteString("    ")
		builder.WriteString(entry.line)
		builder.WriteByte('\n')
	}

	if footer := formatter.Footer(); footer != "" {
		builder.WriteString(footer)
		builder.WriteByte('\n')
	}

	_, err := writer.Write([]byte(builder.String()))
	if err != nil {
		return fmt.Errorf("write diagram: %w", err)
	}

	return nil
}

// diagramIDReplacer collapses characters that are invalid or problematic in
// Mermaid/PlantUML node identifiers into underscores.
//
//nolint:gochecknoglobals // Reusable strings.Replacer, safe to share.
var diagramIDReplacer = strings.NewReplacer(
	"-", "_",
	" ", "_",
	"/", "_",
	".", "_",
	"*", "_",
	"[", "_",
	"]", "_",
)

// sanitizeDiagramID builds a node identifier from scopeID and serviceName that
// is valid in both Mermaid and PlantUML: separators become underscores and any
// remaining non-identifier character is stripped. Returns "node" if the result
// would be empty.
func sanitizeDiagramID(scopeID, serviceName string) string {
	raw := diagramIDReplacer.Replace(scopeID + "_" + serviceName)

	var b strings.Builder
	b.Grow(len(raw))

	for _, r := range raw {
		if isDiagramIdentRune(r) {
			b.WriteRune(r)
		}
	}

	if b.Len() == 0 {
		return "node"
	}

	return b.String()
}

// isDiagramIdentRune reports whether r is valid in a Mermaid/PlantUML node
// identifier (ASCII letter, digit, or underscore).
func isDiagramIdentRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '_'
}

// mermaidLabelReplacer escapes characters that break Mermaid node labels.
//
//nolint:gochecknoglobals // Reusable strings.Replacer, safe to share.
var mermaidLabelReplacer = strings.NewReplacer(
	`"`, "'",
	"[", "(",
	"]", ")",
	"{", "(",
	"}", ")",
	"\n", "<br>",
)

// mermaidLabel escapes special characters for a Mermaid display label.
func mermaidLabel(label string) string {
	return mermaidLabelReplacer.Replace(label)
}

// plantumlLabel escapes a double quote so a service name is safe inside a
// PlantUML quoted component declaration.
func plantumlLabel(label string) string {
	return strings.ReplaceAll(label, `"`, "'")
}

type mermaidFormatter struct{}

func (mermaidFormatter) Header() string {
	return `%%{init: {'theme':'base', 'themeVariables': {'primaryColor':'#e8a838', 'primaryTextColor':'#14110d', 'primaryBorderColor':'#4a4030', 'lineColor':'#9a8d78', 'fontSize':'14px'}}}%%
flowchart TD`
}
func (mermaidFormatter) Footer() string { return "" }
func (mermaidFormatter) NodeID(scopeID, serviceName string) string {
	return sanitizeDiagramID(scopeID, serviceName)
}

func (mermaidFormatter) NodeDecl(id, label string) string {
	return fmt.Sprintf("%s[%s]", id, mermaidLabel(label))
}

func (mermaidFormatter) EdgeDecl(fromID, toID string) string {
	return fmt.Sprintf("%s --> %s", fromID, toID)
}

type plantumlFormatter struct{}

func (plantumlFormatter) Header() string {
	return `@startuml
skinparam component {
  BackgroundColor #e8a838
  FontColor #14110d
  BorderColor #4a4030
}
skinparam arrow {
  Color #9a8d78
}
skinparam defaultTextAlignment left`
}
func (plantumlFormatter) Footer() string { return "@enduml" }
func (plantumlFormatter) NodeID(scopeID, serviceName string) string {
	return sanitizeDiagramID(scopeID, serviceName)
}

func (plantumlFormatter) NodeDecl(id, label string) string {
	return fmt.Sprintf(`component "%s" as %s`, plantumlLabel(label), id)
}

func (plantumlFormatter) EdgeDecl(fromID, toID string) string {
	return fmt.Sprintf("%s --> %s", fromID, toID)
}

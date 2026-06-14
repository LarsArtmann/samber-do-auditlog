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

// writeDiagram writes a dependency graph using the supplied formatter.
// It deduplicates nodes and edges and sorts the output for stable,
// deterministic reports.
func writeDiagram(writer io.Writer, report Report, formatter diagramFormatter) error {
	_, err := fmt.Fprintln(writer, formatter.Header())
	if err != nil {
		return fmt.Errorf("write diagram header: %w", err)
	}

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

	for _, entry := range entries {
		_, err := fmt.Fprintln(writer, "    "+entry.line)
		if err != nil {
			return fmt.Errorf("write diagram line: %w", err)
		}
	}

	if formatter.Footer() != "" {
		_, err = fmt.Fprintln(writer, formatter.Footer())
		if err != nil {
			return fmt.Errorf("write diagram footer: %w", err)
		}
	}

	return nil
}

// diagramNodeID returns a sanitized node ID for diagram output.
func diagramNodeID(scopeID, serviceName string) string {
	return strings.NewReplacer(
		"-", "_",
		" ", "_",
		"/", "_",
		".", "_",
	).Replace(scopeID + "_" + serviceName)
}

type mermaidFormatter struct{}

func (mermaidFormatter) Header() string {
	return `%%{init: {'theme':'base', 'themeVariables': {'primaryColor':'#e8a838', 'primaryTextColor':'#14110d', 'primaryBorderColor':'#4a4030', 'lineColor':'#9a8d78', 'fontSize':'14px'}}}%%
flowchart TD`
}
func (mermaidFormatter) Footer() string { return "" }
func (mermaidFormatter) NodeID(scopeID, serviceName string) string {
	return diagramNodeID(scopeID, serviceName)
}

func (mermaidFormatter) NodeDecl(id, label string) string {
	return fmt.Sprintf("%s[%s]", id, label)
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
	return strings.NewReplacer(
		"-", "_",
		" ", "_",
		"/", "_",
		".", "_",
		"*", "_",
		"[", "_",
		"]", "_",
	).Replace(scopeID + "_" + serviceName)
}

func (plantumlFormatter) NodeDecl(id, label string) string {
	return fmt.Sprintf(`component "%s" as %s`, label, id)
}

func (plantumlFormatter) EdgeDecl(fromID, toID string) string {
	return fmt.Sprintf("%s --> %s", fromID, toID)
}

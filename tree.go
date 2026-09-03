package auditlog

import (
	"fmt"
	"html/template"
	"io"
	"strings"
)

// treeNode is a minimal tree structure served to the ASCII and HTML tree
// renderers. It replaces the go-output TreeNode for the Go 1.23 branch.
// Fields are exported so html/template can read them.
type treeNode struct {
	ID       string
	Label    string
	Children []*treeNode
}

func newTreeNode(id, label string) *treeNode {
	return &treeNode{ID: id, Label: label, Children: nil}
}

func (n *treeNode) addChild(child *treeNode) {
	n.Children = append(n.Children, child)
}

// addTreeChildren recursively adds dependent services as children to the parent
// treeNode, using the provided lookup map and visited set to avoid cycles.
func addTreeChildren(
	parent *treeNode,
	svc ServiceInfo,
	byKey map[string]ServiceInfo,
	visited map[string]struct{},
) {
	key := serviceKey(svc.ScopeID, svc.ServiceName)
	if _, ok := visited[key]; ok {
		return
	}

	visited[key] = struct{}{}

	for _, depRef := range svc.Dependents {
		childSvc, ok := byKey[serviceKey(depRef.ScopeID, depRef.ServiceName)]
		if !ok {
			continue
		}

		childNode := newTreeNode(
			diagramNodeID(childSvc.ScopeID, childSvc.ServiceName),
			serviceLabel(childSvc),
		)
		parent.addChild(childNode)
		addTreeChildren(childNode, childSvc, byKey, visited)
	}
}

// buildServiceTreeNodes constructs a forest of treeNodes from the service
// dependency graph. Root nodes are services with no dependencies; children are
// their dependents (services that depend on the parent). The result is wrapped
// in a single root node for the renderer.
func (r Report) buildServiceTreeNodes() *treeNode {
	title := string(r.ContainerID)
	if title == "" {
		title = "container"
	}

	forestRoot := newTreeNode("container", title)

	if len(r.Services) == 0 {
		return forestRoot
	}

	byKey := make(map[string]ServiceInfo, len(r.Services))
	for _, svc := range r.Services {
		byKey[serviceKey(svc.ScopeID, svc.ServiceName)] = svc
	}

	var roots []ServiceInfo

	for _, svc := range r.Services {
		if len(svc.Dependencies) == 0 {
			roots = append(roots, svc)
		}
	}

	if len(roots) == 0 && len(r.Services) > 0 {
		roots = append(roots, r.Services[0])
	}

	visited := make(map[string]struct{})

	for _, rootSvc := range roots {
		rootNode := newTreeNode(
			diagramNodeID(rootSvc.ScopeID, rootSvc.ServiceName),
			serviceLabel(rootSvc),
		)
		forestRoot.addChild(rootNode)
		addTreeChildren(rootNode, rootSvc, byKey, visited)
	}

	return forestRoot
}

// renderASCIITree renders the tree with box-drawing connectors, matching the
// go-output ASCII tree renderer's output (no color; color required a TTY which
// file/string captures never have).
func renderASCIITree(root *treeNode) string {
	if root == nil {
		return ""
	}

	var b strings.Builder

	renderASCIINode(&b, root, "", true)

	return b.String()
}

// renderASCIINode recursively writes one node and its children.
func renderASCIINode(b *strings.Builder, node *treeNode, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	b.WriteString(prefix)
	b.WriteString(connector)
	b.WriteString(node.Label)
	b.WriteString("\n")

	childPrefix := prefix + "│   "
	if isLast {
		childPrefix = prefix + "    "
	}

	for i, child := range node.Children {
		renderASCIINode(b, child, childPrefix, i == len(node.Children)-1)
	}
}

// htmlTreeTemplate renders the tree as nested HTML lists, ported verbatim
// from the go-output markup.HTMLTreeRenderer (html/template auto-escapes
// labels).
//
//nolint:gochecknoglobals // Parsed once at package init; immutable.
var htmlTreeTemplate = template.Must(template.New("treeNode").Parse(
	`<ul class="tree">
{{template "treeNodeRec" .}}
</ul>
` + treeNodeRecTemplate,
))

// treeNodeRecTemplate is the recursive body of the HTML tree template.
const treeNodeRecTemplate = `{{define "treeNodeRec"}}<li>{{.Label}}
{{- if .Children}}
<ul>
{{- range .Children}}
{{template "treeNodeRec" .}}
{{- end}}
</ul>
{{- end}}
</li>
{{end}}`

// renderHTMLTree renders the tree as an HTML nested list.
func renderHTMLTree(root *treeNode) (string, error) {
	if root == nil {
		return `<ul class="tree"></ul>`, nil
	}

	var b strings.Builder

	if err := htmlTreeTemplate.Execute(&b, root); err != nil {
		return "", fmt.Errorf("render html tree: %w", err)
	}

	return b.String(), nil
}

// WriteTree writes the service dependency DAG as an ASCII tree.
// Nodes are labeled with service name and provider-type icon.
func (r Report) WriteTree(writer io.Writer) error {
	out := renderASCIITree(r.buildServiceTreeNodes())

	if _, err := fmt.Fprintln(writer, out); err != nil {
		return fmt.Errorf("write tree output: %w", err)
	}

	return nil
}

// WriteHTMLTree writes the service dependency DAG as an HTML nested list tree.
// Nodes are labeled with service name and provider-type icon.
func (r Report) WriteHTMLTree(writer io.Writer) error {
	out, err := renderHTMLTree(r.buildServiceTreeNodes())
	if err != nil {
		return fmt.Errorf("render tree: %w", err)
	}

	if _, err := fmt.Fprintln(writer, out); err != nil {
		return fmt.Errorf("write tree output: %w", err)
	}

	return nil
}

// WriteTreeString returns the ASCII tree as a string.
func (r Report) WriteTreeString() (string, error) {
	return writeTreeToString(r.WriteTree)
}

// WriteHTMLTreeString returns the HTML tree as a string.
func (r Report) WriteHTMLTreeString() (string, error) {
	return writeTreeToString(r.WriteHTMLTree)
}

// writeTreeToString writes a tree to a strings.Builder via the supplied writer
// function and returns the resulting string. Centralizes the shared
// Builder + write + return idiom of the String() helpers.
func writeTreeToString(write func(io.Writer) error) (string, error) {
	var buf strings.Builder

	if err := write(&buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

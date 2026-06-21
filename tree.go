package auditlog

import (
	"fmt"
	"io"
	"strings"

	"github.com/larsartmann/go-output"
	"github.com/larsartmann/go-output/markup"
	"github.com/larsartmann/go-output/tree"
)

// addTreeChildren recursively adds dependent services as children to the parent
// TreeNode, using the provided lookup map and visited set to avoid cycles.
func addTreeChildren(
	parent *output.TreeNode,
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

		childNode := output.NewTreeNode(
			diagramNodeID(childSvc.ScopeID, childSvc.ServiceName),
			serviceLabel(childSvc),
		)
		parent.AddChild(childNode)
		addTreeChildren(childNode, childSvc, byKey, visited)
	}
}

// buildServiceTreeNodes constructs a forest of TreeNodes from the service
// dependency graph. Root nodes are services with no dependencies; children are
// their dependents (services that depend on the parent). The result is wrapped
// in a single root node for the renderer.
func (r Report) buildServiceTreeNodes() *output.TreeNode {
	title := r.ContainerID
	if title == "" {
		title = "container"
	}

	forestRoot := output.NewTreeNode("container", title)

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
		rootNode := output.NewTreeNode(
			diagramNodeID(rootSvc.ScopeID, rootSvc.ServiceName),
			serviceLabel(rootSvc),
		)
		forestRoot.AddChild(rootNode)
		addTreeChildren(rootNode, rootSvc, byKey, visited)
	}

	return forestRoot
}

// WriteTree writes the service dependency DAG as an ASCII tree.
// Nodes are labeled with service name and provider-type icon.
func (r Report) WriteTree(writer io.Writer) error {
	root := r.buildServiceTreeNodes()

	renderer := tree.NewASCIITreeRenderer()
	renderer.SetRoot(root)

	out, err := renderer.Render()
	if err != nil {
		return fmt.Errorf("render tree: %w", err)
	}

	_, err = fmt.Fprintln(writer, out)
	if err != nil {
		return fmt.Errorf("write tree output: %w", err)
	}

	return nil
}

// WriteHTMLTree writes the service dependency DAG as an HTML nested list tree.
// Nodes are labeled with service name and provider-type icon.
func (r Report) WriteHTMLTree(writer io.Writer) error {
	root := r.buildServiceTreeNodes()

	renderer := markup.NewHTMLTreeRenderer()
	renderer.SetRoot(root)

	out, err := renderer.Render()
	if err != nil {
		return fmt.Errorf("render html tree: %w", err)
	}

	_, err = fmt.Fprintln(writer, out)
	if err != nil {
		return fmt.Errorf("write html tree output: %w", err)
	}

	return nil
}

// WriteTreeString returns the ASCII tree as a string.
func (r Report) WriteTreeString() (string, error) {
	var buf strings.Builder

	err := r.WriteTree(&buf)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// WriteHTMLTreeString returns the HTML tree as a string.
func (r Report) WriteHTMLTreeString() (string, error) {
	var buf strings.Builder

	err := r.WriteHTMLTree(&buf)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

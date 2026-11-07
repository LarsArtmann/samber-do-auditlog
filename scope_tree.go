package auditlog

import (
	"cmp"
	"slices"
)

func (r *Recorder) buildScopeTreeLocked() ScopeNode {
	sortedScopes := sortedScopes(r.scopes)

	return buildScopeTreeFromMeta(
		sortedScopes,
		scopeMetaID, scopeMetaName, scopeMetaParentID,
		scopeServicesForServices(r.services),
	)
}

// scopeMetaID, scopeMetaName, scopeMetaParentID are field accessors for scopeMeta.
func scopeMetaID(m scopeMeta) ScopeID       { return m.id }
func scopeMetaName(m scopeMeta) string      { return m.name }
func scopeMetaParentID(m scopeMeta) ScopeID { return m.parentID }

// scopeServicesForServices groups service names by their scopeID.
func scopeServicesForServices(services map[svcKey]*serviceRecord) map[ScopeID][]ServiceName {
	scopeServices := make(map[ScopeID][]ServiceName)
	for _, rec := range services {
		scopeServices[rec.scopeID] = append(scopeServices[rec.scopeID], rec.serviceName)
	}

	for id, names := range scopeServices {
		slices.Sort(names)
		scopeServices[id] = names
	}

	return scopeServices
}

// findRootScope returns the first meta with an empty parentID, or false
// if none found. Sorted iteration keeps the result deterministic.
func findRootScope[T any](sorted []T, parentOf func(T) ScopeID) (T, bool) {
	for _, meta := range sorted {
		if parentOf(meta) == "" {
			return meta, true
		}
	}

	var zero T

	return zero, false
}

// buildScopeChildren constructs the child scope tree below parentID. The
// cycle guard (metaID(meta) != parentID) prevents infinite recursion on
// self-referential entries where both IDs are empty.
func buildScopeChildren[T any](
	parentID ScopeID,
	sorted []T,
	metaID func(T) ScopeID,
	metaName func(T) string,
	metaParent func(T) ScopeID,
	scopeServices map[ScopeID][]ServiceName,
) []ScopeNode {
	var children []ScopeNode

	for _, meta := range sorted {
		if metaParent(meta) != parentID {
			continue
		}

		if metaID(meta) == parentID {
			continue
		}

		scopeID := metaID(meta)

		children = append(children, ScopeNode{
			ID:       scopeID,
			Name:     metaName(meta),
			Services: scopeServices[scopeID],
			Children: buildScopeChildren(scopeID, sorted, metaID, metaName, metaParent, scopeServices),
		})
	}

	return children
}

// buildScopeTreeFromMeta assembles a ScopeNode tree from sorted scope
// metadata using the provided field accessors. The first scope with an
// empty parentID is the root; remaining scopes become children of
// whichever scope matches their parentID.
func buildScopeTreeFromMeta[T any](
	sorted []T,
	metaID func(T) ScopeID,
	metaName func(T) string,
	metaParent func(T) ScopeID,
	scopeServices map[ScopeID][]ServiceName,
) ScopeNode {
	if len(sorted) == 0 {
		return ScopeNode{} //nolint:exhaustruct_v5
	}

	root, ok := findRootScope(sorted, metaParent)
	if !ok {
		root = sorted[0]
	}

	rootID := metaID(root)

	return ScopeNode{
		ID:       rootID,
		Name:     metaName(root),
		Services: scopeServices[rootID],
		Children: sortScopeNodes(buildScopeChildren(rootID, sorted, metaID, metaName, metaParent, scopeServices)),
	}
}

func sortedScopes(scopes map[ScopeID]scopeMeta) []scopeMeta {
	result := make([]scopeMeta, 0, len(scopes))

	for _, meta := range scopes {
		result = append(result, meta)
	}

	slices.SortFunc(result, func(a, b scopeMeta) int {
		return cmp.Compare(a.id, b.id)
	})

	return result
}

func sortScopeNodes(nodes []ScopeNode) []ScopeNode {
	slices.SortFunc(nodes, func(a, b ScopeNode) int {
		return cmp.Compare(a.Name, b.Name)
	})

	for i := range nodes {
		nodes[i].Children = sortScopeNodes(nodes[i].Children)
	}

	return nodes
}

// --- Capability enrichment (samber/do ExplainInjector) ---

// enrichCapabilities populates IsHealthchecker and IsShutdowner on each ServiceInfo
// by calling do.ExplainInjector on each stored scope reference. Must be called
// outside the recorder mutex to avoid deadlocking with samber/do's internal locks.

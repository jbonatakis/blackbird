package plan

import "sort"

// TaskTree captures ordered parent/child relationships for rendering.
type TaskTree struct {
	Roots    []string
	Children map[string][]string
}

// BuildTaskTree derives a hierarchy from parentId references and orders siblings
// deterministically, using parent childIds ordering when available.
func BuildTaskTree(g WorkGraph) TaskTree {
	roots := make([]string, 0)
	children := map[string][]string{}

	for id, it := range g.Items {
		parentID := ""
		if it.ParentID != nil {
			parentID = *it.ParentID
		}
		if parentID == "" {
			roots = append(roots, id)
			continue
		}
		if _, ok := g.Items[parentID]; !ok {
			roots = append(roots, id)
			continue
		}
		children[parentID] = append(children[parentID], id)
	}

	sort.Strings(roots)

	for parentID, ids := range children {
		parent, ok := g.Items[parentID]
		if !ok {
			sort.Strings(ids)
			children[parentID] = ids
			continue
		}
		children[parentID] = orderChildren(parent.ChildIDs, ids)
	}

	return TaskTree{
		Roots:    roots,
		Children: children,
	}
}

func orderChildren(preferred []string, actual []string) []string {
	if len(actual) <= 1 {
		return append([]string{}, actual...)
	}

	remaining := make(map[string]bool, len(actual))
	for _, id := range actual {
		remaining[id] = true
	}

	out := make([]string, 0, len(actual))
	seen := map[string]bool{}
	for _, id := range preferred {
		if remaining[id] && !seen[id] {
			out = append(out, id)
			delete(remaining, id)
			seen[id] = true
		}
	}

	if len(remaining) == 0 {
		return out
	}

	rest := make([]string, 0, len(remaining))
	for id := range remaining {
		rest = append(rest, id)
	}
	sort.Strings(rest)
	out = append(out, rest...)
	return out
}

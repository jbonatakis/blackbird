package plan

import (
	"sort"
)

type DepEdge struct {
	From string
	To   string
}

type DiffSummary struct {
	Added       []string
	Removed     []string
	Updated     []string
	Moved       []string
	DepsAdded   []DepEdge
	DepsRemoved []DepEdge
}

func Diff(before, after WorkGraph) DiffSummary {
	summary := DiffSummary{}

	beforeIDs := map[string]WorkItem{}
	afterIDs := map[string]WorkItem{}
	for id, it := range before.Items {
		beforeIDs[id] = it
	}
	for id, it := range after.Items {
		afterIDs[id] = it
	}

	for id := range afterIDs {
		if _, ok := beforeIDs[id]; !ok {
			summary.Added = append(summary.Added, id)
		}
	}
	for id := range beforeIDs {
		if _, ok := afterIDs[id]; !ok {
			summary.Removed = append(summary.Removed, id)
		}
	}

	for id, beforeItem := range beforeIDs {
		afterItem, ok := afterIDs[id]
		if !ok {
			continue
		}
		if parentChanged(beforeItem.ParentID, afterItem.ParentID) {
			summary.Moved = append(summary.Moved, id)
		}
		if !itemContentEqual(beforeItem, afterItem) {
			summary.Updated = append(summary.Updated, id)
		}
	}

	beforeEdges := depEdges(before)
	afterEdges := depEdges(after)
	for edge := range afterEdges {
		if !beforeEdges[edge] {
			summary.DepsAdded = append(summary.DepsAdded, parseEdge(edge))
		}
	}
	for edge := range beforeEdges {
		if !afterEdges[edge] {
			summary.DepsRemoved = append(summary.DepsRemoved, parseEdge(edge))
		}
	}

	sort.Strings(summary.Added)
	sort.Strings(summary.Removed)
	sort.Strings(summary.Updated)
	sort.Strings(summary.Moved)
	sort.Slice(summary.DepsAdded, func(i, j int) bool {
		if summary.DepsAdded[i].From == summary.DepsAdded[j].From {
			return summary.DepsAdded[i].To < summary.DepsAdded[j].To
		}
		return summary.DepsAdded[i].From < summary.DepsAdded[j].From
	})
	sort.Slice(summary.DepsRemoved, func(i, j int) bool {
		if summary.DepsRemoved[i].From == summary.DepsRemoved[j].From {
			return summary.DepsRemoved[i].To < summary.DepsRemoved[j].To
		}
		return summary.DepsRemoved[i].From < summary.DepsRemoved[j].From
	})

	return summary
}

func parentChanged(a, b *string) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil || b == nil {
		return true
	}
	return *a != *b
}

func itemContentEqual(a, b WorkItem) bool {
	if a.Title != b.Title ||
		a.Description != b.Description ||
		a.Prompt != b.Prompt ||
		a.Status != b.Status {
		return false
	}
	if !stringSliceEqual(a.AcceptanceCriteria, b.AcceptanceCriteria) {
		return false
	}
	if !stringSliceEqual(a.Deps, b.Deps) {
		return false
	}
	if !notesEqual(a.Notes, b.Notes) {
		return false
	}
	if !rationaleEqual(a.DepRationale, b.DepRationale) {
		return false
	}
	return true
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func notesEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func rationaleEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func depEdges(g WorkGraph) map[string]bool {
	out := map[string]bool{}
	for id, it := range g.Items {
		for _, depID := range it.Deps {
			if depID == "" {
				continue
			}
			out[id+"->"+depID] = true
		}
	}
	return out
}

func parseEdge(edge string) DepEdge {
	for i := 0; i < len(edge); i++ {
		if edge[i] == '-' && i+1 < len(edge) && edge[i+1] == '>' {
			return DepEdge{From: edge[:i], To: edge[i+2:]}
		}
	}
	return DepEdge{From: edge, To: ""}
}

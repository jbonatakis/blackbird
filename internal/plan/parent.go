package plan

import "time"

// PropagateParentCompletion sets any parent to done when all of its children are done,
// and recurses up the hierarchy so grandparents are updated too. Call this after
// setting an item's status to done so that tasks depending on parent containers
// (e.g. a top-level task that depends on "chess-core" and "cli-interface") can
// become ready when all children of those parents are complete.
// g is mutated in place.
func PropagateParentCompletion(g *WorkGraph, childID string, now time.Time) {
	if g == nil || g.Items == nil {
		return
	}
	it, ok := g.Items[childID]
	if !ok {
		return
	}
	if it.ParentID == nil || *it.ParentID == "" {
		return
	}
	parentID := *it.ParentID
	parent, ok := g.Items[parentID]
	if !ok {
		return
	}
	for _, cid := range parent.ChildIDs {
		if c, ok := g.Items[cid]; !ok || c.Status != StatusDone {
			return
		}
	}
	parent.Status = StatusDone
	parent.UpdatedAt = now
	g.Items[parentID] = parent
	PropagateParentCompletion(g, parentID, now)
}

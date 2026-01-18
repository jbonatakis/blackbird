package plan

import (
	"fmt"
	"sort"
	"time"
)

type DepCycleError struct {
	Cycle []string
}

func (e DepCycleError) Error() string {
	return fmt.Sprintf("dependency cycle detected: %s", joinCycle(e.Cycle))
}

type HierarchyCycleError struct {
	Cycle []string
}

func (e HierarchyCycleError) Error() string {
	return fmt.Sprintf("hierarchy cycle detected: %s", joinCycle(e.Cycle))
}

func AddItem(g *WorkGraph, it WorkItem, parentID *string, index *int, now time.Time) error {
	if g.Items == nil {
		g.Items = map[string]WorkItem{}
	}
	if it.ID == "" {
		return fmt.Errorf("id is required")
	}
	if _, ok := g.Items[it.ID]; ok {
		return fmt.Errorf("id already exists: %q", it.ID)
	}
	if it.Title == "" {
		return fmt.Errorf("title is required")
	}

	it.ParentID = nil
	if parentID != nil && *parentID != "" {
		if _, ok := g.Items[*parentID]; !ok {
			return fmt.Errorf("unknown parent id %q", *parentID)
		}
		pid := *parentID
		it.ParentID = &pid
	}

	// Ensure required slices exist.
	if it.AcceptanceCriteria == nil {
		it.AcceptanceCriteria = []string{}
	}
	if it.ChildIDs == nil {
		it.ChildIDs = []string{}
	}
	if it.Deps == nil {
		it.Deps = []string{}
	}
	it.CreatedAt = now
	it.UpdatedAt = now

	g.Items[it.ID] = it

	// Link into parent if needed.
	if it.ParentID != nil && *it.ParentID != "" {
		parent := g.Items[*it.ParentID]
		children, err := insertID(parent.ChildIDs, it.ID, index)
		if err != nil {
			delete(g.Items, it.ID)
			return err
		}
		parent.ChildIDs = children
		parent.UpdatedAt = now
		g.Items[parent.ID] = parent
	}

	return nil
}

func MoveItem(g *WorkGraph, id string, newParentID *string, index *int, now time.Time) error {
	it, ok := g.Items[id]
	if !ok {
		return fmt.Errorf("unknown id %q", id)
	}

	var oldParentID *string
	if it.ParentID != nil && *it.ParentID != "" {
		pid := *it.ParentID
		oldParentID = &pid
	}
	var oldParent planWorkItemSnapshot
	if oldParentID != nil {
		oldParent = snapshot(g.Items[*oldParentID])
	}

	var nextParentID *string
	if newParentID != nil && *newParentID != "" {
		if _, ok := g.Items[*newParentID]; !ok {
			return fmt.Errorf("unknown parent id %q", *newParentID)
		}
		if *newParentID == id {
			return HierarchyCycleError{Cycle: []string{id, id}}
		}
		pid := *newParentID
		nextParentID = &pid
	}

	// Reject hierarchy cycles preemptively for a clearer error.
	if nextParentID != nil {
		if cycle := parentCycleIfMove(g, id, *nextParentID); len(cycle) > 0 {
			return HierarchyCycleError{Cycle: cycle}
		}
	}

	// Detach from old parent.
	if oldParentID != nil {
		op := g.Items[*oldParentID]
		newChildren, _ := removeID(op.ChildIDs, id)
		op.ChildIDs = newChildren
		op.UpdatedAt = now
		g.Items[op.ID] = op
	}

	// Attach to new parent.
	it.ParentID = nil
	if nextParentID != nil && *nextParentID != "" {
		parent := g.Items[*nextParentID]
		children, err := insertID(parent.ChildIDs, id, index)
		if err != nil {
			// rollback old parent detach for a clean error.
			if oldParentID != nil {
				restore(g, oldParent)
				pid := *oldParentID
				it.ParentID = &pid
			}
			return err
		}
		parent.ChildIDs = children
		parent.UpdatedAt = now
		g.Items[parent.ID] = parent
		pid := *nextParentID
		it.ParentID = &pid
	}

	it.UpdatedAt = now
	g.Items[id] = it

	// Final safety net.
	if errs := Validate(*g); len(errs) != 0 {
		return fmt.Errorf("invalid move (would violate plan invariants)")
	}
	return nil
}

type planWorkItemSnapshot struct {
	ID   string
	Item WorkItem
}

func snapshot(it WorkItem) planWorkItemSnapshot {
	return planWorkItemSnapshot{ID: it.ID, Item: it}
}

func restore(g *WorkGraph, snap planWorkItemSnapshot) {
	if snap.ID == "" {
		return
	}
	g.Items[snap.ID] = snap.Item
}

func AddDep(g *WorkGraph, id string, depID string, now time.Time) error {
	it, ok := g.Items[id]
	if !ok {
		return fmt.Errorf("unknown id %q", id)
	}
	if _, ok := g.Items[depID]; !ok {
		return fmt.Errorf("unknown dep id %q", depID)
	}
	if depID == id {
		return fmt.Errorf("cannot add self-dependency: %q", id)
	}

	if containsID(it.Deps, depID) {
		return nil
	}
	before := append([]string{}, it.Deps...)
	it.Deps = append(it.Deps, depID)
	it.UpdatedAt = now
	g.Items[id] = it

	if cycle := DepCycle(*g); len(cycle) > 0 {
		// rollback
		it.Deps = before
		it.UpdatedAt = now
		g.Items[id] = it
		return DepCycleError{Cycle: cycle}
	}
	return nil
}

func RemoveDep(g *WorkGraph, id string, depID string, now time.Time) error {
	it, ok := g.Items[id]
	if !ok {
		return fmt.Errorf("unknown id %q", id)
	}
	if depID == "" {
		return fmt.Errorf("dep id must be non-empty")
	}

	next, removed := removeID(it.Deps, depID)
	if !removed {
		return nil
	}
	it.Deps = next
	if it.DepRationale != nil {
		delete(it.DepRationale, depID)
		if len(it.DepRationale) == 0 {
			it.DepRationale = nil
		}
	}
	it.UpdatedAt = now
	g.Items[id] = it

	// Removing deps cannot introduce cycles; still ensure structural validity.
	if errs := Validate(*g); len(errs) != 0 {
		return fmt.Errorf("invalid dep removal (would violate plan invariants)")
	}
	return nil
}

func SetDeps(g *WorkGraph, id string, deps []string, now time.Time) error {
	it, ok := g.Items[id]
	if !ok {
		return fmt.Errorf("unknown id %q", id)
	}

	seen := map[string]bool{}
	out := make([]string, 0, len(deps))
	for _, depID := range deps {
		if depID == "" {
			return fmt.Errorf("dep id must be non-empty")
		}
		if depID == id {
			return fmt.Errorf("cannot add self-dependency: %q", id)
		}
		if seen[depID] {
			continue
		}
		seen[depID] = true
		if _, ok := g.Items[depID]; !ok {
			return fmt.Errorf("unknown dep id %q", depID)
		}
		out = append(out, depID)
	}

	beforeDeps := append([]string{}, it.Deps...)
	beforeRationale := copyRationale(it.DepRationale)

	it.Deps = out
	if it.DepRationale != nil {
		// Drop rationale for deps no longer present.
		for depID := range it.DepRationale {
			if !seen[depID] {
				delete(it.DepRationale, depID)
			}
		}
		if len(it.DepRationale) == 0 {
			it.DepRationale = nil
		}
	}
	it.UpdatedAt = now
	g.Items[id] = it

	if cycle := DepCycle(*g); len(cycle) > 0 {
		// rollback
		it.Deps = beforeDeps
		it.DepRationale = beforeRationale
		it.UpdatedAt = now
		g.Items[id] = it
		return DepCycleError{Cycle: cycle}
	}
	return nil
}

type DeleteResult struct {
	DeletedIDs  []string
	UpdatedIDs  []string
	DetachedIDs []string // ids whose deps were removed due to --force
}

func DeleteItem(g *WorkGraph, id string, cascadeChildren bool, force bool, now time.Time) (DeleteResult, error) {
	it, ok := g.Items[id]
	if !ok {
		return DeleteResult{}, fmt.Errorf("unknown id %q", id)
	}
	if !cascadeChildren && len(it.ChildIDs) > 0 {
		return DeleteResult{}, fmt.Errorf("refusing to delete %q: has children (use --cascade-children)", id)
	}

	toDeleteSet := map[string]bool{id: true}
	if cascadeChildren {
		for _, sid := range subtreeIDs(*g, id) {
			toDeleteSet[sid] = true
		}
	}

	// Find dependents outside the deletion set.
	dependentMap := map[string][]string{} // deletedID -> dependents
	for deletedID := range toDeleteSet {
		for _, dep := range Dependents(*g, deletedID) {
			if !toDeleteSet[dep] {
				dependentMap[deletedID] = append(dependentMap[deletedID], dep)
			}
		}
	}
	if !force {
		var blockedOn []string
		for deletedID, deps := range dependentMap {
			if len(deps) == 0 {
				continue
			}
			blockedOn = append(blockedOn, fmt.Sprintf("%s (dependents: %s)", deletedID, joinComma(deps)))
		}
		sort.Strings(blockedOn)
		if len(blockedOn) > 0 {
			return DeleteResult{}, fmt.Errorf("refusing to delete: would break dependents (%s). Use --force to remove those dep edges", joinComma(blockedOn))
		}
	}

	res := DeleteResult{}

	// If force, remove dep edges from remaining nodes.
	if force {
		for deletedID, dependents := range dependentMap {
			for _, depID := range dependents {
				depItem := g.Items[depID]
				next, removed := removeID(depItem.Deps, deletedID)
				if removed {
					depItem.Deps = next
					if depItem.DepRationale != nil {
						delete(depItem.DepRationale, deletedID)
						if len(depItem.DepRationale) == 0 {
							depItem.DepRationale = nil
						}
					}
					depItem.UpdatedAt = now
					g.Items[depID] = depItem
					res.DetachedIDs = append(res.DetachedIDs, depID)
				}
			}
		}
	}

	// Detach from parents and delete nodes.
	for deletedID := range toDeleteSet {
		d := g.Items[deletedID]
		if d.ParentID != nil && *d.ParentID != "" && !toDeleteSet[*d.ParentID] {
			parent := g.Items[*d.ParentID]
			next, removed := removeID(parent.ChildIDs, deletedID)
			if removed {
				parent.ChildIDs = next
				parent.UpdatedAt = now
				g.Items[parent.ID] = parent
				res.UpdatedIDs = append(res.UpdatedIDs, parent.ID)
			}
		}
		delete(g.Items, deletedID)
		res.DeletedIDs = append(res.DeletedIDs, deletedID)
	}

	sort.Strings(res.DeletedIDs)
	sort.Strings(res.UpdatedIDs)
	sort.Strings(res.DetachedIDs)

	if errs := Validate(*g); len(errs) != 0 {
		return DeleteResult{}, fmt.Errorf("invalid delete (would violate plan invariants)")
	}

	return res, nil
}

func parentCycleIfMove(g *WorkGraph, movingID string, newParentID string) []string {
	chain := []string{newParentID}
	cur := newParentID
	for {
		it, ok := g.Items[cur]
		if !ok {
			return nil
		}
		if it.ParentID == nil || *it.ParentID == "" {
			return nil
		}
		parent := *it.ParentID
		if parent == movingID {
			// cycle is movingID -> chain... -> movingID
			cycle := append([]string{movingID}, chain...)
			cycle = append(cycle, movingID)
			return cycle
		}
		chain = append(chain, parent)
		cur = parent
	}
}

func subtreeIDs(g WorkGraph, rootID string) []string {
	root, ok := g.Items[rootID]
	if !ok {
		return nil
	}
	var out []string
	queue := append([]string{}, root.ChildIDs...)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if id == "" {
			continue
		}
		out = append(out, id)
		it, ok := g.Items[id]
		if !ok {
			continue
		}
		queue = append(queue, it.ChildIDs...)
	}
	return out
}

func insertID(ss []string, id string, index *int) ([]string, error) {
	if id == "" {
		return ss, fmt.Errorf("id must be non-empty")
	}
	if containsID(ss, id) {
		return ss, nil
	}
	if index == nil {
		return append(ss, id), nil
	}
	if *index < 0 || *index > len(ss) {
		return ss, fmt.Errorf("index out of range: %d (valid: 0..%d)", *index, len(ss))
	}
	out := make([]string, 0, len(ss)+1)
	out = append(out, ss[:*index]...)
	out = append(out, id)
	out = append(out, ss[*index:]...)
	return out, nil
}

func removeID(ss []string, id string) ([]string, bool) {
	for i := range ss {
		if ss[i] == id {
			out := make([]string, 0, len(ss)-1)
			out = append(out, ss[:i]...)
			out = append(out, ss[i+1:]...)
			return out, true
		}
	}
	return ss, false
}

func containsID(ss []string, id string) bool {
	for _, s := range ss {
		if s == id {
			return true
		}
	}
	return false
}

func joinComma(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	out := ss[0]
	for i := 1; i < len(ss); i++ {
		out += ", " + ss[i]
	}
	return out
}

func copyRationale(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

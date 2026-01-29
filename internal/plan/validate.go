package plan

import (
	"fmt"
)

type ValidationError struct {
	Path    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

func Validate(g WorkGraph) []ValidationError {
	var errs []ValidationError

	if g.SchemaVersion == 0 {
		errs = append(errs, ValidationError{Path: "$.schemaVersion", Message: "required"})
	} else if g.SchemaVersion != SchemaVersion {
		errs = append(errs, ValidationError{
			Path:    "$.schemaVersion",
			Message: fmt.Sprintf("unsupported schemaVersion %d (expected %d)", g.SchemaVersion, SchemaVersion),
		})
	}

	if g.Items == nil {
		errs = append(errs, ValidationError{Path: "$.items", Message: "required (must be an object/map)"})
		return errs
	}

	// Validate basic item invariants, and build helper maps.
	for key, it := range g.Items {
		path := fmt.Sprintf("$.items[%q]", key)

		if key == "" {
			errs = append(errs, ValidationError{Path: "$.items", Message: "item key must be non-empty"})
		}

		// Required fields: present + basic constraints.
		if it.ID == "" {
			errs = append(errs, ValidationError{Path: path + ".id", Message: "required"})
		}
		if it.ID != "" && it.ID != key {
			errs = append(errs, ValidationError{Path: path + ".id", Message: fmt.Sprintf("must match map key %q", key)})
		}

		if it.Title == "" {
			errs = append(errs, ValidationError{Path: path + ".title", Message: "required"})
		}

		// description may be empty, but field must exist (json decoding covers existence).
		// prompt may be empty, but field must exist.

		if it.AcceptanceCriteria == nil {
			errs = append(errs, ValidationError{Path: path + ".acceptanceCriteria", Message: "required (use [] if none)"})
		}
		if it.ChildIDs == nil {
			errs = append(errs, ValidationError{Path: path + ".childIds", Message: "required (use [] if none)"})
		}
		if it.Deps == nil {
			errs = append(errs, ValidationError{Path: path + ".deps", Message: "required (use [] if none)"})
		}

		if !isValidStatus(it.Status) {
			errs = append(errs, ValidationError{
				Path:    path + ".status",
				Message: fmt.Sprintf("invalid status %q", it.Status),
			})
		}

		if it.CreatedAt.IsZero() {
			errs = append(errs, ValidationError{Path: path + ".createdAt", Message: "required (RFC3339 timestamp)"})
		}
		if it.UpdatedAt.IsZero() {
			errs = append(errs, ValidationError{Path: path + ".updatedAt", Message: "required (RFC3339 timestamp)"})
		}
		if !it.CreatedAt.IsZero() && !it.UpdatedAt.IsZero() && it.UpdatedAt.Before(it.CreatedAt) {
			errs = append(errs, ValidationError{Path: path + ".updatedAt", Message: "must be >= createdAt"})
		}
	}

	// Reference existence and parent/children consistency.
	for id, it := range g.Items {
		path := fmt.Sprintf("$.items[%q]", id)

		// parent reference exists
		if it.ParentID != nil {
			if *it.ParentID == "" {
				errs = append(errs, ValidationError{Path: path + ".parentId", Message: "must be null or a non-empty ID"})
			} else if _, ok := g.Items[*it.ParentID]; !ok {
				errs = append(errs, ValidationError{Path: path + ".parentId", Message: fmt.Sprintf("unknown parent id %q", *it.ParentID)})
			}
		}

		// children references exist; no duplicates
		seenChild := map[string]bool{}
		for i, childID := range it.ChildIDs {
			cpath := fmt.Sprintf("%s.childIds[%d]", path, i)
			if childID == "" {
				errs = append(errs, ValidationError{Path: cpath, Message: "child id must be non-empty"})
				continue
			}
			if seenChild[childID] {
				errs = append(errs, ValidationError{Path: cpath, Message: fmt.Sprintf("duplicate child id %q", childID)})
				continue
			}
			seenChild[childID] = true

			child, ok := g.Items[childID]
			if !ok {
				errs = append(errs, ValidationError{Path: cpath, Message: fmt.Sprintf("unknown child id %q", childID)})
				continue
			}
			if child.ParentID == nil || *child.ParentID != id {
				errs = append(errs, ValidationError{
					Path:    cpath,
					Message: fmt.Sprintf("child %q must have parentId %q", childID, id),
				})
			}
		}

		// deps references exist; no duplicates
		seenDep := map[string]bool{}
		for i, depID := range it.Deps {
			dpath := fmt.Sprintf("%s.deps[%d]", path, i)
			if depID == "" {
				errs = append(errs, ValidationError{Path: dpath, Message: "dep id must be non-empty"})
				continue
			}
			if seenDep[depID] {
				errs = append(errs, ValidationError{Path: dpath, Message: fmt.Sprintf("duplicate dep id %q", depID)})
				continue
			}
			seenDep[depID] = true

			if _, ok := g.Items[depID]; !ok {
				errs = append(errs, ValidationError{Path: dpath, Message: fmt.Sprintf("unknown dep id %q", depID)})
			}
		}

		// If depRationale is present, ensure keys refer to existing deps.
		for depID := range it.DepRationale {
			if _, ok := g.Items[depID]; !ok {
				errs = append(errs, ValidationError{Path: path + ".depRationale", Message: fmt.Sprintf("unknown dep id %q", depID)})
				continue
			}
			if !contains(it.Deps, depID) {
				errs = append(errs, ValidationError{
					Path:    path + ".depRationale",
					Message: fmt.Sprintf("depRationale key %q must also appear in deps", depID),
				})
			}
		}
	}

	// Ensure every item's parent has this item in its childIds (when parentId is set).
	for id, it := range g.Items {
		if it.ParentID == nil || *it.ParentID == "" {
			continue
		}
		parent, ok := g.Items[*it.ParentID]
		if !ok {
			// already reported above
			continue
		}
		if !contains(parent.ChildIDs, id) {
			errs = append(errs, ValidationError{
				Path:    fmt.Sprintf("$.items[%q].parentId", id),
				Message: fmt.Sprintf("parent %q must include %q in childIds", *it.ParentID, id),
			})
		}
	}

	// Hierarchy cycle detection using parent pointers (each node has at most one parent).
	// Detect a cycle like A.parent=B, B.parent=C, C.parent=A.
	state := map[string]visitState{} // per node
	for id := range g.Items {
		if state[id] == visitDone {
			continue
		}
		if cycle := findParentCycle(id, g.Items, state); len(cycle) > 0 {
			errs = append(errs, ValidationError{
				Path:    "$.items",
				Message: fmt.Sprintf("hierarchy cycle detected: %s", joinCycle(cycle)),
			})
			// One cycle is enough to fail; keep going to report other errors too.
		}
	}

	// Dependency DAG cycle detection.
	if cycle := DepCycle(g); len(cycle) > 0 {
		errs = append(errs, ValidationError{
			Path:    "$.items",
			Message: fmt.Sprintf("dependency cycle detected: %s", joinCycle(cycle)),
		})
	}

	return errs
}

type visitState uint8

const (
	visitNew visitState = iota
	visitVisiting
	visitDone
)

func findParentCycle(start string, items map[string]WorkItem, state map[string]visitState) []string {
	var stack []string
	onStack := map[string]int{} // id -> index in stack

	cur := start
	for {
		if state[cur] == visitDone {
			for _, n := range stack {
				state[n] = visitDone
			}
			return nil
		}

		if idx, ok := onStack[cur]; ok {
			// cycle is stack[idx:] plus cur at end to show closure
			cycle := append([]string{}, stack[idx:]...)
			cycle = append(cycle, cur)
			for _, n := range stack {
				state[n] = visitDone
			}
			return cycle
		}

		state[cur] = visitVisiting
		onStack[cur] = len(stack)
		stack = append(stack, cur)

		it, ok := items[cur]
		if !ok {
			for _, n := range stack {
				state[n] = visitDone
			}
			return nil
		}
		if it.ParentID == nil || *it.ParentID == "" {
			for _, n := range stack {
				state[n] = visitDone
			}
			return nil
		}

		cur = *it.ParentID
	}
}

func joinCycle(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	out := ids[0]
	for i := 1; i < len(ids); i++ {
		out += " -> " + ids[i]
	}
	return out
}

func contains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}

func isValidStatus(s Status) bool {
	switch s {
	case StatusTodo, StatusQueued, StatusInProgress, StatusWaitingUser, StatusBlocked, StatusDone, StatusFailed, StatusSkipped:
		return true
	default:
		return false
	}
}

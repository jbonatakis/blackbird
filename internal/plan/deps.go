package plan

import "sort"

// Dependents returns all item IDs that directly depend on id (reverse deps).
// Output is sorted for stable display.
func Dependents(g WorkGraph, id string) []string {
	out := make([]string, 0)
	for otherID, it := range g.Items {
		for _, depID := range it.Deps {
			if depID == id {
				out = append(out, otherID)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

// UnmetDeps returns prerequisite IDs whose status is not done.
// The result preserves the order of it.Deps.
func UnmetDeps(g WorkGraph, it WorkItem) []string {
	if len(it.Deps) == 0 {
		return nil
	}
	out := make([]string, 0, len(it.Deps))
	for _, depID := range it.Deps {
		dep, ok := g.Items[depID]
		if !ok {
			// Validation will report unknown deps; treat as unmet here too.
			out = append(out, depID)
			continue
		}
		if dep.Status != StatusDone {
			out = append(out, depID)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func depsSatisfied(g WorkGraph, it WorkItem) bool {
	return len(UnmetDeps(g, it)) == 0
}

// DepCycle returns a dependency cycle path if one exists, else nil.
//
// The returned slice includes the starting node again at the end to show closure,
// e.g. ["A", "B", "C", "A"].
func DepCycle(g WorkGraph) []string {
	state := map[string]visitState{} // visitNew default
	onStack := map[string]int{}      // id -> index in stack
	var stack []string
	var cycle []string

	var dfs func(id string)
	dfs = func(id string) {
		if len(cycle) > 0 {
			return
		}

		state[id] = visitVisiting
		onStack[id] = len(stack)
		stack = append(stack, id)

		it, ok := g.Items[id]
		if ok {
			for _, depID := range it.Deps {
				if len(cycle) > 0 {
					return
				}
				// Unknown deps are handled by validation; ignore them for cycle detection.
				if _, ok := g.Items[depID]; !ok {
					continue
				}

				switch state[depID] {
				case visitNew:
					dfs(depID)
				case visitVisiting:
					idx := onStack[depID]
					cycle = append([]string{}, stack[idx:]...)
					cycle = append(cycle, depID)
					return
				case visitDone:
					// nothing
				}
			}
		}

		stack = stack[:len(stack)-1]
		delete(onStack, id)
		state[id] = visitDone
	}

	for id := range g.Items {
		if state[id] == visitDone {
			continue
		}
		if state[id] == visitNew {
			dfs(id)
			if len(cycle) > 0 {
				return cycle
			}
		}
	}

	return nil
}

package execution

import (
	"sort"

	"github.com/jbonatakis/blackbird/internal/plan"
)

// ReadyTasks returns task IDs that are eligible for execution.
// A task is ready when it is todo, has all deps satisfied, and is not manually blocked or skipped.
func ReadyTasks(g plan.WorkGraph) []string {
	ids := make([]string, 0, len(g.Items))
	for id, it := range g.Items {
		if it.Status != plan.StatusTodo {
			continue
		}
		if len(plan.UnmetDeps(g, it)) != 0 {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

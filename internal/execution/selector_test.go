package execution

import (
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestReadyTasksFiltersAndSorts(t *testing.T) {
	now := time.Date(2026, 1, 28, 0, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"a": {
				ID:        "a",
				Title:     "Done dep",
				Status:    plan.StatusDone,
				CreatedAt: now,
				UpdatedAt: now,
			},
			"b": {
				ID:        "b",
				Title:     "Ready",
				Status:    plan.StatusTodo,
				Deps:      []string{"a"},
				CreatedAt: now,
				UpdatedAt: now,
			},
			"c": {
				ID:        "c",
				Title:     "Blocked",
				Status:    plan.StatusBlocked,
				CreatedAt: now,
				UpdatedAt: now,
			},
			"d": {
				ID:        "d",
				Title:     "Skipped",
				Status:    plan.StatusSkipped,
				CreatedAt: now,
				UpdatedAt: now,
			},
			"e": {
				ID:        "e",
				Title:     "Missing dep",
				Status:    plan.StatusTodo,
				Deps:      []string{"missing"},
				CreatedAt: now,
				UpdatedAt: now,
			},
			"f": {
				ID:        "f",
				Title:     "Ready 2",
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	ready := ReadyTasks(g)
	if len(ready) != 2 {
		t.Fatalf("expected 2 ready tasks, got %v", ready)
	}
	if ready[0] != "b" || ready[1] != "f" {
		t.Fatalf("unexpected ready order: %v", ready)
	}
}

func TestReadyTasksEmpty(t *testing.T) {
	g := plan.WorkGraph{SchemaVersion: plan.SchemaVersion, Items: map[string]plan.WorkItem{}}
	ready := ReadyTasks(g)
	if len(ready) != 0 {
		t.Fatalf("expected empty ready list, got %v", ready)
	}
}

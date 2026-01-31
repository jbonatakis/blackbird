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
			"p": {
				ID:        "p",
				Title:     "Parent container",
				Status:    plan.StatusTodo,
				ChildIDs:  []string{"c1"},
				CreatedAt: now,
				UpdatedAt: now,
			},
			"c1": {
				ID:        "c1",
				Title:     "Leaf child",
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	ready := ReadyTasks(g)
	if len(ready) != 3 {
		t.Fatalf("expected 3 ready tasks, got %v", ready)
	}
	if ready[0] != "b" || ready[1] != "c1" || ready[2] != "f" {
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

func TestReadyTasksLeafOnly(t *testing.T) {
	now := time.Date(2026, 1, 28, 0, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"dep": {
				ID:        "dep",
				Title:     "Dependency",
				Status:    plan.StatusDone,
				CreatedAt: now,
				UpdatedAt: now,
			},
			"parent": {
				ID:        "parent",
				Title:     "Parent container",
				Status:    plan.StatusTodo,
				Deps:      []string{"dep"},
				ChildIDs:  []string{"leaf"},
				CreatedAt: now,
				UpdatedAt: now,
			},
			"leaf": {
				ID:        "leaf",
				Title:     "Leaf task",
				Status:    plan.StatusTodo,
				Deps:      []string{"dep"},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	ready := ReadyTasks(g)
	if len(ready) != 1 {
		t.Fatalf("expected 1 ready task, got %v", ready)
	}
	if ready[0] != "leaf" {
		t.Fatalf("expected leaf task to be ready, got %v", ready)
	}
}

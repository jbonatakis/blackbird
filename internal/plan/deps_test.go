package plan

import (
	"testing"
	"time"
)

func TestUnmetDeps(t *testing.T) {
	now := time.Now()
	g := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"A": {
				ID:                 "A",
				Title:              "A",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             StatusDone,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"B": {
				ID:                 "B",
				Title:              "B",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{"A"},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"C": {
				ID:                 "C",
				Title:              "C",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{"B", "A"},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}

	unmet := UnmetDeps(g, g.Items["C"])
	if len(unmet) != 1 || unmet[0] != "B" {
		t.Fatalf("UnmetDeps(C) = %#v, want [\"B\"]", unmet)
	}
}

func TestDependents_Sorted(t *testing.T) {
	now := time.Now()
	g := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"A": {
				ID:                 "A",
				Title:              "A",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"B": {
				ID:                 "B",
				Title:              "B",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{"A"},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"C": {
				ID:                 "C",
				Title:              "C",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{"A"},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}

	got := Dependents(g, "A")
	if len(got) != 2 || got[0] != "B" || got[1] != "C" {
		t.Fatalf("Dependents(A) = %#v, want [\"B\", \"C\"]", got)
	}
}

func TestDepCycle(t *testing.T) {
	now := time.Now()
	g := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"A": {
				ID:                 "A",
				Title:              "A",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{"B"},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"B": {
				ID:                 "B",
				Title:              "B",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{"C"},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"C": {
				ID:                 "C",
				Title:              "C",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{"A"},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}

	cycle := DepCycle(g)
	if len(cycle) == 0 {
		t.Fatalf("expected a cycle")
	}
	if cycle[len(cycle)-1] != cycle[0] {
		t.Fatalf("expected cycle closure, got %#v", cycle)
	}
}

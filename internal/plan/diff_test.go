package plan

import (
	"testing"
	"time"
)

func TestDiffSummary(t *testing.T) {
	now := time.Now().UTC()
	parentID := "a"

	before := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"a": {
				ID:                 "a",
				Title:              "Root",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{"b"},
				Deps:               []string{},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"b": {
				ID:                 "b",
				Title:              "Child",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           &parentID,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}

	after := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"a": {
				ID:                 "a",
				Title:              "Root",
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
			"b": {
				ID:                 "b",
				Title:              "Child updated",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{"a"},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"c": {
				ID:                 "c",
				Title:              "New root",
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
		},
	}

	summary := Diff(before, after)
	if len(summary.Added) != 1 || summary.Added[0] != "c" {
		t.Fatalf("expected added [c], got %v", summary.Added)
	}
	if len(summary.Moved) != 1 || summary.Moved[0] != "b" {
		t.Fatalf("expected moved [b], got %v", summary.Moved)
	}
	if len(summary.Updated) != 1 || summary.Updated[0] != "b" {
		t.Fatalf("expected updated [b], got %v", summary.Updated)
	}
	if len(summary.DepsAdded) != 1 || summary.DepsAdded[0].From != "b" || summary.DepsAdded[0].To != "a" {
		t.Fatalf("expected deps added [b->a], got %v", summary.DepsAdded)
	}
	if len(summary.Removed) != 0 || len(summary.DepsRemoved) != 0 {
		t.Fatalf("expected no removals, got removed=%v depsRemoved=%v", summary.Removed, summary.DepsRemoved)
	}
}

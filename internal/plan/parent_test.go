package plan

import (
	"testing"
	"time"
)

func TestPropagateParentCompletion(t *testing.T) {
	now := time.Date(2026, 1, 28, 12, 0, 0, 0, time.UTC)

	t.Run("no parent", func(t *testing.T) {
		g := &WorkGraph{
			SchemaVersion: SchemaVersion,
			Items: map[string]WorkItem{
				"leaf": {
					ID: "leaf", Title: "Leaf", Status: StatusDone,
					ParentID: nil, ChildIDs: []string{},
					CreatedAt: now, UpdatedAt: now,
				},
			},
		}
		PropagateParentCompletion(g, "leaf", now)
		if g.Items["leaf"].Status != StatusDone {
			t.Fatalf("leaf should remain done")
		}
	})

	t.Run("parent not all children done", func(t *testing.T) {
		parentID := "parent"
		g := &WorkGraph{
			SchemaVersion: SchemaVersion,
			Items: map[string]WorkItem{
				"parent": {
					ID: "parent", Title: "Parent", Status: StatusTodo,
					ChildIDs:  []string{"a", "b"},
					CreatedAt: now, UpdatedAt: now,
				},
				"a": {
					ID: "a", Title: "A", Status: StatusDone,
					ParentID: &parentID, ChildIDs: []string{},
					CreatedAt: now, UpdatedAt: now,
				},
				"b": {
					ID: "b", Title: "B", Status: StatusTodo,
					ParentID: &parentID, ChildIDs: []string{},
					CreatedAt: now, UpdatedAt: now,
				},
			},
		}
		PropagateParentCompletion(g, "a", now)
		if g.Items["parent"].Status != StatusTodo {
			t.Fatalf("parent should stay todo when not all children done, got %s", g.Items["parent"].Status)
		}
	})

	t.Run("parent all children done", func(t *testing.T) {
		parentID := "parent"
		g := &WorkGraph{
			SchemaVersion: SchemaVersion,
			Items: map[string]WorkItem{
				"parent": {
					ID: "parent", Title: "Parent", Status: StatusTodo,
					ChildIDs:  []string{"a", "b"},
					CreatedAt: now, UpdatedAt: now,
				},
				"a": {
					ID: "a", Title: "A", Status: StatusDone,
					ParentID: &parentID, ChildIDs: []string{},
					CreatedAt: now, UpdatedAt: now,
				},
				"b": {
					ID: "b", Title: "B", Status: StatusDone,
					ParentID: &parentID, ChildIDs: []string{},
					CreatedAt: now, UpdatedAt: now,
				},
			},
		}
		PropagateParentCompletion(g, "b", now)
		if g.Items["parent"].Status != StatusDone {
			t.Fatalf("parent should be done when all children done, got %s", g.Items["parent"].Status)
		}
		if g.Items["parent"].UpdatedAt != now {
			t.Fatalf("parent UpdatedAt should be set")
		}
	})

	t.Run("grandparent chain", func(t *testing.T) {
		parentID := "parent"
		grandparentID := "grandparent"
		g := &WorkGraph{
			SchemaVersion: SchemaVersion,
			Items: map[string]WorkItem{
				"grandparent": {
					ID: "grandparent", Title: "Grandparent", Status: StatusTodo,
					ChildIDs:  []string{"parent"},
					CreatedAt: now, UpdatedAt: now,
				},
				"parent": {
					ID: "parent", Title: "Parent", Status: StatusTodo,
					ParentID: &grandparentID, ChildIDs: []string{"leaf"},
					CreatedAt: now, UpdatedAt: now,
				},
				"leaf": {
					ID: "leaf", Title: "Leaf", Status: StatusDone,
					ParentID: &parentID, ChildIDs: []string{},
					CreatedAt: now, UpdatedAt: now,
				},
			},
		}
		PropagateParentCompletion(g, "leaf", now)
		if g.Items["parent"].Status != StatusDone {
			t.Fatalf("parent should be done, got %s", g.Items["parent"].Status)
		}
		if g.Items["grandparent"].Status != StatusDone {
			t.Fatalf("grandparent should be done, got %s", g.Items["grandparent"].Status)
		}
	})
}

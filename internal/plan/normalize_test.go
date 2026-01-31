package plan

import (
	"testing"
	"time"
)

func TestNormalizeWorkGraphTimestamps(t *testing.T) {
	now := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)
	before := time.Date(2026, 1, 30, 10, 0, 0, 0, time.UTC)

	parent := "parent"
	notes := "note"
	g := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"parent": {
				ID:                 "parent",
				Title:              "Parent",
				Description:        "Parent item",
				AcceptanceCriteria: []string{"a"},
				Prompt:             "Do parent",
				ParentID:           nil,
				ChildIDs:           []string{"child"},
				Deps:               []string{},
				Status:             StatusTodo,
				CreatedAt:          before,
				UpdatedAt:          before,
				Notes:              &notes,
			},
			"child": {
				ID:                 "child",
				Title:              "Child",
				Description:        "Child item",
				AcceptanceCriteria: []string{"b"},
				Prompt:             "Do child",
				ParentID:           &parent,
				ChildIDs:           []string{},
				Deps:               []string{"parent"},
				Status:             StatusBlocked,
				CreatedAt:          before.Add(-time.Hour),
				UpdatedAt:          before.Add(-time.Hour),
			},
		},
	}

	normalized := NormalizeWorkGraphTimestamps(g, now)

	if !normalized.Items["parent"].CreatedAt.Equal(now) || !normalized.Items["parent"].UpdatedAt.Equal(now) {
		t.Fatalf("parent timestamps not normalized: got %s/%s want %s", normalized.Items["parent"].CreatedAt, normalized.Items["parent"].UpdatedAt, now)
	}
	if !normalized.Items["child"].CreatedAt.Equal(now) || !normalized.Items["child"].UpdatedAt.Equal(now) {
		t.Fatalf("child timestamps not normalized: got %s/%s want %s", normalized.Items["child"].CreatedAt, normalized.Items["child"].UpdatedAt, now)
	}

	if normalized.Items["parent"].Title != g.Items["parent"].Title {
		t.Fatalf("parent title changed: got %q want %q", normalized.Items["parent"].Title, g.Items["parent"].Title)
	}
	if normalized.Items["child"].ParentID == nil || *normalized.Items["child"].ParentID != parent {
		t.Fatalf("child parentId changed: got %#v want %q", normalized.Items["child"].ParentID, parent)
	}

	if g.Items["parent"].CreatedAt.Equal(now) || g.Items["parent"].UpdatedAt.Equal(now) {
		t.Fatalf("original plan mutated")
	}
}

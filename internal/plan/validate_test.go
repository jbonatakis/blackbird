package plan

import (
	"testing"
	"time"
)

func TestValidate_EmptyGraphIsValid(t *testing.T) {
	g := NewEmptyWorkGraph()
	if errs := Validate(g); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_RejectsMissingSchemaVersion(t *testing.T) {
	g := WorkGraph{Items: map[string]WorkItem{}}
	errs := Validate(g)
	if len(errs) == 0 {
		t.Fatalf("expected errors")
	}
}

func TestValidate_RejectsUnknownRefs(t *testing.T) {
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
				ParentID:           ptr("NOPE"),
				ChildIDs:           []string{"B"},
				Deps:               []string{"C"},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}

	errs := Validate(g)
	if len(errs) == 0 {
		t.Fatalf("expected errors")
	}
}

func TestValidate_ChildParentConsistency(t *testing.T) {
	now := time.Now()
	g := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			"P": {
				ID:                 "P",
				Title:              "parent",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil,
				ChildIDs:           []string{"C"},
				Deps:               []string{},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"C": {
				ID:                 "C",
				Title:              "child",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "",
				ParentID:           nil, // should be P
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}

	errs := Validate(g)
	if len(errs) == 0 {
		t.Fatalf("expected errors")
	}
}

func TestValidate_DetectsParentCycle(t *testing.T) {
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
				ParentID:           ptr("B"),
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
				ParentID:           ptr("A"),
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}

	errs := Validate(g)
	if len(errs) == 0 {
		t.Fatalf("expected errors")
	}
}

func ptr(s string) *string { return &s }

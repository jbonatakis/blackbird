package planquality

import (
	"reflect"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestLeafTaskIDsDeterministicAndLeafOnly(t *testing.T) {
	now := time.Date(2026, 2, 6, 0, 0, 0, 0, time.UTC)
	parentID := "parent"

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"leaf-z": {
				ID:        "leaf-z",
				Title:     "Leaf Z",
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
			"leaf-a": {
				ID:        "leaf-a",
				Title:     "Leaf A",
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
			parentID: {
				ID:        parentID,
				Title:     "Parent",
				ChildIDs:  []string{"leaf-a", "leaf-z"},
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
			"leaf-b": {
				ID:        "leaf-b",
				Title:     "Leaf B",
				ParentID:  &parentID,
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
			"container-2": {
				ID:        "container-2",
				Title:     "Container 2",
				ChildIDs:  []string{"leaf-b"},
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	want := []string{"leaf-a", "leaf-b", "leaf-z"}
	for i := 0; i < 100; i++ {
		got := LeafTaskIDs(g)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("LeafTaskIDs() = %v, want %v", got, want)
		}
	}
}

func TestNormalizeText(t *testing.T) {
	input := "\tTODO:  Improve/Handle errors!!!\n"
	got := NormalizeText(input)
	want := "todo improve handle errors"
	if got != want {
		t.Fatalf("NormalizeText() = %q, want %q", got, want)
	}
}

func TestNormalizeTextsDropsEmpty(t *testing.T) {
	got := NormalizeTexts([]string{"  TBD ", "", "   ", "Ship tests"})
	want := []string{"tbd", "ship tests"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeTexts() = %v, want %v", got, want)
	}
}

func TestContainsAnyNormalizedPhrase(t *testing.T) {
	if !ContainsAnyNormalizedPhrase("Need to improve error handling", []string{"works well", "improve"}) {
		t.Fatalf("expected phrase match")
	}
	if ContainsAnyNormalizedPhrase("this is an improvement", []string{"improve"}) {
		t.Fatalf("unexpected substring-only match")
	}
}

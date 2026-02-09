package execution

import (
	"reflect"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestParentCandidateDiscovery(t *testing.T) {
	now := time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		changedChildID string
		graph          plan.WorkGraph
		want           []string
	}{
		{
			name:           "single parent when all children done",
			changedChildID: "child-b",
			graph: plan.WorkGraph{
				SchemaVersion: plan.SchemaVersion,
				Items: map[string]plan.WorkItem{
					"parent":  parentCandidateItem("parent", plan.StatusTodo, nil, now, "child-a", "child-b"),
					"child-a": parentCandidateItem("child-a", plan.StatusDone, strPtr("parent"), now),
					"child-b": parentCandidateItem("child-b", plan.StatusDone, strPtr("parent"), now),
				},
			},
			want: []string{"parent"},
		},
		{
			name:           "nested parents traverse nearest to furthest",
			changedChildID: "leaf-b",
			graph: plan.WorkGraph{
				SchemaVersion: plan.SchemaVersion,
				Items: map[string]plan.WorkItem{
					"grand":   parentCandidateItem("grand", plan.StatusTodo, nil, now, "parent", "sibling"),
					"parent":  parentCandidateItem("parent", plan.StatusDone, strPtr("grand"), now, "leaf-a", "leaf-b"),
					"sibling": parentCandidateItem("sibling", plan.StatusDone, strPtr("grand"), now),
					"leaf-a":  parentCandidateItem("leaf-a", plan.StatusDone, strPtr("parent"), now),
					"leaf-b":  parentCandidateItem("leaf-b", plan.StatusDone, strPtr("parent"), now),
				},
			},
			want: []string{"parent", "grand"},
		},
		{
			name:           "partially done child set returns no candidates",
			changedChildID: "child-a",
			graph: plan.WorkGraph{
				SchemaVersion: plan.SchemaVersion,
				Items: map[string]plan.WorkItem{
					"parent":  parentCandidateItem("parent", plan.StatusTodo, nil, now, "child-a", "child-b"),
					"child-a": parentCandidateItem("child-a", plan.StatusDone, strPtr("parent"), now),
					"child-b": parentCandidateItem("child-b", plan.StatusInProgress, strPtr("parent"), now),
				},
			},
			want: []string{},
		},
		{
			name:           "ignores non-container parent with empty child ids",
			changedChildID: "leaf",
			graph: plan.WorkGraph{
				SchemaVersion: plan.SchemaVersion,
				Items: map[string]plan.WorkItem{
					"grand":  parentCandidateItem("grand", plan.StatusTodo, nil, now, "middle"),
					"middle": parentCandidateItem("middle", plan.StatusDone, strPtr("grand"), now),
					"leaf":   parentCandidateItem("leaf", plan.StatusDone, strPtr("middle"), now),
				},
			},
			want: []string{"grand"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotFirst := ParentReviewCandidateIDs(tc.graph, tc.changedChildID)
			if !reflect.DeepEqual(gotFirst, tc.want) {
				t.Fatalf("ParentReviewCandidateIDs first call = %v, want %v", gotFirst, tc.want)
			}

			gotSecond := ParentReviewCandidateIDs(tc.graph, tc.changedChildID)
			if !reflect.DeepEqual(gotSecond, tc.want) {
				t.Fatalf("ParentReviewCandidateIDs second call = %v, want %v", gotSecond, tc.want)
			}
		})
	}
}

func parentCandidateItem(id string, status plan.Status, parentID *string, now time.Time, childIDs ...string) plan.WorkItem {
	return plan.WorkItem{
		ID:                 id,
		Title:              "Task " + id,
		Description:        "",
		AcceptanceCriteria: []string{},
		Prompt:             "do it",
		ParentID:           parentID,
		ChildIDs:           append([]string{}, childIDs...),
		Deps:               []string{},
		Status:             status,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func strPtr(v string) *string {
	return &v
}

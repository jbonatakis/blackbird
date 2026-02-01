package plan

import (
	"reflect"
	"testing"
	"time"
)

func TestBuildTaskTree_OrdersChildrenByChildIDs(t *testing.T) {
	now := time.Now()
	parentID := "parent"
	childA := "a"
	childB := "b"
	childC := "c"

	graph := WorkGraph{
		SchemaVersion: 1,
		Items: map[string]WorkItem{
			parentID: {
				ID:        parentID,
				Title:     "Parent",
				ChildIDs:  []string{childB, childA},
				CreatedAt: now,
				UpdatedAt: now,
			},
			childA: {
				ID:        childA,
				Title:     "Child A",
				ParentID:  &parentID,
				ChildIDs:  []string{},
				CreatedAt: now,
				UpdatedAt: now,
			},
			childB: {
				ID:        childB,
				Title:     "Child B",
				ParentID:  &parentID,
				ChildIDs:  []string{},
				CreatedAt: now,
				UpdatedAt: now,
			},
			childC: {
				ID:        childC,
				Title:     "Child C",
				ParentID:  &parentID,
				ChildIDs:  []string{},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	tree := BuildTaskTree(graph)
	want := []string{childB, childA, childC}
	if !reflect.DeepEqual(tree.Children[parentID], want) {
		t.Fatalf("children order = %#v, want %#v", tree.Children[parentID], want)
	}
}

func TestBuildTaskTree_AppendsUnlistedChildrenSorted(t *testing.T) {
	now := time.Now()
	parentID := "parent"
	childA := "a"
	childB := "b"
	childC := "c"

	graph := WorkGraph{
		SchemaVersion: 1,
		Items: map[string]WorkItem{
			parentID: {
				ID:        parentID,
				Title:     "Parent",
				ChildIDs:  []string{childB},
				CreatedAt: now,
				UpdatedAt: now,
			},
			childA: {
				ID:        childA,
				Title:     "Child A",
				ParentID:  &parentID,
				ChildIDs:  []string{},
				CreatedAt: now,
				UpdatedAt: now,
			},
			childB: {
				ID:        childB,
				Title:     "Child B",
				ParentID:  &parentID,
				ChildIDs:  []string{},
				CreatedAt: now,
				UpdatedAt: now,
			},
			childC: {
				ID:        childC,
				Title:     "Child C",
				ParentID:  &parentID,
				ChildIDs:  []string{},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	tree := BuildTaskTree(graph)
	want := []string{childB, childA, childC}
	if !reflect.DeepEqual(tree.Children[parentID], want) {
		t.Fatalf("children order = %#v, want %#v", tree.Children[parentID], want)
	}
}

func TestBuildTaskTree_MissingParentBecomesRoot(t *testing.T) {
	now := time.Now()
	orphanID := "orphan"

	graph := WorkGraph{
		SchemaVersion: 1,
		Items: map[string]WorkItem{
			orphanID: {
				ID:        orphanID,
				Title:     "Orphan",
				ParentID:  strPtr("missing-parent"),
				ChildIDs:  []string{},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	tree := BuildTaskTree(graph)
	want := []string{orphanID}
	if !reflect.DeepEqual(tree.Roots, want) {
		t.Fatalf("roots = %#v, want %#v", tree.Roots, want)
	}
}

func TestBuildTaskTree_RootOrderStable(t *testing.T) {
	now := time.Now()
	rootA := "a"
	rootB := "b"
	orphan := "c"

	graph := WorkGraph{
		SchemaVersion: 1,
		Items: map[string]WorkItem{
			rootB: {
				ID:        rootB,
				Title:     "Root B",
				ParentID:  nil,
				ChildIDs:  []string{},
				CreatedAt: now,
				UpdatedAt: now,
			},
			rootA: {
				ID:        rootA,
				Title:     "Root A",
				ParentID:  nil,
				ChildIDs:  []string{},
				CreatedAt: now,
				UpdatedAt: now,
			},
			orphan: {
				ID:        orphan,
				Title:     "Orphan",
				ParentID:  strPtr("missing-parent"),
				ChildIDs:  []string{},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	tree := BuildTaskTree(graph)
	want := []string{rootA, rootB, orphan}
	if !reflect.DeepEqual(tree.Roots, want) {
		t.Fatalf("roots = %#v, want %#v", tree.Roots, want)
	}
}

func strPtr(s string) *string {
	return &s
}

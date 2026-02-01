package plan

import (
	"testing"
	"time"
)

func TestParseStatus(t *testing.T) {
	tests := []struct {
		value string
		want  Status
		ok    bool
	}{
		{"todo", StatusTodo, true},
		{"queued", StatusQueued, true},
		{"in_progress", StatusInProgress, true},
		{"waiting_user", StatusWaitingUser, true},
		{"blocked", StatusBlocked, true},
		{"done", StatusDone, true},
		{"failed", StatusFailed, true},
		{"skipped", StatusSkipped, true},
		{"unknown", "", false},
	}
	for _, tt := range tests {
		got, ok := ParseStatus(tt.value)
		if ok != tt.ok {
			t.Fatalf("ParseStatus(%q) ok=%v want %v", tt.value, ok, tt.ok)
		}
		if got != tt.want {
			t.Fatalf("ParseStatus(%q)=%q want %q", tt.value, got, tt.want)
		}
	}
}

func TestSetStatusUpdatesAndPropagates(t *testing.T) {
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	parentID := "parent"
	childID := "child"
	graph := WorkGraph{
		SchemaVersion: SchemaVersion,
		Items: map[string]WorkItem{
			parentID: {
				ID:                 parentID,
				Title:              "Parent",
				AcceptanceCriteria: []string{},
				ChildIDs:           []string{childID},
				Status:             StatusTodo,
				CreatedAt:          now.Add(-time.Hour),
				UpdatedAt:          now.Add(-time.Hour),
			},
			childID: {
				ID:                 childID,
				Title:              "Child",
				AcceptanceCriteria: []string{},
				ParentID:           &parentID,
				ChildIDs:           []string{},
				Status:             StatusTodo,
				CreatedAt:          now.Add(-time.Hour),
				UpdatedAt:          now.Add(-time.Hour),
			},
		},
	}

	if err := SetStatus(&graph, childID, StatusDone, now); err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}
	child := graph.Items[childID]
	if child.Status != StatusDone {
		t.Fatalf("child status=%s want %s", child.Status, StatusDone)
	}
	if !child.UpdatedAt.Equal(now) {
		t.Fatalf("child updatedAt=%s want %s", child.UpdatedAt, now)
	}
	parent := graph.Items[parentID]
	if parent.Status != StatusDone {
		t.Fatalf("parent status=%s want %s", parent.Status, StatusDone)
	}
	if !parent.UpdatedAt.Equal(now) {
		t.Fatalf("parent updatedAt=%s want %s", parent.UpdatedAt, now)
	}
}

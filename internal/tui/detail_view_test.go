package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRenderDetailView(t *testing.T) {
	now := time.Date(2026, 1, 29, 10, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"dep-1": {
				ID:        "dep-1",
				Title:     "Dependency",
				Status:    plan.StatusDone,
				CreatedAt: now,
				UpdatedAt: now,
			},
			"task-1": {
				ID:                 "task-1",
				Title:              "Build detail view",
				Description:        "Render the full details pane.",
				AcceptanceCriteria: []string{"shows metadata", "shows readiness"},
				Prompt:             "Implement the UI.",
				Deps:               []string{"dep-1"},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"next-1": {
				ID:        "next-1",
				Title:     "Follow-up",
				Deps:      []string{"task-1"},
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	model := Model{
		plan:        g,
		selectedID:  "task-1",
		windowWidth: 0,
	}

	out := RenderDetailView(model)

	assertContains(t, out, "ID: task-1")
	assertContains(t, out, "Title: Build detail view")
	assertContains(t, out, "Status: todo")
	assertContains(t, out, "Description")
	assertContains(t, out, "Acceptance criteria")
	assertContains(t, out, "- shows metadata")
	assertContains(t, out, "Dependencies")
	assertContains(t, out, "- dep-1 [done] Dependency")
	assertContains(t, out, "Dependents")
	assertContains(t, out, "- next-1 [todo] Follow-up")
	assertContains(t, out, "Readiness")
	assertContains(t, out, "deps satisfied: yes")
	assertContains(t, out, "actionable now: true")
	assertContains(t, out, "Prompt")
	assertContains(t, out, "Implement the UI.")
}

func TestRenderDetailViewEmptySelection(t *testing.T) {
	model := Model{}
	out := RenderDetailView(model)
	if !strings.Contains(out, "No item selected.") {
		t.Fatalf("expected empty selection message, got %q", out)
	}
}

func assertContains(t *testing.T, value string, substr string) {
	t.Helper()
	if !strings.Contains(value, substr) {
		t.Fatalf("expected output to contain %q, got %q", substr, value)
	}
}

package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestModelViewRendersTreeAndDetail(t *testing.T) {
	now := time.Date(2026, 1, 29, 12, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task-1": {
				ID:          "task-1",
				Title:       "Plan tree",
				Description: "Tree view and details.",
				Status:      plan.StatusTodo,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}

	model := Model{
		plan:         g,
		selectedID:   "task-1",
		viewMode:     ViewModeMain,
		planExists:   true,
		windowWidth:  120,
		windowHeight: 40,
		activePane:   PaneTree,
		filterMode:   FilterModeAll,
	}

	out := model.View()
	if !strings.Contains(out, "task-1") {
		t.Fatalf("expected tree content to include item id, got %q", out)
	}
	if !strings.Contains(out, "Item") {
		t.Fatalf("expected detail content to include Item section, got %q", out)
	}
}

func TestModelViewRendersHomeView(t *testing.T) {
	model := Model{
		viewMode:     ViewModeHome,
		planExists:   false,
		windowWidth:  100,
		windowHeight: 20,
	}

	out := model.View()
	if !strings.Contains(out, "blackbird") {
		t.Fatalf("expected home view title in output, got %q", out)
	}
	if !strings.Contains(out, "Durable, dependency-aware planning and execution") {
		t.Fatalf("expected home view tagline in output, got %q", out)
	}
}

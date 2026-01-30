package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRenderHomeViewNoPlan(t *testing.T) {
	model := Model{
		plan:        plan.NewEmptyWorkGraph(),
		planExists:  false,
		windowWidth: 0,
	}

	out := RenderHomeView(model)

	assertContains(t, out, "blackbird")
	assertContains(t, out, "Durable, dependency-aware planning and execution")
	assertContains(t, out, "No plan found")
	assertContains(t, out, "[g]")
	assertContains(t, out, "Generate plan")
	assertContains(t, out, "[v]")
	assertContains(t, out, "View plan")
	assertContains(t, out, "[e]")
	assertContains(t, out, "Execute")
	assertContains(t, out, "[ctrl+c]")
}

func TestRenderHomeViewWithPlan(t *testing.T) {
	now := time.Date(2026, 1, 30, 10, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task-1": {
				ID:        "task-1",
				Title:     "Ready task",
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
			"task-2": {
				ID:        "task-2",
				Title:     "Blocked task",
				Status:    plan.StatusTodo,
				Deps:      []string{"task-1"},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	model := Model{
		plan:       g,
		planExists: true,
	}

	out := RenderHomeView(model)

	if !strings.Contains(out, "Plan found: 2 items, 1 ready, 1 blocked") {
		t.Fatalf("expected plan status line, got %q", out)
	}
}

func TestRenderHomeViewValidationErrorBanner(t *testing.T) {
	model := Model{
		plan:              plan.NewEmptyWorkGraph(),
		planExists:        true,
		planValidationErr: "items.task-1.title: title is required",
		windowWidth:       0,
		windowHeight:      0,
	}

	out := RenderHomeView(model)

	assertContains(t, out, "Plan has errors")
	assertContains(t, out, "items.task-1.title: title is required")
	assertContains(t, out, "Press [g] to regenerate or [v] to view and fix")
}

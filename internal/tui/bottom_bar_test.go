package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestBottomBarHomeHints(t *testing.T) {
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
		},
	}

	model := Model{
		plan:         g,
		viewMode:     ViewModeHome,
		planExists:   true,
		windowWidth:  80,
		windowHeight: 10,
	}

	out := RenderBottomBar(model)
	expected := []string{"[g]enerate", "[v]iew", "[r]efine", "[e]xecute", "[ctrl+c]quit"}
	for _, hint := range expected {
		if !strings.Contains(out, hint) {
			t.Fatalf("expected home hint %q in bottom bar, got %q", hint, out)
		}
	}
}

func TestBottomBarHomeHidesCountsWhenNoPlan(t *testing.T) {
	model := Model{
		viewMode:     ViewModeHome,
		planExists:   false,
		windowWidth:  80,
		windowHeight: 10,
	}

	out := RenderBottomBar(model)
	if strings.Contains(out, "ready:") || strings.Contains(out, "blocked:") {
		t.Fatalf("expected no status counts on home screen without plan, got %q", out)
	}
	if !strings.Contains(out, "[g]enerate") {
		t.Fatalf("expected generate hint on home screen, got %q", out)
	}
	if strings.Contains(out, "[v]iew") || strings.Contains(out, "[r]efine") {
		t.Fatalf("expected view/refine hints to be hidden without a plan, got %q", out)
	}
}

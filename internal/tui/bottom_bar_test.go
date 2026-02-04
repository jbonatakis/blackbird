package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
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
		windowWidth:  120,
		windowHeight: 10,
	}

	out := RenderBottomBar(model)
	expected := []string{"[g]enerate", "[v]iew", "[r]efine", "[e]xecute", "[s]ettings", "[c]hange", "[ctrl+c]quit"}
	for _, hint := range expected {
		if !strings.Contains(out, hint) {
			t.Fatalf("expected home hint %q in bottom bar, got %q", hint, out)
		}
	}
	if !strings.Contains(out, "agent:") {
		t.Fatalf("expected agent label in bottom bar, got %q", out)
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
	if !strings.Contains(out, "[c]hange") {
		t.Fatalf("expected change agent hint on home screen, got %q", out)
	}
	if !strings.Contains(out, "[s]ettings") {
		t.Fatalf("expected settings hint on home screen, got %q", out)
	}
	if !strings.Contains(out, "[g]enerate") {
		t.Fatalf("expected generate hint on home screen, got %q", out)
	}
	if strings.Contains(out, "[v]iew") || strings.Contains(out, "[r]efine") {
		t.Fatalf("expected view/refine hints to be hidden without a plan, got %q", out)
	}
	if !strings.Contains(out, "agent:") {
		t.Fatalf("expected agent label on home screen without plan, got %q", out)
	}
}

func TestBottomBarMainShowsAgentAndCounts(t *testing.T) {
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
		viewMode:     ViewModeMain,
		planExists:   true,
		windowWidth:  80,
		windowHeight: 10,
	}

	out := RenderBottomBar(model)
	if !strings.Contains(out, "agent:") {
		t.Fatalf("expected agent label in main bottom bar, got %q", out)
	}
	if !containsAny(out, []string{"ready:1", "r:1"}) || !containsAny(out, []string{"blocked:0", "b:0"}) {
		t.Fatalf("expected status counts in main bottom bar, got %q", out)
	}
}

func TestBottomBarHomeShowsSelectedAgentLabel(t *testing.T) {
	model := Model{
		viewMode:   ViewModeHome,
		planExists: false,
		agentSelection: agent.AgentSelection{
			Agent:         agent.AgentInfo{ID: agent.AgentCodex, Label: "Codex"},
			ConfigPresent: true,
		},
		windowWidth:  80,
		windowHeight: 10,
	}

	out := RenderBottomBar(model)
	if !strings.Contains(out, "agent:Codex") {
		t.Fatalf("expected selected agent label in home bottom bar, got %q", out)
	}
}

func TestBottomBarMainShowsSelectedAgentLabel(t *testing.T) {
	model := Model{
		viewMode:   ViewModeMain,
		planExists: true,
		agentSelection: agent.AgentSelection{
			Agent:         agent.AgentInfo{ID: agent.AgentCodex, Label: "Codex"},
			ConfigPresent: true,
		},
		windowWidth:  80,
		windowHeight: 10,
	}

	out := RenderBottomBar(model)
	if !strings.Contains(out, "agent:Codex") {
		t.Fatalf("expected selected agent label in main bottom bar, got %q", out)
	}
}

func containsAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}

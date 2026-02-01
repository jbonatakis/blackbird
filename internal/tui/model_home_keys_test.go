package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestHomeKeyToggleView(t *testing.T) {
	model := Model{
		viewMode:   ViewModeHome,
		planExists: false,
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if updated.(Model).viewMode != ViewModeHome {
		t.Fatalf("expected home view to remain when no plan exists")
	}

	model.planExists = true
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if updated.(Model).viewMode != ViewModeMain {
		t.Fatalf("expected home view to toggle to main when plan exists")
	}

	model.viewMode = ViewModeMain
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if updated.(Model).viewMode != ViewModeHome {
		t.Fatalf("expected main view to toggle back to home")
	}
}

func TestHomeKeyGeneratePlan(t *testing.T) {
	model := Model{
		viewMode:   ViewModeHome,
		planExists: false,
		plan:       plan.NewEmptyWorkGraph(),
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	next := updated.(Model)
	if next.actionMode != ActionModeGeneratePlan {
		t.Fatalf("expected generate plan modal to open from home view")
	}
	if next.planGenerateForm == nil {
		t.Fatalf("expected generate plan form to be initialized")
	}
}

func TestHomeKeyPlanActionGating(t *testing.T) {
	model := Model{
		viewMode:   ViewModeHome,
		planExists: false,
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	if updated.(Model).viewMode != ViewModeHome {
		t.Fatalf("expected view to remain home when no plan exists")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if updated.(Model).actionMode != ActionModeNone {
		t.Fatalf("expected refine to be ignored when no plan exists")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if updated.(Model).actionInProgress {
		t.Fatalf("expected execute to be ignored when no plan exists")
	}
}

func TestHomeKeyPlanActionsWithPlan(t *testing.T) {
	readyPlan := plan.WorkGraph{
		Items: map[string]plan.WorkItem{
			"task-1": {
				ID:     "task-1",
				Status: plan.StatusTodo,
			},
		},
	}

	model := Model{
		viewMode:   ViewModeHome,
		planExists: true,
		plan:       readyPlan,
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	if updated.(Model).viewMode != ViewModeMain {
		t.Fatalf("expected view to switch to main when plan exists")
	}

	model.viewMode = ViewModeHome
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	next := updated.(Model)
	if next.actionMode != ActionModePlanRefine || next.planRefineForm == nil {
		t.Fatalf("expected refine modal to open from home when plan exists")
	}

	model.viewMode = ViewModeHome
	model.actionInProgress = false
	model.actionName = ""
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	next = updated.(Model)
	if !next.actionInProgress || next.actionName != "Executing..." {
		t.Fatalf("expected execute action to start from home when ready tasks exist")
	}
}

func TestHomeKeySelectAgentOpensModal(t *testing.T) {
	model := Model{
		viewMode:   ViewModeHome,
		planExists: false,
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	next := updated.(Model)
	if next.actionMode != ActionModeSelectAgent {
		t.Fatalf("expected agent selection modal to open from home view")
	}
	if next.agentSelectionHighlight < 0 || next.agentSelectionHighlight >= len(agent.AgentRegistry) {
		t.Fatalf("expected valid agent selection highlight index, got %d", next.agentSelectionHighlight)
	}
}

func TestHomeKeyExecuteRequiresReadyTasks(t *testing.T) {
	nonReadyPlan := plan.WorkGraph{
		Items: map[string]plan.WorkItem{
			"task-1": {
				ID:     "task-1",
				Status: plan.StatusDone,
			},
		},
	}

	model := Model{
		viewMode:   ViewModeHome,
		planExists: true,
		plan:       nonReadyPlan,
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	next := updated.(Model)
	if next.actionInProgress {
		t.Fatalf("expected execute to be ignored when no ready tasks exist")
	}
	if next.actionName != "" {
		t.Fatalf("expected actionName to remain empty when no ready tasks exist")
	}
}

func TestHomeKeyCtrlCQuits(t *testing.T) {
	model := Model{
		viewMode: ViewModeHome,
	}
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatalf("expected quit command from ctrl+c on home view")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected quit command to return tea.QuitMsg")
	}
}

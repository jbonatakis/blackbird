package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/plan"
)

// PlanRefineForm represents the state of the plan refine modal
// for capturing a change request.
type PlanRefineForm struct {
	changeRequest textarea.Model
	width         int
	height        int
}

// NewPlanRefineForm creates a new plan refine form.
func NewPlanRefineForm() PlanRefineForm {
	ta := textarea.New()
	ta.Placeholder = "Enter change request (describe desired changes)..."
	ta.Focus()
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(4)

	return PlanRefineForm{
		changeRequest: ta,
		width:         70,
		height:        20,
	}
}

// SetSize updates the form dimensions.
func (f *PlanRefineForm) SetSize(width, height int) {
	f.width = width
	f.height = height

	fieldWidth := width - 10
	if fieldWidth > 80 {
		fieldWidth = 80
	}
	if fieldWidth < 40 {
		fieldWidth = 40
	}

	f.changeRequest.SetWidth(fieldWidth)
}

// Update handles form updates.
func (f PlanRefineForm) Update(msg tea.Msg) (PlanRefineForm, tea.Cmd) {
	var cmd tea.Cmd
	f.changeRequest, cmd = f.changeRequest.Update(msg)
	return f, cmd
}

// GetChangeRequest returns the form value.
func (f PlanRefineForm) GetChangeRequest() string {
	return strings.TrimSpace(f.changeRequest.Value())
}

// HandlePlanRefineKey handles key presses in plan-refine mode.
func HandlePlanRefineKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.planRefineForm == nil {
		m.actionMode = ActionModeNone
		return m, nil
	}

	switch msg.String() {
	case "ctrl+s", "ctrl+enter":
		changeRequest := m.planRefineForm.GetChangeRequest()
		if strings.TrimSpace(changeRequest) == "" {
			return m, nil
		}

		basePlan := plan.Clone(m.plan)
		m.pendingPlanRequest = PendingPlanRequest{
			kind:          PendingPlanRefine,
			changeRequest: changeRequest,
			basePlan:      basePlan,
			questionRound: 0,
		}

		m.actionMode = ActionModeNone
		m.planRefineForm = nil
		m.actionInProgress = true
		m.actionName = "Refining plan..."

		return m, tea.Batch(
			RefinePlanInMemory(context.Background(), changeRequest, basePlan),
			spinnerTickCmd(),
		)
	case "esc":
		m.actionMode = ActionModeNone
		m.planRefineForm = nil
		return m, nil
	default:
		updatedForm, cmd := m.planRefineForm.Update(msg)
		m.planRefineForm = &updatedForm
		return m, cmd
	}
}

// RenderPlanRefineModal renders the plan refine modal.
func RenderPlanRefineModal(m Model, form PlanRefineForm) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helperStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	lines := []string{
		titleStyle.Render("Refine Plan"),
		"",
		labelStyle.Render("Describe the changes you want:"),
		form.changeRequest.View(),
		"",
	}

	if strings.TrimSpace(form.changeRequest.Value()) == "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		lines = append(lines, errorStyle.Render("Change request cannot be empty"), "")
	}

	lines = append(lines, helperStyle.Render("[ctrl+s or ctrl+enter]submit [esc]cancel"))

	modalWidth := form.width
	if modalWidth > 90 {
		modalWidth = 90
	}
	if modalWidth < 60 {
		modalWidth = 60
	}
	if m.windowWidth > 0 && m.windowWidth < modalWidth+4 {
		modalWidth = m.windowWidth - 4
	}

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Width(modalWidth)

	modal := modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	if m.windowHeight > 0 {
		topPadding := (m.windowHeight - lipgloss.Height(modal)) / 2
		if topPadding > 0 {
			padding := lipgloss.NewStyle().PaddingTop(topPadding).Render(modal)
			return padding
		}
	}

	return modal
}

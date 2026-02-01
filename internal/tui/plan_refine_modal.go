package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/plan"
)

// PlanRefineFormField represents the focused field in the refine form.
type PlanRefineFormField int

const (
	RefineFieldChangeRequest PlanRefineFormField = iota
	RefineFieldSubmit
)

// PlanRefineForm represents the state of the plan refine modal
// for capturing a change request.
type PlanRefineForm struct {
	changeRequest textarea.Model
	focusedField  PlanRefineFormField
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
		focusedField:  RefineFieldChangeRequest,
		width:         70,
		height:        20,
	}
}

// SetSize updates the form dimensions.
func (f *PlanRefineForm) SetSize(width, height int) {
	f.width = width
	f.height = height

	fieldWidth := width - 10
	if fieldWidth > 120 {
		fieldWidth = 120
	}
	if fieldWidth < 40 {
		fieldWidth = 40
	}

	f.changeRequest.SetWidth(fieldWidth)
	// Use more lines for the textarea when we have full-screen height
	taHeight := 4
	if height > 20 {
		taHeight = height / 4
		if taHeight > 12 {
			taHeight = 12
		}
	}
	f.changeRequest.SetHeight(taHeight)
}

// Update handles form updates.
func (f PlanRefineForm) Update(msg tea.Msg) (PlanRefineForm, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			return f.focusNext(), nil
		case "shift+tab":
			return f.focusPrev(), nil
		case "enter":
			if f.focusedField == RefineFieldSubmit {
				return f, nil // let HandlePlanRefineKey submit
			}
			// in textarea: pass through for newline
		}
	}

	switch f.focusedField {
	case RefineFieldChangeRequest:
		f.changeRequest, cmd = f.changeRequest.Update(msg)
		return f, cmd
	case RefineFieldSubmit:
		return f, nil
	}
	return f, nil
}

func (f PlanRefineForm) focusNext() PlanRefineForm {
	f.changeRequest.Blur()
	if f.focusedField == RefineFieldChangeRequest {
		f.focusedField = RefineFieldSubmit
	} else {
		f.focusedField = RefineFieldChangeRequest
		f.changeRequest.Focus()
	}
	return f
}

func (f PlanRefineForm) focusPrev() PlanRefineForm {
	f.changeRequest.Blur()
	if f.focusedField == RefineFieldSubmit {
		f.focusedField = RefineFieldChangeRequest
		f.changeRequest.Focus()
	} else {
		f.focusedField = RefineFieldSubmit
	}
	return f
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

	// Enter on Submit button: submit the form
	if msg.String() == "enter" && m.planRefineForm.focusedField == RefineFieldSubmit {
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
	}

	if msg.String() == "esc" {
		m.actionMode = ActionModeNone
		m.planRefineForm = nil
		return m, nil
	}

	updatedForm, cmd := m.planRefineForm.Update(msg)
	m.planRefineForm = &updatedForm
	return m, cmd
}

// RenderPlanRefineModal renders the plan refine modal.
func RenderPlanRefineModal(m Model, form PlanRefineForm) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helperStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	buttonStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("15"))
	focusedButtonStyle := buttonStyle.Copy().
		Background(lipgloss.Color("69")).
		Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	lines := []string{
		titleStyle.Render("Refine Plan"),
		"",
		labelStyle.Render("Describe the changes you want:"),
		form.changeRequest.View(),
		"",
	}

	submitButton := "[ Submit ]"
	if form.focusedField == RefineFieldSubmit {
		submitButton = focusedButtonStyle.Render(submitButton)
	} else {
		submitButton = buttonStyle.Render(submitButton)
	}
	lines = append(lines, submitButton)
	lines = append(lines, "")

	if strings.TrimSpace(form.changeRequest.Value()) == "" && form.focusedField == RefineFieldSubmit {
		lines = append(lines, errorStyle.Render("Change request cannot be empty"), "")
	}

	lines = append(lines, helperStyle.Render("Tab: next • Enter: new line (in text) • Shift+Tab: previous • [esc]cancel"))

	// Full-screen modal
	modalWidth := m.windowWidth - 4
	if modalWidth < 50 {
		modalWidth = 50
	}
	modalHeight := m.windowHeight - 3
	if modalHeight < 10 {
		modalHeight = 10
	}

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight)

	modal := modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return modal
}

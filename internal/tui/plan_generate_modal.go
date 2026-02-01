package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PlanGenerateFormField represents the currently focused field in the form
type PlanGenerateFormField int

const (
	FieldDescription PlanGenerateFormField = iota
	FieldConstraints
	FieldGranularity
	FieldSubmit
)

// PlanGenerateForm represents the state of the plan generation modal form
type PlanGenerateForm struct {
	description  textarea.Model
	constraints  textarea.Model
	granularity  textinput.Model
	focusedField PlanGenerateFormField
	width        int
	height       int
}

// NewPlanGenerateForm creates a new plan generation form
func NewPlanGenerateForm() PlanGenerateForm {
	// Description textarea (required, multi-line)
	descTA := textarea.New()
	descTA.Placeholder = "Enter project description (required)..."
	descTA.Focus()
	descTA.CharLimit = 5000
	descTA.SetWidth(60)
	descTA.SetHeight(5)

	// Constraints textarea (optional, multi-line)
	constraintsTA := textarea.New()
	constraintsTA.Placeholder = "Enter constraints (optional, comma-separated)..."
	constraintsTA.CharLimit = 2000
	constraintsTA.SetWidth(60)
	constraintsTA.SetHeight(3)
	constraintsTA.Blur()

	// Granularity textinput (optional, single-line)
	granularityTI := textinput.New()
	granularityTI.Placeholder = "Enter granularity (optional)..."
	granularityTI.CharLimit = 200
	granularityTI.Width = 60
	granularityTI.Blur()

	return PlanGenerateForm{
		description:  descTA,
		constraints:  constraintsTA,
		granularity:  granularityTI,
		focusedField: FieldDescription,
		width:        70,
		height:       25,
	}
}

// SetSize updates the form dimensions
func (f *PlanGenerateForm) SetSize(width, height int) {
	f.width = width
	f.height = height

	// Adjust field widths based on modal width (full screen uses most of width)
	fieldWidth := width - 10
	if fieldWidth > 120 {
		fieldWidth = 120
	}
	if fieldWidth < 40 {
		fieldWidth = 40
	}

	f.description.SetWidth(fieldWidth)
	f.constraints.SetWidth(fieldWidth)
	f.granularity.Width = fieldWidth

	// Use more vertical space for textareas when we have full-screen height
	descHeight := 5
	constHeight := 3
	if height > 20 {
		descHeight = height / 4
		if descHeight > 12 {
			descHeight = 12
		}
		constHeight = height / 6
		if constHeight > 8 {
			constHeight = 8
		}
	}
	f.description.SetHeight(descHeight)
	f.constraints.SetHeight(constHeight)
}

// Update handles form updates
func (f PlanGenerateForm) Update(msg tea.Msg) (PlanGenerateForm, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			return f.focusNext(), nil
		case "enter":
			// In textarea fields, Enter inserts newline; on other fields, move to next
			if f.focusedField == FieldDescription || f.focusedField == FieldConstraints {
				break // pass to textarea for newline
			}
			return f.focusNext(), nil
		case "shift+tab":
			return f.focusPrev(), nil
		}
	}

	// Update the currently focused field
	switch f.focusedField {
	case FieldDescription:
		f.description, cmd = f.description.Update(msg)
		cmds = append(cmds, cmd)
	case FieldConstraints:
		f.constraints, cmd = f.constraints.Update(msg)
		cmds = append(cmds, cmd)
	case FieldGranularity:
		f.granularity, cmd = f.granularity.Update(msg)
		cmds = append(cmds, cmd)
	}

	return f, tea.Batch(cmds...)
}

// focusNext moves focus to the next field
func (f PlanGenerateForm) focusNext() PlanGenerateForm {
	f.description.Blur()
	f.constraints.Blur()
	f.granularity.Blur()

	switch f.focusedField {
	case FieldDescription:
		f.focusedField = FieldConstraints
		f.constraints.Focus()
	case FieldConstraints:
		f.focusedField = FieldGranularity
		f.granularity.Focus()
	case FieldGranularity:
		f.focusedField = FieldSubmit
	case FieldSubmit:
		f.focusedField = FieldDescription
		f.description.Focus()
	}

	return f
}

// focusPrev moves focus to the previous field
func (f PlanGenerateForm) focusPrev() PlanGenerateForm {
	f.description.Blur()
	f.constraints.Blur()
	f.granularity.Blur()

	switch f.focusedField {
	case FieldDescription:
		f.focusedField = FieldSubmit
	case FieldConstraints:
		f.focusedField = FieldDescription
		f.description.Focus()
	case FieldGranularity:
		f.focusedField = FieldConstraints
		f.constraints.Focus()
	case FieldSubmit:
		f.focusedField = FieldGranularity
		f.granularity.Focus()
	}

	return f
}

// Validate checks if the form is valid for submission
func (f PlanGenerateForm) Validate() error {
	if strings.TrimSpace(f.description.Value()) == "" {
		return &ValidationError{Field: "description", Message: "Project description is required"}
	}
	return nil
}

// GetValues returns the form values
func (f PlanGenerateForm) GetValues() (description string, constraints []string, granularity string) {
	description = strings.TrimSpace(f.description.Value())

	// Parse constraints as comma-separated values
	constraintsStr := strings.TrimSpace(f.constraints.Value())
	if constraintsStr != "" {
		parts := strings.Split(constraintsStr, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				constraints = append(constraints, trimmed)
			}
		}
	}

	granularity = strings.TrimSpace(f.granularity.Value())
	return
}

// ValidationError represents a form validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// HandlePlanGenerateKey handles key presses in plan-generate mode
func HandlePlanGenerateKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.planGenerateForm == nil {
		m.actionMode = ActionModeNone
		return m, nil
	}

	// Handle Enter on Submit field - attempt to submit the form
	if msg.String() == "enter" && m.planGenerateForm.focusedField == FieldSubmit {
		if err := m.planGenerateForm.Validate(); err != nil {
			// Validation failed, keep form open to show error
			return m, nil
		}

		// Get form values
		description, constraints, granularity := m.planGenerateForm.GetValues()

		// Store pending request for question handling
		m.pendingPlanRequest = PendingPlanRequest{
			kind:          PendingPlanGenerate,
			description:   description,
			constraints:   constraints,
			granularity:   granularity,
			questionRound: 0,
		}

		// Close modal and start plan generation
		m.actionMode = ActionModeNone
		m.planGenerateForm = nil
		m.actionInProgress = true
		m.actionName = "Generating plan..."

		// Use the in-memory generation with the form inputs
		return m, tea.Batch(
			GeneratePlanInMemory(context.Background(), description, constraints, granularity),
			spinnerTickCmd(),
		)
	}

	// Update the form with the key message
	updatedForm, cmd := m.planGenerateForm.Update(msg)
	m.planGenerateForm = &updatedForm
	return m, cmd
}

// RenderPlanGenerateModal renders the plan generation modal form
func RenderPlanGenerateModal(m Model, form PlanGenerateForm) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	focusedLabelStyle := labelStyle.Copy().Foreground(lipgloss.Color("69")).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	buttonStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("15"))
	focusedButtonStyle := buttonStyle.Copy().
		Background(lipgloss.Color("69")).
		Bold(true)

	title := titleStyle.Render("Generate Plan")
	helpText := labelStyle.Render("Tab: next field • Enter: new line (in text areas) • Shift+Tab: previous • ESC: cancel")

	// Build form fields
	var lines []string
	lines = append(lines, title)
	lines = append(lines, "")

	// Description field
	descLabel := "Project Description (required):"
	if form.focusedField == FieldDescription {
		descLabel = focusedLabelStyle.Render(descLabel)
	} else {
		descLabel = labelStyle.Render(descLabel)
	}
	lines = append(lines, descLabel)
	lines = append(lines, form.description.View())
	lines = append(lines, "")

	// Constraints field
	constraintsLabel := "Constraints (optional, comma-separated):"
	if form.focusedField == FieldConstraints {
		constraintsLabel = focusedLabelStyle.Render(constraintsLabel)
	} else {
		constraintsLabel = labelStyle.Render(constraintsLabel)
	}
	lines = append(lines, constraintsLabel)
	lines = append(lines, form.constraints.View())
	lines = append(lines, "")

	// Granularity field
	granularityLabel := "Granularity (optional):"
	if form.focusedField == FieldGranularity {
		granularityLabel = focusedLabelStyle.Render(granularityLabel)
	} else {
		granularityLabel = labelStyle.Render(granularityLabel)
	}
	lines = append(lines, granularityLabel)
	lines = append(lines, form.granularity.View())
	lines = append(lines, "")

	// Submit button
	submitButton := "[ Submit ]"
	if form.focusedField == FieldSubmit {
		submitButton = focusedButtonStyle.Render(submitButton)
	} else {
		submitButton = buttonStyle.Render(submitButton)
	}
	lines = append(lines, submitButton)
	lines = append(lines, "")

	// Validation error if any
	if err := form.Validate(); err != nil && form.focusedField == FieldSubmit {
		lines = append(lines, errorStyle.Render("⚠ "+err.Error()))
		lines = append(lines, "")
	}

	lines = append(lines, helpText)

	// Full-screen modal: use full window minus margin and space for bottom bar
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

// HandleConfirmOverwriteKey handles key presses in confirm-overwrite mode
func HandleConfirmOverwriteKey(m Model, key string) (Model, tea.Cmd) {
	switch key {
	case "esc", "n", "N":
		// User declined - return to normal view
		m.actionMode = ActionModeNone
		return m, nil
	case "y", "Y", "enter":
		// User confirmed - proceed to plan generation modal
		m.actionMode = ActionModeNone
		form := NewPlanGenerateForm()
		form.SetSize(m.windowWidth, m.windowHeight)
		m.planGenerateForm = &form
		m.actionMode = ActionModeGeneratePlan
		return m, nil
	}
	return m, nil
}

// RenderConfirmOverwriteModal renders the confirmation modal for overwriting existing plan
func RenderConfirmOverwriteModal(m Model) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	buttonStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Margin(0, 1).
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("15"))
	yesButtonStyle := buttonStyle.Copy().
		Background(lipgloss.Color("46")).
		Bold(true)
	noButtonStyle := buttonStyle.Copy().
		Background(lipgloss.Color("196")).
		Bold(true)

	itemCount := len(m.plan.Items)
	title := titleStyle.Render("⚠ Plan Already Exists")
	message := textStyle.Render(fmt.Sprintf("Plan already exists with %d items. Overwrite?", itemCount))

	yesButton := yesButtonStyle.Render("[ Yes ]")
	noButton := noButtonStyle.Render("[ No ]")
	buttons := lipgloss.JoinHorizontal(lipgloss.Left, yesButton, noButton)

	helpText := helpStyle.Render("Y/Enter: Yes • N/ESC: No")

	lines := []string{
		title,
		"",
		message,
		"",
		buttons,
		"",
		helpText,
	}

	modalWidth := 60
	if m.windowWidth > 0 && m.windowWidth < modalWidth+4 {
		modalWidth = m.windowWidth - 4
	}

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("220")).
		Padding(1, 2).
		Width(modalWidth)

	modal := modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	// Center the modal
	if m.windowHeight > 0 {
		topPadding := (m.windowHeight - lipgloss.Height(modal)) / 2
		if topPadding > 0 {
			padding := lipgloss.NewStyle().PaddingTop(topPadding).Render(modal)
			return padding
		}
	}

	return modal
}

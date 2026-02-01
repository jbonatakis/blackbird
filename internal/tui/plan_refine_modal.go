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

const (
	planRefinePickerChangeRequest FilePickerField = "plan_refine_change_request"
)

// PlanRefineForm represents the state of the plan refine modal
// for capturing a change request.
type PlanRefineForm struct {
	changeRequest textarea.Model
	focusedField  PlanRefineFormField
	filePicker    FilePickerState
	requestAnchor FilePickerAnchor
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
		filePicker:    NewFilePickerState(),
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

// OpenFilePicker opens the @ file picker for the change-request field.
func (f *PlanRefineForm) OpenFilePicker(anchor FilePickerAnchor) bool {
	if f.focusedField != RefineFieldChangeRequest {
		return false
	}
	f.requestAnchor = anchor
	f.filePicker.OpenAt(planRefinePickerChangeRequest, anchor)
	return true
}

// CloseFilePicker closes the @ file picker.
func (f *PlanRefineForm) CloseFilePicker() {
	f.filePicker.Close()
}

// ApplyFilePickerSelection replaces the @query span with the selected path.
func (f *PlanRefineForm) ApplyFilePickerSelection(selectedPath string) bool {
	if selectedPath == "" {
		return false
	}
	if f.filePicker.ActiveField != planRefinePickerChangeRequest {
		return false
	}
	applyFilePickerToTextarea(&f.changeRequest, f.requestAnchor, f.filePicker.Query, selectedPath)
	f.filePicker.Close()
	return true
}

// Update handles form updates.
func (f PlanRefineForm) Update(msg tea.Msg) (PlanRefineForm, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if handled := f.handleFilePickerKey(msg); handled {
			return f, nil
		}
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

func planRefinePickerField(field PlanRefineFormField) (FilePickerField, bool) {
	if field == RefineFieldChangeRequest {
		return planRefinePickerChangeRequest, true
	}
	return FilePickerFieldNone, false
}

func planRefineFieldAnchor(f *PlanRefineForm) FilePickerAnchor {
	if f == nil {
		return FilePickerAnchor{}
	}
	if f.focusedField == RefineFieldChangeRequest {
		return textareaCursorAnchor(f.changeRequest)
	}
	return FilePickerAnchor{}
}

func (f *PlanRefineForm) handleFilePickerKey(msg tea.KeyMsg) bool {
	pickerField, ok := planRefinePickerField(f.focusedField)
	if !ok {
		return false
	}

	anchor := planRefineFieldAnchor(f)
	prevOpen := f.filePicker.Open
	prevQuery := f.filePicker.Query

	result, err := HandleFilePickerKey(&f.filePicker, msg, FilePickerKeyOptions{
		Field:  pickerField,
		Anchor: anchor,
	})
	if err != nil {
		f.filePicker.Close()
		return true
	}

	if !prevOpen && f.filePicker.Open {
		f.requestAnchor = anchor
	}

	if result.Action == FilePickerActionInsert {
		f.filePicker.Query = prevQuery
		if result.Selected != "" {
			f.ApplyFilePickerSelection(result.Selected)
		} else {
			f.filePicker.Close()
		}
		return true
	}

	if result.Action == FilePickerActionCancel {
		f.filePicker.Close()
		if prevOpen && (msg.String() == "tab" || msg.String() == "shift+tab") {
			return false
		}
		if prevOpen {
			return true
		}
	}

	if result.Consumed {
		return true
	}

	return false
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

	if m.planRefineForm.filePicker.Open {
		updatedForm, cmd := m.planRefineForm.Update(msg)
		m.planRefineForm = &updatedForm
		return m, cmd
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
		if m.planRefineForm.filePicker.Open {
			updatedForm, cmd := m.planRefineForm.Update(msg)
			m.planRefineForm = &updatedForm
			return m, cmd
		}
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
	focusedLabelStyle := labelStyle.Copy().Foreground(lipgloss.Color("69")).Bold(true)
	helperStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	buttonStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("15"))
	focusedButtonStyle := buttonStyle.Copy().
		Background(lipgloss.Color("69")).
		Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	title := titleStyle.Render("Refine Plan")
	helpText := helperStyle.Render("Tab: next • Enter: new line (in text) • Shift+Tab: previous • [esc]cancel")

	label := "Describe the changes you want:"
	if form.focusedField == RefineFieldChangeRequest {
		label = focusedLabelStyle.Render(label)
	} else {
		label = labelStyle.Render(label)
	}

	submitButton := "[ Submit ]"
	if form.focusedField == RefineFieldSubmit {
		submitButton = focusedButtonStyle.Render(submitButton)
	} else {
		submitButton = buttonStyle.Render(submitButton)
	}
	// Full-screen modal
	modalWidth := m.windowWidth - 4
	if modalWidth < 50 {
		modalWidth = 50
	}
	modalHeight := m.windowHeight - 3
	if modalHeight < 10 {
		modalHeight = 10
	}

	contentWidth := modalWidth - 6
	if contentWidth < 1 {
		contentWidth = 1
	}

	renderLines := func(picker string) []string {
		lines := []string{title, "", label, form.changeRequest.View()}
		if picker != "" && form.focusedField == RefineFieldChangeRequest {
			lines = append(lines, picker)
		} else {
			lines = append(lines, "")
		}

		lines = append(lines, submitButton, "")

		if strings.TrimSpace(form.changeRequest.Value()) == "" && form.focusedField == RefineFieldSubmit {
			lines = append(lines, errorStyle.Render("Change request cannot be empty"), "")
		}

		lines = append(lines, helpText)
		return lines
	}

	baseLines := renderLines("")
	baseHeight := lipgloss.Height(lipgloss.JoinVertical(lipgloss.Left, baseLines...))

	picker := ""
	if form.filePicker.Open {
		pickerWidth := form.changeRequest.Width()
		if pickerWidth < 1 {
			pickerWidth = contentWidth
		}
		if pickerWidth > contentWidth {
			pickerWidth = contentWidth
		}

		available := modalHeight - baseHeight + 1
		if available < 1 {
			available = 1
		}
		if available > 6 {
			available = 6
		}
		picker = RenderFilePickerList(form.filePicker, pickerWidth, available)
	}

	lines := renderLines(picker)

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight)

	modal := modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return modal
}

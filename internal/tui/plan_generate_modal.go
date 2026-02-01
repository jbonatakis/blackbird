package tui

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

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

const (
	planGeneratePickerDescription FilePickerField = "plan_generate_description"
	planGeneratePickerConstraints FilePickerField = "plan_generate_constraints"
	planGeneratePickerGranularity FilePickerField = "plan_generate_granularity"
)

// PlanGenerateForm represents the state of the plan generation modal form
type PlanGenerateForm struct {
	description       textarea.Model
	constraints       textarea.Model
	granularity       textinput.Model
	focusedField      PlanGenerateFormField
	filePicker        FilePickerState
	descriptionAnchor FilePickerAnchor
	constraintsAnchor FilePickerAnchor
	granularityAnchor FilePickerAnchor
	width             int
	height            int
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
		filePicker:   NewFilePickerState(),
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

// OpenFilePicker opens the @ file picker for the provided field.
func (f *PlanGenerateForm) OpenFilePicker(field PlanGenerateFormField, anchor FilePickerAnchor) bool {
	pickerField, ok := planGeneratePickerField(field)
	if !ok {
		return false
	}
	f.setAnchorForField(field, anchor)
	f.filePicker.OpenAt(pickerField, anchor)
	return true
}

// CloseFilePicker closes the @ file picker.
func (f *PlanGenerateForm) CloseFilePicker() {
	f.filePicker.Close()
}

// ApplyFilePickerSelection replaces the @query span with the selected path.
func (f *PlanGenerateForm) ApplyFilePickerSelection(selectedPath string) bool {
	if selectedPath == "" {
		return false
	}
	switch f.filePicker.ActiveField {
	case planGeneratePickerDescription:
		applyFilePickerToTextarea(&f.description, f.descriptionAnchor, f.filePicker.Query, selectedPath)
	case planGeneratePickerConstraints:
		applyFilePickerToTextarea(&f.constraints, f.constraintsAnchor, f.filePicker.Query, selectedPath)
	case planGeneratePickerGranularity:
		applyFilePickerToTextInput(&f.granularity, f.granularityAnchor, f.filePicker.Query, selectedPath)
	default:
		return false
	}
	f.filePicker.Close()
	return true
}

// Update handles form updates
func (f PlanGenerateForm) Update(msg tea.Msg) (PlanGenerateForm, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if handled := f.handleFilePickerKey(msg); handled {
			return f, nil
		}
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

func planGeneratePickerField(field PlanGenerateFormField) (FilePickerField, bool) {
	switch field {
	case FieldDescription:
		return planGeneratePickerDescription, true
	case FieldConstraints:
		return planGeneratePickerConstraints, true
	case FieldGranularity:
		return planGeneratePickerGranularity, true
	default:
		return FilePickerFieldNone, false
	}
}

func (f *PlanGenerateForm) setAnchorForField(field PlanGenerateFormField, anchor FilePickerAnchor) {
	switch field {
	case FieldDescription:
		f.descriptionAnchor = anchor
	case FieldConstraints:
		f.constraintsAnchor = anchor
	case FieldGranularity:
		f.granularityAnchor = anchor
	}
}

func applyFilePickerToTextarea(ta *textarea.Model, anchor FilePickerAnchor, query string, selectedPath string) {
	updated, cursor := replaceFilePickerSpan(ta.Value(), anchor, query, selectedPath)
	setTextareaValueWithCursor(ta, updated, cursor)
}

func applyFilePickerToTextInput(input *textinput.Model, anchor FilePickerAnchor, query string, selectedPath string) {
	updated, cursor := replaceFilePickerSpan(input.Value(), anchor, query, selectedPath)
	input.SetValue(updated)
	input.SetCursor(cursor)
}

func (f *PlanGenerateForm) handleFilePickerKey(msg tea.KeyMsg) bool {
	pickerField, ok := planGeneratePickerField(f.focusedField)
	if !ok {
		return false
	}

	anchor := planGenerateFieldAnchor(f)
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
		f.setAnchorForField(f.focusedField, anchor)
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

func planGenerateFieldAnchor(f *PlanGenerateForm) FilePickerAnchor {
	if f == nil {
		return FilePickerAnchor{}
	}
	switch f.focusedField {
	case FieldDescription:
		return textareaCursorAnchor(f.description)
	case FieldConstraints:
		return textareaCursorAnchor(f.constraints)
	case FieldGranularity:
		return textInputCursorAnchor(f.granularity)
	default:
		return FilePickerAnchor{}
	}
}

func textareaCursorAnchor(ta textarea.Model) FilePickerAnchor {
	value := ta.Value()
	lines := strings.Split(value, "\n")
	row := ta.Line()
	if row < 0 {
		row = 0
	}
	if row >= len(lines) && len(lines) > 0 {
		row = len(lines) - 1
	}

	lineInfo := ta.LineInfo()
	col := lineInfo.StartColumn + lineInfo.ColumnOffset
	if len(lines) == 0 {
		row = 0
		col = 0
	} else {
		lineLen := utf8.RuneCountInString(lines[row])
		if col > lineLen {
			col = lineLen
		}
		if col < 0 {
			col = 0
		}
	}

	start := 0
	for i := 0; i < row && i < len(lines); i++ {
		start += utf8.RuneCountInString(lines[i]) + 1
	}
	start += col

	return FilePickerAnchor{
		Start:  start,
		Line:   row,
		Column: col,
	}
}

func textInputCursorAnchor(input textinput.Model) FilePickerAnchor {
	pos := input.Position()
	if pos < 0 {
		pos = 0
	}
	return FilePickerAnchor{
		Start:  pos,
		Line:   0,
		Column: pos,
	}
}

func setTextareaValueWithCursor(ta *textarea.Model, value string, cursor int) {
	ta.SetValue(value)
	if cursor < 0 {
		cursor = 0
	}
	lineCount := ta.LineCount()
	if lineCount <= 0 {
		return
	}

	row, col := cursorRowCol(value, cursor)
	if row < 0 {
		row = 0
	}
	if row > lineCount-1 {
		row = lineCount - 1
	}
	if col < 0 {
		col = 0
	}

	ta.CursorStart()
	for i := 0; i < lineCount-1-row; i++ {
		ta.CursorUp()
	}
	ta.SetCursor(col)
}

func cursorRowCol(value string, cursor int) (row int, col int) {
	runes := []rune(value)
	if cursor > len(runes) {
		cursor = len(runes)
	}
	for i := 0; i < cursor; i++ {
		if runes[i] == '\n' {
			row++
			col = 0
			continue
		}
		col++
	}
	return row, col
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

	if msg.String() == "esc" {
		if m.planGenerateForm.filePicker.Open {
			updatedForm, cmd := m.planGenerateForm.Update(msg)
			m.planGenerateForm = &updatedForm
			return m, cmd
		}
		m.actionMode = ActionModeNone
		m.planGenerateForm = nil
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

	// Full-screen modal: use full window minus margin and space for bottom bar
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

	// Build form fields
	descLabel := "Project Description (required):"
	if form.focusedField == FieldDescription {
		descLabel = focusedLabelStyle.Render(descLabel)
	} else {
		descLabel = labelStyle.Render(descLabel)
	}

	constraintsLabel := "Constraints (optional, comma-separated):"
	if form.focusedField == FieldConstraints {
		constraintsLabel = focusedLabelStyle.Render(constraintsLabel)
	} else {
		constraintsLabel = labelStyle.Render(constraintsLabel)
	}

	granularityLabel := "Granularity (optional):"
	if form.focusedField == FieldGranularity {
		granularityLabel = focusedLabelStyle.Render(granularityLabel)
	} else {
		granularityLabel = labelStyle.Render(granularityLabel)
	}

	submitButton := "[ Submit ]"
	if form.focusedField == FieldSubmit {
		submitButton = focusedButtonStyle.Render(submitButton)
	} else {
		submitButton = buttonStyle.Render(submitButton)
	}

	renderLines := func(picker string) []string {
		lines := []string{title, ""}
		appendField := func(label, view string, field PlanGenerateFormField) {
			lines = append(lines, label, view)
			if picker != "" && form.focusedField == field {
				lines = append(lines, picker)
			} else {
				lines = append(lines, "")
			}
		}

		appendField(descLabel, form.description.View(), FieldDescription)
		appendField(constraintsLabel, form.constraints.View(), FieldConstraints)
		appendField(granularityLabel, form.granularity.View(), FieldGranularity)

		lines = append(lines, submitButton, "")

		if err := form.Validate(); err != nil && form.focusedField == FieldSubmit {
			lines = append(lines, errorStyle.Render("⚠ "+err.Error()), "")
		}

		lines = append(lines, helpText)
		return lines
	}

	baseLines := renderLines("")
	baseHeight := lipgloss.Height(lipgloss.JoinVertical(lipgloss.Left, baseLines...))

	picker := ""
	if form.filePicker.Open {
		pickerWidth := contentWidth
		switch form.focusedField {
		case FieldDescription:
			pickerWidth = form.description.Width()
		case FieldConstraints:
			pickerWidth = form.constraints.Width()
		case FieldGranularity:
			pickerWidth = lipgloss.Width(form.granularity.View())
		}
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

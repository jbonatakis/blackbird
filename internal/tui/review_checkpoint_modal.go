package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

type ReviewCheckpointMode int

const (
	ReviewCheckpointChooseAction ReviewCheckpointMode = iota
	ReviewCheckpointRequestChanges
)

const (
	reviewCheckpointPickerChangeRequest FilePickerField = "review_checkpoint_change_request"
)

type ReviewCheckpointForm struct {
	mode           ReviewCheckpointMode
	run            execution.RunRecord
	task           execution.TaskContext
	selectedAction int
	changeRequest  textarea.Model
	filePicker     FilePickerState
	requestAnchor  FilePickerAnchor
	width          int
	height         int
}

func NewReviewCheckpointForm(run execution.RunRecord, g plan.WorkGraph) ReviewCheckpointForm {
	task := run.Context.Task
	if strings.TrimSpace(task.ID) == "" {
		task.ID = run.TaskID
	}
	if strings.TrimSpace(task.Title) == "" {
		if item, ok := g.Items[run.TaskID]; ok {
			task.Title = item.Title
			if strings.TrimSpace(task.Description) == "" {
				task.Description = item.Description
			}
			if len(task.AcceptanceCriteria) == 0 && len(item.AcceptanceCriteria) > 0 {
				task.AcceptanceCriteria = append([]string{}, item.AcceptanceCriteria...)
			}
		}
	}
	if strings.TrimSpace(task.ID) == "" {
		task.ID = run.TaskID
	}

	request := textarea.New()
	request.Placeholder = "Describe requested changes..."
	request.CharLimit = 2000
	request.SetWidth(60)
	request.SetHeight(4)
	request.Blur()

	return ReviewCheckpointForm{
		mode:           ReviewCheckpointChooseAction,
		run:            run,
		task:           task,
		selectedAction: 0,
		changeRequest:  request,
		filePicker:     NewFilePickerState(),
		width:          80,
		height:         30,
	}
}

func (f *ReviewCheckpointForm) SetSize(width, height int) {
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
func (f *ReviewCheckpointForm) OpenFilePicker(anchor FilePickerAnchor) bool {
	if f.mode != ReviewCheckpointRequestChanges {
		return false
	}
	f.requestAnchor = anchor
	f.filePicker.OpenAt(reviewCheckpointPickerChangeRequest, anchor)
	return true
}

// CloseFilePicker closes the @ file picker.
func (f *ReviewCheckpointForm) CloseFilePicker() {
	f.filePicker.Close()
}

func (f ReviewCheckpointForm) Update(msg tea.Msg) (ReviewCheckpointForm, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if f.mode == ReviewCheckpointRequestChanges {
			if handled := f.handleFilePickerKey(msg); handled {
				return f, nil
			}
		}
		switch f.mode {
		case ReviewCheckpointChooseAction:
			switch msg.String() {
			case "up", "k":
				if f.selectedAction > 0 {
					f.selectedAction--
				}
				return f, nil
			case "down", "j":
				if f.selectedAction < 3 {
					f.selectedAction++
				}
				return f, nil
			case "1":
				f.selectedAction = 0
				return f, nil
			case "2":
				f.selectedAction = 1
				return f, nil
			case "3":
				f.selectedAction = 2
				return f, nil
			case "4":
				f.selectedAction = 3
				return f, nil
			}
		case ReviewCheckpointRequestChanges:
			f.changeRequest, cmd = f.changeRequest.Update(msg)
			return f, cmd
		}
	}

	if f.mode == ReviewCheckpointRequestChanges {
		f.changeRequest, cmd = f.changeRequest.Update(msg)
		return f, cmd
	}

	return f, nil
}

func (f ReviewCheckpointForm) ActionState() execution.DecisionState {
	switch f.selectedAction {
	case 0:
		return execution.DecisionStateApprovedContinue
	case 1:
		return execution.DecisionStateApprovedQuit
	case 2:
		return execution.DecisionStateChangesRequested
	case 3:
		return execution.DecisionStateRejected
	default:
		return execution.DecisionStateApprovedContinue
	}
}

func (f ReviewCheckpointForm) GetChangeRequest() string {
	return strings.TrimSpace(f.changeRequest.Value())
}

func (f *ReviewCheckpointForm) ApplyFilePickerSelection(selectedPath string) bool {
	if selectedPath == "" {
		return false
	}
	if f.filePicker.ActiveField != reviewCheckpointPickerChangeRequest {
		return false
	}
	applyFilePickerToTextarea(&f.changeRequest, f.requestAnchor, f.filePicker.Query, selectedPath)
	f.filePicker.Close()
	return true
}

func (f *ReviewCheckpointForm) handleFilePickerKey(msg tea.KeyMsg) bool {
	if f.mode != ReviewCheckpointRequestChanges {
		return false
	}

	anchor := textareaCursorAnchor(f.changeRequest)
	prevOpen := f.filePicker.Open
	prevQuery := f.filePicker.Query

	result, err := HandleFilePickerKey(&f.filePicker, msg, FilePickerKeyOptions{
		Field:  reviewCheckpointPickerChangeRequest,
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

func HandleReviewCheckpointKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.reviewCheckpointForm == nil {
		m.actionMode = ActionModeNone
		return m, nil
	}

	if m.reviewCheckpointForm.mode == ReviewCheckpointChooseAction {
		switch msg.String() {
		case "enter":
			action := m.reviewCheckpointForm.ActionState()
			switch action {
			case execution.DecisionStateChangesRequested:
				updated := *m.reviewCheckpointForm
				updated.mode = ReviewCheckpointRequestChanges
				updated.changeRequest.Focus()
				m.reviewCheckpointForm = &updated
				return m, nil
			default:
				return startReviewDecision(m, action, "")
			}
		default:
			updatedForm, cmd := m.reviewCheckpointForm.Update(msg)
			m.reviewCheckpointForm = &updatedForm
			return m, cmd
		}
	}

	if m.reviewCheckpointForm.mode == ReviewCheckpointRequestChanges {
		if m.reviewCheckpointForm.filePicker.Open {
			updatedForm, cmd := m.reviewCheckpointForm.Update(msg)
			m.reviewCheckpointForm = &updatedForm
			return m, cmd
		}
		switch msg.String() {
		case "ctrl+s", "ctrl+enter":
			feedback := m.reviewCheckpointForm.GetChangeRequest()
			if strings.TrimSpace(feedback) == "" {
				return m, nil
			}
			return startReviewDecision(m, execution.DecisionStateChangesRequested, feedback)
		case "esc":
			updated := *m.reviewCheckpointForm
			updated.mode = ReviewCheckpointChooseAction
			updated.changeRequest.Blur()
			m.reviewCheckpointForm = &updated
			return m, nil
		default:
			updatedForm, cmd := m.reviewCheckpointForm.Update(msg)
			m.reviewCheckpointForm = &updatedForm
			return m, cmd
		}
	}

	return m, nil
}

func startReviewDecision(m Model, action execution.DecisionState, feedback string) (Model, tea.Cmd) {
	if m.reviewCheckpointForm == nil {
		return m, nil
	}

	taskID := m.reviewCheckpointForm.task.ID
	if strings.TrimSpace(taskID) == "" {
		taskID = m.reviewCheckpointForm.run.TaskID
	}
	runID := m.reviewCheckpointForm.run.ID

	m.actionMode = ActionModeNone
	m.reviewCheckpointForm = nil
	m.actionInProgress = true
	m.actionName = reviewDecisionActionName(action)

	ctx, cancel := context.WithCancel(context.Background())
	m.actionCancel = cancel

	if action == execution.DecisionStateChangesRequested {
		streamCh, stdout, stderr := m.startLiveOutput()
		return m, tea.Batch(
			ResolveDecisionCmdWithContextAndStream(
				ctx,
				taskID,
				runID,
				action,
				feedback,
				m.config.Execution.StopAfterEachTask,
				m.config.Execution.ParentReviewEnabled,
				stdout,
				stderr,
				streamCh,
			),
			listenLiveOutputCmd(streamCh),
			spinnerTickCmd(),
		)
	}

	return m, tea.Batch(
		ResolveDecisionCmdWithContext(
			ctx,
			taskID,
			runID,
			action,
			feedback,
			m.config.Execution.StopAfterEachTask,
			m.config.Execution.ParentReviewEnabled,
		),
		spinnerTickCmd(),
	)
}

func reviewDecisionActionName(action execution.DecisionState) string {
	switch action {
	case execution.DecisionStateApprovedContinue:
		return "Recording decision..."
	case execution.DecisionStateApprovedQuit:
		return "Recording decision..."
	case execution.DecisionStateChangesRequested:
		return "Resuming..."
	case execution.DecisionStateRejected:
		return "Recording decision..."
	default:
		return "Recording decision..."
	}
}

func RenderReviewCheckpointModal(m Model, form ReviewCheckpointForm) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("69")).
		Foreground(lipgloss.Color("15")).
		Bold(true)
	unselectedStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("15"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	modalWidth := m.windowWidth - 4
	if modalWidth < 60 {
		modalWidth = 60
	}
	modalHeight := m.windowHeight - 3
	if modalHeight < 12 {
		modalHeight = 12
	}

	contentWidth := modalWidth - 6
	if contentWidth < 1 {
		contentWidth = 1
	}

	lines := make([]string, 0)
	title := titleStyle.Render("Task review checkpoint")
	lines = append(lines, title, "")

	taskLine := form.task.ID
	if strings.TrimSpace(form.task.Title) != "" {
		taskLine = fmt.Sprintf("%s - %s", form.task.ID, form.task.Title)
	}
	if strings.TrimSpace(taskLine) == "" {
		taskLine = form.run.TaskID
	}
	lines = append(lines, labelStyle.Render("Task:")+" "+textStyle.Render(truncateField(taskLine, contentWidth)))
	lines = append(lines, labelStyle.Render("Run status:")+" "+renderRunStatus(form.run.Status))
	lines = append(lines, "")

	lines = append(lines, labelStyle.Render("Review summary:"))
	lines = append(lines, renderReviewSummaryLines(form.reviewSummary(), contentWidth, labelStyle, textStyle, mutedStyle)...)
	lines = append(lines, "")

	if form.mode == ReviewCheckpointChooseAction {
		lines = append(lines, labelStyle.Render("Choose an action:"), "")

		actions := []string{
			"1. Approve and continue",
			"2. Approve and quit",
			"3. Request changes",
			"4. Reject changes",
		}
		for idx, label := range actions {
			if idx == form.selectedAction {
				lines = append(lines, selectedStyle.Render(label))
			} else {
				lines = append(lines, unselectedStyle.Render(label))
			}
		}
		lines = append(lines, "")
		lines = append(lines, mutedStyle.Render("[up/down] navigate  [1-4] select  [enter] confirm  [ctrl+c] quit"))
	}

	if form.mode == ReviewCheckpointRequestChanges {
		lines = append(lines, labelStyle.Render("Change request:"))
		lines = append(lines, form.changeRequest.View())

		picker := ""
		if form.filePicker.Open {
			pickerWidth := form.changeRequest.Width()
			if pickerWidth < 1 {
				pickerWidth = contentWidth
			}
			if pickerWidth > contentWidth {
				pickerWidth = contentWidth
			}
			baseHeight := lipgloss.Height(lipgloss.JoinVertical(lipgloss.Left, append(lines, "", "")...))
			available := modalHeight - baseHeight
			if available < 1 {
				available = 1
			}
			if available > 6 {
				available = 6
			}
			picker = RenderFilePickerList(form.filePicker, pickerWidth, available)
		}
		if picker != "" {
			lines = append(lines, picker)
		} else {
			lines = append(lines, "")
		}

		if strings.TrimSpace(form.changeRequest.Value()) == "" {
			lines = append(lines, errorStyle.Render("Change request cannot be empty"), "")
		}

		lines = append(lines, mutedStyle.Render("[ctrl+s or ctrl+enter] submit  Enter: new line  [esc] back"))
	}

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("214")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight)

	modal := modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return modal
}

func (f ReviewCheckpointForm) reviewSummary() *execution.ReviewSummary {
	return f.run.ReviewSummary
}

func renderReviewSummaryLines(summary *execution.ReviewSummary, width int, labelStyle, textStyle, mutedStyle lipgloss.Style) []string {
	if summary == nil {
		return []string{mutedStyle.Render("No review summary available.")}
	}

	var lines []string

	if len(summary.Files) == 0 {
		lines = append(lines, textStyle.Render("Files: (none)"))
	} else {
		lines = append(lines, labelStyle.Render("Files:"))
		maxFiles := 10
		for i, file := range summary.Files {
			if i >= maxFiles {
				lines = append(lines, mutedStyle.Render(fmt.Sprintf("... and %d more", len(summary.Files)-maxFiles)))
				break
			}
			lines = append(lines, textStyle.Render("- "+truncateField(file, width)))
		}
	}

	diffStat := strings.TrimSpace(summary.DiffStat)
	if diffStat == "" {
		lines = append(lines, textStyle.Render("Diffstat: (none)"))
	} else {
		lines = append(lines, labelStyle.Render("Diffstat:"))
		statLines := strings.Split(diffStat, "\n")
		maxLines := 8
		for i, line := range statLines {
			if i >= maxLines {
				lines = append(lines, mutedStyle.Render(fmt.Sprintf("... and %d more lines", len(statLines)-maxLines)))
				break
			}
			lines = append(lines, textStyle.Render("  "+truncateField(line, width)))
		}
	}

	if len(summary.Snippets) == 0 {
		lines = append(lines, textStyle.Render("Snippets: (none)"))
	} else {
		lines = append(lines, labelStyle.Render("Snippets:"))
		maxSnippets := 2
		for i, snippet := range summary.Snippets {
			if i >= maxSnippets {
				lines = append(lines, mutedStyle.Render(fmt.Sprintf("... and %d more", len(summary.Snippets)-maxSnippets)))
				break
			}
			if strings.TrimSpace(snippet.File) != "" {
				lines = append(lines, textStyle.Render("File: "+truncateField(snippet.File, width)))
			}
			snippetText := strings.TrimSpace(snippet.Snippet)
			if snippetText == "" {
				continue
			}
			for _, line := range strings.Split(snippetText, "\n") {
				if strings.TrimSpace(line) == "" {
					lines = append(lines, textStyle.Render("  "))
					continue
				}
				lines = append(lines, textStyle.Render("  "+truncateField(line, width)))
			}
		}
	}

	return lines
}

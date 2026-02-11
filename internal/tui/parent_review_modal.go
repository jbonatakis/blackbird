package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

type ParentReviewModalAction int

const (
	ParentReviewModalActionNone ParentReviewModalAction = iota
	ParentReviewModalActionContinue
	ParentReviewModalActionResumeAllFailed
	ParentReviewModalActionResumeOneTask
	ParentReviewModalActionQuit
)

type ParentReviewModalMode int

const (
	ParentReviewModalModeActions ParentReviewModalMode = iota
)

const (
	parentReviewActionContinue = iota
	parentReviewActionResumeAllFailed
	parentReviewActionResumeOneTask
	parentReviewActionQuit
)

type ParentReviewForm struct {
	run              execution.RunRecord
	parentTask       execution.TaskContext
	taskResults      execution.ParentReviewTaskResults
	reviewedTaskIDs  []string
	resumeTaskIDs    []string
	selectedAction   int
	selectedTarget   int
	fallbackFeedback string
	actionError      string
	width            int
	height           int
}

func NewParentReviewForm(run execution.RunRecord, g plan.WorkGraph) ParentReviewForm {
	taskResults := execution.ParentReviewTaskResultsForRecord(run)
	reviewedTaskIDs := parentReviewModalReviewedTaskIDs(taskResults)
	resumeTaskIDs := normalizeParentReviewModalTaskIDs(execution.ParentReviewFailedTaskIDs(run))
	selectedTarget := -1
	if len(resumeTaskIDs) > 0 {
		selectedTarget = 0
	}

	return ParentReviewForm{
		run:              run,
		parentTask:       resolveParentReviewModalTaskContext(run, g),
		taskResults:      taskResults,
		reviewedTaskIDs:  reviewedTaskIDs,
		resumeTaskIDs:    resumeTaskIDs,
		selectedAction:   parentReviewActionContinue,
		selectedTarget:   selectedTarget,
		fallbackFeedback: normalizeParentReviewModalFeedback(execution.ParentReviewPrimaryFeedback(run)),
		width:            90,
		height:           30,
	}
}

func (f *ParentReviewForm) SetSize(width, height int) {
	f.width = width
	f.height = height
}

func (f ParentReviewForm) ResumeTargets() []string {
	return append([]string{}, f.resumeTaskIDs...)
}

func (f ParentReviewForm) ReviewedTaskIDs() []string {
	return append([]string{}, f.reviewedTaskIDs...)
}

func (f ParentReviewForm) SelectedAction() int {
	return f.selectedAction
}

func (f ParentReviewForm) SelectedTarget() string {
	if f.selectedTarget < 0 || f.selectedTarget >= len(f.resumeTaskIDs) {
		return ""
	}
	return f.resumeTaskIDs[f.selectedTarget]
}

func (f ParentReviewForm) HasFailedTasks() bool {
	return len(f.resumeTaskIDs) > 0
}

func (f ParentReviewForm) Feedback() string {
	if selected := f.SelectedTarget(); selected != "" {
		if feedback := f.feedbackForTask(selected); feedback != "" {
			return feedback
		}
	}
	return f.fallbackFeedback
}

func (f ParentReviewForm) ResumeFeedbackForTask(taskID string) string {
	return f.feedbackForTask(taskID)
}

func (f ParentReviewForm) Mode() ParentReviewModalMode {
	return ParentReviewModalModeActions
}

func (f *ParentReviewForm) SetActionError(message string) {
	f.actionError = strings.TrimSpace(message)
}

func (f ParentReviewForm) Update(msg tea.Msg) (ParentReviewForm, ParentReviewModalAction) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, ParentReviewModalActionNone
	}
	return f.updateActionSelection(keyMsg)
}

func (f ParentReviewForm) updateActionSelection(keyMsg tea.KeyMsg) (ParentReviewForm, ParentReviewModalAction) {
	switch keyMsg.String() {
	case "up", "k":
		f.actionError = ""
		f.selectedAction = f.prevSelectableAction(f.selectedAction)
		return f, ParentReviewModalActionNone
	case "down", "j":
		f.actionError = ""
		f.selectedAction = f.nextSelectableAction(f.selectedAction)
		return f, ParentReviewModalActionNone
	case "left":
		f.actionError = ""
		if f.selectedTarget > 0 {
			f.selectedTarget--
		}
		return f, ParentReviewModalActionNone
	case "right":
		f.actionError = ""
		if f.selectedTarget >= 0 && f.selectedTarget < len(f.resumeTaskIDs)-1 {
			f.selectedTarget++
		}
		return f, ParentReviewModalActionNone
	case "1":
		f.actionError = ""
		f.selectedAction = parentReviewActionContinue
		return f, ParentReviewModalActionNone
	case "2":
		f.actionError = ""
		if f.isSelectableAction(parentReviewActionResumeAllFailed) {
			f.selectedAction = parentReviewActionResumeAllFailed
		}
		return f, ParentReviewModalActionNone
	case "3":
		f.actionError = ""
		if f.isSelectableAction(parentReviewActionResumeOneTask) {
			f.selectedAction = parentReviewActionResumeOneTask
		}
		return f, ParentReviewModalActionNone
	case "4":
		f.actionError = ""
		f.selectedAction = parentReviewActionQuit
		return f, ParentReviewModalActionNone
	case "enter":
		f.actionError = ""
		return f, f.actionForSelection()
	case "esc":
		f.actionError = ""
		return f, ParentReviewModalActionContinue
	default:
		return f, ParentReviewModalActionNone
	}
}

func (f ParentReviewForm) actionForSelection() ParentReviewModalAction {
	switch f.selectedAction {
	case parentReviewActionContinue:
		return ParentReviewModalActionContinue
	case parentReviewActionResumeAllFailed:
		if !f.isSelectableAction(parentReviewActionResumeAllFailed) {
			return ParentReviewModalActionNone
		}
		return ParentReviewModalActionResumeAllFailed
	case parentReviewActionResumeOneTask:
		if !f.isSelectableAction(parentReviewActionResumeOneTask) {
			return ParentReviewModalActionNone
		}
		if f.SelectedTarget() == "" {
			return ParentReviewModalActionNone
		}
		return ParentReviewModalActionResumeOneTask
	case parentReviewActionQuit:
		return ParentReviewModalActionQuit
	default:
		return ParentReviewModalActionNone
	}
}

func (f ParentReviewForm) isSelectableAction(action int) bool {
	switch action {
	case parentReviewActionContinue, parentReviewActionQuit:
		return true
	case parentReviewActionResumeAllFailed, parentReviewActionResumeOneTask:
		return f.HasFailedTasks()
	default:
		return false
	}
}

func (f ParentReviewForm) nextSelectableAction(current int) int {
	for candidate := current + 1; candidate <= parentReviewActionQuit; candidate++ {
		if f.isSelectableAction(candidate) {
			return candidate
		}
	}
	return current
}

func (f ParentReviewForm) prevSelectableAction(current int) int {
	for candidate := current - 1; candidate >= parentReviewActionContinue; candidate-- {
		if f.isSelectableAction(candidate) {
			return candidate
		}
	}
	return current
}

func (f ParentReviewForm) feedbackForTask(taskID string) string {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ""
	}
	result, ok := f.taskResults[taskID]
	if !ok || result.Status != execution.ParentReviewTaskStatusFailed {
		return ""
	}
	if feedback := normalizeParentReviewModalFeedback(result.Feedback); feedback != "" {
		return feedback
	}
	return f.fallbackFeedback
}

func RenderParentReviewModal(m Model, form ParentReviewForm) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	passStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	selectedStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("69")).
		Foreground(lipgloss.Color("15")).
		Bold(true)
	unselectedStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("15"))
	disabledStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("242"))
	errorStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("196")).
		Background(lipgloss.Color("52")).
		Bold(true)

	modalWidth := m.windowWidth - 4
	if modalWidth < 64 {
		modalWidth = 64
	}
	modalHeight := m.windowHeight - 3
	if modalHeight < 18 {
		modalHeight = 18
	}
	contentWidth := modalWidth - 8
	if contentWidth < 20 {
		contentWidth = 20
	}

	lines := make([]string, 0)
	lines = append(lines, titleStyle.Render("Post-review results"), "")

	parentDetail := parentReviewModalTaskLine(form)
	lines = append(lines, labelStyle.Render("Parent task:")+" "+textStyle.Render(truncateField(parentDetail, contentWidth)))
	lines = append(lines, labelStyle.Render("Review run:")+" "+textStyle.Render(truncateField(parentReviewModalRunID(form.run), contentWidth)))

	outcome := "failed"
	outcomeStyle := failStyle
	if parentReviewModalPassed(form.run) {
		outcome = "passed"
		outcomeStyle = passStyle
	}
	lines = append(lines, labelStyle.Render("Outcome:")+" "+outcomeStyle.Render(outcome))
	lines = append(lines, "")
	if strings.TrimSpace(form.actionError) != "" {
		lines = append(lines, errorStyle.Render(truncateField(form.actionError, contentWidth)))
		lines = append(lines, "")
	}

	lines = append(lines, labelStyle.Render("Reviewed tasks:"))
	if len(form.reviewedTaskIDs) == 0 {
		lines = append(lines, mutedStyle.Render("  (none)"))
	} else {
		taskWidth := contentWidth - 14
		if taskWidth < 8 {
			taskWidth = 8
		}
		feedbackWidth := contentWidth - 18
		if feedbackWidth < 10 {
			feedbackWidth = 10
		}
		for _, taskID := range form.reviewedTaskIDs {
			result := form.taskResults[taskID]
			statusLabel := "[PASS]"
			statusStyle := passStyle
			if result.Status == execution.ParentReviewTaskStatusFailed {
				statusLabel = "[FAIL]"
				statusStyle = failStyle
			}
			lines = append(lines, "  "+statusStyle.Render(statusLabel)+" "+textStyle.Render(truncateField(taskID, taskWidth)))

			taskFeedback := form.feedbackForTask(taskID)
			if taskFeedback == "" {
				continue
			}
			feedbackLines := strings.Split(taskFeedback, "\n")
			for idx, line := range feedbackLines {
				prefix := "      "
				if idx == 0 {
					prefix = "      feedback: "
				}
				lines = append(lines, textStyle.Render(prefix+truncateField(line, feedbackWidth)))
			}
		}
	}
	lines = append(lines, "")

	if !form.HasFailedTasks() {
		lines = append(lines, mutedStyle.Render("No failed tasks were reported; resume actions are disabled."))
		lines = append(lines, "")
	} else {
		lines = append(lines, labelStyle.Render("Resume one target:")+" "+textStyle.Render(form.SelectedTarget()))
		if len(form.resumeTaskIDs) > 1 {
			lines = append(lines, mutedStyle.Render("[←/→] change target"))
		}
		lines = append(lines, "")
	}

	lines = append(lines, labelStyle.Render("Actions:"))
	lines = append(lines, "")
	actions := []struct {
		index int
		label string
	}{
		{index: parentReviewActionContinue, label: "1. Continue"},
		{index: parentReviewActionResumeAllFailed, label: "2. Resume all failed"},
		{index: parentReviewActionResumeOneTask, label: "3. Resume one task"},
		{index: parentReviewActionQuit, label: "4. Quit"},
	}
	for _, action := range actions {
		if !form.isSelectableAction(action.index) {
			lines = append(lines, disabledStyle.Render(action.label+" (disabled)"))
			continue
		}
		if form.selectedAction == action.index {
			lines = append(lines, selectedStyle.Render(action.label))
		} else {
			lines = append(lines, unselectedStyle.Render(action.label))
		}
	}
	lines = append(lines, "")
	if len(form.resumeTaskIDs) > 1 {
		lines = append(lines, mutedStyle.Render("[↑/↓]navigate  [←/→]target  [1-4]select  [enter]confirm  [esc]back"))
	} else {
		lines = append(lines, mutedStyle.Render("[↑/↓]navigate  [1-4]select  [enter]confirm  [esc]back"))
	}

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight)

	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func normalizeParentReviewModalTaskIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeParentReviewModalFeedback(feedback string) string {
	feedback = strings.ReplaceAll(feedback, "\r\n", "\n")
	feedback = strings.TrimSpace(feedback)
	if feedback == "" {
		return ""
	}

	lines := strings.Split(feedback, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func parentReviewModalReviewedTaskIDs(results execution.ParentReviewTaskResults) []string {
	if len(results) == 0 {
		return nil
	}
	ids := make([]string, 0, len(results))
	for taskID, result := range results {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			taskID = strings.TrimSpace(result.TaskID)
		}
		if taskID == "" {
			continue
		}
		ids = append(ids, taskID)
	}
	if len(ids) == 0 {
		return nil
	}
	sort.Strings(ids)
	return ids
}

func resolveParentReviewModalTaskContext(run execution.RunRecord, g plan.WorkGraph) execution.TaskContext {
	task := run.Context.Task
	if strings.TrimSpace(task.ID) == "" {
		task.ID = strings.TrimSpace(run.TaskID)
	}
	if run.Context.ParentReview != nil {
		parent := run.Context.ParentReview
		if strings.TrimSpace(task.ID) == "" {
			task.ID = strings.TrimSpace(parent.ParentTaskID)
		}
		if strings.TrimSpace(task.Title) == "" {
			task.Title = strings.TrimSpace(parent.ParentTaskTitle)
		}
		if len(task.AcceptanceCriteria) == 0 && len(parent.AcceptanceCriteria) > 0 {
			task.AcceptanceCriteria = append([]string{}, parent.AcceptanceCriteria...)
		}
	}
	if item, ok := g.Items[task.ID]; ok {
		if strings.TrimSpace(task.Title) == "" {
			task.Title = item.Title
		}
		if strings.TrimSpace(task.Description) == "" {
			task.Description = item.Description
		}
		if len(task.AcceptanceCriteria) == 0 && len(item.AcceptanceCriteria) > 0 {
			task.AcceptanceCriteria = append([]string{}, item.AcceptanceCriteria...)
		}
	}
	return task
}

func parentReviewModalTaskLine(form ParentReviewForm) string {
	taskID := strings.TrimSpace(form.parentTask.ID)
	if taskID == "" {
		taskID = strings.TrimSpace(form.run.TaskID)
	}
	taskTitle := strings.TrimSpace(form.parentTask.Title)
	if taskID == "" && taskTitle == "" {
		return "unknown"
	}
	if taskID == "" {
		return taskTitle
	}
	if taskTitle == "" {
		return taskID
	}
	return fmt.Sprintf("%s - %s", taskID, taskTitle)
}

func parentReviewModalRunID(run execution.RunRecord) string {
	runID := strings.TrimSpace(run.ID)
	if runID == "" {
		return "unknown"
	}
	return runID
}

func parentReviewModalPassed(run execution.RunRecord) bool {
	if run.ParentReviewPassed != nil {
		return *run.ParentReviewPassed
	}
	return len(execution.ParentReviewFailedTaskIDs(run)) == 0
}

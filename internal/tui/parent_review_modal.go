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
	ParentReviewModalActionResumeSelected
	ParentReviewModalActionResumeAll
	ParentReviewModalActionDismiss
)

type ParentReviewForm struct {
	run            execution.RunRecord
	parentTask     execution.TaskContext
	resumeTaskIDs  []string
	selectedTarget int
	feedback       string
	width          int
	height         int
}

func NewParentReviewForm(run execution.RunRecord, g plan.WorkGraph) ParentReviewForm {
	resumeTaskIDs := normalizeParentReviewModalTaskIDs(run.ParentReviewResumeTaskIDs)
	selectedTarget := -1
	if len(resumeTaskIDs) > 0 {
		selectedTarget = 0
	}

	return ParentReviewForm{
		run:            run,
		parentTask:     resolveParentReviewModalTaskContext(run, g),
		resumeTaskIDs:  resumeTaskIDs,
		selectedTarget: selectedTarget,
		feedback:       normalizeParentReviewModalFeedback(run.ParentReviewFeedback),
		width:          90,
		height:         30,
	}
}

func (f *ParentReviewForm) SetSize(width, height int) {
	f.width = width
	f.height = height
}

func (f ParentReviewForm) ResumeTargets() []string {
	return append([]string{}, f.resumeTaskIDs...)
}

func (f ParentReviewForm) SelectedTarget() string {
	if f.selectedTarget < 0 || f.selectedTarget >= len(f.resumeTaskIDs) {
		return ""
	}
	return f.resumeTaskIDs[f.selectedTarget]
}

func (f ParentReviewForm) Feedback() string {
	return f.feedback
}

func (f ParentReviewForm) Update(msg tea.Msg) (ParentReviewForm, ParentReviewModalAction) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, ParentReviewModalActionNone
	}

	switch keyMsg.String() {
	case "up", "k":
		if f.selectedTarget > 0 {
			f.selectedTarget--
		}
		return f, ParentReviewModalActionNone
	case "down", "j":
		if f.selectedTarget >= 0 && f.selectedTarget < len(f.resumeTaskIDs)-1 {
			f.selectedTarget++
		}
		return f, ParentReviewModalActionNone
	case "1", "enter":
		if f.SelectedTarget() == "" {
			return f, ParentReviewModalActionNone
		}
		return f, ParentReviewModalActionResumeSelected
	case "2":
		if len(f.resumeTaskIDs) == 0 {
			return f, ParentReviewModalActionNone
		}
		return f, ParentReviewModalActionResumeAll
	case "3", "esc":
		return f, ParentReviewModalActionDismiss
	default:
		return f, ParentReviewModalActionNone
	}
}

func RenderParentReviewModal(m Model, form ParentReviewForm) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	selectedStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("69")).
		Foreground(lipgloss.Color("15")).
		Bold(true)
	unselectedStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("15"))

	modalWidth := m.windowWidth - 4
	if modalWidth < 64 {
		modalWidth = 64
	}
	modalHeight := m.windowHeight - 3
	if modalHeight < 16 {
		modalHeight = 16
	}
	contentWidth := modalWidth - 8
	if contentWidth < 20 {
		contentWidth = 20
	}

	lines := make([]string, 0)
	lines = append(lines, titleStyle.Render("Parent review failed"), "")

	parentDetail := parentReviewModalTaskLine(form)
	lines = append(lines, labelStyle.Render("Parent task:")+" "+textStyle.Render(truncateField(parentDetail, contentWidth)))
	lines = append(lines, labelStyle.Render("Review run:")+" "+textStyle.Render(truncateField(parentReviewModalRunID(form.run), contentWidth)))
	lines = append(lines, labelStyle.Render("Outcome:")+" "+errorStyle.Render("failed"))
	lines = append(lines, "")

	lines = append(lines, labelStyle.Render("Resume targets:"))
	if len(form.resumeTaskIDs) == 0 {
		lines = append(lines, mutedStyle.Render("  (none)"))
	} else {
		for idx, taskID := range form.resumeTaskIDs {
			line := fmt.Sprintf("%d. %s", idx+1, taskID)
			if idx == form.selectedTarget {
				lines = append(lines, selectedStyle.Render(line))
			} else {
				lines = append(lines, unselectedStyle.Render(line))
			}
		}
	}
	lines = append(lines, "")

	lines = append(lines, labelStyle.Render("Feedback:"))
	if form.feedback == "" {
		lines = append(lines, mutedStyle.Render("  (none)"))
	} else {
		feedbackWidth := contentWidth - 2
		if feedbackWidth < 10 {
			feedbackWidth = 10
		}
		for _, line := range strings.Split(form.feedback, "\n") {
			lines = append(lines, textStyle.Render("  "+truncateField(line, feedbackWidth)))
		}
	}
	lines = append(lines, "")

	lines = append(lines, labelStyle.Render("Actions:"))
	lines = append(lines, textStyle.Render("  1. Resume selected target"))
	lines = append(lines, textStyle.Render("  2. Resume all targets"))
	lines = append(lines, textStyle.Render("  3. Dismiss"))
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render("[↑/↓] target  [1 or enter] resume selected  [2] resume all  [3 or esc] dismiss"))

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

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
)

func openParentReviewModal(m Model, run execution.RunRecord) Model {
	form := NewParentReviewForm(run, m.plan)
	form.SetSize(m.windowWidth, m.windowHeight)
	m.parentReviewForm = &form
	m.reviewCheckpointForm = nil
	m.actionMode = ActionModeParentReview
	return m
}

func HandleParentReviewKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.parentReviewForm == nil {
		m.actionMode = ActionModeNone
		return m, nil
	}

	updatedForm, action := m.parentReviewForm.Update(msg)
	m.parentReviewForm = &updatedForm

	switch action {
	case ParentReviewModalActionDismiss:
		m.actionMode = ActionModeNone
		m.parentReviewForm = nil
		return m, nil
	case ParentReviewModalActionResumeSelected:
		return m.startFeedbackResumeAction([]string{updatedForm.SelectedTarget()})
	case ParentReviewModalActionResumeAll:
		return m.startFeedbackResumeAction(updatedForm.ResumeTargets())
	default:
		return m, nil
	}
}

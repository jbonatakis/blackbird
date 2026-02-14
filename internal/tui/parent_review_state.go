package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
)

func openParentReviewModal(m Model, run execution.RunRecord) Model {
	form := NewParentReviewForm(run, m.plan)
	form.SetSize(m.windowWidth, m.windowHeight)
	m.parentReviewForm = &form
	m.parentReviewResumeState = nil
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
	case ParentReviewModalActionContinue:
		m = m.releaseLiveParentReviewAckIfExecuting()
		m.actionMode = ActionModeNone
		m.parentReviewForm = nil
		m = m.showNextQueuedParentReview()
		if m.actionMode == ActionModeNone && m.resumeExecuteAfterParentReview {
			m.resumeExecuteAfterParentReview = false
			return m.startExecuteAction(true)
		}
		return m, nil
	case ParentReviewModalActionResumeOneTask:
		m.resumeExecuteAfterParentReview = false
		targetID := strings.TrimSpace(updatedForm.SelectedTarget())
		if targetID == "" {
			return m, nil
		}
		return m.startParentReviewResumeAction(
			[]ResumePendingParentFeedbackTarget{
				{
					TaskID:   targetID,
					Feedback: updatedForm.ResumeFeedbackForTask(targetID),
				},
			},
			updatedForm,
		)
	case ParentReviewModalActionResumeAllFailed:
		m.resumeExecuteAfterParentReview = false
		targetIDs := updatedForm.ResumeTargets()
		targets := make([]ResumePendingParentFeedbackTarget, 0, len(targetIDs))
		for _, taskID := range targetIDs {
			taskID = strings.TrimSpace(taskID)
			if taskID == "" {
				continue
			}
			targets = append(targets, ResumePendingParentFeedbackTarget{
				TaskID:   taskID,
				Feedback: updatedForm.ResumeFeedbackForTask(taskID),
			})
		}
		return m.startParentReviewResumeAction(targets, updatedForm)
	case ParentReviewModalActionQuit:
		m.resumeExecuteAfterParentReview = false
		if m.actionInProgress && m.actionName == "Executing..." {
			m = cancelRunningAction(m)
		}
		m.actionMode = ActionModeNone
		m.parentReviewForm = nil
		m.queuedParentReviewRuns = nil
		return m, nil
	default:
		return m, nil
	}
}

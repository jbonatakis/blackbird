package tui

import (
	"time"

	"github.com/jbonatakis/blackbird/internal/execution"
)

func pendingDecisionRun(m Model) *execution.RunRecord {
	if m.reviewCheckpointForm != nil {
		run := m.reviewCheckpointForm.run
		return &run
	}
	return pendingDecisionRunFromData(m.runData)
}

func pendingDecisionRunFromData(data map[string]execution.RunRecord) *execution.RunRecord {
	var selected *execution.RunRecord
	for _, record := range data {
		if !isDecisionPending(record) {
			continue
		}
		copy := record
		if selected == nil {
			selected = &copy
			continue
		}
		if decisionTimestamp(copy).After(decisionTimestamp(*selected)) {
			selected = &copy
		}
	}
	return selected
}

func isDecisionPending(record execution.RunRecord) bool {
	if !record.DecisionRequired {
		return false
	}
	return record.DecisionState == "" || record.DecisionState == execution.DecisionStatePending
}

func decisionTimestamp(record execution.RunRecord) time.Time {
	if record.DecisionRequestedAt != nil {
		return *record.DecisionRequestedAt
	}
	if record.CompletedAt != nil {
		return *record.CompletedAt
	}
	return record.StartedAt
}

func openReviewCheckpointModal(m Model, run execution.RunRecord) Model {
	form := NewReviewCheckpointForm(run, m.plan)
	form.SetSize(m.windowWidth, m.windowHeight)
	m.reviewCheckpointForm = &form
	m.parentReviewForm = nil
	m.actionMode = ActionModeReviewCheckpoint
	return m
}

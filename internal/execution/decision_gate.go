package execution

import "time"

func isTerminalRunStatus(status RunStatus) bool {
	return status == RunStatusSuccess || status == RunStatusFailed
}

func requiresDecisionGate(stopAfterEachTask bool, status RunStatus) bool {
	if !stopAfterEachTask {
		return false
	}
	return isTerminalRunStatus(status)
}

func markDecisionRequired(record *RunRecord) {
	if record == nil {
		return
	}
	if record.DecisionRequired && record.DecisionState != "" && record.DecisionState != DecisionStatePending {
		return
	}
	now := time.Now().UTC()
	record.DecisionRequired = true
	if record.DecisionRequestedAt == nil {
		record.DecisionRequestedAt = &now
	}
	record.DecisionState = DecisionStatePending
	record.DecisionResolvedAt = nil
	record.DecisionFeedback = ""
}

func isDecisionPending(record RunRecord) bool {
	if !record.DecisionRequired {
		return false
	}
	return record.DecisionState == "" || record.DecisionState == DecisionStatePending
}

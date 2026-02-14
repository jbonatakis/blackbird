package execution

import "strings"

// ExecutionStage is the orchestration stage for live execution UI state.
type ExecutionStage string

const (
	ExecutionStageIdle       ExecutionStage = "idle"
	ExecutionStageExecuting  ExecutionStage = "executing"
	ExecutionStageReviewing  ExecutionStage = "reviewing"
	ExecutionStagePostReview ExecutionStage = "post_review"
)

// ExecutionStageState captures the current orchestration stage and related task IDs.
type ExecutionStageState struct {
	Stage          ExecutionStage `json:"stage"`
	TaskID         string         `json:"taskId,omitempty"`
	ReviewedTaskID string         `json:"reviewedTaskId,omitempty"`
}

func emitExecutionStageState(emit func(ExecutionStageState), state ExecutionStageState) {
	if emit == nil {
		return
	}
	emit(normalizeExecutionStageState(state))
}

func normalizeExecutionStageState(state ExecutionStageState) ExecutionStageState {
	state.TaskID = strings.TrimSpace(state.TaskID)
	state.ReviewedTaskID = strings.TrimSpace(state.ReviewedTaskID)

	if state.Stage == "" {
		state.Stage = ExecutionStageIdle
	}
	if state.Stage != ExecutionStageReviewing {
		state.ReviewedTaskID = ""
	}
	if state.Stage == ExecutionStageIdle {
		state.TaskID = ""
	}

	return state
}

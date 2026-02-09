package tui

import (
	"testing"

	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestCanResume(t *testing.T) {
	tests := []struct {
		name                  string
		selectedID            string
		plan                  plan.WorkGraph
		runData               map[string]execution.RunRecord
		pendingParentFeedback map[string]execution.PendingParentReviewFeedback
		want                  bool
	}{
		{
			name:                  "no selected ID",
			selectedID:            "",
			plan:                  plan.NewEmptyWorkGraph(),
			runData:               map[string]execution.RunRecord{},
			pendingParentFeedback: map[string]execution.PendingParentReviewFeedback{},
			want:                  false,
		},
		{
			name:       "task not in waiting_user status",
			selectedID: "task-1",
			plan: plan.WorkGraph{
				Items: map[string]plan.WorkItem{
					"task-1": {
						ID:     "task-1",
						Title:  "Task 1",
						Status: plan.StatusTodo,
					},
				},
			},
			runData:               map[string]execution.RunRecord{},
			pendingParentFeedback: map[string]execution.PendingParentReviewFeedback{},
			want:                  false,
		},
		{
			name:       "task in waiting_user status but no waiting runs",
			selectedID: "task-1",
			plan: plan.WorkGraph{
				Items: map[string]plan.WorkItem{
					"task-1": {
						ID:     "task-1",
						Title:  "Task 1",
						Status: plan.StatusWaitingUser,
					},
				},
			},
			runData:               map[string]execution.RunRecord{},
			pendingParentFeedback: map[string]execution.PendingParentReviewFeedback{},
			want:                  false,
		},
		{
			name:       "task in waiting_user status with waiting run",
			selectedID: "task-1",
			plan: plan.WorkGraph{
				Items: map[string]plan.WorkItem{
					"task-1": {
						ID:     "task-1",
						Title:  "Task 1",
						Status: plan.StatusWaitingUser,
					},
				},
			},
			runData: map[string]execution.RunRecord{
				"run-1": {
					TaskID: "task-1",
					Status: "waiting_user",
				},
			},
			pendingParentFeedback: map[string]execution.PendingParentReviewFeedback{},
			want:                  true,
		},
		{
			name:       "task in waiting_user status with completed run",
			selectedID: "task-1",
			plan: plan.WorkGraph{
				Items: map[string]plan.WorkItem{
					"task-1": {
						ID:     "task-1",
						Title:  "Task 1",
						Status: plan.StatusWaitingUser,
					},
				},
			},
			runData: map[string]execution.RunRecord{
				"run-1": {
					TaskID: "task-1",
					Status: "completed",
				},
			},
			pendingParentFeedback: map[string]execution.PendingParentReviewFeedback{},
			want:                  false,
		},
		{
			name:       "task with pending parent review feedback can resume without waiting run",
			selectedID: "task-1",
			plan: plan.WorkGraph{
				Items: map[string]plan.WorkItem{
					"task-1": {
						ID:     "task-1",
						Title:  "Task 1",
						Status: plan.StatusDone,
					},
				},
			},
			runData: map[string]execution.RunRecord{},
			pendingParentFeedback: map[string]execution.PendingParentReviewFeedback{
				"task-1": {
					ParentTaskID: "parent-1",
					ReviewRunID:  "review-1",
					Feedback:     "update task",
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				selectedID:            tt.selectedID,
				plan:                  tt.plan,
				runData:               tt.runData,
				pendingParentFeedback: tt.pendingParentFeedback,
			}
			got := CanResume(m)
			if got != tt.want {
				t.Errorf("CanResume() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleSetStatusKey(t *testing.T) {
	tests := []struct {
		name              string
		key               string
		pendingStatusID   string
		wantActionMode    ActionMode
		wantPendingID     string
		wantActionStarted bool
	}{
		{
			name:              "escape cancels",
			key:               "esc",
			pendingStatusID:   "task-1",
			wantActionMode:    ActionModeNone,
			wantPendingID:     "",
			wantActionStarted: false,
		},
		{
			name:              "1 selects todo status",
			key:               "1",
			pendingStatusID:   "task-1",
			wantActionMode:    ActionModeNone,
			wantPendingID:     "",
			wantActionStarted: true,
		},
		{
			name:              "6 selects done status",
			key:               "6",
			pendingStatusID:   "task-1",
			wantActionMode:    ActionModeNone,
			wantPendingID:     "",
			wantActionStarted: true,
		},
		{
			name:              "invalid key does nothing",
			key:               "x",
			pendingStatusID:   "task-1",
			wantActionMode:    ActionModeSetStatus,
			wantPendingID:     "task-1",
			wantActionStarted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				actionMode:      ActionModeSetStatus,
				pendingStatusID: tt.pendingStatusID,
			}
			gotModel, _ := HandleSetStatusKey(m, tt.key)
			if gotModel.actionMode != tt.wantActionMode {
				t.Errorf("actionMode = %v, want %v", gotModel.actionMode, tt.wantActionMode)
			}
			if gotModel.pendingStatusID != tt.wantPendingID {
				t.Errorf("pendingStatusID = %v, want %v", gotModel.pendingStatusID, tt.wantPendingID)
			}
			if tt.wantActionStarted && !gotModel.actionInProgress {
				t.Errorf("expected action to be started")
			}
		})
	}
}

func TestRenderActionOutput(t *testing.T) {
	tests := []struct {
		name   string
		output *ActionOutput
		width  int
		want   string
	}{
		{
			name:   "nil output returns empty string",
			output: nil,
			width:  80,
			want:   "",
		},
		{
			name: "success message renders",
			output: &ActionOutput{
				Message: "Success!",
				IsError: false,
			},
			width: 80,
			want:  "Success!",
		},
		{
			name: "error message renders",
			output: &ActionOutput{
				Message: "Error occurred",
				IsError: true,
			},
			width: 80,
			want:  "Error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderActionOutput(tt.output, tt.width)
			if tt.output == nil && got != "" {
				t.Errorf("expected empty string for nil output, got %q", got)
			}
			if tt.output != nil && got == "" {
				t.Errorf("expected non-empty output, got empty string")
			}
		})
	}
}

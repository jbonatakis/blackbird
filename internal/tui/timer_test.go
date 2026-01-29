package tui

import (
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/execution"
)

func TestHasActiveRuns(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name    string
		runData map[string]execution.RunRecord
		want    bool
	}{
		{name: "empty", runData: map[string]execution.RunRecord{}, want: false},
		{
			name: "running",
			runData: map[string]execution.RunRecord{
				"task-1": {Status: execution.RunStatusRunning, StartedAt: now},
			},
			want: true,
		},
		{
			name: "waiting-user",
			runData: map[string]execution.RunRecord{
				"task-1": {Status: execution.RunStatusWaitingUser, StartedAt: now},
			},
			want: true,
		},
		{
			name: "completed",
			runData: map[string]execution.RunRecord{
				"task-1": {Status: execution.RunStatusSuccess, StartedAt: now},
			},
			want: false,
		},
	}

	for _, tc := range cases {
		if got := hasActiveRuns(tc.runData); got != tc.want {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, got)
		}
	}
}

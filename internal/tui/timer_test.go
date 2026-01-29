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

func TestFormatElapsed(t *testing.T) {
	// Mock the time.Now function for predictable testing
	originalTimeNow := timeNow
	defer func() { timeNow = originalTimeNow }()

	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return baseTime }

	tests := []struct {
		name        string
		startedAt   time.Time
		completedAt *time.Time
		want        string
	}{
		{
			name:        "zero duration",
			startedAt:   baseTime,
			completedAt: nil,
			want:        "00:00:00",
		},
		{
			name:        "1 second",
			startedAt:   baseTime.Add(-1 * time.Second),
			completedAt: nil,
			want:        "00:00:01",
		},
		{
			name:        "30 seconds",
			startedAt:   baseTime.Add(-30 * time.Second),
			completedAt: nil,
			want:        "00:00:30",
		},
		{
			name:        "1 minute",
			startedAt:   baseTime.Add(-1 * time.Minute),
			completedAt: nil,
			want:        "00:01:00",
		},
		{
			name:        "5 minutes 30 seconds",
			startedAt:   baseTime.Add(-5*time.Minute - 30*time.Second),
			completedAt: nil,
			want:        "00:05:30",
		},
		{
			name:        "1 hour",
			startedAt:   baseTime.Add(-1 * time.Hour),
			completedAt: nil,
			want:        "01:00:00",
		},
		{
			name:        "2 hours 15 minutes 45 seconds",
			startedAt:   baseTime.Add(-2*time.Hour - 15*time.Minute - 45*time.Second),
			completedAt: nil,
			want:        "02:15:45",
		},
		{
			name:        "completed run",
			startedAt:   baseTime.Add(-5 * time.Minute),
			completedAt: timePtr(baseTime.Add(-2 * time.Minute)),
			want:        "00:03:00",
		},
		{
			name:        "completed instantly",
			startedAt:   baseTime,
			completedAt: timePtr(baseTime),
			want:        "00:00:00",
		},
		{
			name:        "end before start (defensive)",
			startedAt:   baseTime,
			completedAt: timePtr(baseTime.Add(-1 * time.Minute)),
			want:        "00:00:00",
		},
		{
			name:        "10 hours",
			startedAt:   baseTime.Add(-10 * time.Hour),
			completedAt: nil,
			want:        "10:00:00",
		},
		{
			name:        "truncates milliseconds",
			startedAt:   baseTime.Add(-1*time.Second - 500*time.Millisecond),
			completedAt: nil,
			want:        "00:00:01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatElapsed(tt.startedAt, tt.completedAt)
			if got != tt.want {
				t.Errorf("formatElapsed() = %q, want %q", got, tt.want)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRenderExecutionViewActiveRun(t *testing.T) {
	now := time.Date(2026, 1, 29, 12, 0, 0, 0, time.UTC)
	originalTimeNow := timeNow
	timeNow = func() time.Time { return now }
	t.Cleanup(func() { timeNow = originalTimeNow })

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task-1": {
				ID:        "task-1",
				Title:     "Ready task",
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
			"task-2": {
				ID:        "task-2",
				Title:     "Blocked task",
				Status:    plan.StatusTodo,
				Deps:      []string{"task-1"},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	stdout := makeLines("out", 25)
	stderr := makeLines("err", 5)
	started := now.Add(-90 * time.Second)
	runData := map[string]execution.RunRecord{
		"task-1": {
			ID:        "run-1",
			TaskID:    "task-1",
			StartedAt: started,
			Status:    execution.RunStatusRunning,
			Stdout:    stdout,
			Stderr:    stderr,
		},
	}

	model := Model{
		plan:    g,
		runData: runData,
	}

	out := RenderExecutionView(model)

	assertContains(t, out, "Active Run")
	assertContains(t, out, "Task: task-1")
	assertContains(t, out, "Elapsed: 00:01:30")
	assertContains(t, out, "Log Output")
	assertContains(t, out, "STDOUT:")
	assertContains(t, out, "out-25")
	if strings.Contains(out, "out-01") {
		t.Fatalf("expected stdout tail to exclude out-01, got %q", out)
	}
	assertContains(t, out, "STDERR:")
	assertContains(t, out, "err-05")
	assertContains(t, out, "Task Summary")
	assertContains(t, out, "Ready: 1")
	assertContains(t, out, "Blocked: 1")
}

func TestRenderExecutionViewEmptyState(t *testing.T) {
	model := Model{}
	out := RenderExecutionView(model)
	assertContains(t, out, "No active runs.")
	assertContains(t, out, "(no logs)")
}

func TestRenderExecutionViewLiveOutput(t *testing.T) {
	model := Model{
		actionInProgress: true,
		actionName:       "Executing...",
		liveStdout:       makeLines("live", 3),
		liveStderr:       "boom",
	}
	out := RenderExecutionView(model)
	assertContains(t, out, "Status: running")
	assertContains(t, out, "STDOUT:")
	assertContains(t, out, "live-03")
	assertContains(t, out, "STDERR:")
	assertContains(t, out, "boom")
	if strings.Contains(out, "(no logs)") {
		t.Fatalf("expected live output, got %q", out)
	}
}

func TestRenderExecutionViewLiveOutputOverridesRunLogs(t *testing.T) {
	runData := map[string]execution.RunRecord{
		"task-1": {
			ID:        "run-1",
			TaskID:    "task-1",
			StartedAt: time.Now().Add(-30 * time.Second),
			Status:    execution.RunStatusRunning,
			Stdout:    makeLines("run", 2),
			Stderr:    "run-err",
		},
	}

	model := Model{
		actionInProgress: true,
		actionName:       "Resuming...",
		liveStdout:       makeLines("live", 2),
		liveStderr:       "live-err",
		runData:          runData,
	}

	out := RenderExecutionView(model)
	assertContains(t, out, "STDOUT:")
	assertContains(t, out, "live-02")
	if strings.Contains(out, "run-02") {
		t.Fatalf("expected live stdout to override run stdout, got %q", out)
	}
	assertContains(t, out, "STDERR:")
	assertContains(t, out, "live-err")
	if strings.Contains(out, "run-err") {
		t.Fatalf("expected live stderr to override run stderr, got %q", out)
	}
}

func TestRenderExecutionViewRunOutputWhenNotInProgress(t *testing.T) {
	runData := map[string]execution.RunRecord{
		"task-1": {
			ID:        "run-1",
			TaskID:    "task-1",
			StartedAt: time.Now().Add(-30 * time.Second),
			Status:    execution.RunStatusRunning,
			Stdout:    makeLines("run", 2),
			Stderr:    "run-err",
		},
	}

	model := Model{
		actionInProgress: false,
		actionName:       "",
		liveStdout:       makeLines("live", 2),
		liveStderr:       "live-err",
		runData:          runData,
	}

	out := RenderExecutionView(model)
	assertContains(t, out, "STDOUT:")
	assertContains(t, out, "run-02")
	if strings.Contains(out, "live-02") {
		t.Fatalf("expected run stdout when not in progress, got %q", out)
	}
	assertContains(t, out, "STDERR:")
	assertContains(t, out, "run-err")
	if strings.Contains(out, "live-err") {
		t.Fatalf("expected run stderr when not in progress, got %q", out)
	}
}

func makeLines(prefix string, count int) string {
	lines := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		lines = append(lines, fmtLine(prefix, i))
	}
	return strings.Join(lines, "\n")
}

func fmtLine(prefix string, index int) string {
	return fmt.Sprintf("%s-%02d", prefix, index)
}

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestRunExecuteSingleTask(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("BLACKBIRD_AGENT_CMD", "cat")

	now := time.Date(2026, 1, 28, 20, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": {
				ID:                 "task",
				Title:              "Task",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "do it",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	loaded, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if errs := plan.Validate(loaded); len(errs) != 0 {
		t.Fatalf("plan invalid after save: %v", errs)
	}

	if _, err := captureStdout(func() error { return runExecute([]string{}) }); err != nil {
		t.Fatalf("runExecute: %v", err)
	}

	updated, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items["task"].Status != plan.StatusDone {
		t.Fatalf("expected task done, got %s", updated.Items["task"].Status)
	}

	runsDir := filepath.Join(tempDir, ".blackbird", "runs", "task")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("read runs dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 run record, got %d", len(entries))
	}
}

func TestRunExecuteFailureContinues(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("BLACKBIRD_AGENT_CMD", "exit 2")

	now := time.Date(2026, 1, 28, 20, 30, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"a": {
				ID:                 "a",
				Title:              "Task A",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "fail",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"b": {
				ID:                 "b",
				Title:              "Task B",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "fail",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	loaded, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if errs := plan.Validate(loaded); len(errs) != 0 {
		t.Fatalf("plan invalid after save: %v", errs)
	}

	if _, err := captureStdout(func() error { return runExecute([]string{}) }); err != nil {
		t.Fatalf("runExecute: %v", err)
	}

	updated, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items["a"].Status != plan.StatusFailed {
		t.Fatalf("expected task a failed, got %s", updated.Items["a"].Status)
	}
	if updated.Items["b"].Status != plan.StatusFailed {
		t.Fatalf("expected task b failed, got %s", updated.Items["b"].Status)
	}

	runsDir := filepath.Join(tempDir, ".blackbird", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("read runs dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 run task dirs, got %d", len(entries))
	}
}

func TestRunExecuteDecisionApproveQuitStops(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("BLACKBIRD_AGENT_CMD", "cat")

	configDir := filepath.Join(tempDir, ".blackbird")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"schemaVersion":1,"execution":{"stopAfterEachTask":true}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	oldTerminal := isTerminal
	isTerminal = func(fd uintptr) bool { return false }
	t.Cleanup(func() { isTerminal = oldTerminal })

	setPromptReader(strings.NewReader("2\n"))
	t.Cleanup(func() { setPromptReader(os.Stdin) })

	now := time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"a": {
				ID:                 "a",
				Title:              "Task A",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "do it",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"b": {
				ID:                 "b",
				Title:              "Task B",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "do it",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	output, err := captureStdout(func() error { return runExecute([]string{}) })
	if err != nil {
		t.Fatalf("runExecute: %v", err)
	}
	if !strings.Contains(output, "Review summary:") {
		t.Fatalf("expected review prompt in output, got %q", output)
	}

	updated, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items["a"].Status != plan.StatusDone {
		t.Fatalf("expected task a done, got %s", updated.Items["a"].Status)
	}
	if updated.Items["b"].Status != plan.StatusTodo {
		t.Fatalf("expected task b todo, got %s", updated.Items["b"].Status)
	}

	latest, err := execution.GetLatestRun(filepath.Dir(plan.PlanPath()), "a")
	if err != nil {
		t.Fatalf("GetLatestRun: %v", err)
	}
	if latest == nil || latest.DecisionState != execution.DecisionStateApprovedQuit {
		t.Fatalf("expected approved quit decision, got %#v", latest)
	}
}

func TestRunExecuteDecisionApproveContinueRunsNext(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("BLACKBIRD_AGENT_CMD", "cat")

	configDir := filepath.Join(tempDir, ".blackbird")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"schemaVersion":1,"execution":{"stopAfterEachTask":true}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	oldTerminal := isTerminal
	isTerminal = func(fd uintptr) bool { return false }
	t.Cleanup(func() { isTerminal = oldTerminal })

	setPromptReader(strings.NewReader("1\n2\n"))
	t.Cleanup(func() { setPromptReader(os.Stdin) })

	now := time.Date(2026, 1, 20, 11, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"a": {
				ID:                 "a",
				Title:              "Task A",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "do it",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			"b": {
				ID:                 "b",
				Title:              "Task B",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "do it",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	if _, err := captureStdout(func() error { return runExecute([]string{}) }); err != nil {
		t.Fatalf("runExecute: %v", err)
	}

	updated, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items["a"].Status != plan.StatusDone {
		t.Fatalf("expected task a done, got %s", updated.Items["a"].Status)
	}
	if updated.Items["b"].Status != plan.StatusDone {
		t.Fatalf("expected task b done, got %s", updated.Items["b"].Status)
	}

	runA, err := execution.GetLatestRun(filepath.Dir(plan.PlanPath()), "a")
	if err != nil {
		t.Fatalf("GetLatestRun a: %v", err)
	}
	if runA == nil || runA.DecisionState != execution.DecisionStateApprovedContinue {
		t.Fatalf("expected approved continue for a, got %#v", runA)
	}

	runB, err := execution.GetLatestRun(filepath.Dir(plan.PlanPath()), "b")
	if err != nil {
		t.Fatalf("GetLatestRun b: %v", err)
	}
	if runB == nil || runB.DecisionState != execution.DecisionStateApprovedQuit {
		t.Fatalf("expected approved quit for b, got %#v", runB)
	}
}

func TestRunExecuteDecisionRequestChangesCancelReturnsToPrompt(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("BLACKBIRD_AGENT_CMD", "cat")

	configDir := filepath.Join(tempDir, ".blackbird")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"schemaVersion":1,"execution":{"stopAfterEachTask":true}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	oldTerminal := isTerminal
	isTerminal = func(fd uintptr) bool { return false }
	t.Cleanup(func() { isTerminal = oldTerminal })

	setPromptReader(strings.NewReader("3\n/cancel\n2\n"))
	t.Cleanup(func() { setPromptReader(os.Stdin) })

	now := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"a": {
				ID:                 "a",
				Title:              "Task A",
				Description:        "",
				AcceptanceCriteria: []string{},
				Prompt:             "do it",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	if _, err := captureStdout(func() error { return runExecute([]string{}) }); err != nil {
		t.Fatalf("runExecute: %v", err)
	}

	runsDir := filepath.Join(tempDir, ".blackbird", "runs", "a")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("read runs dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 run record after cancel, got %d", len(entries))
	}

	latest, err := execution.GetLatestRun(filepath.Dir(plan.PlanPath()), "a")
	if err != nil {
		t.Fatalf("GetLatestRun: %v", err)
	}
	if latest == nil || latest.DecisionState != execution.DecisionStateApprovedQuit {
		t.Fatalf("expected approved quit decision, got %#v", latest)
	}
	if latest.DecisionFeedback != "" {
		t.Fatalf("expected no decision feedback, got %q", latest.DecisionFeedback)
	}
}

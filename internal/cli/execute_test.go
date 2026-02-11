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

func TestRunExecuteParentReviewPassContinuesWithoutPause(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("BLACKBIRD_AGENT_PROVIDER", "codex")
	t.Setenv("BLACKBIRD_AGENT_CMD", `printf '{"passed":true}'`)

	configDir := filepath.Join(tempDir, ".blackbird")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(configDir, "config.json"),
		[]byte(`{"schemaVersion":1,"execution":{"parentReviewEnabled":true}}`),
		0o644,
	); err != nil {
		t.Fatalf("write config: %v", err)
	}

	now := time.Date(2026, 1, 20, 15, 0, 0, 0, time.UTC)
	parentID := "parent"
	childID := "child"
	otherID := "other"

	parent := plan.WorkItem{
		ID:                 parentID,
		Title:              "Parent Review",
		Description:        "",
		AcceptanceCriteria: []string{"Parent acceptance criteria"},
		Prompt:             "Review child output",
		ParentID:           nil,
		ChildIDs:           []string{childID},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	childParentID := parentID
	child := plan.WorkItem{
		ID:                 childID,
		Title:              "Child",
		Description:        "",
		AcceptanceCriteria: []string{"Do child work"},
		Prompt:             "Implement child",
		ParentID:           &childParentID,
		ChildIDs:           []string{},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	other := plan.WorkItem{
		ID:                 otherID,
		Title:              "Other",
		Description:        "",
		AcceptanceCriteria: []string{"Other work"},
		Prompt:             "Implement other",
		ParentID:           nil,
		ChildIDs:           []string{},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID: parent,
			childID:  child,
			otherID:  other,
		},
	}
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	output, err := captureStdout(func() error { return runExecute([]string{}) })
	if err != nil {
		t.Fatalf("runExecute: %v", err)
	}

	wantLines := []string{
		"starting child",
		"completed child",
		"starting other",
		"completed other",
		"no ready tasks remaining",
	}
	for _, line := range wantLines {
		if !strings.Contains(output, line) {
			t.Fatalf("expected output to contain %q, got %q", line, output)
		}
	}
	if strings.Contains(output, "parent review failed") {
		t.Fatalf("expected no parent review pause output, got %q", output)
	}
	if strings.Contains(output, "next step: blackbird resume") {
		t.Fatalf("expected no resume guidance for passing parent review, got %q", output)
	}

	updated, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items[childID].Status != plan.StatusDone {
		t.Fatalf("expected %s done, got %s", childID, updated.Items[childID].Status)
	}
	if updated.Items[otherID].Status != plan.StatusDone {
		t.Fatalf("expected %s done, got %s", otherID, updated.Items[otherID].Status)
	}

	parentRuns, err := execution.ListRuns(tempDir, parentID)
	if err != nil {
		t.Fatalf("ListRuns(%s): %v", parentID, err)
	}
	if len(parentRuns) != 1 {
		t.Fatalf("expected 1 parent review run, got %d", len(parentRuns))
	}
	if parentRuns[0].Type != execution.RunTypeReview {
		t.Fatalf("parent run type = %q, want %q", parentRuns[0].Type, execution.RunTypeReview)
	}
	if parentRuns[0].ParentReviewPassed == nil || !*parentRuns[0].ParentReviewPassed {
		t.Fatalf("parent run passed flag = %#v, want true", parentRuns[0].ParentReviewPassed)
	}

	pending, err := execution.LoadPendingParentReviewFeedback(tempDir, childID)
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(%s): %v", childID, err)
	}
	if pending != nil {
		t.Fatalf("expected no pending feedback for passing review, got %#v", pending)
	}
}

func TestRunExecuteParentReviewFailureSummary(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("BLACKBIRD_AGENT_PROVIDER", "codex")
	t.Setenv(
		"BLACKBIRD_AGENT_CMD",
		`printf '{"passed":false,"resumeTaskIds":[" child-b ","child-a"],"feedbackForResume":"  Child outputs miss required validation paths.  "}'`,
	)

	configDir := filepath.Join(tempDir, ".blackbird")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(configDir, "config.json"),
		[]byte(`{"schemaVersion":1,"execution":{"parentReviewEnabled":true}}`),
		0o644,
	); err != nil {
		t.Fatalf("write config: %v", err)
	}

	now := time.Date(2026, 1, 20, 16, 0, 0, 0, time.UTC)
	parentID := "parent"
	childAID := "child-a"
	childBID := "child-b"
	otherID := "other"

	parent := plan.WorkItem{
		ID:                 parentID,
		Title:              "Parent Review",
		Description:        "",
		AcceptanceCriteria: []string{"Parent acceptance criteria"},
		Prompt:             "Review child output",
		ParentID:           nil,
		ChildIDs:           []string{childAID, childBID},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	childAParentID := parentID
	childA := plan.WorkItem{
		ID:                 childAID,
		Title:              "Child A",
		Description:        "",
		AcceptanceCriteria: []string{"Do A"},
		Prompt:             "Implement A",
		ParentID:           &childAParentID,
		ChildIDs:           []string{},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	childBParentID := parentID
	childB := plan.WorkItem{
		ID:                 childBID,
		Title:              "Child B",
		Description:        "",
		AcceptanceCriteria: []string{"Do B"},
		Prompt:             "Implement B",
		ParentID:           &childBParentID,
		ChildIDs:           []string{},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	other := plan.WorkItem{
		ID:                 otherID,
		Title:              "Other",
		Description:        "",
		AcceptanceCriteria: []string{"Other work"},
		Prompt:             "Implement other",
		ParentID:           nil,
		ChildIDs:           []string{},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID: parent,
			childAID: childA,
			childBID: childB,
			otherID:  other,
		},
	}
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	output, err := captureStdout(func() error { return runExecute([]string{}) })
	if err != nil {
		t.Fatalf("runExecute: %v", err)
	}

	wantLines := []string{
		"starting child-a",
		"completed child-a",
		"starting child-b",
		"completed child-b",
		"running parent review for parent",
		"parent review failed for parent",
		"resume tasks: child-a, child-b",
		"feedback: Child outputs miss required validation paths.",
		"next step: blackbird resume child-a",
		"next step: blackbird resume child-b",
	}
	for _, line := range wantLines {
		if !strings.Contains(output, line) {
			t.Fatalf("expected output to contain %q, got %q", line, output)
		}
	}

	firstResume := strings.Index(output, "next step: blackbird resume child-a")
	secondResume := strings.Index(output, "next step: blackbird resume child-b")
	if firstResume == -1 || secondResume == -1 {
		t.Fatalf("missing resume instructions in output: %q", output)
	}
	if firstResume > secondResume {
		t.Fatalf("resume instructions out of order: %q", output)
	}
	if strings.Contains(output, "starting other") {
		t.Fatalf("expected parent review pause before running %s, got %q", otherID, output)
	}

	updated, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items[childAID].Status != plan.StatusDone {
		t.Fatalf("expected %s done, got %s", childAID, updated.Items[childAID].Status)
	}
	if updated.Items[childBID].Status != plan.StatusDone {
		t.Fatalf("expected %s done, got %s", childBID, updated.Items[childBID].Status)
	}
	if updated.Items[otherID].Status != plan.StatusTodo {
		t.Fatalf("expected %s todo, got %s", otherID, updated.Items[otherID].Status)
	}
}

func TestRunExecuteParentReviewDisabledSkipsGate(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("BLACKBIRD_AGENT_PROVIDER", "codex")
	t.Setenv(
		"BLACKBIRD_AGENT_CMD",
		`printf '{"passed":false,"resumeTaskIds":["child"],"feedbackForResume":"Fix child output before continuing."}'`,
	)

	now := time.Date(2026, 1, 20, 17, 0, 0, 0, time.UTC)
	parentID := "parent"
	childID := "child"
	otherID := "other"

	parent := plan.WorkItem{
		ID:                 parentID,
		Title:              "Parent Review",
		Description:        "",
		AcceptanceCriteria: []string{"Parent acceptance criteria"},
		Prompt:             "Review child output",
		ParentID:           nil,
		ChildIDs:           []string{childID},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	childParentID := parentID
	child := plan.WorkItem{
		ID:                 childID,
		Title:              "Child",
		Description:        "",
		AcceptanceCriteria: []string{"Do child work"},
		Prompt:             "Implement child",
		ParentID:           &childParentID,
		ChildIDs:           []string{},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	other := plan.WorkItem{
		ID:                 otherID,
		Title:              "Other",
		Description:        "",
		AcceptanceCriteria: []string{"Other work"},
		Prompt:             "Implement other",
		ParentID:           nil,
		ChildIDs:           []string{},
		Deps:               []string{},
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID: parent,
			childID:  child,
			otherID:  other,
		},
	}
	if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	output, err := captureStdout(func() error { return runExecute([]string{}) })
	if err != nil {
		t.Fatalf("runExecute: %v", err)
	}

	wantLines := []string{
		"starting child",
		"completed child",
		"starting other",
		"completed other",
		"no ready tasks remaining",
	}
	for _, line := range wantLines {
		if !strings.Contains(output, line) {
			t.Fatalf("expected output to contain %q, got %q", line, output)
		}
	}
	if strings.Contains(output, "running parent review") {
		t.Fatalf("expected no parent review output when disabled, got %q", output)
	}

	updated, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if updated.Items[childID].Status != plan.StatusDone {
		t.Fatalf("expected %s done, got %s", childID, updated.Items[childID].Status)
	}
	if updated.Items[otherID].Status != plan.StatusDone {
		t.Fatalf("expected %s done, got %s", otherID, updated.Items[otherID].Status)
	}

	parentRuns, err := execution.ListRuns(tempDir, parentID)
	if err != nil {
		t.Fatalf("ListRuns(%s): %v", parentID, err)
	}
	if len(parentRuns) != 0 {
		t.Fatalf("expected 0 parent review runs when disabled, got %d", len(parentRuns))
	}
}

func TestRunExecuteParentReviewResolvedConfigPrecedence(t *testing.T) {
	tests := []struct {
		name             string
		globalConfigJSON string
		localConfigJSON  string
		expectReviewRun  bool
	}{
		{
			name:             "local false overrides global true and skips parent review",
			globalConfigJSON: `{"schemaVersion":1,"execution":{"parentReviewEnabled":true}}`,
			localConfigJSON:  `{"schemaVersion":1,"execution":{"parentReviewEnabled":false}}`,
			expectReviewRun:  false,
		},
		{
			name:             "local true overrides global false and runs parent review",
			globalConfigJSON: `{"schemaVersion":1,"execution":{"parentReviewEnabled":false}}`,
			localConfigJSON:  `{"schemaVersion":1,"execution":{"parentReviewEnabled":true}}`,
			expectReviewRun:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			oldWD, err := os.Getwd()
			if err != nil {
				t.Fatalf("getwd: %v", err)
			}
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("chdir temp: %v", err)
			}
			t.Cleanup(func() { _ = os.Chdir(oldWD) })

			homeDir := t.TempDir()
			t.Setenv("HOME", homeDir)
			t.Setenv("BLACKBIRD_AGENT_PROVIDER", "codex")
			t.Setenv(
				"BLACKBIRD_AGENT_CMD",
				`printf '{"passed":false,"resumeTaskIds":["child"],"feedbackForResume":"Fix child output before continuing."}'`,
			)

			globalConfigPath := filepath.Join(homeDir, ".blackbird", "config.json")
			if err := os.MkdirAll(filepath.Dir(globalConfigPath), 0o755); err != nil {
				t.Fatalf("mkdir global config dir: %v", err)
			}
			if err := os.WriteFile(globalConfigPath, []byte(tt.globalConfigJSON), 0o644); err != nil {
				t.Fatalf("write global config: %v", err)
			}

			localConfigPath := filepath.Join(tempDir, ".blackbird", "config.json")
			if err := os.MkdirAll(filepath.Dir(localConfigPath), 0o755); err != nil {
				t.Fatalf("mkdir local config dir: %v", err)
			}
			if err := os.WriteFile(localConfigPath, []byte(tt.localConfigJSON), 0o644); err != nil {
				t.Fatalf("write local config: %v", err)
			}

			now := time.Date(2026, 1, 20, 18, 0, 0, 0, time.UTC)
			parentID := "parent"
			childID := "child"
			otherID := "other"

			parent := plan.WorkItem{
				ID:                 parentID,
				Title:              "Parent Review",
				Description:        "",
				AcceptanceCriteria: []string{"Parent acceptance criteria"},
				Prompt:             "Review child output",
				ParentID:           nil,
				ChildIDs:           []string{childID},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			}
			childParentID := parentID
			child := plan.WorkItem{
				ID:                 childID,
				Title:              "Child",
				Description:        "",
				AcceptanceCriteria: []string{"Do child work"},
				Prompt:             "Implement child",
				ParentID:           &childParentID,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			}
			other := plan.WorkItem{
				ID:                 otherID,
				Title:              "Other",
				Description:        "",
				AcceptanceCriteria: []string{"Other work"},
				Prompt:             "Implement other",
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			}

			g := plan.WorkGraph{
				SchemaVersion: plan.SchemaVersion,
				Items: map[string]plan.WorkItem{
					parentID: parent,
					childID:  child,
					otherID:  other,
				},
			}
			if errs := plan.Validate(g); len(errs) != 0 {
				t.Fatalf("invalid fixture: %v", errs)
			}
			if err := plan.SaveAtomic(plan.PlanPath(), g); err != nil {
				t.Fatalf("save plan: %v", err)
			}

			output, err := captureStdout(func() error { return runExecute([]string{}) })
			if err != nil {
				t.Fatalf("runExecute: %v", err)
			}

			updated, err := plan.Load(plan.PlanPath())
			if err != nil {
				t.Fatalf("load plan: %v", err)
			}
			if updated.Items[childID].Status != plan.StatusDone {
				t.Fatalf("expected %s done, got %s", childID, updated.Items[childID].Status)
			}

			parentRuns, err := execution.ListRuns(tempDir, parentID)
			if err != nil {
				t.Fatalf("ListRuns(%s): %v", parentID, err)
			}

			if tt.expectReviewRun {
				if !strings.Contains(output, "running parent review for "+parentID) {
					t.Fatalf("expected parent review output, got %q", output)
				}
				if updated.Items[otherID].Status != plan.StatusTodo {
					t.Fatalf("expected %s todo after parent-review pause, got %s", otherID, updated.Items[otherID].Status)
				}
				if len(parentRuns) != 1 {
					t.Fatalf("expected 1 parent review run, got %d", len(parentRuns))
				}
				return
			}

			if strings.Contains(output, "running parent review") {
				t.Fatalf("expected no parent review output when resolved false, got %q", output)
			}
			if updated.Items[otherID].Status != plan.StatusDone {
				t.Fatalf("expected %s done when parent review disabled, got %s", otherID, updated.Items[otherID].Status)
			}
			if len(parentRuns) != 0 {
				t.Fatalf("expected 0 parent review runs, got %d", len(parentRuns))
			}
		})
	}
}

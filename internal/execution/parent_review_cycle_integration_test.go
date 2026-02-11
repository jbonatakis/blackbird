package execution

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestParentReviewCycleIntegrationExecuteReviewFailurePersistsTargetedFeedback(t *testing.T) {
	fixture := newParentReviewCycleFixture(t)

	result, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:            fixture.planPath,
		Runtime:             fixture.runtime,
		ParentReviewEnabled: true,
	})
	if err != nil {
		t.Fatalf(
			"RunExecute: %v (plan validation errors: %v)",
			err,
			parentReviewCycleValidatePlanFile(t, fixture.planPath),
		)
	}
	if result.Reason != ExecuteReasonParentReviewRequired {
		t.Fatalf("result.Reason = %q, want %q", result.Reason, ExecuteReasonParentReviewRequired)
	}
	if result.TaskID != fixture.parentID {
		t.Fatalf("result.TaskID = %q, want %q", result.TaskID, fixture.parentID)
	}
	if result.Run == nil {
		t.Fatalf("expected parent review run in execute result")
	}
	if result.Run.Type != RunTypeReview {
		t.Fatalf("result.Run.Type = %q, want %q", result.Run.Type, RunTypeReview)
	}
	if result.Run.ParentReviewPassed == nil || *result.Run.ParentReviewPassed {
		t.Fatalf("result.Run.ParentReviewPassed = %#v, want false", result.Run.ParentReviewPassed)
	}
	if len(result.Run.ParentReviewResumeTaskIDs) != 1 || result.Run.ParentReviewResumeTaskIDs[0] != fixture.childBID {
		t.Fatalf("result.Run.ParentReviewResumeTaskIDs = %#v, want [%s]", result.Run.ParentReviewResumeTaskIDs, fixture.childBID)
	}
	if result.Run.ParentReviewFeedback != fixture.reviewFeedback {
		t.Fatalf("result.Run.ParentReviewFeedback = %q, want %q", result.Run.ParentReviewFeedback, fixture.reviewFeedback)
	}
	if strings.TrimSpace(result.Run.ParentReviewCompletionSignature) == "" {
		t.Fatalf("expected non-empty completion signature")
	}

	if got := parentReviewCycleReadReviewCount(t, fixture.reviewCountPath); got != 1 {
		t.Fatalf("review invocation count = %d, want 1", got)
	}

	reviewRuns, err := ListReviewRuns(fixture.baseDir, fixture.parentID)
	if err != nil {
		t.Fatalf("ListReviewRuns: %v", err)
	}
	if len(reviewRuns) != 1 {
		t.Fatalf("review run count = %d, want 1", len(reviewRuns))
	}
	firstSignature := reviewRuns[0].ParentReviewCompletionSignature
	if strings.TrimSpace(firstSignature) == "" {
		t.Fatalf("expected persisted parent review signature")
	}

	shouldRun, err := ShouldRunParentReviewForSignature(fixture.baseDir, fixture.parentID, firstSignature)
	if err != nil {
		t.Fatalf("ShouldRunParentReviewForSignature: %v", err)
	}
	if shouldRun {
		t.Fatalf("expected idempotence check to skip rerun for signature %q", firstSignature)
	}

	_, pauseRun, err := runParentReviewGateForCompletedTask(context.Background(), ExecuteConfig{
		PlanPath:            fixture.planPath,
		Runtime:             fixture.runtime,
		ParentReviewEnabled: true,
	}, fixture.childBID)
	if err != nil {
		t.Fatalf("runParentReviewGateForCompletedTask: %v", err)
	}
	if pauseRun != nil {
		t.Fatalf("unexpected pause run on idempotence re-check: %#v", pauseRun)
	}
	if got := parentReviewCycleReadReviewCount(t, fixture.reviewCountPath); got != 1 {
		t.Fatalf("review invocation count after idempotence re-check = %d, want 1", got)
	}

	reviewRunsAfter, err := ListReviewRuns(fixture.baseDir, fixture.parentID)
	if err != nil {
		t.Fatalf("ListReviewRuns after idempotence re-check: %v", err)
	}
	if len(reviewRunsAfter) != 1 {
		t.Fatalf("review run count after idempotence re-check = %d, want 1", len(reviewRunsAfter))
	}

	pendingA, err := LoadPendingParentReviewFeedback(fixture.baseDir, fixture.childAID)
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(%s): %v", fixture.childAID, err)
	}
	if pendingA != nil {
		t.Fatalf("%s should not have pending feedback, got %#v", fixture.childAID, pendingA)
	}

	pendingB, err := LoadPendingParentReviewFeedback(fixture.baseDir, fixture.childBID)
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(%s): %v", fixture.childBID, err)
	}
	if pendingB == nil {
		t.Fatalf("expected pending feedback for %s", fixture.childBID)
	}
	if pendingB.ParentTaskID != fixture.parentID {
		t.Fatalf("pending parent task id = %q, want %q", pendingB.ParentTaskID, fixture.parentID)
	}
	if pendingB.ReviewRunID != result.Run.ID {
		t.Fatalf("pending review run id = %q, want %q", pendingB.ReviewRunID, result.Run.ID)
	}
	if pendingB.Feedback != fixture.reviewFeedback {
		t.Fatalf("pending feedback = %q, want %q", pendingB.Feedback, fixture.reviewFeedback)
	}

	updatedPlan, err := plan.Load(fixture.planPath)
	if err != nil {
		t.Fatalf("plan.Load: %v", err)
	}
	if updatedPlan.Items[fixture.childAID].Status != plan.StatusDone {
		t.Fatalf("%s status = %q, want %q", fixture.childAID, updatedPlan.Items[fixture.childAID].Status, plan.StatusDone)
	}
	if updatedPlan.Items[fixture.childBID].Status != plan.StatusDone {
		t.Fatalf("%s status = %q, want %q", fixture.childBID, updatedPlan.Items[fixture.childBID].Status, plan.StatusDone)
	}
}

func TestParentReviewCycleIntegrationResumeFeedbackAndRerunWithNewSignature(t *testing.T) {
	fixture := newParentReviewCycleFixture(t)

	executeResult, err := RunExecute(context.Background(), ExecuteConfig{
		PlanPath:            fixture.planPath,
		Runtime:             fixture.runtime,
		ParentReviewEnabled: true,
	})
	if err != nil {
		t.Fatalf(
			"RunExecute: %v (plan validation errors: %v)",
			err,
			parentReviewCycleValidatePlanFile(t, fixture.planPath),
		)
	}
	if executeResult.Reason != ExecuteReasonParentReviewRequired {
		t.Fatalf("executeResult.Reason = %q, want %q", executeResult.Reason, ExecuteReasonParentReviewRequired)
	}
	if executeResult.Run == nil {
		t.Fatalf("expected parent review run from execute result")
	}
	firstReviewRunID := executeResult.Run.ID
	firstSignature := executeResult.Run.ParentReviewCompletionSignature
	if strings.TrimSpace(firstSignature) == "" {
		t.Fatalf("expected initial non-empty signature")
	}

	resumeRecord, err := RunResume(context.Background(), ResumeConfig{
		PlanPath: fixture.planPath,
		TaskID:   fixture.childBID,
		Runtime:  fixture.runtime,
	})
	if err != nil {
		t.Fatalf("RunResume: %v", err)
	}
	if resumeRecord.Status != RunStatusSuccess {
		t.Fatalf("resumeRecord.Status = %q, want %q", resumeRecord.Status, RunStatusSuccess)
	}
	if resumeRecord.Context.ParentReviewFeedback == nil {
		t.Fatalf("expected parent review feedback context in resumed run")
	}
	wantFeedback := ParentReviewFeedbackContext{
		ParentTaskID: fixture.parentID,
		ReviewRunID:  firstReviewRunID,
		Feedback:     fixture.reviewFeedback,
	}
	if *resumeRecord.Context.ParentReviewFeedback != wantFeedback {
		t.Fatalf(
			"resume parent review feedback = %#v, want %#v",
			*resumeRecord.Context.ParentReviewFeedback,
			wantFeedback,
		)
	}

	consumedFeedback := strings.TrimSpace(parentReviewCycleReadFile(t, fixture.resumeFeedbackPath))
	if consumedFeedback != fixture.reviewFeedback {
		t.Fatalf("consumed resume feedback = %q, want %q", consumedFeedback, fixture.reviewFeedback)
	}

	pendingAfterResume, err := LoadPendingParentReviewFeedback(fixture.baseDir, fixture.childBID)
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(%s): %v", fixture.childBID, err)
	}
	if pendingAfterResume != nil {
		t.Fatalf("expected pending feedback to be cleared after resume, got %#v", pendingAfterResume)
	}

	graph, err := plan.Load(fixture.planPath)
	if err != nil {
		t.Fatalf("plan.Load: %v", err)
	}
	child := graph.Items[fixture.childBID]
	child.UpdatedAt = fixture.baseTime.Add(6 * time.Hour)
	graph.Items[fixture.childBID] = child
	if err := plan.SaveAtomic(fixture.planPath, graph); err != nil {
		t.Fatalf("plan.SaveAtomic: %v", err)
	}

	_, pauseRun, err := runParentReviewGateForCompletedTask(context.Background(), ExecuteConfig{
		PlanPath:            fixture.planPath,
		Runtime:             fixture.runtime,
		ParentReviewEnabled: true,
	}, fixture.childBID)
	if err != nil {
		t.Fatalf("runParentReviewGateForCompletedTask: %v", err)
	}
	if pauseRun != nil {
		t.Fatalf("unexpected pause run for passing second review: %#v", pauseRun)
	}
	if got := parentReviewCycleReadReviewCount(t, fixture.reviewCountPath); got != 2 {
		t.Fatalf("review invocation count after re-review = %d, want 2", got)
	}

	reviewRuns, err := ListReviewRuns(fixture.baseDir, fixture.parentID)
	if err != nil {
		t.Fatalf("ListReviewRuns: %v", err)
	}
	if len(reviewRuns) != 2 {
		t.Fatalf("review run count after re-review = %d, want 2", len(reviewRuns))
	}

	var latestReview *RunRecord
	for i := range reviewRuns {
		run := reviewRuns[i]
		if run.ID == firstReviewRunID {
			continue
		}
		candidate := run
		latestReview = &candidate
	}
	if latestReview == nil {
		t.Fatalf("expected second review run distinct from %q", firstReviewRunID)
	}
	if latestReview.ParentReviewPassed == nil || !*latestReview.ParentReviewPassed {
		t.Fatalf("latestReview.ParentReviewPassed = %#v, want true", latestReview.ParentReviewPassed)
	}
	if strings.TrimSpace(latestReview.ParentReviewCompletionSignature) == "" {
		t.Fatalf("expected non-empty second review signature")
	}
	if latestReview.ParentReviewCompletionSignature == firstSignature {
		t.Fatalf("expected second signature to differ from first signature %q", firstSignature)
	}

	_, pauseRun, err = runParentReviewGateForCompletedTask(context.Background(), ExecuteConfig{
		PlanPath:            fixture.planPath,
		Runtime:             fixture.runtime,
		ParentReviewEnabled: true,
	}, fixture.childBID)
	if err != nil {
		t.Fatalf("runParentReviewGateForCompletedTask second idempotence check: %v", err)
	}
	if pauseRun != nil {
		t.Fatalf("unexpected pause run on second idempotence check: %#v", pauseRun)
	}
	if got := parentReviewCycleReadReviewCount(t, fixture.reviewCountPath); got != 2 {
		t.Fatalf("review invocation count after second idempotence check = %d, want 2", got)
	}

	finalReviewRuns, err := ListReviewRuns(fixture.baseDir, fixture.parentID)
	if err != nil {
		t.Fatalf("ListReviewRuns final: %v", err)
	}
	if len(finalReviewRuns) != 2 {
		t.Fatalf("final review run count = %d, want 2", len(finalReviewRuns))
	}

	shouldRun, err := ShouldRunParentReviewForSignature(
		fixture.baseDir,
		fixture.parentID,
		latestReview.ParentReviewCompletionSignature,
	)
	if err != nil {
		t.Fatalf("ShouldRunParentReviewForSignature latest signature: %v", err)
	}
	if shouldRun {
		t.Fatalf("expected latest signature idempotence check to skip rerun")
	}
}

type parentReviewCycleFixture struct {
	baseDir            string
	planPath           string
	runtime            agent.Runtime
	baseTime           time.Time
	parentID           string
	childAID           string
	childBID           string
	reviewFeedback     string
	reviewCountPath    string
	resumeFeedbackPath string
}

func newParentReviewCycleFixture(t *testing.T) parentReviewCycleFixture {
	t.Helper()

	baseDir := t.TempDir()
	planPath := filepath.Join(baseDir, plan.DefaultPlanFilename)
	baseTime := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

	parentID := "parent"
	childAID := "child-a"
	childBID := "child-b"
	reviewFeedback := "Address parent review feedback."

	graph := parentReviewCycleGraph(baseTime, parentID, childAID, childBID)
	if errs := plan.Validate(graph); len(errs) != 0 {
		t.Fatalf("invalid graph fixture: %v", errs)
	}
	if err := plan.SaveAtomic(planPath, graph); err != nil {
		t.Fatalf("plan.SaveAtomic: %v", err)
	}

	scriptPath, reviewCountPath, resumeFeedbackPath := writeParentReviewCycleScriptFixture(
		t,
		baseDir,
		childAID,
		childBID,
		reviewFeedback,
	)

	return parentReviewCycleFixture{
		baseDir:            baseDir,
		planPath:           planPath,
		runtime:            agent.Runtime{Provider: "codex", Command: scriptPath, Timeout: 2 * time.Second},
		baseTime:           baseTime,
		parentID:           parentID,
		childAID:           childAID,
		childBID:           childBID,
		reviewFeedback:     reviewFeedback,
		reviewCountPath:    reviewCountPath,
		resumeFeedbackPath: resumeFeedbackPath,
	}
}

func parentReviewCycleGraph(baseTime time.Time, parentID, childAID, childBID string) plan.WorkGraph {
	childAParent := parentID
	childBParent := parentID

	return plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			parentID: {
				ID:                 parentID,
				Title:              "Parent Review",
				Description:        "Parent task that reviews child outcomes.",
				AcceptanceCriteria: []string{"Child outputs satisfy parent acceptance criteria."},
				Prompt:             "Review child outputs and decide if resume is required.",
				ParentID:           nil,
				ChildIDs:           []string{childAID, childBID},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          baseTime,
				UpdatedAt:          baseTime,
			},
			childAID: {
				ID:                 childAID,
				Title:              "Child A",
				Description:        "",
				AcceptanceCriteria: []string{"Implement child A."},
				Prompt:             "Implement child A.",
				ParentID:           &childAParent,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          baseTime,
				UpdatedAt:          baseTime,
			},
			childBID: {
				ID:                 childBID,
				Title:              "Child B",
				Description:        "",
				AcceptanceCriteria: []string{"Implement child B."},
				Prompt:             "Implement child B.",
				ParentID:           &childBParent,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          baseTime,
				UpdatedAt:          baseTime,
			},
		},
	}
}

func writeParentReviewCycleScriptFixture(
	t *testing.T,
	baseDir string,
	childAID string,
	childBID string,
	reviewFeedback string,
) (string, string, string) {
	t.Helper()

	scriptPath := filepath.Join(baseDir, "agent-fixture.sh")
	reviewCountPath := filepath.Join(baseDir, "review-count.txt")
	resumeFeedbackPath := filepath.Join(baseDir, "resume-feedback.txt")
	argsLogPath := filepath.Join(baseDir, "args.log")

	script := fmt.Sprintf(`#!/bin/sh
set -eu

review_count_path=%q
resume_feedback_path=%q
args_log_path=%q

printf '%%s\n' "$*" >> "$args_log_path"

if [ "${3:-}" = "resume" ]; then
	cat - > "$resume_feedback_path"
	printf 'resume ok\n'
	exit 0
fi

payload="$(cat)"

if printf '%%s' "$payload" | grep -q '"parentReview":'; then
	review_count=0
	if [ -f "$review_count_path" ]; then
		review_count="$(cat "$review_count_path")"
	fi
	review_count=$((review_count + 1))
	printf '%%s' "$review_count" > "$review_count_path"

	if [ "$review_count" -eq 1 ]; then
		printf '{"passed":false,"resumeTaskIds":["%s"],"feedbackForResume":"%s"}\n'
		exit 0
	fi

	printf '{"passed":true}\n'
	exit 0
fi

if printf '%%s' "$payload" | grep -q '"task":{"id":"%s"'; then
	printf '%s completed\n'
	exit 0
fi

if printf '%%s' "$payload" | grep -q '"task":{"id":"%s"'; then
	printf '%s completed\n'
	exit 0
fi

printf 'unexpected payload\n' >&2
exit 71
`,
		reviewCountPath,
		resumeFeedbackPath,
		argsLogPath,
		childBID,
		reviewFeedback,
		childAID,
		childAID,
		childBID,
		childBID,
	)

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script fixture: %v", err)
	}

	return scriptPath, reviewCountPath, resumeFeedbackPath
}

func parentReviewCycleReadReviewCount(t *testing.T, path string) int {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatalf("read review count: %v", err)
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parse review count: %v", err)
	}
	return count
}

func parentReviewCycleReadFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	return string(data)
}

func parentReviewCycleValidatePlanFile(t *testing.T, planPath string) []plan.ValidationError {
	t.Helper()

	graph, err := plan.Load(planPath)
	if err != nil {
		t.Fatalf("plan.Load(%s): %v", planPath, err)
	}
	return plan.Validate(graph)
}

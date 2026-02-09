package execution

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestParentReviewContextIncludesParentTaskAndReviewerInstructions(t *testing.T) {
	baseDir := t.TempDir()
	now := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	parentID := "parent-checkout"

	g := parentReviewContextTestGraph(
		now,
		parentID,
		"Parent Checkout Review",
		[]string{
			"Checkout flow supports guest and signed-in users.",
			"Order totals are accurate across taxes and discounts.",
			"Payment failure states are surfaced with clear recovery guidance.",
		},
		[]string{"child-a", "child-b"},
	)

	fixtures := []RunRecord{
		parentReviewContextTestRun("child-a", "run-a-1", now.Add(1*time.Minute), "child-a summary", []string{"checkout/form.go"}),
		parentReviewContextTestRun("child-b", "run-b-1", now.Add(2*time.Minute), "child-b summary", []string{"checkout/payment.go"}),
	}
	for _, fixture := range fixtures {
		if err := SaveRun(baseDir, fixture); err != nil {
			t.Fatalf("SaveRun(%s): %v", fixture.ID, err)
		}
	}

	pack, err := BuildParentReviewContext(g, baseDir, parentID, ParentReviewContextOptions{})
	if err != nil {
		t.Fatalf("BuildParentReviewContext: %v", err)
	}
	if pack.Task.ID != "parent-checkout" {
		t.Fatalf("pack.Task.ID = %q, want %q", pack.Task.ID, "parent-checkout")
	}
	if pack.Task.Title != "Parent Checkout Review" {
		t.Fatalf("pack.Task.Title = %q, want %q", pack.Task.Title, "Parent Checkout Review")
	}
	wantCriteria := []string{
		"Checkout flow supports guest and signed-in users.",
		"Order totals are accurate across taxes and discounts.",
		"Payment failure states are surfaced with clear recovery guidance.",
	}
	if !reflect.DeepEqual(pack.Task.AcceptanceCriteria, wantCriteria) {
		t.Fatalf("pack.Task.AcceptanceCriteria = %#v, want %#v", pack.Task.AcceptanceCriteria, wantCriteria)
	}
	if pack.ParentReview == nil {
		t.Fatalf("expected parent review payload")
	}
	if pack.ParentReview.ParentTaskID != "parent-checkout" {
		t.Fatalf("pack.ParentReview.ParentTaskID = %q, want %q", pack.ParentReview.ParentTaskID, "parent-checkout")
	}
	if pack.ParentReview.ParentTaskTitle != "Parent Checkout Review" {
		t.Fatalf("pack.ParentReview.ParentTaskTitle = %q, want %q", pack.ParentReview.ParentTaskTitle, "Parent Checkout Review")
	}
	if !reflect.DeepEqual(pack.ParentReview.AcceptanceCriteria, wantCriteria) {
		t.Fatalf("pack.ParentReview.AcceptanceCriteria = %#v, want %#v", pack.ParentReview.AcceptanceCriteria, wantCriteria)
	}
	const expectedReviewerInstructions = "Act as a reviewer only. Assess the parent acceptance criteria against child outputs, flag major correctness or security issues, and map failures to child task IDs with actionable feedback."
	if pack.ParentReview.ReviewerInstructions != expectedReviewerInstructions {
		t.Fatalf("pack.ParentReview.ReviewerInstructions = %q, want %q", pack.ParentReview.ReviewerInstructions, expectedReviewerInstructions)
	}
	const expectedSystemPrompt = "You are running a parent-task review. Evaluate whether child-task outputs satisfy the parent acceptance criteria. Do not implement code changes."
	if pack.SystemPrompt != expectedSystemPrompt {
		t.Fatalf("pack.SystemPrompt = %q, want %q", pack.SystemPrompt, expectedSystemPrompt)
	}
}

func TestParentReviewContextOrdersChildEntriesByIDWithLatestRunData(t *testing.T) {
	baseDir := t.TempDir()
	now := time.Date(2026, 2, 9, 13, 0, 0, 0, time.UTC)
	parentID := "parent-unsorted"

	g := parentReviewContextTestGraph(
		now,
		parentID,
		"Parent Ordering Review",
		[]string{"All child work is integrated consistently."},
		[]string{"child-c", "child-a", "child-b"},
	)

	fixtures := []RunRecord{
		parentReviewContextTestRun("child-a", "run-a-1", now.Add(1*time.Minute), "summary-a-old", []string{"a-old.go"}),
		parentReviewContextTestRun("child-a", "run-a-2", now.Add(4*time.Minute), "summary-a-latest", []string{"a-2.go", "a-1.go", "a-2.go"}),
		parentReviewContextTestRun("child-b", "run-b-1", now.Add(3*time.Minute), "summary-b-latest", []string{"b-1.go"}),
		parentReviewContextTestRun("child-c", "run-c-1", now.Add(2*time.Minute), "summary-c-latest", []string{"c-1.go"}),
	}
	for _, fixture := range fixtures {
		if err := SaveRun(baseDir, fixture); err != nil {
			t.Fatalf("SaveRun(%s): %v", fixture.ID, err)
		}
	}

	pack, err := BuildParentReviewContext(g, baseDir, parentID, ParentReviewContextOptions{})
	if err != nil {
		t.Fatalf("BuildParentReviewContext: %v", err)
	}
	if pack.ParentReview == nil {
		t.Fatalf("expected parent review payload")
	}
	children := pack.ParentReview.Children
	if len(children) != 3 {
		t.Fatalf("expected 3 child context entries, got %d", len(children))
	}

	if children[0].ChildID != "child-a" || children[1].ChildID != "child-b" || children[2].ChildID != "child-c" {
		t.Fatalf("child order = %#v, want child-a, child-b, child-c", children)
	}

	if children[0].LatestRunID != "run-a-2" {
		t.Fatalf("child-a LatestRunID = %q, want %q", children[0].LatestRunID, "run-a-2")
	}
	if children[0].LatestRunSummary != "summary-a-latest" {
		t.Fatalf("child-a summary = %q, want %q", children[0].LatestRunSummary, "summary-a-latest")
	}
	wantARefs := []string{
		filepath.ToSlash(filepath.Join(runsDirName, "child-a", "run-a-2.json")),
		"a-1.go",
		"a-2.go",
	}
	if !reflect.DeepEqual(children[0].ArtifactRefs, wantARefs) {
		t.Fatalf("child-a ArtifactRefs = %#v, want %#v", children[0].ArtifactRefs, wantARefs)
	}

	if children[1].LatestRunID != "run-b-1" || children[2].LatestRunID != "run-c-1" {
		t.Fatalf("unexpected latest run ids for child-b/child-c: %#v", children)
	}
	if children[1].LatestRunSummary != "summary-b-latest" || children[2].LatestRunSummary != "summary-c-latest" {
		t.Fatalf("unexpected latest summaries for child-b/child-c: %#v", children)
	}
	wantBRefs := []string{
		filepath.ToSlash(filepath.Join(runsDirName, "child-b", "run-b-1.json")),
		"b-1.go",
	}
	wantCRefs := []string{
		filepath.ToSlash(filepath.Join(runsDirName, "child-c", "run-c-1.json")),
		"c-1.go",
	}
	if !reflect.DeepEqual(children[1].ArtifactRefs, wantBRefs) {
		t.Fatalf("child-b ArtifactRefs = %#v, want %#v", children[1].ArtifactRefs, wantBRefs)
	}
	if !reflect.DeepEqual(children[2].ArtifactRefs, wantCRefs) {
		t.Fatalf("child-c ArtifactRefs = %#v, want %#v", children[2].ArtifactRefs, wantCRefs)
	}
}

func TestParentReviewContextBoundsChildSummariesDeterministically(t *testing.T) {
	baseDir := t.TempDir()
	now := time.Date(2026, 2, 9, 14, 0, 0, 0, time.UTC)
	parentID := "parent-bounds"

	g := parentReviewContextTestGraph(
		now,
		parentID,
		"Parent Bounds Review",
		[]string{"Child summaries stay bounded."},
		[]string{"child-a"},
	)

	longSummary := strings.Repeat("1234567890", 25)
	if err := SaveRun(baseDir, parentReviewContextTestRun("child-a", "run-a-1", now.Add(1*time.Minute), longSummary, []string{"a.go"})); err != nil {
		t.Fatalf("SaveRun(run-a-1): %v", err)
	}

	limit := 37
	first, err := BuildParentReviewContext(g, baseDir, parentID, ParentReviewContextOptions{MaxChildSummaryBytes: limit})
	if err != nil {
		t.Fatalf("first BuildParentReviewContext: %v", err)
	}
	second, err := BuildParentReviewContext(g, baseDir, parentID, ParentReviewContextOptions{MaxChildSummaryBytes: limit})
	if err != nil {
		t.Fatalf("second BuildParentReviewContext: %v", err)
	}
	if first.ParentReview == nil || second.ParentReview == nil {
		t.Fatalf("expected parent review payload for both calls")
	}
	if len(first.ParentReview.Children) != 1 || len(second.ParentReview.Children) != 1 {
		t.Fatalf("expected one child context entry")
	}

	summaryOne := first.ParentReview.Children[0].LatestRunSummary
	summaryTwo := second.ParentReview.Children[0].LatestRunSummary
	if len(summaryOne) > limit {
		t.Fatalf("summary length = %d, want <= %d", len(summaryOne), limit)
	}
	want := longSummary[:limit]
	if summaryOne != want {
		t.Fatalf("summaryOne = %q, want %q", summaryOne, want)
	}
	if summaryTwo != want {
		t.Fatalf("summaryTwo = %q, want %q", summaryTwo, want)
	}
}

func TestParentReviewContextMissingOptionalSummaryUsesFallback(t *testing.T) {
	baseDir := t.TempDir()
	now := time.Date(2026, 2, 9, 15, 0, 0, 0, time.UTC)
	parentID := "parent-missing-summary"

	g := parentReviewContextTestGraph(
		now,
		parentID,
		"Parent Missing Summary Review",
		[]string{"Missing child summaries should not fail payload generation."},
		[]string{"child-a"},
	)

	if err := SaveRun(baseDir, parentReviewContextTestRun("child-a", "run-a-1", now.Add(1*time.Minute), "", nil)); err != nil {
		t.Fatalf("SaveRun(run-a-1): %v", err)
	}

	pack, err := BuildParentReviewContext(g, baseDir, parentID, ParentReviewContextOptions{MaxChildSummaryBytes: 20})
	if err != nil {
		t.Fatalf("BuildParentReviewContext: %v", err)
	}
	if pack.ParentReview == nil {
		t.Fatalf("expected parent review payload")
	}
	if len(pack.ParentReview.Children) != 1 {
		t.Fatalf("expected 1 child context entry, got %d", len(pack.ParentReview.Children))
	}

	child := pack.ParentReview.Children[0]
	if child.ChildID != "child-a" {
		t.Fatalf("child.ChildID = %q, want %q", child.ChildID, "child-a")
	}
	if child.LatestRunID != "run-a-1" {
		t.Fatalf("child.LatestRunID = %q, want %q", child.LatestRunID, "run-a-1")
	}
	if child.LatestRunSummary != "" {
		t.Fatalf("child.LatestRunSummary = %q, want empty summary fallback", child.LatestRunSummary)
	}
	wantRefs := []string{filepath.ToSlash(filepath.Join(runsDirName, "child-a", "run-a-1.json"))}
	if !reflect.DeepEqual(child.ArtifactRefs, wantRefs) {
		t.Fatalf("child.ArtifactRefs = %#v, want %#v", child.ArtifactRefs, wantRefs)
	}
}

func parentReviewContextTestGraph(
	now time.Time,
	parentID string,
	parentTitle string,
	parentAcceptance []string,
	childIDs []string,
) plan.WorkGraph {
	items := make(map[string]plan.WorkItem, len(childIDs)+1)
	items[parentID] = plan.WorkItem{
		ID:                 parentID,
		Title:              parentTitle,
		Description:        "Parent review task.",
		AcceptanceCriteria: append([]string{}, parentAcceptance...),
		Prompt:             "Review completed child work.",
		ChildIDs:           append([]string{}, childIDs...),
		Status:             plan.StatusTodo,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	for _, childID := range childIDs {
		parentRef := parentID
		items[childID] = plan.WorkItem{
			ID:        childID,
			Title:     "Child " + childID,
			Prompt:    "Implement child task.",
			ParentID:  &parentRef,
			Status:    plan.StatusDone,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	return plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items:         items,
	}
}

func parentReviewContextTestRun(taskID, runID string, startedAt time.Time, diffStat string, files []string) RunRecord {
	completedAt := startedAt.Add(2 * time.Minute)
	var summary *ReviewSummary
	if strings.TrimSpace(diffStat) != "" || len(files) > 0 {
		summary = &ReviewSummary{
			Files:    append([]string{}, files...),
			DiffStat: diffStat,
		}
	}

	return RunRecord{
		ID:          runID,
		TaskID:      taskID,
		Type:        RunTypeExecute,
		StartedAt:   startedAt,
		CompletedAt: &completedAt,
		Status:      RunStatusSuccess,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task: TaskContext{
				ID:    taskID,
				Title: "Task " + taskID,
			},
		},
		ReviewSummary: summary,
	}
}

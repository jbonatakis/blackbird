package execution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestBuildContextIncludesTaskAndDeps(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	snapshotPath := filepath.Join(tempDir, ".blackbird", "snapshot.md")
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0o755); err != nil {
		t.Fatalf("mkdir snapshot: %v", err)
	}
	if err := os.WriteFile(snapshotPath, []byte("snapshot"), 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	now := time.Date(2026, 1, 28, 18, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"dep": {
				ID:        "dep",
				Title:     "Dependency",
				Status:    plan.StatusDone,
				CreatedAt: now,
				UpdatedAt: now,
			},
			"task": {
				ID:                 "task",
				Title:              "Task",
				Description:        "desc",
				AcceptanceCriteria: []string{"a", "b"},
				Prompt:             "do it",
				Deps:               []string{"dep"},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}

	ctx, err := BuildContext(g, "task")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if ctx.Task.ID != "task" || ctx.Task.Title != "Task" || ctx.Task.Prompt != "do it" {
		t.Fatalf("unexpected task context: %#v", ctx.Task)
	}
	if ctx.SystemPrompt == "" {
		t.Fatalf("expected system prompt")
	}
	if len(ctx.Dependencies) != 1 || ctx.Dependencies[0].ID != "dep" {
		t.Fatalf("unexpected deps: %#v", ctx.Dependencies)
	}
	if ctx.ProjectSnapshot != "snapshot" {
		t.Fatalf("unexpected snapshot: %q", ctx.ProjectSnapshot)
	}
}

func TestBuildContextErrorsOnUnknownTask(t *testing.T) {
	g := plan.WorkGraph{SchemaVersion: plan.SchemaVersion, Items: map[string]plan.WorkItem{}}
	_, err := BuildContext(g, "missing")
	if err == nil {
		t.Fatalf("expected error for unknown task")
	}
}

func TestMergeParentReviewFeedbackContextAddsFeedbackWithoutMutatingBase(t *testing.T) {
	base := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		Task: TaskContext{
			ID:                 "child-a",
			Title:              "Child A",
			AcceptanceCriteria: []string{"child criterion"},
		},
		Dependencies: []DependencyContext{{
			ID:        "dep-1",
			Title:     "Dependency",
			Status:    "done",
			Artifacts: []string{"dep/output.txt"},
		}},
		ParentReview: &ParentReviewContext{
			ParentTaskID:       "parent-a",
			AcceptanceCriteria: []string{"parent criterion"},
			Children: []ParentReviewChildContext{{
				ChildID:          "child-a",
				LatestRunID:      "run-child-a-1",
				LatestRunSummary: "summary",
				ArtifactRefs:     []string{"runs/child-a/run-child-a-1.json"},
			}},
		},
		Questions: []agent.Question{{
			ID:      "q1",
			Prompt:  "Choose",
			Options: []string{"a", "b"},
		}},
		Answers: []agent.Answer{{
			ID:    "q1",
			Value: "a",
		}},
	}

	merged, err := MergeParentReviewFeedbackContext(base, ParentReviewFeedbackContext{
		ParentTaskID: "  parent-review-1 ",
		ReviewRunID:  " review-run-9 ",
		Feedback:     "  fix validation and retry  ",
	})
	if err != nil {
		t.Fatalf("MergeParentReviewFeedbackContext: %v", err)
	}
	if merged.ParentReviewFeedback == nil {
		t.Fatalf("expected parent review feedback context")
	}
	if merged.ParentReviewFeedback.ParentTaskID != "parent-review-1" {
		t.Fatalf("ParentTaskID = %q, want %q", merged.ParentReviewFeedback.ParentTaskID, "parent-review-1")
	}
	if merged.ParentReviewFeedback.ReviewRunID != "review-run-9" {
		t.Fatalf("ReviewRunID = %q, want %q", merged.ParentReviewFeedback.ReviewRunID, "review-run-9")
	}
	if merged.ParentReviewFeedback.Feedback != "fix validation and retry" {
		t.Fatalf("Feedback = %q, want %q", merged.ParentReviewFeedback.Feedback, "fix validation and retry")
	}
	if base.ParentReviewFeedback != nil {
		t.Fatalf("base context should remain unchanged, got %#v", base.ParentReviewFeedback)
	}

	merged.Task.AcceptanceCriteria[0] = "changed child criterion"
	merged.Dependencies[0].Artifacts[0] = "changed-artifact"
	merged.ParentReview.AcceptanceCriteria[0] = "changed parent criterion"
	merged.ParentReview.Children[0].ArtifactRefs[0] = "changed-ref"
	merged.Questions[0].Options[0] = "changed-option"
	merged.Answers[0].Value = "changed-answer"

	if base.Task.AcceptanceCriteria[0] != "child criterion" {
		t.Fatalf("base task acceptance criteria mutated: %#v", base.Task.AcceptanceCriteria)
	}
	if base.Dependencies[0].Artifacts[0] != "dep/output.txt" {
		t.Fatalf("base dependency artifacts mutated: %#v", base.Dependencies[0].Artifacts)
	}
	if base.ParentReview.AcceptanceCriteria[0] != "parent criterion" {
		t.Fatalf("base parent review criteria mutated: %#v", base.ParentReview.AcceptanceCriteria)
	}
	if base.ParentReview.Children[0].ArtifactRefs[0] != "runs/child-a/run-child-a-1.json" {
		t.Fatalf("base parent review child refs mutated: %#v", base.ParentReview.Children[0].ArtifactRefs)
	}
	if base.Questions[0].Options[0] != "a" {
		t.Fatalf("base questions mutated: %#v", base.Questions[0].Options)
	}
	if base.Answers[0].Value != "a" {
		t.Fatalf("base answers mutated: %#v", base.Answers[0])
	}
}

func TestMergePendingParentReviewFeedbackContext(t *testing.T) {
	base := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		Task: TaskContext{
			ID:    "child-a",
			Title: "Child A",
		},
	}

	merged, err := MergePendingParentReviewFeedbackContext(base, PendingParentReviewFeedback{
		ParentTaskID: "parent-a",
		ReviewRunID:  "review-run-1",
		Feedback:     "Address API timeout handling.",
	})
	if err != nil {
		t.Fatalf("MergePendingParentReviewFeedbackContext: %v", err)
	}

	if merged.ParentReviewFeedback == nil {
		t.Fatalf("expected parent review feedback context")
	}
	if merged.ParentReviewFeedback.ParentTaskID != "parent-a" {
		t.Fatalf("ParentTaskID = %q, want %q", merged.ParentReviewFeedback.ParentTaskID, "parent-a")
	}
	if merged.ParentReviewFeedback.ReviewRunID != "review-run-1" {
		t.Fatalf("ReviewRunID = %q, want %q", merged.ParentReviewFeedback.ReviewRunID, "review-run-1")
	}
	if merged.ParentReviewFeedback.Feedback != "Address API timeout handling." {
		t.Fatalf("Feedback = %q", merged.ParentReviewFeedback.Feedback)
	}
}

func TestMergeParentReviewFeedbackContextValidation(t *testing.T) {
	base := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		Task: TaskContext{
			ID:    "child-a",
			Title: "Child A",
		},
	}

	cases := []struct {
		name     string
		feedback ParentReviewFeedbackContext
		wantErr  string
	}{
		{
			name: "missing parent task id",
			feedback: ParentReviewFeedbackContext{
				ReviewRunID: "review-1",
				Feedback:    "fix this",
			},
			wantErr: "parent task id required",
		},
		{
			name: "missing review run id",
			feedback: ParentReviewFeedbackContext{
				ParentTaskID: "parent-1",
				Feedback:     "fix this",
			},
			wantErr: "review run id required",
		},
		{
			name: "missing feedback",
			feedback: ParentReviewFeedbackContext{
				ParentTaskID: "parent-1",
				ReviewRunID:  "review-1",
			},
			wantErr: "feedback required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := MergeParentReviewFeedbackContext(base, tc.feedback)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestBuildContextErrorsOnUnknownDep(t *testing.T) {
	now := time.Now().UTC()
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": {
				ID:        "task",
				Title:     "Task",
				Deps:      []string{"missing"},
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	_, err := BuildContext(g, "task")
	if err == nil {
		t.Fatalf("expected error for unknown dependency")
	}
}

package execution

import (
	"bytes"
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
)

func TestRunParentReviewPersistsPassedOutcomeAndKeepsPendingFeedbackUnchanged(t *testing.T) {
	baseDir := t.TempDir()
	planPath := filepath.Join(baseDir, "blackbird.plan.json")
	now := time.Date(2026, 2, 9, 16, 0, 0, 0, time.UTC)
	parentID := "parent-review"

	g := parentReviewContextTestGraph(
		now,
		parentID,
		"Parent Review Task",
		[]string{
			"Child outputs satisfy parent acceptance criteria.",
			"Major issues are called out with actionable feedback.",
		},
		[]string{"child-a", "child-b"},
	)

	fixtures := []RunRecord{
		parentReviewContextTestRun("child-a", "run-a-1", now.Add(1*time.Minute), "child-a summary", []string{"a.go"}),
		parentReviewContextTestRun("child-b", "run-b-1", now.Add(2*time.Minute), "child-b summary", []string{"b.go"}),
	}
	for _, fixture := range fixtures {
		if err := SaveRun(baseDir, fixture); err != nil {
			t.Fatalf("SaveRun(%s): %v", fixture.ID, err)
		}
	}
	if _, err := upsertPendingParentReviewFeedback(
		baseDir,
		"child-a",
		"old-parent",
		"old-review",
		"existing pending feedback",
		now.Add(-5*time.Minute),
	); err != nil {
		t.Fatalf("upsertPendingParentReviewFeedback(child-a): %v", err)
	}
	beforeChildA, err := LoadPendingParentReviewFeedback(baseDir, "child-a")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(child-a) before run: %v", err)
	}
	beforeChildB, err := LoadPendingParentReviewFeedback(baseDir, "child-b")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(child-b) before run: %v", err)
	}

	var streamed bytes.Buffer
	record, err := RunParentReview(context.Background(), ParentReviewRunConfig{
		PlanPath:            planPath,
		Graph:               g,
		ParentTaskID:        parentID,
		CompletionSignature: "children:child-a,child-b",
		Runtime: agent.Runtime{
			Provider: "codex",
			Command:  "printf 'review complete\\n{\"passed\":true}\\n'",
			UseShell: true,
			Timeout:  2 * time.Second,
		},
		StreamStdout: &streamed,
	})
	if err != nil {
		t.Fatalf("RunParentReview: %v", err)
	}
	if record.Type != RunTypeReview {
		t.Fatalf("record.Type = %q, want %q", record.Type, RunTypeReview)
	}
	if record.Status != RunStatusSuccess {
		t.Fatalf("record.Status = %q, want %q", record.Status, RunStatusSuccess)
	}
	if record.Provider != "codex" {
		t.Fatalf("record.Provider = %q, want %q", record.Provider, "codex")
	}
	if record.ProviderSessionRef == "" || record.ProviderSessionRef != record.ID {
		t.Fatalf("record.ProviderSessionRef = %q, want run id %q", record.ProviderSessionRef, record.ID)
	}
	if record.ParentReviewCompletionSignature != "children:child-a,child-b" {
		t.Fatalf(
			"record.ParentReviewCompletionSignature = %q, want %q",
			record.ParentReviewCompletionSignature,
			"children:child-a,child-b",
		)
	}
	if record.ParentReviewPassed == nil || !*record.ParentReviewPassed {
		t.Fatalf("record.ParentReviewPassed = %#v, want true", record.ParentReviewPassed)
	}
	if len(record.ParentReviewResumeTaskIDs) != 0 {
		t.Fatalf("record.ParentReviewResumeTaskIDs = %#v, want empty", record.ParentReviewResumeTaskIDs)
	}
	if record.ParentReviewFeedback != "" {
		t.Fatalf("record.ParentReviewFeedback = %q, want empty", record.ParentReviewFeedback)
	}
	if len(record.ParentReviewResults) != 2 {
		t.Fatalf("len(record.ParentReviewResults) = %d, want 2", len(record.ParentReviewResults))
	}
	for _, childID := range []string{"child-a", "child-b"} {
		result, ok := record.ParentReviewResults[childID]
		if !ok {
			t.Fatalf("missing record.ParentReviewResults[%s]", childID)
		}
		if result.Status != ParentReviewTaskStatusPassed {
			t.Fatalf("record.ParentReviewResults[%s].Status = %q, want %q", childID, result.Status, ParentReviewTaskStatusPassed)
		}
		if result.Feedback != "" {
			t.Fatalf("record.ParentReviewResults[%s].Feedback = %q, want empty", childID, result.Feedback)
		}
	}
	if record.Context.ParentReview == nil {
		t.Fatalf("expected parent review context payload")
	}
	if !strings.Contains(record.Context.SystemPrompt, "Do not implement code changes.") {
		t.Fatalf("system prompt missing no-implementation constraint: %q", record.Context.SystemPrompt)
	}
	if !strings.Contains(record.Context.ParentReview.ReviewerInstructions, "Act as a reviewer only.") {
		t.Fatalf("reviewer instructions missing reviewer-only constraint: %q", record.Context.ParentReview.ReviewerInstructions)
	}
	if record.Stdout == "" {
		t.Fatalf("expected captured stdout")
	}
	if streamed.String() == "" {
		t.Fatalf("expected streamed stdout")
	}

	loaded, err := LoadRun(baseDir, parentID, record.ID)
	if err != nil {
		t.Fatalf("LoadRun: %v", err)
	}
	if loaded.Type != RunTypeReview {
		t.Fatalf("loaded.Type = %q, want %q", loaded.Type, RunTypeReview)
	}
	if loaded.Provider != "codex" {
		t.Fatalf("loaded.Provider = %q, want %q", loaded.Provider, "codex")
	}
	if loaded.ProviderSessionRef != record.ProviderSessionRef {
		t.Fatalf("loaded.ProviderSessionRef = %q, want %q", loaded.ProviderSessionRef, record.ProviderSessionRef)
	}
	if loaded.ParentReviewCompletionSignature != "children:child-a,child-b" {
		t.Fatalf(
			"loaded.ParentReviewCompletionSignature = %q, want %q",
			loaded.ParentReviewCompletionSignature,
			"children:child-a,child-b",
		)
	}
	if loaded.ParentReviewPassed == nil || !*loaded.ParentReviewPassed {
		t.Fatalf("loaded.ParentReviewPassed = %#v, want true", loaded.ParentReviewPassed)
	}
	if len(loaded.ParentReviewResumeTaskIDs) != 0 {
		t.Fatalf("loaded.ParentReviewResumeTaskIDs = %#v, want empty", loaded.ParentReviewResumeTaskIDs)
	}
	if loaded.ParentReviewFeedback != "" {
		t.Fatalf("loaded.ParentReviewFeedback = %q, want empty", loaded.ParentReviewFeedback)
	}
	if len(loaded.ParentReviewResults) != 2 {
		t.Fatalf("len(loaded.ParentReviewResults) = %d, want 2", len(loaded.ParentReviewResults))
	}
	for _, childID := range []string{"child-a", "child-b"} {
		result, ok := loaded.ParentReviewResults[childID]
		if !ok {
			t.Fatalf("missing loaded.ParentReviewResults[%s]", childID)
		}
		if result.Status != ParentReviewTaskStatusPassed {
			t.Fatalf("loaded.ParentReviewResults[%s].Status = %q, want %q", childID, result.Status, ParentReviewTaskStatusPassed)
		}
		if result.Feedback != "" {
			t.Fatalf("loaded.ParentReviewResults[%s].Feedback = %q, want empty", childID, result.Feedback)
		}
	}
	if loaded.Stdout != record.Stdout {
		t.Fatalf("loaded.Stdout mismatch")
	}

	afterChildA, err := LoadPendingParentReviewFeedback(baseDir, "child-a")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(child-a) after run: %v", err)
	}
	afterChildB, err := LoadPendingParentReviewFeedback(baseDir, "child-b")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(child-b) after run: %v", err)
	}
	if !reflect.DeepEqual(afterChildA, beforeChildA) {
		t.Fatalf("child-a pending feedback changed on passed review: before=%#v after=%#v", beforeChildA, afterChildA)
	}
	if !reflect.DeepEqual(afterChildB, beforeChildB) {
		t.Fatalf("child-b pending feedback changed on passed review: before=%#v after=%#v", beforeChildB, afterChildB)
	}
}

func TestRunParentReviewPersistsFailedOutcomeAndLinksResumeTargets(t *testing.T) {
	baseDir := t.TempDir()
	planPath := filepath.Join(baseDir, "blackbird.plan.json")
	now := time.Date(2026, 2, 9, 16, 30, 0, 0, time.UTC)
	parentID := "parent-review-fail"

	g := parentReviewContextTestGraph(
		now,
		parentID,
		"Parent Review Task",
		[]string{
			"Child outputs satisfy parent acceptance criteria.",
			"Major issues are called out with actionable feedback.",
		},
		[]string{"child-a", "child-b", "child-c"},
	)

	fixtures := []RunRecord{
		parentReviewContextTestRun("child-a", "run-a-1", now.Add(1*time.Minute), "child-a summary", []string{"a.go"}),
		parentReviewContextTestRun("child-b", "run-b-1", now.Add(2*time.Minute), "child-b summary", []string{"b.go"}),
		parentReviewContextTestRun("child-c", "run-c-1", now.Add(3*time.Minute), "child-c summary", []string{"c.go"}),
	}
	for _, fixture := range fixtures {
		if err := SaveRun(baseDir, fixture); err != nil {
			t.Fatalf("SaveRun(%s): %v", fixture.ID, err)
		}
	}

	record, err := RunParentReview(context.Background(), ParentReviewRunConfig{
		PlanPath:            planPath,
		Graph:               g,
		ParentTaskID:        parentID,
		CompletionSignature: "children:child-a,child-b,child-c",
		Runtime: agent.Runtime{
			Provider: "codex",
			Command:  "printf 'review complete\\n{\"passed\":false,\"resumeTaskIds\":[\" child-b \",\"child-a\"],\"feedbackForResume\":\"  Child outputs miss required validation paths.  \"}\\n'",
			UseShell: true,
			Timeout:  2 * time.Second,
		},
	})
	if err != nil {
		t.Fatalf("RunParentReview: %v", err)
	}
	if record.Type != RunTypeReview {
		t.Fatalf("record.Type = %q, want %q", record.Type, RunTypeReview)
	}
	if record.Status != RunStatusSuccess {
		t.Fatalf("record.Status = %q, want %q", record.Status, RunStatusSuccess)
	}
	if record.ParentReviewPassed == nil || *record.ParentReviewPassed {
		t.Fatalf("record.ParentReviewPassed = %#v, want false", record.ParentReviewPassed)
	}
	if !reflect.DeepEqual(record.ParentReviewResumeTaskIDs, []string{"child-a", "child-b"}) {
		t.Fatalf("record.ParentReviewResumeTaskIDs = %#v, want [child-a child-b]", record.ParentReviewResumeTaskIDs)
	}
	if record.ParentReviewFeedback != "Child outputs miss required validation paths." {
		t.Fatalf("record.ParentReviewFeedback = %q", record.ParentReviewFeedback)
	}
	if len(record.ParentReviewResults) != 3 {
		t.Fatalf("len(record.ParentReviewResults) = %d, want 3", len(record.ParentReviewResults))
	}
	for _, childID := range []string{"child-a", "child-b"} {
		result, ok := record.ParentReviewResults[childID]
		if !ok {
			t.Fatalf("missing record.ParentReviewResults[%s]", childID)
		}
		if result.Status != ParentReviewTaskStatusFailed {
			t.Fatalf("record.ParentReviewResults[%s].Status = %q, want %q", childID, result.Status, ParentReviewTaskStatusFailed)
		}
		if result.Feedback != "Child outputs miss required validation paths." {
			t.Fatalf("record.ParentReviewResults[%s].Feedback = %q", childID, result.Feedback)
		}
	}
	if childC := record.ParentReviewResults["child-c"]; childC.Status != ParentReviewTaskStatusPassed || childC.Feedback != "" {
		t.Fatalf("record.ParentReviewResults[child-c] = %#v, want passed with empty feedback", childC)
	}
	if record.ParentReviewCompletionSignature != "children:child-a,child-b,child-c" {
		t.Fatalf(
			"record.ParentReviewCompletionSignature = %q, want %q",
			record.ParentReviewCompletionSignature,
			"children:child-a,child-b,child-c",
		)
	}

	loaded, loadErr := LoadRun(baseDir, parentID, record.ID)
	if loadErr != nil {
		t.Fatalf("LoadRun: %v", loadErr)
	}
	if loaded.ParentReviewPassed == nil || *loaded.ParentReviewPassed {
		t.Fatalf("loaded.ParentReviewPassed = %#v, want false", loaded.ParentReviewPassed)
	}
	if !reflect.DeepEqual(loaded.ParentReviewResumeTaskIDs, []string{"child-a", "child-b"}) {
		t.Fatalf("loaded.ParentReviewResumeTaskIDs = %#v, want [child-a child-b]", loaded.ParentReviewResumeTaskIDs)
	}
	if loaded.ParentReviewFeedback != "Child outputs miss required validation paths." {
		t.Fatalf("loaded.ParentReviewFeedback = %q", loaded.ParentReviewFeedback)
	}
	if len(loaded.ParentReviewResults) != 3 {
		t.Fatalf("len(loaded.ParentReviewResults) = %d, want 3", len(loaded.ParentReviewResults))
	}
	for _, childID := range []string{"child-a", "child-b"} {
		result, ok := loaded.ParentReviewResults[childID]
		if !ok {
			t.Fatalf("missing loaded.ParentReviewResults[%s]", childID)
		}
		if result.Status != ParentReviewTaskStatusFailed {
			t.Fatalf("loaded.ParentReviewResults[%s].Status = %q, want %q", childID, result.Status, ParentReviewTaskStatusFailed)
		}
		if result.Feedback != "Child outputs miss required validation paths." {
			t.Fatalf("loaded.ParentReviewResults[%s].Feedback = %q", childID, result.Feedback)
		}
	}
	if childC := loaded.ParentReviewResults["child-c"]; childC.Status != ParentReviewTaskStatusPassed || childC.Feedback != "" {
		t.Fatalf("loaded.ParentReviewResults[child-c] = %#v, want passed with empty feedback", childC)
	}

	for _, childID := range []string{"child-a", "child-b"} {
		pending, err := LoadPendingParentReviewFeedback(baseDir, childID)
		if err != nil {
			t.Fatalf("LoadPendingParentReviewFeedback(%s): %v", childID, err)
		}
		if pending == nil {
			t.Fatalf("expected pending feedback for %s", childID)
		}
		if pending.ParentTaskID != parentID {
			t.Fatalf("pending.ParentTaskID (%s) = %q, want %q", childID, pending.ParentTaskID, parentID)
		}
		if pending.ReviewRunID != record.ID {
			t.Fatalf("pending.ReviewRunID (%s) = %q, want %q", childID, pending.ReviewRunID, record.ID)
		}
		if pending.Feedback != "Child outputs miss required validation paths." {
			t.Fatalf("pending.Feedback (%s) = %q", childID, pending.Feedback)
		}
		if _, err := LoadRun(baseDir, parentID, pending.ReviewRunID); err != nil {
			t.Fatalf("pending feedback for %s points to missing run record %q: %v", childID, pending.ReviewRunID, err)
		}
	}

	pendingChildC, err := LoadPendingParentReviewFeedback(baseDir, "child-c")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(child-c): %v", err)
	}
	if pendingChildC != nil {
		t.Fatalf("child-c should not receive pending feedback, got %#v", pendingChildC)
	}
}

func TestRunParentReviewPersistsTaskSpecificFeedbackPerResumeTarget(t *testing.T) {
	baseDir := t.TempDir()
	planPath := filepath.Join(baseDir, "blackbird.plan.json")
	now := time.Date(2026, 2, 9, 16, 45, 0, 0, time.UTC)
	parentID := "parent-review-targeted-feedback"

	g := parentReviewContextTestGraph(
		now,
		parentID,
		"Parent Review Task",
		[]string{
			"Child outputs satisfy parent acceptance criteria.",
		},
		[]string{"child-a", "child-b"},
	)

	fixtures := []RunRecord{
		parentReviewContextTestRun("child-a", "run-a-1", now.Add(1*time.Minute), "child-a summary", []string{"a.go"}),
		parentReviewContextTestRun("child-b", "run-b-1", now.Add(2*time.Minute), "child-b summary", []string{"b.go"}),
	}
	for _, fixture := range fixtures {
		if err := SaveRun(baseDir, fixture); err != nil {
			t.Fatalf("SaveRun(%s): %v", fixture.ID, err)
		}
	}

	record, err := RunParentReview(context.Background(), ParentReviewRunConfig{
		PlanPath:            planPath,
		Graph:               g,
		ParentTaskID:        parentID,
		CompletionSignature: "children:child-a,child-b",
		Runtime: agent.Runtime{
			Provider: "codex",
			Command:  `printf '{"passed":false,"resumeTaskIds":["child-a","child-b"],"feedbackForResume":"global fallback","reviewResults":[{"taskId":"child-a","status":"failed","feedback":"fix child-a"},{"taskId":"child-b","status":"failed","feedback":"fix child-b"}]}'`,
			UseShell: true,
			Timeout:  2 * time.Second,
		},
	})
	if err != nil {
		t.Fatalf("RunParentReview: %v", err)
	}

	if got := record.ParentReviewResults["child-a"]; got.Status != ParentReviewTaskStatusFailed || got.Feedback != "fix child-a" {
		t.Fatalf("record.ParentReviewResults[child-a] = %#v, want failed with task-specific feedback", got)
	}
	if got := record.ParentReviewResults["child-b"]; got.Status != ParentReviewTaskStatusFailed || got.Feedback != "fix child-b" {
		t.Fatalf("record.ParentReviewResults[child-b] = %#v, want failed with task-specific feedback", got)
	}

	pendingA, err := LoadPendingParentReviewFeedback(baseDir, "child-a")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(child-a): %v", err)
	}
	if pendingA == nil || pendingA.Feedback != "fix child-a" {
		t.Fatalf("pending child-a feedback = %#v, want %q", pendingA, "fix child-a")
	}

	pendingB, err := LoadPendingParentReviewFeedback(baseDir, "child-b")
	if err != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(child-b): %v", err)
	}
	if pendingB == nil || pendingB.Feedback != "fix child-b" {
		t.Fatalf("pending child-b feedback = %#v, want %q", pendingB, "fix child-b")
	}
}

func TestRunParentReviewPersistsFailedReviewRun(t *testing.T) {
	baseDir := t.TempDir()
	planPath := filepath.Join(baseDir, "blackbird.plan.json")
	now := time.Date(2026, 2, 9, 17, 0, 0, 0, time.UTC)
	parentID := "parent-failed-review"

	g := parentReviewContextTestGraph(
		now,
		parentID,
		"Parent Failed Review Task",
		[]string{"Parent review failures are persisted with diagnostics."},
		[]string{"child-a"},
	)
	if err := SaveRun(baseDir, parentReviewContextTestRun("child-a", "run-a-1", now.Add(1*time.Minute), "child-a summary", []string{"a.go"})); err != nil {
		t.Fatalf("SaveRun(run-a-1): %v", err)
	}

	record, err := RunParentReview(context.Background(), ParentReviewRunConfig{
		PlanPath:            planPath,
		Graph:               g,
		ParentTaskID:        parentID,
		CompletionSignature: "children:child-a",
		Runtime: agent.Runtime{
			Provider: "codex",
			Command:  "printf 'review-stdout'; printf 'review-stderr' >&2; exit 7",
			UseShell: true,
			Timeout:  2 * time.Second,
		},
	})
	if err == nil {
		t.Fatalf("expected RunParentReview error for failing command")
	}
	if record.Type != RunTypeReview {
		t.Fatalf("record.Type = %q, want %q", record.Type, RunTypeReview)
	}
	if record.Status != RunStatusFailed {
		t.Fatalf("record.Status = %q, want %q", record.Status, RunStatusFailed)
	}
	if record.ExitCode == nil || *record.ExitCode != 7 {
		t.Fatalf("record.ExitCode = %#v, want 7", record.ExitCode)
	}
	if !strings.Contains(record.Stdout, "review-stdout") {
		t.Fatalf("record.Stdout missing expected output: %q", record.Stdout)
	}
	if !strings.Contains(record.Stderr, "review-stderr") {
		t.Fatalf("record.Stderr missing expected output: %q", record.Stderr)
	}

	loaded, loadErr := LoadRun(baseDir, parentID, record.ID)
	if loadErr != nil {
		t.Fatalf("LoadRun: %v", loadErr)
	}
	if loaded.Type != RunTypeReview {
		t.Fatalf("loaded.Type = %q, want %q", loaded.Type, RunTypeReview)
	}
	if loaded.Status != RunStatusFailed {
		t.Fatalf("loaded.Status = %q, want %q", loaded.Status, RunStatusFailed)
	}
	if loaded.ExitCode == nil || *loaded.ExitCode != 7 {
		t.Fatalf("loaded.ExitCode = %#v, want 7", loaded.ExitCode)
	}
	if loaded.ParentReviewCompletionSignature != "children:child-a" {
		t.Fatalf(
			"loaded.ParentReviewCompletionSignature = %q, want %q",
			loaded.ParentReviewCompletionSignature,
			"children:child-a",
		)
	}
	pendingChildA, pendingErr := LoadPendingParentReviewFeedback(baseDir, "child-a")
	if pendingErr != nil {
		t.Fatalf("LoadPendingParentReviewFeedback(child-a): %v", pendingErr)
	}
	if pendingChildA != nil {
		t.Fatalf("child-a should not receive pending feedback when review run fails to execute, got %#v", pendingChildA)
	}
}
